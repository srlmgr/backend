package service

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/service/standings"
)

type (
	//nolint:lll // readability
	cachedService struct {
		Service
		opts                    *cacheOptions
		standingsCache          *cache.Cache[CacheStandingsKey, *model.SeasonStandingsContainer]
		resultsOverviewCache    *cache.Cache[CacheResultsKey, *model.SeasonResultsOverviewContainer]
		seasonParticipantsCache *cache.Cache[CacheSeasonParticipantsKey, *model.SeasonParticipantsContainer]
	}
	cacheOptions struct {
		//nolint:containedctx // bounds lifetime of cache eviction goroutines
		cacheCtx     context.Context
		cacheConfig  *cache.Config
		cacheManager *cache.CacheManager
		l            *log.Logger
	}
	CacheOption func(*cacheOptions)

	CacheStandingsKey struct {
		SeasonID int
		EventID  int
		SkipMode standings.SkipModeType
	}
	CacheResultsKey struct {
		SeasonID int
		ClassID  int
	}
	CacheSeasonParticipantsKey struct {
		SeasonID int
	}
)

const (
	CacheStandings          = "htmlStandings"
	CacheResultsOverview    = "htmlResultsOverview"
	CacheSeasonParticipants = "htmlSeasonParticipants"
)

// WithCache enables in-memory read-through caching for cache-aware repositories.
// ctx bounds the lifetime of any background eviction goroutines the cache starts.
// cfg may be nil, in which case caches are created without TTL/capacity bounds.
//
//nolint:whitespace // editor/linter issue
func WithCache(
	ctx context.Context,
	cfg *cache.Config,
	cm *cache.CacheManager,
) CacheOption {
	return func(o *cacheOptions) {
		o.cacheCtx = ctx
		o.cacheConfig = cfg
		o.cacheManager = cm
	}
}

func NewCachedService(svc Service, opts ...CacheOption) Service {
	o := &cacheOptions{
		l: log.Default().Named("html.service.cache"),
	}
	for _, opt := range opts {
		opt(o)
	}

	ret := &cachedService{
		Service: svc,
		opts:    o,
	}
	ret.setupStandingsCaches()
	ret.setupResultsCaches()
	ret.setupSeasonParticipantsCaches()
	return ret
}

//nolint:whitespace // editor/linter issue
func (s *cachedService) GetSeasonStandings(
	ctx context.Context,
	seasonID int,
	skipMode standings.SkipModeType,
) (*model.SeasonStandingsContainer, error) {
	key := CacheStandingsKey{SeasonID: seasonID, SkipMode: skipMode, EventID: 0}
	if v, ok := s.standingsCache.Get(ctx, key); ok {
		return v, nil
	}
	ret, err := s.Service.GetSeasonStandings(ctx, seasonID, skipMode)
	if err == nil {
		cloned, _ := cache.Clone(ret)
		s.standingsCache.Set(ctx, key, ret)
		ret = cloned
	}
	return ret, err
}

//nolint:whitespace // editor/linter issue
func (s *cachedService) GetEventStandings(
	ctx context.Context,
	eventID int,
	skipMode standings.SkipModeType,
) (*model.SeasonStandingsContainer, error) {
	key := CacheStandingsKey{SeasonID: 0, SkipMode: skipMode, EventID: eventID}
	if v, ok := s.standingsCache.Get(ctx, key); ok {
		return v, nil
	}
	ret, err := s.Service.GetEventStandings(ctx, eventID, skipMode)
	if err == nil {
		cloned, _ := cache.Clone(ret)
		s.standingsCache.Set(ctx, key, ret)
		ret = cloned
	}
	return ret, err
}

//nolint:whitespace // editor/linter issue
func (s *cachedService) GetResultsOverview(
	ctx context.Context,
	seasonID, classID int,
) (*model.SeasonResultsOverviewContainer, error) {
	key := CacheResultsKey{SeasonID: seasonID, ClassID: classID}
	if v, ok := s.resultsOverviewCache.Get(ctx, key); ok {
		return v, nil
	}
	ret, err := s.Service.GetResultsOverview(ctx, seasonID, classID)
	if err == nil {
		s.resultsOverviewCache.Set(ctx, key, ret)
	}
	return ret, err
}

//nolint:whitespace // editor/linter issue
func (s *cachedService) GetSeasonParticipants(
	ctx context.Context,
	seasonID int,
) (*model.SeasonParticipantsContainer, error) {
	key := CacheSeasonParticipantsKey{SeasonID: seasonID}
	if v, ok := s.seasonParticipantsCache.Get(ctx, key); ok {
		return v, nil
	}
	ret, err := s.Service.GetSeasonParticipants(ctx, seasonID)
	if err == nil {
		s.seasonParticipantsCache.Set(ctx, key, ret)
	}
	return ret, err
}

func (s *cachedService) setupStandingsCaches() {
	settings := s.opts.cacheConfig.SettingsFor(CacheStandings)
	opts := cache.OptionsFromSettings[
		CacheStandingsKey, *model.SeasonStandingsContainer](settings)
	opts = append(opts,
		cache.WithCacheManager[CacheStandingsKey, *model.SeasonStandingsContainer](
			s.opts.cacheManager, CacheStandings),
		cache.WithClone[CacheStandingsKey, *model.SeasonStandingsContainer](),
	)
	if c, err := cache.New(
		s.opts.cacheCtx,
		CacheStandings,
		opts...); err == nil {
		s.opts.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, s.opts.l))
		s.standingsCache = c
	}
}

func (s *cachedService) setupResultsCaches() {
	settings := s.opts.cacheConfig.SettingsFor(CacheResultsOverview)
	opts := cache.OptionsFromSettings[
		CacheResultsKey, *model.SeasonResultsOverviewContainer](settings)
	opts = append(opts,
		cache.WithCacheManager[CacheResultsKey, *model.SeasonResultsOverviewContainer](
			s.opts.cacheManager, CacheResultsOverview),
		// cache.WithClone[CacheResultsKey, *model.SeasonResultsOverviewContainer](),
	)
	if c, err := cache.New(
		s.opts.cacheCtx,
		CacheResultsOverview,
		opts...); err == nil {
		s.opts.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, s.opts.l))
		s.resultsOverviewCache = c
	}
}

func (s *cachedService) setupSeasonParticipantsCaches() {
	settings := s.opts.cacheConfig.SettingsFor(CacheSeasonParticipants)
	opts := cache.OptionsFromSettings[
		CacheSeasonParticipantsKey, *model.SeasonParticipantsContainer](settings)
	opts = append(opts,
		cache.WithCacheManager[
			CacheSeasonParticipantsKey, *model.SeasonParticipantsContainer](
			s.opts.cacheManager, CacheSeasonParticipants),
	)
	if c, err := cache.New(
		s.opts.cacheCtx,
		CacheSeasonParticipants,
		opts...); err == nil {
		s.opts.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, s.opts.l))
		s.seasonParticipantsCache = c
	}
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
		l.Debug("flushing html cache due to change",
			log.String("name", c.Name()),
			log.Int32("ref_id", refID),
			log.String("entity", evt.EntityName),
		)
		c.Flush()
	}
}
