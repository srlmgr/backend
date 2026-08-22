// Package cars provides repositories for the cars migration group.
package cars

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/repository/pgbob"
)

// Repository exposes repositories for the cars migration group.
type Repository interface {
	CarManufacturers() CarManufacturersRepository
	CarModels() CarModelsRepository
	CarModelVariants() CarModelVariantsRepository
	CarClasses() CarClassesRepository
	SimulationCarAliases() SimulationCarAliasesRepository
}

type repository struct {
	carManufacturers     CarManufacturersRepository
	carModels            CarModelsRepository
	carModelVariants     CarModelVariantsRepository
	carClasses           CarClassesRepository
	simulationCarAliases SimulationCarAliasesRepository
}

type (
	carManufacturersRepository     struct{ exec *pgbob.Executor }
	carModelsRepository            struct{ exec *pgbob.Executor }
	carModelVariantsRepository     struct{ exec *pgbob.Executor }
	carClassesRepository           struct{ exec *pgbob.Executor }
	simulationCarAliasesRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		carManufacturers:     &carManufacturersRepository{exec: pgbob.New(pool)},
		carModels:            &carModelsRepository{exec: pgbob.New(pool)},
		carModelVariants:     &carModelVariantsRepository{exec: pgbob.New(pool)},
		carClasses:           &carClassesRepository{exec: pgbob.New(pool)},
		simulationCarAliases: &simulationCarAliasesRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) CarManufacturers() CarManufacturersRepository {
	return r.carManufacturers
}

func (r *repository) CarModels() CarModelsRepository {
	return r.carModels
}

func (r *repository) CarModelVariants() CarModelVariantsRepository {
	return r.carModelVariants
}

func (r *repository) CarClasses() CarClassesRepository {
	return r.carClasses
}

func (r *repository) SimulationCarAliases() SimulationCarAliasesRepository {
	return r.simulationCarAliases
}
