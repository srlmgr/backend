package authn

import "time"

// Config holds all authentication configuration.
type Config struct {
	JWT      JWTConfig
	APIToken APITokenConfig
}

// JWTConfig holds JWT validation settings.
type JWTConfig struct {
	// Issuer is the expected iss claim value.
	Issuer string
	// Audience is the expected aud claim value.
	Audience string
	// JWKSUrl is the URL to fetch the JSON Web Key Set for signature verification.
	JWKSUrl string
	// CacheRefreshInterval is how often to refresh the JWKS cache.
	CacheRefreshInterval time.Duration
	// ClockSkew is the allowed clock skew for time-based claim validation.
	ClockSkew time.Duration
}

// APITokenConfig holds filesystem api-token settings.
type APITokenConfig struct {
	// FilePath is the path to the api-token store file (YAML).
	FilePath string
	// ReloadInterval is how often to reload the token file (0 = startup only).
	ReloadInterval time.Duration
}
