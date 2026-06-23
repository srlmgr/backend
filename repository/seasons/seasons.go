// Package seasons provides repositories for the seasons migration group.
//
//nolint:lll,whitespace,dupl // repository implementations can be verbose
package seasons

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/opt/omit"
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

// Repository defines persistence operations for Season entities.
type Repository interface {
	LoadAll(ctx context.Context) ([]*models.Season, error)
	LoadActiveAt(ctx context.Context, refDate time.Time) ([]*models.Season, error)
	LoadBySeriesID(ctx context.Context, seriesID int32) ([]*models.Season, error)
	LoadByID(ctx context.Context, id int32) (*models.Season, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.SeasonSetter) (*models.Season, error)
	Update(ctx context.Context, id int32, input *models.SeasonSetter) (*models.Season, error)
	SetCarClasses(ctx context.Context, seasonID int32, carClassIDs []int32) error
	SetCarModels(ctx context.Context, seasonID int32, carModelIDs []int32) error

	DeleteCarClasses(ctx context.Context, seasonID int32) error
	DeleteCarModels(ctx context.Context, seasonID int32) error
}

type seasonsRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &seasonsRepository{exec: pgbob.New(pool)}
}

func (r *seasonsRepository) LoadAll(ctx context.Context) ([]*models.Season, error) {
	return models.Seasons.Query().All(ctx, r.getExecutor(ctx))
}

func (r *seasonsRepository) LoadActiveAt(
	ctx context.Context,
	refDate time.Time,
) ([]*models.Season, error) {
	return models.Seasons.Query(
		sm.Where(
			models.Seasons.Columns.StartsAt.IsNull().
				Or(models.Seasons.Columns.StartsAt.LTE(psql.Arg(refDate))),
		),
		sm.Where(
			models.Seasons.Columns.EndsAt.IsNull().
				Or(models.Seasons.Columns.EndsAt.GTE(psql.Arg(refDate))),
		),
	).All(ctx, r.getExecutor(ctx))
}

func (r *seasonsRepository) LoadBySeriesID(
	ctx context.Context,
	seriesID int32,
) ([]*models.Season, error) {
	return models.Seasons.Query(
		sm.Where(models.Seasons.Columns.SeriesID.EQ(psql.Arg(seriesID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *seasonsRepository) LoadByID(ctx context.Context, id int32) (*models.Season, error) {
	entity, err := models.Seasons.Query(sm.Where(models.Seasons.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("season %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *seasonsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Seasons.Delete(dm.Where(models.Seasons.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonsRepository) Create(
	ctx context.Context,
	input *models.SeasonSetter,
) (*models.Season, error) {
	return models.Seasons.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *seasonsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeasonSetter,
) (*models.Season, error) {
	entity, err := models.Seasons.Update(
		input.UpdateMod(),
		um.Where(models.Seasons.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("season %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *seasonsRepository) SetCarClasses(
	ctx context.Context,
	seasonID int32,
	carClassIDs []int32,
) error {
	// Delete existing car classes for the season
	if err := r.DeleteCarClasses(ctx, seasonID); err != nil {
		return err
	}

	setters := make([]*models.SeasonCarClassSetter, len(carClassIDs))
	for i, carClassID := range carClassIDs {
		setters[i] = &models.SeasonCarClassSetter{
			SeasonID:   omit.From(seasonID),
			CarClassID: omit.From(carClassID),
			Pos:        omit.From(int32(i)), // Set position based on the order in the input slice
		}
	}

	if len(setters) == 0 {
		return nil
	}

	_, err := models.SeasonCarClasses.Insert(bob.ToMods(setters...)).All(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonsRepository) SetCarModels(
	ctx context.Context,
	seasonID int32,
	carModelIDs []int32,
) error {
	// Delete existing car models for the season
	if err := r.DeleteCarModels(ctx, seasonID); err != nil {
		return err
	}

	setters := make([]*models.SeasonCarModelSetter, len(carModelIDs))
	for i, carModelID := range carModelIDs {
		setters[i] = &models.SeasonCarModelSetter{
			SeasonID:   omit.From(seasonID),
			CarModelID: omit.From(carModelID),
			Pos:        omit.From(int32(i)), // Set position based on the order in the input slice
		}
	}

	if len(setters) == 0 {
		return nil
	}

	_, err := models.SeasonCarModels.Insert(bob.ToMods(setters...)).All(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *seasonsRepository) DeleteCarClasses(
	ctx context.Context,
	seasonID int32,
) error {
	_, err := models.SeasonCarClasses.Delete(
		dm.Where(models.SeasonCarClasses.Columns.SeasonID.EQ(psql.Arg(seasonID))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *seasonsRepository) DeleteCarModels(
	ctx context.Context,
	seasonID int32,
) error {
	_, err := models.SeasonCarModels.Delete(
		dm.Where(models.SeasonCarModels.Columns.SeasonID.EQ(psql.Arg(seasonID))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
