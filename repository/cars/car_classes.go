//nolint:lll,dupl // readability,duplicate:false positive
package cars

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// CarClassesRepository defines persistence operations for CarClass entities.
type CarClassesRepository interface {
	LoadAll(context.Context) ([]*models.CarClass, error)
	LoadByID(context.Context, int32) (*models.CarClass, error)
	LoadBySeasonID(context.Context, int32) ([]*models.CarClass, error)
	DeleteByID(context.Context, int32) error
	Create(context.Context, *models.CarClassSetter) (*models.CarClass, error)
	Update(context.Context, int32, *models.CarClassSetter) (*models.CarClass, error)
	AssignCarModelVariant(context.Context, int32, int32) error
	UnassignCarModelVariant(context.Context, int32, int32) error
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) LoadAll(
	ctx context.Context,
) ([]*models.CarClass, error) {
	return models.CarClasses.Query().All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) LoadByID(
	ctx context.Context, id int32,
) (*models.CarClass, error) {
	entity, err := models.CarClasses.Query(
		sm.Where(models.CarClasses.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car class %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) LoadBySeasonID(
	ctx context.Context,
	id int32,
) ([]*models.CarClass, error) {
	return models.CarClasses.Query(
		sm.InnerJoin(models.SeasonCarClasses.Name()).
			On(models.SeasonCarClasses.Columns.CarClassID.EQ(models.CarClasses.Columns.ID)),
		sm.Where(models.SeasonCarClasses.Columns.SeasonID.EQ(psql.Arg(id))),
		sm.OrderBy(models.SeasonCarClasses.Columns.Pos).Asc(),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carClassesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarClasses.Delete(
		dm.Where(
			models.CarClasses.Columns.ID.EQ(psql.Arg(id)),
		),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) Create(
	ctx context.Context,
	input *models.CarClassSetter,
) (*models.CarClass, error) {
	return models.CarClasses.Insert(input).One(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarClassSetter,
) (*models.CarClass, error) {
	entity, err := models.CarClasses.Update(
		input.UpdateMod(), um.Where(models.CarClasses.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car class %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) AssignCarModelVariant(
	ctx context.Context,
	classID, variantID int32,
) error {
	if _, err := models.CarClassesToCarModels.Query(
		sm.Where(models.CarClassesToCarModels.Columns.CarClassID.EQ(psql.Arg(classID))),
		sm.Where(models.CarClassesToCarModels.Columns.CarModelVariantID.EQ(psql.Arg(variantID))),
	).One(ctx, r.getExecutor(ctx)); err == nil {
		return nil
	}
	_, err := models.CarClassesToCarModels.Insert(
		&models.CarClassesToCarModelSetter{
			CarClassID:        omit.From(classID),
			CarModelVariantID: omit.From(variantID),
		},
	).One(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) UnassignCarModelVariant(
	ctx context.Context,
	classID, variantID int32,
) error {
	_, err := models.CarClassesToCarModels.Delete(
		dm.Where(models.CarClassesToCarModels.Columns.CarClassID.EQ(psql.Arg(classID))),
		dm.Where(models.CarClassesToCarModels.Columns.CarModelVariantID.EQ(psql.Arg(variantID))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *carClassesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
