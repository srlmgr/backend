package server

import (
	"github.com/dgraph-io/badger/v4"

	"github.com/srlmgr/backend/cmd/config"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/support/cache"
	"github.com/srlmgr/backend/support/iracing/irdata"
	"github.com/srlmgr/backend/support/iracing/irdata/auth"
)

type (
	iRacingClient struct {
		API  *irdata.IrData
		opts appOptions
	}
	appOptions struct {
		cachOpts cacheOptions
	}
	cacheOptions interface {
		cacheOpen() (cache.Cache, error)
		cacheClose() error
	}
	badgerCacheOptions struct {
		db *badger.DB
	}
	noopCacheOptions struct{}

	Option func(*appOptions)
)

var (
	badgerCache cacheOptions = &badgerCacheOptions{}
	noopCache   cacheOptions = &noopCacheOptions{}
)

func (b *badgerCacheOptions) cacheOpen() (cache.Cache, error) {
	var dbErr error
	b.db, dbErr = badger.Open(badger.DefaultOptions(config.IRacingCfg.CacheDir))
	if dbErr != nil {
		log.Error("failed to open cache database", log.ErrorField(dbErr))
		return nil, dbErr
	}

	badgerCache, cacheErr := cache.NewBadgerCache(b.db)
	if cacheErr != nil {
		log.Error("failed to create cache", log.ErrorField(cacheErr))
		return nil, cacheErr
	}
	return badgerCache, nil
}

func (b *badgerCacheOptions) cacheClose() error {
	if err := b.db.Close(); err != nil {
		log.Error("failed to close cache database", log.ErrorField(err))
		return err
	}
	return nil
}

func (n *noopCacheOptions) cacheOpen() (cache.Cache, error) {
	return cache.NewNoopCache(), nil
}

func (n *noopCacheOptions) cacheClose() error {
	return nil
}

func WithNoopCache() Option {
	return func(opts *appOptions) {
		opts.cachOpts = noopCache
	}
}

func WithBadgerCache() Option {
	return func(opts *appOptions) {
		opts.cachOpts = badgerCache
	}
}

func InitIRacingClient(opts ...Option) (*iRacingClient, error) {
	tm, tmErr := auth.NewTokenManager(auth.WithAuthConfig(&config.IRacingCfg.AuthConfig))
	if tmErr != nil {
		log.Error("failed to create token manager", log.ErrorField(tmErr))
		return nil, tmErr
	}
	if loginErr := tm.Login(); loginErr != nil {
		log.Error("failed to login", log.ErrorField(loginErr))
		return nil, loginErr
	}

	appOpts := appOptions{
		cachOpts: badgerCache,
	}
	for _, opt := range opts {
		opt(&appOpts)
	}

	irCache, cacheErr := appOpts.cachOpts.cacheOpen()
	if cacheErr != nil {
		log.Error("failed to open cache", log.ErrorField(cacheErr))
		return nil, cacheErr
	}

	ir, irErr := irdata.NewIrData(
		irdata.WithTokenProvider(tm.GetAccessToken),
		irdata.WithCache(irCache),
	)
	if irErr != nil {
		log.Error("failed to create iRData instance", log.ErrorField(irErr))
		return nil, irErr
	}
	return &iRacingClient{API: ir, opts: appOpts}, nil
}

func (a *iRacingClient) Close() {
	if a.opts.cachOpts != nil {
		if err := a.opts.cachOpts.cacheClose(); err != nil {
			log.Error("failed to close cache", log.ErrorField(err))
		}
	}
}
