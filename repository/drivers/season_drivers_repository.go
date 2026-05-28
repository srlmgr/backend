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

// SeasonDriversRepository defines persistence operations for SeasonDriver entities.
type SeasonDriversRepository interface {
	LoadAll(ctx context.Context) ([]*models.SeasonDriver, error)
	LoadByID(ctx context.Context, id int32) (*models.SeasonDriver, error)
	LoadByDriverID(ctx context.Context, driverID int32) ([]*models.SeasonDriver, error)
	LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.SeasonDriver, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.SeasonDriverSetter) (*models.SeasonDriver, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SeasonDriverSetter,
	) (*models.SeasonDriver, error)
}

func (r *seasonDriversRepository) LoadAll(ctx context.Context) ([]*models.SeasonDriver, error) {
	return models.SeasonDrivers.Query().All(ctx, r.getExecutor(ctx))
}

func (r *seasonDriversRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SeasonDriver, error) {
	entity, err := models.SeasonDrivers.Query(sm.Where(models.SeasonDrivers.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("season driver %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *seasonDriversRepository) LoadByDriverID(
	ctx context.Context,
	driverID int32,
) ([]*models.SeasonDriver, error) {
	return models.SeasonDrivers.Query(
		sm.Where(models.SeasonDrivers.Columns.DriverID.EQ(psql.Arg(driverID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *seasonDriversRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.SeasonDriver, error) {
	return models.SeasonDrivers.Query(
		sm.Where(models.SeasonDrivers.Columns.SeasonID.EQ(psql.Arg(seasonID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *seasonDriversRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SeasonDrivers.Delete(dm.Where(models.SeasonDrivers.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonDriversRepository) Create(
	ctx context.Context,
	input *models.SeasonDriverSetter,
) (*models.SeasonDriver, error) {
	return models.SeasonDrivers.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *seasonDriversRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeasonDriverSetter,
) (*models.SeasonDriver, error) {
	entity, err := models.SeasonDrivers.Update(
		input.UpdateMod(),
		um.Where(models.SeasonDrivers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("season driver %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *seasonDriversRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
