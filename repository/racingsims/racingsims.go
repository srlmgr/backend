// Package racingsims provides repositories for the racing_sims migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package racingsims

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

// RacingSimsRepository defines persistence operations for RacingSim entities.
type Repository interface {
	LoadByID(ctx context.Context, id int32) (*models.RacingSim, error)
	LoadAll(ctx context.Context) ([]*models.RacingSim, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.RacingSimSetter) (*models.RacingSim, error)
	Update(ctx context.Context, id int32, input *models.RacingSimSetter) (*models.RacingSim, error)
}

type racingSimsRepository struct {
	exec *pgbob.Executor
}

var _ Repository = (*racingSimsRepository)(nil)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &racingSimsRepository{exec: pgbob.New(pool)}
}

func (r *racingSimsRepository) LoadByID(ctx context.Context, id int32) (*models.RacingSim, error) {
	entity, err := models.RacingSims.Query(
		sm.Where(models.RacingSims.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("racing sim %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *racingSimsRepository) LoadAll(ctx context.Context) ([]*models.RacingSim, error) {
	return models.RacingSims.Query().All(ctx, r.getExecutor(ctx))
}

func (r *racingSimsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.RacingSims.Delete(
		dm.Where(models.RacingSims.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *racingSimsRepository) Create(
	ctx context.Context,
	input *models.RacingSimSetter,
) (*models.RacingSim, error) {
	return models.RacingSims.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *racingSimsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.RacingSimSetter,
) (*models.RacingSim, error) {
	entity, err := models.RacingSims.Update(
		input.UpdateMod(),
		um.Where(models.RacingSims.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("racing sim %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *racingSimsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
