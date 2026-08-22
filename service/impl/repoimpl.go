package impl

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
	"github.com/srlmgr/backend/service/impl/cached"
)

type options struct {
	//nolint:containedctx // bounds lifetime of cache eviction goroutines
	cacheCtx     context.Context
	cacheConfig  *cache.Config
	cacheManager *cache.CacheManager
	l            *log.Logger
}

// Option configures optional behavior of New.
type Option func(*options)

// WithCache enables in-memory read-through caching for cache-aware repositories.
// ctx bounds the lifetime of any background eviction goroutines the cache starts.
// cfg may be nil, in which case caches are created without TTL/capacity bounds.
func WithCache(ctx context.Context, cfg *cache.Config, cm *cache.CacheManager) Option {
	return func(o *options) {
		o.cacheCtx = ctx
		o.cacheConfig = cfg
		o.cacheManager = cm
	}
}

type (
	serviceImpl struct {
		r                repository.Repository
		logger           *log.Logger
		tracer           trace.Tracer
		opts             *options
		standingsService service.StandingsService
	}
)

const (
	CacheStandings = "standings"
	CacheSummary   = "summary"
)

var _ service.Service = (*serviceImpl)(nil)

// creates a new service implementation based on the provided
// repository, logger, and tracer.
// this implementation is used on a running production instance
//
//nolint:whitespace //editor/linter issue
func New(
	r repository.Repository,
	logger *log.Logger,
	opts ...Option,
) service.Service {
	o := &options{}
	o.l = log.Default().Named("svc")
	for _, opt := range opts {
		opt(o)
	}

	ret := &serviceImpl{
		r:      r,
		logger: logger,
		tracer: otel.Tracer("backend.service"),
		opts:   o,
	}
	ret.standingsService = ret.setupStandingsService()
	return ret
}

func (s *serviceImpl) StandingsService() service.StandingsService {
	return s.standingsService
}

func (s *serviceImpl) ParticipantsService() service.ParticipantsService {
	return newParticipantsImpl(s.r, s.logger, s.tracer)
}

func (s *serviceImpl) SummaryService() service.SummaryService {
	return newSummaryImpl(s.r, s.logger, s.tracer)
}

func (s *serviceImpl) BookingsService() service.BookingsService {
	return nil
}

//nolint:lll // readability
func (s *serviceImpl) setupStandingsService() service.StandingsService {
	s.standingsService = newStandingsImpl(s.r, s.logger, s.tracer)
	if s.opts.cacheCtx != nil && s.opts.cacheConfig != nil && s.opts.cacheManager != nil {
		settings := s.opts.cacheConfig.SettingsFor(CacheStandings)
		opts := cache.OptionsFromSettings[cached.CacheStandingsKey, *service.StandingsContainer](
			settings,
		)
		opts = append(opts,
			cache.WithCacheManager[cached.CacheStandingsKey, *service.StandingsContainer](
				s.opts.cacheManager, CacheStandings))
		if c, err := cache.New(
			s.opts.cacheCtx,
			CacheStandings,
			opts...); err == nil {
			s.opts.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, s.opts.l))
			s.standingsService = cached.NewCachedStandings(s.standingsService, c)
		}
	}
	return s.standingsService
}

//nolint:whitespace // editor/linter issue
func flushOnChanges[K comparable, E any](
	c *cache.Cache[K, E],
	l *log.Logger,
) cache.EventHandler {
	return func(ctx context.Context, evt cache.InvalidationEvent) {
		refID, ok := evt.EntityID.(int32)
		if !ok {
			return
		}
		l.Debug("flushing cache due to change",
			log.String("name", c.Name()),
			log.Int32("ref_id", refID),
			log.String("entity", evt.EntityName),
		)
		c.Flush()
	}
}
