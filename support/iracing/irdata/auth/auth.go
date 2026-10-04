package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/srlmgr/backend/log"
)

type (
	//nolint:tagliatelle // external definition
	tokenData struct {
		AccessToken           string `json:"access_token"`
		TokenType             string `json:"token_type"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	}
	//nolint:lll // readability
	AuthConfig struct {
		ClientID           string
		ClientSecret       string
		Username           string
		Password           string
		AuthFile           string
		RefreshGuard       time.Duration // duration before token expiration to attempt refresh
		TokenCheckInterval time.Duration // interval to check token expiration, default 1 minute
	}

	Option             func(*tokenManagerConfig)
	tokenManagerConfig struct {
		authConfig *AuthConfig
	}
	TokenManager struct {
		cfg   tokenManagerConfig
		ctx   context.Context
		token *tokenData
		log   *log.Logger
	}
)

const (
	//nolint:gosec // this is the official iRacing API endpoint
	tokenURL = "https://oauth.iracing.com/oauth2/token"
)

func NewTokenManager(opts ...Option) (*TokenManager, error) {
	tm := tokenManagerConfig{}
	for _, opt := range opts {
		opt(&tm)
	}
	return &TokenManager{
		cfg: tm,
		ctx: context.Background(),
		log: log.Default().Named("irauth"),
	}, nil
}

func WithAuthConfig(authConfig *AuthConfig) Option {
	return func(tm *tokenManagerConfig) {
		tm.authConfig = authConfig
	}
}

//nolint:nestif // quite complex logic
func (tm *TokenManager) Login() error {
	if tm.cfg.authConfig.AuthFile != "" {
		tm.log.Debug("auth file path provided, trying to load auth info from file",
			log.String("auth-file", tm.cfg.authConfig.AuthFile))
		if err := tm.loadAuthInfo(); err != nil {
			tm.log.Debug("failed to load auth info from file, will try to login",
				log.ErrorField(err))
			return tm.doLogin()
		}
		tm.log.Info("successfully loaded auth info from file")
		exp := tm.getExpiresIn(tm.token.AccessToken)
		if exp.After(time.Now()) {
			tm.log.Debug("token is valid")
			tm.setupTokenRefresh()
			return nil
		}
		tm.log.Debug("token is expired, refreshing...")

		if refreshErr := tm.doRefresh(); refreshErr != nil {
			tm.log.Debug("failed to refresh access token, will try to login with credentials",
				log.ErrorField(refreshErr))
			if loginErr := tm.doLogin(); loginErr != nil {
				return loginErr
			}
			tm.setupTokenRefresh()
			return nil
		}
		tm.log.Info("successfully refreshed access token")
		tm.setupTokenRefresh()
		return nil
	}
	if loginErr := tm.doLogin(); loginErr != nil {
		return loginErr
	}
	tm.setupTokenRefresh()
	return nil
}

func (tm *TokenManager) GetAccessToken() (string, error) {
	if tm.token == nil {
		return "", fmt.Errorf("not logged in")
	}
	return tm.token.AccessToken, nil
}

func (tm *TokenManager) getExpiresIn(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		log.Debug("invalid token format")
		return time.Time{}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		log.Debug("failed to decode token payload", log.ErrorField(err))
		return time.Time{}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		tm.log.Debug("failed to unmarshal token claims", log.ErrorField(err))
		return time.Time{}
	}

	if exp, ok := claims["exp"].(float64); ok {
		tm.log.Debug("token expiration", log.Any("exp_time", time.Unix(int64(exp), 0)))
		return time.Unix(int64(exp), 0)
	}
	return time.Time{}
}

func (tm *TokenManager) setupTokenRefresh() {
	go func() {
		for {
			select {
			case <-tm.ctx.Done():
				tm.log.Debug("stopping token refresh goroutine")
				return
			case <-time.After(tm.cfg.authConfig.TokenCheckInterval):
				tm.log.Debug("checking token expiration for refresh")

				if tm.token == nil {
					time.Sleep(time.Second)
					continue
				}

				exp := tm.getExpiresIn(tm.token.AccessToken)
				pre := tm.cfg.authConfig.RefreshGuard
				if pre <= 0 {
					pre = 1 * time.Minute
				}
				if time.Until(exp) <= pre {
					tm.log.Debug(
						"token is close to expiration, refreshing...",
						log.Time("exp", exp),
					)
					if err := tm.doRefresh(); err != nil {
						tm.log.Debug("token refresh failed", log.ErrorField(err))
					}
				}
			}
		}
	}()
}

func (tm *TokenManager) doLogin() error {
	client := &http.Client{}
	data := url.Values{}
	data.Set("username", tm.cfg.authConfig.Username)
	data.Set("password",
		tm.hashSecret(tm.cfg.authConfig.Password, tm.cfg.authConfig.Username))
	data.Set("client_id", tm.cfg.authConfig.ClientID)
	data.Set("client_secret",
		tm.hashSecret(tm.cfg.authConfig.ClientSecret, tm.cfg.authConfig.ClientID))
	data.Set("grant_type", "password_limited")
	data.Set("scope", "iracing.auth")

	req, err := http.NewRequestWithContext(
		tm.ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var token tokenData
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	tm.token = &token

	if tm.cfg.authConfig.AuthFile != "" {
		tm.log.Debug("saving auth info to file",
			log.String("auth-file", tm.cfg.authConfig.AuthFile))
		if err := tm.saveAuthInfo(); err != nil {
			return err
		}
	}
	return nil
}

func (tm *TokenManager) doRefresh() error {
	client := &http.Client{}
	data := url.Values{}
	data.Set("refresh_token", tm.token.RefreshToken)
	data.Set("client_id", tm.cfg.authConfig.ClientID)
	data.Set("client_secret",
		tm.hashSecret(tm.cfg.authConfig.ClientSecret, tm.cfg.authConfig.ClientID))
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(
		tm.ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var token tokenData
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	tm.token = &token

	if tm.cfg.authConfig.AuthFile != "" {
		tm.log.Debug("saving auth info to file",
			log.String("auth-file", tm.cfg.authConfig.AuthFile))
		if err := tm.saveAuthInfo(); err != nil {
			return err
		}
	}
	return nil
}

func (tm *TokenManager) hashSecret(secret, id string) string {
	h := sha256.New()
	h.Write([]byte(secret + strings.ToLower(id)))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (tm *TokenManager) saveAuthInfo() error {
	//nolint:gosec // need secret pattern here
	data, err := json.Marshal(tm.token)
	if err != nil {
		return err
	}
	return os.WriteFile(tm.cfg.authConfig.AuthFile, data, 0o600)
}

func (tm *TokenManager) loadAuthInfo() error {
	data, err := os.ReadFile(tm.cfg.authConfig.AuthFile)
	if err != nil {
		return err
	}
	token := &tokenData{}
	if err := json.Unmarshal(data, token); err != nil {
		return err
	}
	tm.token = token
	return nil
}
