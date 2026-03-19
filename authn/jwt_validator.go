package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type jwtValidator struct {
	cfg      JWTConfig
	jwkCache *jwk.Cache
}

func newJWTValidator(ctx context.Context, cfg JWTConfig) (*jwtValidator, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.Issuer == "" || cfg.Audience == "" || cfg.JWKSURL == "" {
		return nil, fmt.Errorf("jwt authn requires issuer, audience and jwks-url")
	}

	cache, err := jwk.NewCache(ctx, httprc.NewClient())
	if err != nil {
		return nil, fmt.Errorf("create jwks cache: %w", err)
	}

	registerOptions := []jwk.RegisterOption{jwk.WithWaitReady(true)}
	if cfg.RefreshInterval > 0 {
		registerOptions = append(
			registerOptions,
			jwk.WithMinInterval(cfg.RefreshInterval),
			jwk.WithMaxInterval(cfg.RefreshInterval),
		)
	}

	if err := cache.Register(ctx, cfg.JWKSURL, registerOptions...); err != nil {
		return nil, fmt.Errorf("register jwks url: %w", err)
	}

	return &jwtValidator{cfg: cfg, jwkCache: cache}, nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (v *jwtValidator) validate(
	ctx context.Context,
	rawToken string,
) (Principal, error) {
	if v == nil {
		return Principal{}, errors.New("jwt validation is not configured")
	}

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		return v.lookupVerificationKey(ctx, token)
	},
		jwt.WithIssuer(v.cfg.Issuer),
		jwt.WithAudience(v.cfg.Audience),
		jwt.WithLeeway(v.cfg.ClockSkew),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithNotBeforeRequired(),
	)
	if err != nil {
		return Principal{}, fmt.Errorf("jwt validation failed: %w", err)
	}

	principal, err := mapJWTClaims(claims)
	if err != nil {
		return Principal{}, err
	}
	principal.Source = "jwt"
	return principal, nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (v *jwtValidator) lookupVerificationKey(
	ctx context.Context,
	token *jwt.Token,
) (any, error) {
	kid, _ := token.Header["kid"].(string)
	if strings.TrimSpace(kid) == "" {
		return nil, fmt.Errorf("jwt header is missing kid")
	}

	set, err := v.jwkCache.Lookup(ctx, v.cfg.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("lookup jwks set: %w", err)
	}
	key, ok := set.LookupKeyID(kid)
	if !ok {
		return nil, fmt.Errorf("no jwk found for kid")
	}

	var rawKey any
	if err := jwk.Export(key, &rawKey); err != nil {
		return nil, fmt.Errorf("convert jwk to raw key: %w", err)
	}
	return rawKey, nil
}

func mapJWTClaims(claims jwt.MapClaims) (Principal, error) {
	id := stringFromAny(claims["id"])
	if id == "" {
		id = stringFromAny(claims["sub"])
	}
	tenant := stringFromAny(claims["tenant"])
	if id == "" || tenant == "" {
		return Principal{}, fmt.Errorf("jwt missing required id/sub or tenant claim")
	}

	roles := stringSliceFromAny(claims["roles"])

	scopeSimulationIDs := stringSliceFromAny(claims["simulationIDs"])
	if len(scopeSimulationIDs) == 0 {
		scopeSimulationIDs = stringSliceFromAny(claims["simulationsIDs"])
	}
	scopeSeriesIDs := stringSliceFromAny(claims["seriesIDs"])

	if scopesRaw, ok := claims["scopes"].(map[string]any); ok {
		scopeSimulationIDs = resolveSimulationScope(scopeSimulationIDs, scopesRaw)
		scopeSeriesIDs = resolveSeriesScope(scopeSeriesIDs, scopesRaw)
	}

	return Principal{
		ID:            id,
		Tenant:        tenant,
		Roles:         roles,
		SimulationIDs: scopeSimulationIDs,
		SeriesIDs:     scopeSeriesIDs,
	}, nil
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func stringSliceFromAny(v any) []string {
	switch typed := v.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if val := stringFromAny(item); val != "" {
				out = append(out, val)
			}
		}
		return out
	default:
		return nil
	}
}

func resolveSimulationScope(current []string, scopesRaw map[string]any) []string {
	if len(current) > 0 {
		return current
	}

	simulationIDs := stringSliceFromAny(scopesRaw["simulationIDs"])
	if len(simulationIDs) > 0 {
		return simulationIDs
	}

	return stringSliceFromAny(scopesRaw["simulationsIDs"])
}

func resolveSeriesScope(current []string, scopesRaw map[string]any) []string {
	if len(current) > 0 {
		return current
	}

	return stringSliceFromAny(scopesRaw["seriesIDs"])
}
