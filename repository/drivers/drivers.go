// Package drivers provides repositories for the drivers migration group.
//

package drivers

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/repository/pgbob"
)

// Repository exposes repositories for the drivers migration group.
type Repository interface {
	Drivers() DriversRepository
	SimulationDriverAliases() SimulationDriverAliasesRepository
	SeasonDrivers() SeasonDriversRepository
}

type repository struct {
	drivers                 DriversRepository
	simulationDriverAliases SimulationDriverAliasesRepository
	seasonDrivers           SeasonDriversRepository
}

type (
	driversRepository                 struct{ exec *pgbob.Executor }
	simulationDriverAliasesRepository struct{ exec *pgbob.Executor }
	seasonDriversRepository           struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		drivers:                 &driversRepository{exec: pgbob.New(pool)},
		simulationDriverAliases: &simulationDriverAliasesRepository{exec: pgbob.New(pool)},
		seasonDrivers:           &seasonDriversRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) Drivers() DriversRepository { return r.drivers }
func (r *repository) SimulationDriverAliases() SimulationDriverAliasesRepository {
	return r.simulationDriverAliases
}
func (r *repository) SeasonDrivers() SeasonDriversRepository { return r.seasonDrivers }
