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

// CarManufacturersRepository defines persistence operations
// for CarManufacturer entities.
//
//nolint:lll // readability
type CarManufacturersRepository interface {
	LoadAll(context.Context) ([]*models.CarManufacturer, error)
	LoadByID(context.Context, int32) (*models.CarManufacturer, error)
	DeleteByID(context.Context, int32) error
	Create(context.Context, *models.CarManufacturerSetter) (*models.CarManufacturer, error)
	Update(context.Context, int32, *models.CarManufacturerSetter) (*models.CarManufacturer, error)
}

//nolint:whitespace // editor/linter issue
func (r *carManufacturersRepository) LoadAll(
	ctx context.Context,
) ([]*models.CarManufacturer, error) {
	return models.CarManufacturers.Query().All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carManufacturersRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarManufacturer, error) {
	entity, err := models.CarManufacturers.Query(
		sm.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car manufacturer %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carManufacturersRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarManufacturers.Delete(
		dm.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *carManufacturersRepository) Create(
	ctx context.Context,
	input *models.CarManufacturerSetter,
) (*models.CarManufacturer, error) {
	return models.CarManufacturers.Insert(input).One(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *carManufacturersRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarManufacturerSetter,
) (*models.CarManufacturer, error) {
	entity, err := models.CarManufacturers.Update(
		input.UpdateMod(),
		um.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car manufacturer %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carManufacturersRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
