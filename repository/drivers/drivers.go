// Package drivers provides repositories for the drivers migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package drivers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// DriversRepository defines persistence operations for Driver entities.
type DriversRepository interface {
	LoadAll(ctx context.Context) ([]*models.Driver, error)
	LoadByID(ctx context.Context, id int32) (*models.Driver, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.DriverSetter) (*models.Driver, error)
	Update(ctx context.Context, id int32, input *models.DriverSetter) (*models.Driver, error)
	FindByName(ctx context.Context, arg string) (*models.Driver, error)
}

// SimulationDriverAliasesRepository defines persistence operations for SimulationDriverAlias entities.
type SimulationDriverAliasesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SimulationDriverAlias, error)
	LoadBySimulationID(ctx context.Context, simID int32) ([]*models.SimulationDriverAlias, error)
	FindBySimID(
		ctx context.Context,
		simID int32,
		aliases ...string,
	) (*models.SimulationDriverAlias, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.SimulationDriverAliasSetter,
	) (*models.SimulationDriverAlias, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SimulationDriverAliasSetter,
	) (*models.SimulationDriverAlias, error)
}

// Repository exposes repositories for the drivers migration group.
type Repository interface {
	Drivers() DriversRepository
	SimulationDriverAliases() SimulationDriverAliasesRepository
}

type repository struct {
	drivers                 DriversRepository
	simulationDriverAliases SimulationDriverAliasesRepository
}

type (
	driversRepository                 struct{ exec *pgbob.Executor }
	simulationDriverAliasesRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		drivers:                 &driversRepository{exec: pgbob.New(pool)},
		simulationDriverAliases: &simulationDriverAliasesRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) Drivers() DriversRepository { return r.drivers }
func (r *repository) SimulationDriverAliases() SimulationDriverAliasesRepository {
	return r.simulationDriverAliases
}

func (r *driversRepository) LoadAll(ctx context.Context) ([]*models.Driver, error) {
	return models.Drivers.Query().All(ctx, r.getExecutor(ctx))
}

func (r *driversRepository) LoadByID(ctx context.Context, id int32) (*models.Driver, error) {
	entity, err := models.Drivers.Query(sm.Where(models.Drivers.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("driver %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *driversRepository) FindByName(ctx context.Context, arg string) (*models.Driver, error) {
	entity, err := models.Drivers.Query(sm.Where(models.Drivers.Columns.Name.EQ(psql.Arg(arg)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("driver with name %q: %w", arg, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *driversRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Drivers.Delete(dm.Where(models.Drivers.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *driversRepository) Create(
	ctx context.Context,
	input *models.DriverSetter,
) (*models.Driver, error) {
	return models.Drivers.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *driversRepository) Update(
	ctx context.Context,
	id int32,
	input *models.DriverSetter,
) (*models.Driver, error) {
	entity, err := models.Drivers.Update(
		input.UpdateMod(),
		um.Where(models.Drivers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("driver %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *driversRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *simulationDriverAliasesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.Query(sm.Where(models.SimulationDriverAliases.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation driver alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationDriverAliasesRepository) LoadBySimulationID(
	ctx context.Context,
	simID int32,
) ([]*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.
		Query(
			sm.Where(
				models.SimulationDriverAliases.Columns.SimulationID.EQ(psql.Arg(simID)))).
		All(ctx, r.getExecutor(ctx))

	return entity, err
}

func (r *simulationDriverAliasesRepository) FindBySimID(
	ctx context.Context,
	simID int32,
	aliases ...string,
) (*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.Query(
		sm.Where(models.SimulationDriverAliases.Columns.SimulationID.EQ(psql.Arg(simID))),
		sm.Where(
			models.SimulationDriverAliases.Columns.SimulationDriverID.EQ(
				psql.F("ANY", psql.Arg(aliases)),
			),
		),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"simulation driver alias %q for simulation %d: %w",
			aliases,
			simID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *simulationDriverAliasesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SimulationDriverAliases.Delete(dm.Where(models.SimulationDriverAliases.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationDriverAliasesRepository) Create(
	ctx context.Context,
	input *models.SimulationDriverAliasSetter,
) (*models.SimulationDriverAlias, error) {
	return models.SimulationDriverAliases.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *simulationDriverAliasesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SimulationDriverAliasSetter,
) (*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.Update(
		input.UpdateMod(),
		um.Where(models.SimulationDriverAliases.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation driver alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationDriverAliasesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
