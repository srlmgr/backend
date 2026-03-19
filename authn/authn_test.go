package authn_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/log"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func testLogger(t *testing.T) *log.Logger {
	t.Helper()
	return log.New(log.WithLogLevel("warn"))
}

func writeTokenFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tokens.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	return path
}

func newAuthenticator(t *testing.T, cfg authn.Config) *authn.Authenticator {
	t.Helper()
	a, err := authn.NewAuthenticator(context.Background(), cfg, testLogger(t))
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}
	return a
}

// authenticate is a convenience wrapper around AuthenticateHeaders.
func authenticate(
	a *authn.Authenticator,
	procedure string,
	header http.Header,
) (*authn.Principal, error) {
	return a.AuthenticateHeaders(context.Background(), procedure, header)
}

const validTokenYAML = `
tokens:
  - token: "secret-token-value"
    id: "user-123"
    tenant: "tenant-a"
    active: true
    roles:
      - "admin"
    scopes:
      simulationIDs:
        - "sim-1"
      seriesIDs:
        - "series-1"
`

// ─── anonymous procedure policy ──────────────────────────────────────────────

func TestIsAnonymousProcedure_QueryReadOps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		procedure string
		want      bool
	}{
		{"/backend.query.v1.QueryService/ListSimulations", true},
		{"/backend.query.v1.QueryService/GetDriverStandings", true},
		{"/backend.query.v1.QueryService/GetTeamStandings", true},
		{"/backend.query.v1.QueryService/GetEventResults", true},
		{"/backend.query.v1.QueryService/GetEventBookingEntries", true},
		{"/backend.query.v1.QueryService/ListCarManufacturers", true},
		{"/backend.admin.v1.AdminService/MarkResultState", false},
		{"/backend.command.v1.CommandService/CreateSimulation", false},
		{"/unknown/Procedure", false},
	}
	for _, tc := range cases {
		got := authn.IsAnonymousProcedure(tc.procedure)
		if got != tc.want {
			t.Errorf("IsAnonymousProcedure(%q) = %v, want %v",
				tc.procedure, got, tc.want)
		}
	}
}

// ─── principal context helpers ──────────────────────────────────────────────

func TestPrincipalRoundTrip(t *testing.T) {
	t.Parallel()
	p := &authn.Principal{
		ID:     "user-1",
		Tenant: "tenant-a",
		Roles:  []string{"admin"},
		Scopes: authn.Scopes{
			SimulationIDs: []string{"sim-1"},
			SeriesIDs:     []string{"series-1"},
		},
	}
	ctx := authn.WithPrincipal(context.Background(), p)
	got := authn.PrincipalFromContext(ctx)
	if got == nil {
		t.Fatal("expected principal, got nil")
	}
	if got.ID != p.ID || got.Tenant != p.Tenant {
		t.Errorf("principal mismatch: got %+v, want %+v", got, p)
	}
}

func TestPrincipalFromContext_Nil(t *testing.T) {
	t.Parallel()
	if p := authn.PrincipalFromContext(context.Background()); p != nil {
		t.Errorf("expected nil principal, got %+v", p)
	}
}

// ─── auth source selection ───────────────────────────────────────────────────

func TestAuthenticate_AnonymousProcedure_NoCredentials(t *testing.T) {
	t.Parallel()
	const proc = "/backend.query.v1.QueryService/ListSimulations"
	a := newAuthenticator(t, authn.Config{})
	p, err := authenticate(a, proc, http.Header{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil principal for anonymous procedure, got %+v", p)
	}
}

func TestAuthenticate_ProtectedProcedure_NoCredentials(t *testing.T) {
	t.Parallel()
	const proc = "/backend.admin.v1.AdminService/MarkResultState"
	a := newAuthenticator(t, authn.Config{})
	_, err := authenticate(a, proc, http.Header{})
	if err == nil {
		t.Fatal("expected error for missing credentials on protected procedure")
	}
}

func TestAuthenticate_AmbiguousAuth_Rejected(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, validTokenYAML)
	a := newAuthenticator(t, authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	})
	h := http.Header{}
	h.Set("Authorization", "Bearer sometoken")
	h.Set("api-token", "secret-token-value")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error when both auth headers present")
	}
}

// ─── Bearer token extraction ─────────────────────────────────────────────────

func TestAuthenticate_BearerScheme_Required(t *testing.T) {
	t.Parallel()
	a := newAuthenticator(t, authn.Config{})
	h := http.Header{}
	h.Set("Authorization", "Basic dXNlcjpwYXNz")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error for non-Bearer Authorization scheme")
	}
}

func TestAuthenticate_EmptyBearerToken_Rejected(t *testing.T) {
	t.Parallel()
	a := newAuthenticator(t, authn.Config{})
	h := http.Header{}
	h.Set("Authorization", "Bearer   ")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error for empty bearer token")
	}
}

func TestAuthenticate_JWTNotConfigured_Rejected(t *testing.T) {
	t.Parallel()
	a := newAuthenticator(t, authn.Config{})
	h := http.Header{}
	h.Set("Authorization", "Bearer sometoken")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error when JWT not configured")
	}
}

// ─── api-token validation ────────────────────────────────────────────────────

func TestAuthenticate_APIToken_Valid(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, validTokenYAML)
	a := newAuthenticator(t, authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	})
	h := http.Header{}
	h.Set("api-token", "secret-token-value")
	p, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected principal, got nil")
	}
	if p.ID != "user-123" {
		t.Errorf("ID = %q, want user-123", p.ID)
	}
}

func TestAuthenticate_APIToken_InactiveToken_Rejected(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, `
tokens:
  - token: "inactive-token"
    id: "user-inactive"
    tenant: "tenant-a"
    active: false
`)
	a := newAuthenticator(t, authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	})
	h := http.Header{}
	h.Set("api-token", "inactive-token")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error for inactive token")
	}
}

func TestAuthenticate_APIToken_UnknownToken_Rejected(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, validTokenYAML)
	a := newAuthenticator(t, authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	})
	h := http.Header{}
	h.Set("api-token", "not-a-real-token")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
}

func TestAuthenticate_APITokenNotConfigured_Rejected(t *testing.T) {
	t.Parallel()
	// No APIToken config: api-token header should fail.
	a := newAuthenticator(t, authn.Config{})
	h := http.Header{}
	h.Set("api-token", "some-token")
	_, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err == nil {
		t.Fatal("expected error when api-token not configured")
	}
}

// ─── principal claim mapping ─────────────────────────────────────────────────

func TestAuthenticate_APIToken_ClaimMapping(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, `
tokens:
  - token: "mapped-token"
    id: "u-999"
    tenant: "org-x"
    active: true
    roles:
      - "editor"
      - "viewer"
    scopes:
      simulationIDs:
        - "sim-a"
      seriesIDs:
        - "sr-1"
`)
	a := newAuthenticator(t, authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	})
	h := http.Header{}
	h.Set("api-token", "mapped-token")
	p, err := authenticate(a, "/backend.admin.v1.AdminService/MarkResultState", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected principal")
	}
	if p.ID != "u-999" {
		t.Errorf("ID = %q, want u-999", p.ID)
	}
	if p.Tenant != "org-x" {
		t.Errorf("Tenant = %q, want org-x", p.Tenant)
	}
	if len(p.Roles) != 2 || p.Roles[0] != "editor" {
		t.Errorf("Roles = %v, want [editor viewer]", p.Roles)
	}
	if len(p.Scopes.SimulationIDs) != 1 {
		t.Errorf("SimulationIDs = %v, want 1 entry", p.Scopes.SimulationIDs)
	}
	if len(p.Scopes.SeriesIDs) != 1 || p.Scopes.SeriesIDs[0] != "sr-1" {
		t.Errorf("SeriesIDs = %v, want [sr-1]", p.Scopes.SeriesIDs)
	}
}

// ─── token file validation errors ────────────────────────────────────────────

func TestNewAuthenticator_DuplicateToken_Error(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, `
tokens:
  - token: "dup"
    id: "user-1"
    tenant: "t"
    active: true
  - token: "dup"
    id: "user-2"
    tenant: "t"
    active: true
`)
	_, err := authn.NewAuthenticator(context.Background(), authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	}, testLogger(t))
	if err == nil {
		t.Fatal("expected error for duplicate token entries")
	}
}

func TestNewAuthenticator_MissingTokenValue_Error(t *testing.T) {
	t.Parallel()
	path := writeTokenFile(t, `
tokens:
  - token: ""
    id: "user-1"
    tenant: "t"
    active: true
`)
	_, err := authn.NewAuthenticator(context.Background(), authn.Config{
		APIToken: authn.APITokenConfig{FilePath: path},
	}, testLogger(t))
	if err == nil {
		t.Fatal("expected error for empty token value")
	}
}
