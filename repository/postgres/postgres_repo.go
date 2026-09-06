// Package postgres wires the repository interfaces to postgres-backed implementations.
//
//nolint:lll,dupl // readability, many similar code in this package
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/bookingentries"
	"github.com/srlmgr/backend/repository/cars"
	"github.com/srlmgr/backend/repository/drivers"
	"github.com/srlmgr/backend/repository/eventprocessingaudit"
	"github.com/srlmgr/backend/repository/events"
	"github.com/srlmgr/backend/repository/importbatches"
	"github.com/srlmgr/backend/repository/pointsystems"
	"github.com/srlmgr/backend/repository/queries"
	"github.com/srlmgr/backend/repository/races"
	"github.com/srlmgr/backend/repository/racingsims"
	"github.com/srlmgr/backend/repository/resultentries"
	"github.com/srlmgr/backend/repository/seasons"
	"github.com/srlmgr/backend/repository/series"
	"github.com/srlmgr/backend/repository/standings"
	"github.com/srlmgr/backend/repository/teams"
	"github.com/srlmgr/backend/repository/tracks"
)

type options struct {
	cacheCtx     context.Context //nolint:containedctx // bounds lifetime of cache eviction goroutines
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

const (
	CacheSerieses             = "serieses"
	CacheSeasons              = "seasons"
	CacheEvents               = "events"
	CacheBookingEntries       = "booking_entries"
	CacheResultEntries        = "result_entries"
	CacheDrivers              = "drivers"
	CacheSeasonDrivers        = "season_drivers"
	CacheCarManufacturers     = "car_manufacturers"
	CacheCarModels            = "car_models"
	CacheCarModelVariants     = "car_model_variants"
	CacheCarClasses           = "car_classes"
	CacheSimulationCarAliases = "simulation_car_aliases"
	CacheTeams                = "teams"
	CacheSeasonTeams          = "season_teams"
	CacheTeamDrivers          = "team_drivers"
)

type repository struct {
	racingSims           racingsims.Repository
	pointSystems         pointsystems.Repository
	drivers              drivers.Repository
	tracks               tracks.Repository
	cars                 cars.Repository
	series               series.Repository
	seasons              seasons.Repository
	events               events.Repository
	races                races.Repository
	teams                teams.Repository
	importBatches        importbatches.Repository
	resultEntries        resultentries.Repository
	bookingEntries       bookingentries.Repository
	eventProcessingAudit eventprocessingaudit.Repository
	standings            standings.Repository
	queries              rootrepo.Queries
}

// New returns the root postgres-backed repository aggregate.
//

func New(pool *pgxpool.Pool, opts ...Option) rootrepo.Repository {
	var o options
	o.l = log.Default().Named("repo")
	for _, opt := range opts {
		opt(&o)
	}

	seasonsRepo := setupSeasonsRepo(pool, o)
	eventsRepo := setupEventsRepo(pool, o)
	bookingEntriesRepo := setupBookingEntriesRepo(pool, o)
	resultEntriesRepo := setupResultEntriesRepo(pool, o)
	driversRepo := setupDriversRepo(pool, o)
	carsRepo := setupCarsRepo(pool, o)
	teamsRepo := setupTeamsRepo(pool, o)

	return &repository{
		racingSims:           racingsims.New(pool),
		pointSystems:         pointsystems.New(pool),
		drivers:              driversRepo,
		tracks:               tracks.New(pool),
		cars:                 carsRepo,
		series:               series.New(pool),
		seasons:              seasonsRepo,
		events:               eventsRepo,
		races:                races.New(pool),
		teams:                teamsRepo,
		importBatches:        importbatches.New(pool),
		resultEntries:        resultEntriesRepo,
		bookingEntries:       bookingEntriesRepo,
		eventProcessingAudit: eventprocessingaudit.New(pool),
		standings:            standings.New(pool),
		queries:              queries.New(pool),
	}
}

func (r *repository) RacingSims() racingsims.Repository         { return r.racingSims }
func (r *repository) PointSystems() pointsystems.Repository     { return r.pointSystems }
func (r *repository) Drivers() drivers.Repository               { return r.drivers }
func (r *repository) Tracks() tracks.Repository                 { return r.tracks }
func (r *repository) Cars() cars.Repository                     { return r.cars }
func (r *repository) Series() series.Repository                 { return r.series }
func (r *repository) Seasons() seasons.Repository               { return r.seasons }
func (r *repository) Events() events.Repository                 { return r.events }
func (r *repository) Races() races.Repository                   { return r.races }
func (r *repository) Teams() teams.Repository                   { return r.teams }
func (r *repository) ImportBatches() importbatches.Repository   { return r.importBatches }
func (r *repository) ResultEntries() resultentries.Repository   { return r.resultEntries }
func (r *repository) BookingEntries() bookingentries.Repository { return r.bookingEntries }
func (r *repository) EventProcessingAudit() eventprocessingaudit.Repository {
	return r.eventProcessingAudit
}
func (r *repository) Standings() standings.Repository { return r.standings }
func (r *repository) Queries() rootrepo.Queries       { return r.queries }

func setupSeasonsRepo(pool *pgxpool.Pool, o options) seasons.Repository {
	ret := seasons.New(pool)
	if o.cacheCtx != nil {
		settings := o.cacheConfig.SettingsFor(CacheSeasons)
		opts := cache.OptionsFromSettings[int32, *models.Season](settings)
		opts = append(opts,
			cache.WithCacheManager[int32, *models.Season](o.cacheManager, CacheSeasons))
		if c, err := cache.New(o.cacheCtx, CacheSeasons, opts...); err == nil {
			ret = seasons.NewCached(ret, c)
			o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, o.l))
		}
	}
	return ret
}

func setupEventsRepo(pool *pgxpool.Pool, o options) events.Repository {
	ret := events.New(pool)
	if o.cacheCtx != nil {
		settings := o.cacheConfig.SettingsFor(CacheEvents)
		opts := cache.OptionsFromSettings[int32, *models.Event](settings)
		opts = append(opts,
			cache.WithCacheManager[int32, *models.Event](o.cacheManager, CacheEvents))
		if c, err := cache.New(o.cacheCtx, CacheEvents, opts...); err == nil {
			ret = events.NewCached(ret, c)
			o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, o.l))
		}
	}
	return ret
}

func setupBookingEntriesRepo(pool *pgxpool.Pool, o options) bookingentries.Repository {
	ret := bookingentries.New(pool)
	if o.cacheCtx != nil {
		settings := o.cacheConfig.SettingsFor(CacheBookingEntries)
		opts := cache.OptionsFromSettings[cache.CompositeKey[int32], []*models.BookingEntry](
			settings,
		)
		opts = append(
			opts,
			cache.WithCacheManager[cache.CompositeKey[int32], []*models.BookingEntry](
				o.cacheManager,
				CacheBookingEntries,
			),
		)
		if c, err := cache.New(o.cacheCtx, CacheBookingEntries, opts...); err == nil {
			ret = bookingentries.NewCached(ret, c)
			o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, o.l))
		}
	}
	return ret
}

func setupResultEntriesRepo(pool *pgxpool.Pool, o options) resultentries.Repository {
	ret := resultentries.New(pool)
	if o.cacheCtx != nil {
		settings := o.cacheConfig.SettingsFor(CacheResultEntries)
		opts := cache.OptionsFromSettings[cache.CompositeKey[int32], []*models.ResultEntry](
			settings,
		)
		opts = append(
			opts,
			cache.WithCacheManager[cache.CompositeKey[int32], []*models.ResultEntry](
				o.cacheManager,
				CacheResultEntries,
			),
		)
		if c, err := cache.New(o.cacheCtx, CacheResultEntries, opts...); err == nil {
			ret = resultentries.NewCached(ret, c)
			o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(c, o.l))
		}
	}
	return ret
}

func setupDriversRepo(pool *pgxpool.Pool, o options) drivers.Repository {
	ret := drivers.New(pool)
	if o.cacheCtx != nil {
		settings := o.cacheConfig.SettingsFor(CacheDrivers)
		opts := cache.OptionsFromSettings[int32, *models.Driver](settings)
		opts = append(opts,
			cache.WithCacheManager[int32, *models.Driver](o.cacheManager, CacheDrivers))
		dCache, _ := cache.New(o.cacheCtx, CacheDrivers, opts...)

		sdSettings := o.cacheConfig.SettingsFor(CacheSeasonDrivers)
		sdOpts := cache.OptionsFromSettings[int32, []*models.SeasonDriver](sdSettings)
		sdOpts = append(sdOpts,
			cache.WithCacheManager[int32, []*models.SeasonDriver](
				o.cacheManager, CacheSeasonDrivers,
			))
		sdCache, _ := cache.New(o.cacheCtx, CacheSeasonDrivers, sdOpts...)
		o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(sdCache, o.l))

		ret = drivers.NewCached(ret, dCache, sdCache)
	}
	return ret
}

//nolint:funlen // lots of caches to wire up
func setupCarsRepo(pool *pgxpool.Pool, o options) cars.Repository {
	ret := cars.New(pool)
	if o.cacheCtx == nil {
		return ret
	}

	manufacturerSettings := o.cacheConfig.SettingsFor(CacheCarManufacturers)
	manufacturerOpts := cache.OptionsFromSettings[int32, *models.CarManufacturer](
		manufacturerSettings,
	)
	manufacturerOpts = append(manufacturerOpts,
		cache.WithCacheManager[int32, *models.CarManufacturer](
			o.cacheManager, CacheCarManufacturers,
		))
	manufacturersCache, err := cache.New(o.cacheCtx, CacheCarManufacturers, manufacturerOpts...)
	if err != nil {
		return ret
	}

	modelSettings := o.cacheConfig.SettingsFor(CacheCarModels)
	modelOpts := cache.OptionsFromSettings[int32, *models.CarModel](modelSettings)
	modelOpts = append(modelOpts,
		cache.WithCacheManager[int32, *models.CarModel](o.cacheManager, CacheCarModels))
	modelsCache, err := cache.New(o.cacheCtx, CacheCarModels, modelOpts...)
	if err != nil {
		return ret
	}

	variantSettings := o.cacheConfig.SettingsFor(CacheCarModelVariants)
	variantOpts := cache.OptionsFromSettings[int32, *models.CarModelVariant](variantSettings)
	variantOpts = append(variantOpts,
		cache.WithCacheManager[int32, *models.CarModelVariant](
			o.cacheManager, CacheCarModelVariants,
		))
	variantsCache, err := cache.New(o.cacheCtx, CacheCarModelVariants, variantOpts...)
	if err != nil {
		return ret
	}

	classSettings := o.cacheConfig.SettingsFor(CacheCarClasses)
	classOpts := cache.OptionsFromSettings[int32, *models.CarClass](classSettings)
	classOpts = append(classOpts,
		cache.WithCacheManager[int32, *models.CarClass](o.cacheManager, CacheCarClasses))
	classesCache, err := cache.New(o.cacheCtx, CacheCarClasses, classOpts...)
	if err != nil {
		return ret
	}

	aliasSettings := o.cacheConfig.SettingsFor(CacheSimulationCarAliases)
	aliasOpts := cache.OptionsFromSettings[int32, *models.SimulationCarAlias](aliasSettings)
	aliasOpts = append(aliasOpts,
		cache.WithCacheManager[int32, *models.SimulationCarAlias](
			o.cacheManager, CacheSimulationCarAliases,
		))
	aliasesCache, err := cache.New(o.cacheCtx, CacheSimulationCarAliases, aliasOpts...)
	if err != nil {
		return ret
	}

	caches := &cars.Caches{
		CarManufacturers:     manufacturersCache,
		CarModels:            modelsCache,
		CarModelVariants:     variantsCache,
		CarClasses:           classesCache,
		SimulationCarAliases: aliasesCache,
	}
	for _, c := range []cache.EventHandler{
		flushOnChanges(manufacturersCache, o.l),
		flushOnChanges(modelsCache, o.l),
		flushOnChanges(variantsCache, o.l),
		flushOnChanges(classesCache, o.l),
		flushOnChanges(aliasesCache, o.l),
	} {
		o.cacheManager.Subscribe(cache.BroadcastChannel, c)
	}

	return cars.NewCached(ret, caches)
}

//nolint:funlen // lots of caches to wire up
func setupTeamsRepo(pool *pgxpool.Pool, o options) teams.Repository {
	ret := teams.New(pool)
	if o.cacheCtx == nil {
		return ret
	}

	teamSettings := o.cacheConfig.SettingsFor(CacheTeams)
	teamOpts := cache.OptionsFromSettings[int32, *models.Team](teamSettings)
	teamOpts = append(teamOpts,
		cache.WithCacheManager[int32, *models.Team](o.cacheManager, CacheTeams))
	teamsCache, err := cache.New(o.cacheCtx, CacheTeams, teamOpts...)
	if err != nil {
		return ret
	}
	seasonTeamOpts := cache.OptionsFromSettings[int32, []*models.Team](teamSettings)
	seasonTeamOpts = append(seasonTeamOpts,
		cache.WithCacheManager[int32, []*models.Team](o.cacheManager, CacheSeasonTeams))
	seasonTeamsCache, err := cache.New(o.cacheCtx, CacheSeasonTeams, seasonTeamOpts...)
	if err != nil {
		return ret
	}

	teamDriverSettings := o.cacheConfig.SettingsFor(CacheTeamDrivers)
	teamDriverOpts := cache.OptionsFromSettings[int32, []*models.TeamDriver](
		teamDriverSettings,
	)
	teamDriverOpts = append(teamDriverOpts,
		cache.WithCacheManager[int32, []*models.TeamDriver](
			o.cacheManager,
			CacheTeamDrivers,
		))
	teamDriversCache, err := cache.New(
		o.cacheCtx,
		CacheTeamDrivers,
		teamDriverOpts...,
	)
	if err != nil {
		return ret
	}

	o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(teamsCache, o.l))
	o.cacheManager.Subscribe(cache.BroadcastChannel, flushOnChanges(teamDriversCache, o.l))

	return teams.NewCached(ret, &teams.Caches{
		Teams:       teamsCache,
		SeasonTeams: seasonTeamsCache,
		TeamDrivers: teamDriversCache,
	})
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
