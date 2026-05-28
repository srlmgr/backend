//nolint:lll,whitespace // repository implementations can be verbose
package drivers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	LoadByIDs(ctx context.Context, ids []int32) ([]*models.Driver, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.DriverSetter) (*models.Driver, error)
	Update(ctx context.Context, id int32, input *models.DriverSetter) (*models.Driver, error)
	FindByName(ctx context.Context, arg string) (*models.Driver, error)
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

func (r *driversRepository) LoadByIDs(ctx context.Context, ids []int32) ([]*models.Driver, error) {
	entity, err := models.Drivers.Query(
		sm.Where(models.Drivers.Columns.ID.EQ(psql.F("ANY", psql.Arg(ids))))).
		All(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("drivers %v: %w", ids, repoerrors.ErrNotFound)
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
