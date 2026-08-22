package cars

import (
	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedRepository struct {
	carManufacturers     CarManufacturersRepository
	carModels            CarModelsRepository
	carModelVariants     CarModelVariantsRepository
	carClasses           CarClassesRepository
	simulationCarAliases SimulationCarAliasesRepository
}

// Caches contains the read-through caches used by Repository.
type Caches struct {
	CarManufacturers *cache.Cache[
		int32,
		*models.CarManufacturer,
	]
	CarModels *cache.Cache[int32,
		*models.CarModel]
	CarModelVariants *cache.Cache[
		int32,
		*models.CarModelVariant,
	]
	CarClasses           *cache.Cache[int32, *models.CarClass]
	SimulationCarAliases *cache.Cache[
		int32,
		*models.SimulationCarAlias,
	]
}

// NewCached wraps repo with cached child repositories.
func NewCached(repo Repository, caches *Caches) Repository {
	return &cachedRepository{
		carManufacturers: NewCachedCarManufacturersRepository(
			repo.CarManufacturers(),
			caches.CarManufacturers,
		),
		carModels: NewCachedCarModelsRepository(
			repo.CarModels(),
			caches.CarModels,
		),
		carModelVariants: NewCachedCarModelVariantsRepository(
			repo.CarModelVariants(),
			caches.CarModelVariants,
		),
		carClasses: NewCachedCarClassesRepository(
			repo.CarClasses(),
			caches.CarClasses,
		),
		simulationCarAliases: NewCachedSimulationCarAliasesRepository(
			repo.SimulationCarAliases(),
			caches.SimulationCarAliases,
		),
	}
}

func (r *cachedRepository) CarManufacturers() CarManufacturersRepository {
	return r.carManufacturers
}

func (r *cachedRepository) CarModels() CarModelsRepository { return r.carModels }

func (r *cachedRepository) CarModelVariants() CarModelVariantsRepository {
	return r.carModelVariants
}

func (r *cachedRepository) CarClasses() CarClassesRepository { return r.carClasses }

func (r *cachedRepository) SimulationCarAliases() SimulationCarAliasesRepository {
	return r.simulationCarAliases
}
