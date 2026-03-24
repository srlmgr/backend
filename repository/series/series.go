// Package series provides repositories for the series migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package series

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

// Repository defines persistence operations for Series entities.
type Repository interface {
	LoadAll(ctx context.Context) ([]*models.Series, error)
	LoadBySimulationID(ctx context.Context, simulationID int32) ([]*models.Series, error)
	LoadByID(ctx context.Context, id int32) (*models.Series, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.SeriesSetter) (*models.Series, error)
	Update(ctx context.Context, id int32, input *models.SeriesSetter) (*models.Series, error)
}

type seriesRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &seriesRepository{exec: pgbob.New(pool)}
}

func (r *seriesRepository) LoadAll(ctx context.Context) ([]*models.Series, error) {
	return models.Serieses.Query().All(ctx, r.getExecutor(ctx))
}

func (r *seriesRepository) LoadBySimulationID(
	ctx context.Context,
	simulationID int32,
) ([]*models.Series, error) {
	return models.Serieses.Query(
		sm.Where(models.Serieses.Columns.SimulationID.EQ(psql.Arg(simulationID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *seriesRepository) LoadByID(ctx context.Context, id int32) (*models.Series, error) {
	entity, err := models.Serieses.Query(sm.Where(models.Serieses.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("series %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *seriesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Serieses.Delete(dm.Where(models.Serieses.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seriesRepository) Create(
	ctx context.Context,
	input *models.SeriesSetter,
) (*models.Series, error) {
	return models.Serieses.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *seriesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeriesSetter,
) (*models.Series, error) {
	return models.Serieses.Update(
		input.UpdateMod(),
		um.Where(models.Serieses.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
}

func (r *seriesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
