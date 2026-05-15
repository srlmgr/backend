package authn

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	connect "connectrpc.com/connect"

	"github.com/srlmgr/backend/log"
)

type manager struct {
	cfg        *Config
	idpClient  *oidcClient
	apiToken   *apiTokenStore
	sessions   SessionStore
	states     *loginStateStore
	logger     *log.Logger
	callback   string
	cookieName string
}

type authError struct {
	err         error
	clearCookie bool
}

type currentUserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (e *authError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func NewManager(ctx context.Context, cfg *Config, l *log.Logger) (*manager, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.WithDefaults()

	m := &manager{
		cfg:        cfg,
		logger:     l.Named("authn"),
		states:     newLoginStateStore(),
		cookieName: cfg.IDP.CookieName,
		sessions:   cfg.Store,
	}
	if m.sessions == nil {
		m.sessions = newInMemorySessionStore()
	}

	if cfg.IDP.Enabled {
		client, err := newOIDCClient(ctx, &cfg.IDP)
		if err != nil {
			return nil, err
		}
		m.idpClient = client

		callback, err := callbackPath(cfg.IDP.CallbackURL)
		if err != nil {
			return nil, err
		}
		m.callback = callback
	}

	store, err := newAPITokenStore(ctx, cfg.APIToken, m.logger)
	if err != nil {
		return nil, err
	}
	m.apiToken = store

	return m, nil
}

func callbackPath(callbackURL string) (string, error) {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("parse callback url: %w", err)
	}
	if parsed.Path == "" {
		return "", fmt.Errorf("callback url must include a path")
	}
	if parsed.RawQuery != "" {
		return parsed.Path + "?" + parsed.RawQuery, nil
	}
	return parsed.Path, nil
}

//nolint:whitespace // multiline signature for line-length compliance
func NewInterceptor(
	ctx context.Context,
	cfg *Config,
	l *log.Logger,
) (connect.Interceptor, error) {
	m, err := NewManager(ctx, cfg, l)
	if err != nil {
		return nil, err
	}
	return m.Interceptor(), nil
}

//nolint:whitespace // multiline callback signature for line-length compliance
func (m *manager) Interceptor() connect.Interceptor {
	if !m.cfg.Enabled {
		return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return next
		})
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			principalCtx, err := m.authenticate(ctx, req)
			if err != nil {
				return nil, m.toConnectError(err)
			}
			return next(principalCtx, req)
		}
	})
}

func (m *manager) RegisterHTTPHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/currentuser", m.handleCurrentUser)

	if !m.cfg.IDP.Enabled {
		return
	}

	mux.HandleFunc("/login", m.handleLogin)
	mux.HandleFunc(m.callback, m.handleCallback)
	mux.HandleFunc("/logout", m.handleLogout)
}

func (m *manager) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	principal, found, err := m.CurrentPrincipalFromRequest(r.Context(), r)
	if err != nil {
		m.logger.Warn("resolve current user from session", log.ErrorField(err))
		http.Error(w, "unable to resolve current user", http.StatusInternalServerError)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(currentUserResponse{
		ID:        principal.ID,
		Name:      principal.Name,
		FirstName: principal.FirstName,
		LastName:  principal.LastName,
	}); err != nil {
		m.logger.Warn("encode current user response", log.ErrorField(err))
	}
}

// CurrentPrincipalFromRequest resolves the current principal from session cookie data.
//
//nolint:whitespace // multiline signature for line-length compliance
func (m *manager) CurrentPrincipalFromRequest(
	ctx context.Context,
	r *http.Request,
) (Principal, bool, error) {
	if r == nil {
		return Principal{}, false, nil
	}

	sessionID, err := m.readSessionID(r.Header)
	if err != nil {
		return Principal{}, false, err
	}
	if sessionID == "" {
		return Principal{}, false, nil
	}

	session, found, err := m.sessionFromID(ctx, sessionID)
	if err != nil {
		return Principal{}, false, err
	}
	if !found {
		return Principal{}, false, nil
	}

	return session.Principal, true, nil
}

func (m *manager) toConnectError(err error) error {
	var authErr *authError
	if !errors.As(err, &authErr) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	connectErr := connect.NewError(connect.CodeUnauthenticated, authErr.err)
	if authErr.clearCookie {
		connectErr.Meta().Add("Set-Cookie", m.clearSessionCookie().String())
	}
	return connectErr
}

//nolint:whitespace // auth flow intentionally explicit; editor/linter issue
func (m *manager) authenticate(
	ctx context.Context,
	req connect.AnyRequest,
) (context.Context, error) {
	cred, err := selectCredential(req.Header())
	if err != nil {
		return nil, err
	}

	if cred.source == authSourceAPIToken {
		principal, tokenErr := m.validateAPIToken(cred.token)
		if tokenErr != nil {
			return nil, tokenErr
		}
		return AddPrincipal(ctx, &principal), nil
	}

	if cred.source != authSourceNone {
		return nil, fmt.Errorf("unsupported authentication source")
	}

	sessionID, err := m.readSessionID(req.Header())
	if err != nil {
		return nil, err
	}
	if sessionID == "" {
		if IsAnonymousProcedure(req.Spec().Procedure) {
			return ctx, nil
		}
		return nil, errors.New("missing authentication credentials")
	}

	session, found, err := m.sessionFromID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &authError{err: errors.New("session not found"), clearCookie: true}
	}

	return AddPrincipal(ctx, &session.Principal), nil
}

//nolint:nestif,whitespace // false positive
func (m *manager) sessionFromID(
	ctx context.Context,
	sessionID string,
) (Session, bool, error) {
	session, err := m.sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			return Session{}, false, nil
		}
		return Session{}, false, err
	}

	if m.cfg.IDP.Enabled && shouldRefreshToken(session.Expiry, m.cfg.IDP.RefreshSkew) {
		if refreshErr := m.refreshSession(ctx, &session); refreshErr != nil {
			_ = m.sessions.Delete(ctx, session.ID)
			//nolint:nilerr // session could not be refreshed. nil return ok
			return Session{}, false, nil
		}

		session, err = m.sessions.Get(ctx, sessionID)
		if err != nil {
			if errors.Is(err, errSessionNotFound) {
				return Session{}, false, nil
			}
			return Session{}, false, err
		}
	}

	return session, true, nil
}

func shouldRefreshToken(expiry time.Time, skew time.Duration) bool {
	if expiry.IsZero() {
		return false
	}

	return time.Now().Add(skew).After(expiry)
}

func (m *manager) validateAPIToken(token string) (Principal, error) {
	if m.apiToken == nil {
		return Principal{}, fmt.Errorf("api-token authentication is disabled")
	}
	return m.apiToken.validate(token)
}

func (m *manager) readSessionID(headers http.Header) (string, error) {
	req := &http.Request{Header: headers}
	cookie, err := req.Cookie(m.cookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read session cookie: %w", err)
	}
	return strings.TrimSpace(cookie.Value), nil
}

func (m *manager) refreshSession(ctx context.Context, session *Session) error {
	if m.idpClient == nil {
		return errors.New("session refresh is not configured")
	}
	if strings.TrimSpace(session.RefreshToken) == "" {
		return errors.New("session refresh token is missing")
	}

	bundle, err := m.idpClient.refresh(ctx, session.RefreshToken)
	if err != nil {
		return err
	}

	session.AccessToken = bundle.AccessToken
	session.RefreshToken = bundle.RefreshToken
	session.Expiry = bundle.Expiry
	if bundle.Principal.ID != "" {
		session.Principal = bundle.Principal
	}
	if m.cfg.IDP.SessionTTL > 0 {
		session.ExpiresAt = time.Now().Add(m.cfg.IDP.SessionTTL)
	}

	return m.sessions.Put(ctx, *session)
}

func (m *manager) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "unable to initialize auth flow", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "unable to initialize auth flow", http.StatusInternalServerError)
		return
	}

	m.states.Put(state, loginState{
		Nonce:     nonce,
		ExpiresAt: time.Now().Add(m.cfg.IDP.StateTTL),
	})
	http.Redirect(w, r, m.idpClient.authCodeURL(state, nonce), http.StatusFound)
}

//nolint:funlen // callback flow keeps error handling explicit
func (m *manager) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	stateEntry, err := m.states.Consume(state)
	if err != nil {
		http.Error(w, "invalid state", http.StatusUnauthorized)
		return
	}

	bundle, err := m.idpClient.exchange(r.Context(), code)
	if err != nil {
		m.logger.Warn("oidc code exchange failed", log.ErrorField(err))
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}
	if stateEntry.Nonce != "" && stateEntry.Nonce != bundle.Nonce {
		http.Error(w, "invalid nonce", http.StatusUnauthorized)
		return
	}

	sessionID, err := randomToken(32)
	if err != nil {
		http.Error(w, "unable to create session", http.StatusInternalServerError)
		return
	}

	session := Session{
		ID:           sessionID,
		Principal:    bundle.Principal,
		AccessToken:  bundle.AccessToken,
		RefreshToken: bundle.RefreshToken,
		Expiry:       bundle.Expiry,
	}
	if m.cfg.IDP.SessionTTL > 0 {
		session.ExpiresAt = time.Now().Add(m.cfg.IDP.SessionTTL)
	}
	if err := m.sessions.Put(r.Context(), session); err != nil {
		http.Error(w, "unable to persist session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, m.sessionCookie(session.ID))
	http.Redirect(w, r, m.cfg.IDP.FrontendURL, http.StatusFound)
}

func (m *manager) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := m.readSessionID(r.Header)
	if err != nil {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}
	if sessionID != "" {
		_ = m.sessions.Delete(r.Context(), sessionID)
	}

	http.SetCookie(w, m.clearSessionCookie())
	if strings.TrimSpace(m.cfg.IDP.FrontendURL) != "" {
		http.Redirect(w, r, m.cfg.IDP.FrontendURL, http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *manager) sessionCookie(sessionID string) *http.Cookie {
	cookie := &http.Cookie{
		Name:     m.cookieName,
		Value:    sessionID,
		Path:     "/",
		Secure:   m.cfg.IDP.CookieSecure,
		HttpOnly: m.cfg.IDP.CookieHTTPOnly,
		SameSite: m.cfg.IDP.CookieSameSite,
	}
	if m.cfg.IDP.SessionTTL > 0 {
		cookie.MaxAge = int(m.cfg.IDP.SessionTTL.Seconds())
		cookie.Expires = time.Now().Add(m.cfg.IDP.SessionTTL)
	}
	return cookie
}

func (m *manager) clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		Secure:   m.cfg.IDP.CookieSecure,
		HttpOnly: m.cfg.IDP.CookieHTTPOnly,
		SameSite: m.cfg.IDP.CookieSameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
