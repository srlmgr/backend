package authn

import "context"

// contextKey is the type for context keys in this package.
type contextKey int

const principalKey contextKey = iota

// Scopes holds the simulation and series IDs accessible to a principal.
type Scopes struct {
	SimulationIDs []string
	SeriesIDs     []string
}

// Principal is the normalized identity attached to request context.
type Principal struct {
	ID     string
	Tenant string
	Roles  []string
	Scopes Scopes
}

// WithPrincipal returns a new context with the given principal attached.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFromContext extracts the principal from context.
// Returns nil if no principal is present.
func PrincipalFromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey).(*Principal)
	return p
}
