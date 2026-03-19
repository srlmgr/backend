//nolint:funlen,lll // tests with multiple cases
package authn

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	connect "connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/srlmgr/backend/log"
)

func TestAuthenticatorValidateCredential(t *testing.T) {
	t.Parallel()

	principal := Principal{ID: "principal-1", Tenant: "tenant-1", Source: "api-token"}
	a := &authenticator{
		apiToken: &apiTokenStore{tokens: map[string]Principal{"valid-token": principal}},
		logger:   log.New(),
	}

	tests := []struct {
		name          string
		cred          selectedCredential
		expectErr     string
		expectSuccess *Principal
	}{
		{
			name:      "no credentials",
			cred:      selectedCredential{source: authSourceNone},
			expectErr: "no authentication credentials supplied",
		},
		{
			name:      "jwt disabled",
			cred:      selectedCredential{source: authSourceJWT, token: "jwt"},
			expectErr: "jwt authentication is disabled",
		},
		{
			name:          "api token success",
			cred:          selectedCredential{source: authSourceAPIToken, token: "valid-token"},
			expectSuccess: &principal,
		},
		{
			name:      "unknown source",
			cred:      selectedCredential{source: authSource(999), token: "token"},
			expectErr: "no supported authentication source found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := a.validateCredential(context.Background(), tc.cred)
			if tc.expectErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.expectErr)
				}
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected error containing %q, got %q", tc.expectErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("validateCredential() error = %v", err)
			}
			if tc.expectSuccess == nil {
				t.Fatal("test case missing expectSuccess")
			}
			if !reflect.DeepEqual(got, *tc.expectSuccess) {
				t.Fatalf("validateCredential() = %#v, want %#v", got, *tc.expectSuccess)
			}
		})
	}
}

func TestAuthenticatorAuthenticate(t *testing.T) {
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
		name          string
		headers       map[string]string
		authenticator *authenticator
		expectErr     string
		expectAuthn   *Principal
	}{
		{
			name:          "invalid metadata",
			headers:       map[string]string{authorizationHeader: "Basic abc"},
			authenticator: &authenticator{logger: log.New()},
			expectErr:     "authorization header must use bearer scheme",
		},
		{
			name:          "missing credentials",
			authenticator: &authenticator{logger: log.New()},
			expectErr:     "missing authentication credentials",
		},
		{
			name:    "api token authenticated",
			headers: map[string]string{apiTokenHeader: "valid-token"},
			authenticator: &authenticator{
				apiToken: &apiTokenStore{
					tokens: map[string]Principal{"valid-token": principal},
				},
				logger: log.New(),
			},
			expectAuthn: &principal,
		},
		{
			name:    "invalid api token credentials",
			headers: map[string]string{apiTokenHeader: "invalid-token"},
			authenticator: &authenticator{
				apiToken: &apiTokenStore{
					tokens: map[string]Principal{"valid-token": principal},
				},
				logger: log.New(),
			},
			expectErr: "api-token not found",
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

			gotCtx, err := tc.authenticator.authenticate(context.Background(), req)

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

			if tc.expectAuthn == nil {
				t.Fatal("test case missing expectAuthn")
			}

			gotPrincipal, ok := PrincipalFromContext(gotCtx)
			if !ok {
				t.Fatal("expected principal in context")
			}
			if !reflect.DeepEqual(gotPrincipal, *tc.expectAuthn) {
				t.Fatalf("principal = %#v, want %#v", gotPrincipal, *tc.expectAuthn)
			}
		})
	}
}

func TestNewInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("disabled passthrough", func(t *testing.T) {
		t.Parallel()

		interceptor, err := NewInterceptor(context.Background(), Config{Enabled: false}, log.New())
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

		interceptor, err := NewInterceptor(context.Background(), Config{Enabled: true}, log.New())
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
