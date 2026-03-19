package authn

import (
	"strings"
	"time"
)

// Config configures authentication behavior.
type Config struct {
	Enabled  bool
	JWT      JWTConfig
	APIToken APITokenConfig
}

// JWTConfig configures JWT validation via remote JWKS.
type JWTConfig struct {
	Enabled         bool
	Issuer          string
	Audience        string
	JWKSURL         string
	ClockSkew       time.Duration
	RefreshInterval time.Duration
}

// APITokenConfig configures filesystem-backed api-token validation.
type APITokenConfig struct {
	FilePath        string
	RefreshInterval time.Duration
}

// IsAnonymousProcedure returns true for procedures that allow anonymous access.
func IsAnonymousProcedure(procedure string) bool {
	return strings.HasPrefix(procedure, "/backend.query.v1.QueryService/")
}
