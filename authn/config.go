package authn

import (
	"net/http"
	"strings"
	"time"
)

// Config configures authentication behavior.
type Config struct {
	Enabled  bool
	IDP      IDPConfig
	APIToken APITokenConfig
	Store    SessionStore
}

// IDPConfig configures OIDC login and backend session handling.
type IDPConfig struct {
	Enabled        bool
	IssuerURL      string
	ClientID       string
	ClientSecret   string
	CallbackURL    string
	FrontendURL    string
	RefreshSkew    time.Duration
	SessionTTL     time.Duration
	StateTTL       time.Duration
	CookieName     string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite http.SameSite
}

// APITokenConfig configures filesystem-backed api-token validation.
type APITokenConfig struct {
	FilePath        string
	RefreshInterval time.Duration
}

// WithDefaults applies secure defaults to optional IDP settings.
func (c *Config) WithDefaults() {
	if c == nil {
		return
	}

	if c.IDP.CookieName == "" {
		c.IDP.CookieName = "backend_session"
	}
	if c.IDP.SessionTTL <= 0 {
		c.IDP.SessionTTL = 24 * time.Hour
	}
	if c.IDP.StateTTL <= 0 {
		c.IDP.StateTTL = 10 * time.Minute
	}
	if c.IDP.RefreshSkew <= 0 {
		c.IDP.RefreshSkew = 30 * time.Second
	}
	if c.IDP.CookieSameSite == 0 {
		c.IDP.CookieSameSite = http.SameSiteLaxMode
	}
}

// IsAnonymousProcedure returns true for procedures that allow anonymous access.
func IsAnonymousProcedure(procedure string) bool {
	return strings.HasPrefix(procedure, "/backend.query.v1.QueryService/")
}
