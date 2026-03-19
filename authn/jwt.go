package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/srlmgr/backend/log"
)

// jwtClaims extends registered claims with backend-specific fields.
type jwtClaims struct {
	jwtlib.RegisteredClaims
	Tenant        string   `json:"tenant"`
	Roles         []string `json:"roles"`
	SimulationIDs []string `json:"simulationIDs"`
	SeriesIDs     []string `json:"seriesIDs"`
}

// jwtValidator validates JWT tokens.
type jwtValidator struct {
	cfg   JWTConfig
	cache *jwksCache
}

// newJWTValidator creates a new JWT validator.
// ctx is used for the initial JWKS fetch (if JWKSUrl is configured).
//
//nolint:whitespace //editor/linter issue
func newJWTValidator(
	ctx context.Context,
	cfg JWTConfig,
	logger *log.Logger,
) (*jwtValidator, error) {
	v := &jwtValidator{cfg: cfg}
	if cfg.JWKSUrl == "" {
		return v, nil
	}
	cache, err := newJWKSCache(ctx, cfg.JWKSUrl, cfg.CacheRefreshInterval, logger)
	if err != nil {
		return nil, fmt.Errorf("create JWKS cache: %w", err)
	}
	v.cache = cache
	return v, nil
}

// Validate parses and validates the JWT, returning a Principal on success.
func (v *jwtValidator) Validate(tokenStr string) (*Principal, error) {
	opts := []jwtlib.ParserOption{
		jwtlib.WithExpirationRequired(),
	}
	if v.cfg.ClockSkew > 0 {
		opts = append(opts, jwtlib.WithLeeway(v.cfg.ClockSkew))
	}
	if v.cfg.Issuer != "" {
		opts = append(opts, jwtlib.WithIssuer(v.cfg.Issuer))
	}
	if v.cfg.Audience != "" {
		opts = append(opts, jwtlib.WithAudience(v.cfg.Audience))
	}

	claims := &jwtClaims{}
	token, err := jwtlib.ParseWithClaims(tokenStr, claims, v.keyFunc, opts...)
	if err != nil {
		return nil, fmt.Errorf("parse JWT: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid JWT")
	}
	sub, err := claims.GetSubject()
	if err != nil || strings.TrimSpace(sub) == "" {
		return nil, errors.New("JWT missing subject claim")
	}
	return &Principal{
		ID:     sub,
		Tenant: claims.Tenant,
		Roles:  claims.Roles,
		Scopes: Scopes{
			SimulationIDs: claims.SimulationIDs,
			SeriesIDs:     claims.SeriesIDs,
		},
	}, nil
}

func (v *jwtValidator) keyFunc(token *jwtlib.Token) (any, error) {
	if v.cache == nil {
		return nil, errors.New("no JWKS configured")
	}
	if _, ok := token.Method.(*jwtlib.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("JWT missing kid header")
	}
	key, found := v.cache.getKey(kid)
	if !found {
		return nil, fmt.Errorf("unknown kid: %s", kid)
	}
	return key, nil
}
