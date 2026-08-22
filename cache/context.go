package cache

import "context"

type (
	cacheContextKey struct{}
	CacheContext    struct {
		CacheManager *CacheManager
		Config       *Config
	}
)

var ctxKey = cacheContextKey{}

//nolint:whitespace //editor/linter issue
func AddCacheConfigToContext(
	ctx context.Context,
	cm *CacheManager,
	cfg *Config,
) context.Context {
	return context.WithValue(ctx, ctxKey, &CacheContext{
		CacheManager: cm,
		Config:       cfg,
	})
}

func GetCacheConfigFromContext(ctx context.Context) *CacheContext {
	if v, ok := ctx.Value(ctxKey).(*CacheContext); ok {
		return v
	}
	return nil
}
