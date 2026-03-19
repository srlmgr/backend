// Package races provides repositories for the races migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package races

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

// RacesRepository defines persistence operations for Race entities.
type RacesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.Race, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.RaceSetter) (*models.Race, error)
	Update(ctx context.Context, id int32, input *models.RaceSetter) (*models.Race, error)
}

// Repository exposes repositories for the races migration group.
type Repository interface {
	Races() RacesRepository
}

type (
	repository      struct{ races RacesRepository }
	racesRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{races: &racesRepository{exec: pgbob.New(pool)}}
}

func (r *repository) Races() RacesRepository { return r.races }

func (r *racesRepository) LoadByID(ctx context.Context, id int32) (*models.Race, error) {
	entity, err := models.Races.Query(sm.Where(models.Races.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("race %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *racesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Races.Delete(dm.Where(models.Races.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *racesRepository) Create(
	ctx context.Context,
	input *models.RaceSetter,
) (*models.Race, error) {
	return models.Races.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *racesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.RaceSetter,
) (*models.Race, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *racesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
