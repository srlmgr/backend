//nolint:lll,dupl // readability,duplicate:false positive
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

// CarModelVariantsRepository defines persistence operations
// for CarModelVariant entities.
//
//nolint:lll // readability
type CarModelVariantsRepository interface {
	LoadAll(context.Context) ([]*models.CarModelVariant, error)
	LoadByManufacturerID(context.Context, int32) ([]*models.CarModelVariant, error)
	LoadByCarClassID(context.Context, int32) ([]*models.CarModelVariant, error)
	LoadBySeasonID(context.Context, int32) ([]*models.CarModelVariant, error)
	LoadByModelID(context.Context, int32) ([]*models.CarModelVariant, error)
	LoadByID(context.Context, int32) (*models.CarModelVariant, error)
	DeleteByID(context.Context, int32) error
	Create(context.Context, *models.CarModelVariantSetter) (*models.CarModelVariant, error)
	Update(context.Context, int32, *models.CarModelVariantSetter) (*models.CarModelVariant, error)
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) LoadAll(
	ctx context.Context,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query().All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) LoadByManufacturerID(
	ctx context.Context,
	id int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.CarModelID.In(
			psql.Select(
				sm.Columns(models.CarModels.Columns.ID),
				sm.From(models.CarModels.Name()),
				sm.Where(models.CarModels.Columns.ManufacturerID.EQ(psql.Arg(id))),
			))),
	).All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) LoadByCarClassID(
	ctx context.Context,
	id int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.ID.In(
			psql.Select(
				sm.Columns(models.CarClassesToCarModels.Columns.CarModelVariantID),
				sm.From(models.CarClassesToCarModels.Name()),
				sm.Where(models.CarClassesToCarModels.Columns.CarClassID.EQ(psql.Arg(id))),
			))),
	).All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) LoadBySeasonID(
	ctx context.Context,
	id int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.InnerJoin(
			models.SeasonCarModelVariants.Name()).On(
			models.SeasonCarModelVariants.Columns.CarModelVariantID.
				EQ(models.CarModelVariants.Columns.ID)),
		sm.Where(models.SeasonCarModelVariants.Columns.SeasonID.EQ(psql.Arg(id))),
		sm.OrderBy(models.SeasonCarModelVariants.Columns.Pos).Asc(),
	).All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) LoadByModelID(
	ctx context.Context,
	id int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.CarModelID.EQ(psql.Arg(id))),
	).All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarModelVariant, error) {
	entity, err := models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model variant %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelVariantsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarModelVariants.Delete(
		dm.Where(models.CarModelVariants.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) Create(
	ctx context.Context,
	input *models.CarModelVariantSetter,
) (*models.CarModelVariant, error) {
	return models.CarModelVariants.Insert(input).One(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carModelVariantsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarModelVariantSetter,
) (*models.CarModelVariant, error) {
	entity, err := models.CarModelVariants.Update(
		input.UpdateMod(),
		um.Where(models.CarModelVariants.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model variant %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelVariantsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
