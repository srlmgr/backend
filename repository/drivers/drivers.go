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
}

// DriverSimulationIDsRepository defines persistence operations for DriverSimulationID entities.
type DriverSimulationIDsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.DriverSimulationID, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.DriverSimulationIDSetter,
	) (*models.DriverSimulationID, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.DriverSimulationIDSetter,
	) (*models.DriverSimulationID, error)
}

// Repository exposes repositories for the drivers migration group.
type Repository interface {
	Drivers() DriversRepository
	DriverSimulationIDs() DriverSimulationIDsRepository
}

type repository struct {
	drivers             DriversRepository
	driverSimulationIDs DriverSimulationIDsRepository
}

type (
	driversRepository             struct{ exec *pgbob.Executor }
	driverSimulationIDsRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		drivers:             &driversRepository{exec: pgbob.New(pool)},
		driverSimulationIDs: &driverSimulationIDsRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) Drivers() DriversRepository { return r.drivers }
func (r *repository) DriverSimulationIDs() DriverSimulationIDsRepository {
	return r.driverSimulationIDs
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
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *driversRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *driverSimulationIDsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.DriverSimulationID, error) {
	entity, err := models.DriverSimulationIds.Query(sm.Where(models.DriverSimulationIds.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("driver simulation id %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *driverSimulationIDsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.DriverSimulationIds.Delete(dm.Where(models.DriverSimulationIds.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *driverSimulationIDsRepository) Create(
	ctx context.Context,
	input *models.DriverSimulationIDSetter,
) (*models.DriverSimulationID, error) {
	return models.DriverSimulationIds.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *driverSimulationIDsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.DriverSimulationIDSetter,
) (*models.DriverSimulationID, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *driverSimulationIDsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
