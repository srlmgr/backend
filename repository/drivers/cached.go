package drivers

import (
	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedRepository struct {
	drivers                 DriversRepository
	seasonDrivers           SeasonDriversRepository
	simulationDriverAliases SimulationDriverAliasesRepository
}

//nolint:whitespace // editor/linter issue
func NewCached(
	repo Repository,
	driversCache *cache.Cache[int32, *models.Driver],
	seasonDriversCache *cache.Cache[int32, []*models.SeasonDriver],
) Repository {
	return &cachedRepository{
		drivers: NewCachedDriversRepository(repo.Drivers(), driversCache),
		seasonDrivers: NewCachedSeasonDriversRepository(
			repo.SeasonDrivers(),
			seasonDriversCache,
		),
		simulationDriverAliases: repo.SimulationDriverAliases(),
	}
}

func (r *cachedRepository) Drivers() DriversRepository {
	return r.drivers
}

func (r *cachedRepository) SimulationDriverAliases() SimulationDriverAliasesRepository {
	return r.simulationDriverAliases
}

func (r *cachedRepository) SeasonDrivers() SeasonDriversRepository {
	return r.seasonDrivers
}
