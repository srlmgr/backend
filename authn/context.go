package authn

import "context"

type principalContextKey struct{}

var principalKey = principalContextKey{}

// AddPrincipal adds the normalized principal to context.
func AddPrincipal(ctx context.Context, principal *Principal) context.Context {
	if principal == nil {
		return ctx
	}

	return context.WithValue(ctx, principalKey, *principal)
}

// PrincipalFromContext returns the normalized principal if present.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(principalKey).(Principal)
	return principal, ok
}
