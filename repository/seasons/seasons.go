// Package seasons provides repositories for the seasons migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package seasons

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

// SeasonsRepository defines persistence operations for Season entities.
type SeasonsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.Season, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.SeasonSetter) (*models.Season, error)
	Update(ctx context.Context, id int32, input *models.SeasonSetter) (*models.Season, error)
}

// Repository exposes repositories for the seasons migration group.
type Repository interface {
	Seasons() SeasonsRepository
}

type (
	repository        struct{ seasons SeasonsRepository }
	seasonsRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{seasons: &seasonsRepository{exec: pgbob.New(pool)}}
}

func (r *repository) Seasons() SeasonsRepository { return r.seasons }

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
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *seasonsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
