//nolint:funlen,noctx // ok for testcode here
package authn

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	connect "connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/srlmgr/backend/log"
)

//nolint:funlen // table-driven auth scenarios are intentionally compact
func TestManagerAuthenticate(t *testing.T) {
	t.Parallel()

	principal := Principal{
		ID:            "principal-1",
		Tenant:        "tenant-1",
		Roles:         []string{"admin"},
		SimulationIDs: []string{"sim-1"},
		SeriesIDs:     []string{"series-1"},
		Source:        "api-token",
	}

	tests := []struct {
		name      string
		headers   map[string]string
		manager   *manager
		expectErr string
		expect    *Principal
	}{
		{
			name:    "authorization header unsupported",
			headers: map[string]string{"Authorization": "Bearer abc"},
			manager: &manager{
				logger:     log.New(),
				cfg:        &Config{Enabled: true},
				cookieName: "backend_session",
				sessions:   newInMemorySessionStore(),
			},
			expectErr: "authorization header is not supported",
		},
		{
			name: "missing credentials",
			manager: &manager{
				logger:     log.New(),
				cfg:        &Config{Enabled: true},
				cookieName: "backend_session",
				sessions:   newInMemorySessionStore(),
			},
			expectErr: "missing authentication credentials",
		},
		{
			name:    "api token authenticated",
			headers: map[string]string{apiTokenHeader: "valid-token"},
			manager: &manager{
				logger: log.New(),
				cfg:    &Config{Enabled: true},
				apiToken: &apiTokenStore{
					tokens: map[string]Principal{"valid-token": principal},
				},
				cookieName: "backend_session",
				sessions:   newInMemorySessionStore(),
			},
			expect: &principal,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := connect.NewRequest(&emptypb.Empty{})
			for k, v := range tc.headers {
				req.Header().Set(k, v)
			}

			gotCtx, err := tc.manager.authenticate(context.Background(), req)
			if tc.expectErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.expectErr)
				}
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected error containing %q, got %q", tc.expectErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("authenticate() error = %v", err)
			}

			if tc.expect == nil {
				t.Fatal("missing expected principal")
			}
			gotPrincipal, ok := PrincipalFromContext(gotCtx)
			if !ok {
				t.Fatal("expected principal in context")
			}
			if !reflect.DeepEqual(gotPrincipal, *tc.expect) {
				t.Fatalf("principal = %#v, want %#v", gotPrincipal, *tc.expect)
			}
		})
	}
}

//nolint:funlen // keeps two key interceptor behaviors explicit
func TestNewInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("disabled passthrough", func(t *testing.T) {
		t.Parallel()

		interceptor, err := NewInterceptor(
			context.Background(),
			&Config{Enabled: false},
			log.New(),
		)
		if err != nil {
			t.Fatalf("NewInterceptor() error = %v", err)
		}

		called := false
		wrapped := interceptor.WrapUnary(
			func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				called = true
				return connect.NewResponse(&emptypb.Empty{}), nil
			},
		)

		_, err = wrapped(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err != nil {
			t.Fatalf("wrapped unary error = %v", err)
		}
		if !called {
			t.Fatal("expected next unary function to be called")
		}
	})

	t.Run("missing credentials unauthenticated", func(t *testing.T) {
		t.Parallel()

		interceptor, err := NewInterceptor(
			context.Background(),
			&Config{Enabled: true},
			log.New(),
		)
		if err != nil {
			t.Fatalf("NewInterceptor() error = %v", err)
		}

		wrapped := interceptor.WrapUnary(
			func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				return connect.NewResponse(&emptypb.Empty{}), nil
			},
		)

		_, err = wrapped(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err == nil {
			t.Fatal("expected unauthenticated error")
		}

		var connectErr *connect.Error
		if !errors.As(err, &connectErr) {
			t.Fatalf("expected connect.Error, got %T", err)
		}
		if connectErr.Code() != connect.CodeUnauthenticated {
			t.Fatalf("error code = %v, want %v", connectErr.Code(), connect.CodeUnauthenticated)
		}
	})
}

func TestCurrentPrincipalFromRequest(t *testing.T) {
	t.Parallel()

	sessions := newInMemorySessionStore()
	const sessionID = "session-1"

	principal := Principal{ID: "user-1", Name: "Ada Lovelace"}
	if err := sessions.Put(context.Background(), Session{
		ID:        sessionID,
		Principal: principal,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("put session: %v", err)
	}

	m := &manager{
		cfg:        &Config{IDP: IDPConfig{CookieName: "backend_session"}},
		cookieName: "backend_session",
		sessions:   sessions,
		logger:     log.New(),
	}

	t.Run("cookie maps to principal", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/currentuser", http.NoBody)
		req.AddCookie(m.sessionCookie(sessionID))

		got, found, err := m.CurrentPrincipalFromRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("CurrentPrincipalFromRequest() error = %v", err)
		}
		if !found {
			t.Fatal("CurrentPrincipalFromRequest() found = false, want true")
		}
		if !reflect.DeepEqual(got, principal) {
			t.Fatalf("principal = %#v, want %#v", got, principal)
		}
	})

	t.Run("missing cookie returns not found", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/currentuser", http.NoBody)

		_, found, err := m.CurrentPrincipalFromRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("CurrentPrincipalFromRequest() error = %v", err)
		}
		if found {
			t.Fatal("CurrentPrincipalFromRequest() found = true, want false")
		}
	})

	t.Run("unknown session returns not found", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/currentuser", http.NoBody)
		req.AddCookie(m.sessionCookie("missing-session"))

		_, found, err := m.CurrentPrincipalFromRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("CurrentPrincipalFromRequest() error = %v", err)
		}
		if found {
			t.Fatal("CurrentPrincipalFromRequest() found = true, want false")
		}
	})
}

func TestRegisterHTTPHandlersCurrentUser(t *testing.T) {
	t.Parallel()

	sessions := newInMemorySessionStore()
	const sessionID = "session-2"
	principal := Principal{ID: "user-2", Name: "Grace Hopper"}
	if err := sessions.Put(context.Background(), Session{
		ID:        sessionID,
		Principal: principal,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("put session: %v", err)
	}

	m := &manager{
		cfg:        &Config{IDP: IDPConfig{CookieName: "backend_session"}},
		cookieName: "backend_session",
		sessions:   sessions,
		logger:     log.New(),
	}

	mux := http.NewServeMux()
	m.RegisterHTTPHandlers(mux)

	t.Run("returns no content when session cookie missing", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/currentuser", http.NoBody)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)

		if got := res.Code; got != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", got, http.StatusNoContent)
		}
	})

	t.Run("returns id and name when session exists", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/currentuser", http.NoBody)
		req.AddCookie(m.sessionCookie(sessionID))
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)

		if got := res.Code; got != http.StatusOK {
			t.Fatalf("status = %d, want %d", got, http.StatusOK)
		}

		var body currentUserResponse
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if body.ID != principal.ID || body.Name != principal.Name {
			t.Fatalf("body = %#v, want id=%q name=%q", body, principal.ID, principal.Name)
		}
	})

	t.Run("returns method not allowed for non GET", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/currentuser", http.NoBody)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)

		if got := res.Code; got != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", got, http.StatusMethodNotAllowed)
		}
	})
}
