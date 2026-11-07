//nolint:funlen // test cases are explicit
package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestValidatedRedirectURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		frontendURL string
		redirectURL string
		wantValid   bool
	}{
		{
			name:        "same frontend path",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://frontend.example/app",
			wantValid:   true,
		},
		{
			name:        "deep link below frontend path",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://frontend.example/app/orders?id=42#item",
			wantValid:   true,
		},
		{
			name:        "frontend root permits paths",
			frontendURL: "https://frontend.example",
			redirectURL: "https://frontend.example/deep/link",
			wantValid:   true,
		},
		{
			name:        "different host",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://other.example/app",
		},
		{
			name:        "host suffix is not a match",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://frontend.example.evil/app",
		},
		{
			name:        "path prefix boundary is required",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://frontend.example/application",
		},
		{
			name:        "path traversal cannot escape frontend path",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://frontend.example/app/../admin",
		},
		{
			name:        "encoded path traversal cannot escape frontend path",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://frontend.example/app/%2e%2e%5cadmin",
		},
		{
			name:        "different scheme",
			frontendURL: "https://frontend.example/app",
			redirectURL: "http://frontend.example/app",
		},
		{
			name:        "userinfo is not allowed",
			frontendURL: "https://frontend.example/app",
			redirectURL: "https://attacker@frontend.example/app",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotURL, gotValid := validatedRedirectURL(tc.frontendURL, tc.redirectURL)
			if gotValid != tc.wantValid {
				t.Fatalf("valid = %v, want %v", gotValid, tc.wantValid)
			}
			if gotValid && gotURL != tc.redirectURL {
				t.Fatalf("redirect URL = %q, want %q", gotURL, tc.redirectURL)
			}
			if !gotValid && gotURL != "" {
				t.Fatalf("rejected redirect URL = %q, want empty", gotURL)
			}
		})
	}
}

func TestLoginCarriesValidatedRedirectInState(t *testing.T) {
	t.Parallel()

	const (
		frontendURL = "https://frontend.example/app"
		redirectURL = "https://frontend.example/app/orders?id=42#item"
	)
	m := &manager{
		cfg: &Config{
			IDP: IDPConfig{
				FrontendURL: frontendURL,
				StateTTL:    time.Minute,
			},
		},
		idpClient: &oidcClient{
			oauthConfig: &oauth2.Config{
				Endpoint: oauth2.Endpoint{AuthURL: "https://idp.example/authorize"},
			},
		},
		states: newLoginStateStore(),
	}

	query := url.Values{"callback_url": []string{redirectURL}}
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/login?"+query.Encode(), http.NoBody,
	)
	res := httptest.NewRecorder()
	m.handleLogin(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	authURL, err := url.Parse(res.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	state := authURL.Query().Get("state")
	if state == "" {
		t.Fatal("authorization URL has no state")
	}

	stateEntry, err := m.states.Consume(state)
	if err != nil {
		t.Fatalf("consume login state: %v", err)
	}
	if stateEntry.RedirectURL != redirectURL {
		t.Fatalf("state redirect URL = %q, want %q", stateEntry.RedirectURL, redirectURL)
	}

	// No redirect URI remains the signal for the callback to use FrontendURL.
	req = httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/login", http.NoBody,
	)
	res = httptest.NewRecorder()
	m.handleLogin(res, req)
	authURL, err = url.Parse(res.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse default authorization URL: %v", err)
	}
	stateEntry, err = m.states.Consume(authURL.Query().Get("state"))
	if err != nil {
		t.Fatalf("consume default login state: %v", err)
	}
	if stateEntry.RedirectURL != "" {
		t.Fatalf("default state redirect URL = %q, want empty", stateEntry.RedirectURL)
	}
}
