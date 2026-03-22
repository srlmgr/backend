// Package resultentries provides repositories for the result_entries migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package resultentries

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

// Repository defines persistence operations for ResultEntry entities.
type Repository interface {
	LoadAll(ctx context.Context) ([]*models.ResultEntry, error)
	LoadByID(ctx context.Context, id int32) (*models.ResultEntry, error)
	LoadByRaceID(ctx context.Context, raceID int32) ([]*models.ResultEntry, error)
	LoadByImportBatchID(ctx context.Context, importBatchID int32) ([]*models.ResultEntry, error)
	LoadByState(ctx context.Context, state string) ([]*models.ResultEntry, error)
	DeleteByID(ctx context.Context, id int32) error
	DeleteByRaceID(ctx context.Context, raceID int32) error
	Create(ctx context.Context, input *models.ResultEntrySetter) (*models.ResultEntry, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.ResultEntrySetter,
	) (*models.ResultEntry, error)
}

type resultEntriesRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &resultEntriesRepository{exec: pgbob.New(pool)}
}

func (r *resultEntriesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.ResultEntry, error) {
	entity, err := models.ResultEntries.Query(sm.Where(models.ResultEntries.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("result entry %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *resultEntriesRepository) LoadAll(ctx context.Context) ([]*models.ResultEntry, error) {
	return models.ResultEntries.Query().All(ctx, r.getExecutor(ctx))
}

func (r *resultEntriesRepository) LoadByRaceID(
	ctx context.Context,
	raceID int32,
) ([]*models.ResultEntry, error) {
	return models.ResultEntries.Query(
		sm.Where(models.ResultEntries.Columns.RaceID.EQ(psql.Arg(raceID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *resultEntriesRepository) LoadByImportBatchID(
	ctx context.Context,
	importBatchID int32,
) ([]*models.ResultEntry, error) {
	return models.ResultEntries.Query(
		sm.Where(models.ResultEntries.Columns.ImportBatchID.EQ(psql.Arg(importBatchID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *resultEntriesRepository) LoadByState(
	ctx context.Context,
	state string,
) ([]*models.ResultEntry, error) {
	return models.ResultEntries.Query(
		sm.Where(models.ResultEntries.Columns.State.EQ(psql.Arg(state))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *resultEntriesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.ResultEntries.Delete(dm.Where(models.ResultEntries.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *resultEntriesRepository) DeleteByRaceID(ctx context.Context, raceID int32) error {
	_, err := models.ResultEntries.Delete(dm.Where(models.ResultEntries.Columns.RaceID.EQ(psql.Arg(raceID)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *resultEntriesRepository) Create(
	ctx context.Context,
	input *models.ResultEntrySetter,
) (*models.ResultEntry, error) {
	return models.ResultEntries.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *resultEntriesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.ResultEntrySetter,
) (*models.ResultEntry, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *resultEntriesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
