package cars

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

// CarModelsRepository defines persistence operations for CarModel entities.
type CarModelsRepository interface {
	LoadAll(context.Context) ([]*models.CarModel, error)
	LoadByManufacturerID(context.Context, int32) ([]*models.CarModel, error)
	LoadByID(context.Context, int32) (*models.CarModel, error)
	DeleteByID(context.Context, int32) error
	Create(context.Context, *models.CarModelSetter) (*models.CarModel, error)
	Update(context.Context, int32, *models.CarModelSetter) (*models.CarModel, error)
}

func (r *carModelsRepository) LoadAll(ctx context.Context) ([]*models.CarModel, error) {
	return models.CarModels.Query().All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelsRepository) LoadByManufacturerID(
	ctx context.Context,
	id int32,
) ([]*models.CarModel, error) {
	return models.CarModels.Query(
		sm.Where(models.CarModels.Columns.ManufacturerID.EQ(psql.Arg(id))),
	).
		All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelsRepository) LoadByID(
	ctx context.Context, id int32,
) (*models.CarModel, error) {
	entity, err := models.CarModels.Query(
		sm.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model v2 %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarModels.Delete(
		dm.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *carModelsRepository) Create(
	ctx context.Context,
	input *models.CarModelSetter,
) (*models.CarModel, error) {
	return models.CarModels.Insert(input).One(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarModelSetter,
) (*models.CarModel, error) {
	entity, err := models.CarModels.Update(
		input.UpdateMod(),
		um.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model  %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
