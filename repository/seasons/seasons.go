// Package seasons provides repositories for the seasons migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package seasons

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	LoadBySeriesID(ctx context.Context, seriesID int32) ([]*models.Season, error)
	LoadByID(ctx context.Context, id int32) (*models.Season, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.SeasonSetter) (*models.Season, error)
	Update(ctx context.Context, id int32, input *models.SeasonSetter) (*models.Season, error)
	AssignCarClass(ctx context.Context, seasonID, carClassID int32) error
	UnassignCarClass(ctx context.Context, seasonID, carClassID int32) error
}

type seasonsRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &seasonsRepository{exec: pgbob.New(pool)}
}

func (r *seasonsRepository) LoadAll(ctx context.Context) ([]*models.Season, error) {
	return models.Seasons.Query().All(ctx, r.getExecutor(ctx))
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
func (r *seasonsRepository) AssignCarClass(
	ctx context.Context,
	seasonID, carClassID int32,
) error {
	if _, checkErr := models.SeasonCarClasses.Query(
		sm.Where(models.SeasonCarClasses.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		sm.Where(models.SeasonCarClasses.Columns.CarClassID.EQ(psql.Arg(carClassID))),
	).One(ctx, r.getExecutor(ctx)); checkErr == nil {
		return nil
	}
	_, err := models.SeasonCarClasses.Insert(&models.SeasonCarClassSetter{
		SeasonID:   omit.From(seasonID),
		CarClassID: omit.From(carClassID),
	}).One(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *seasonsRepository) UnassignCarClass(
	ctx context.Context,
	seasonID, carClassID int32,
) error {
	_, err := models.SeasonCarClasses.Delete(
		dm.Where(models.SeasonCarClasses.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		dm.Where(models.SeasonCarClasses.Columns.CarClassID.EQ(psql.Arg(carClassID))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
