// Package importbatches provides repositories for the import_batches migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package importbatches

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

// Repository defines persistence operations for ImportBatch entities.
type Repository interface {
	LoadByID(ctx context.Context, id int32) (*models.ImportBatch, error)
	LoadByEventIDAndRaceID(
		ctx context.Context,
		eventID, raceID int32,
	) ([]*models.ImportBatch, error)
	LoadByRaceID(
		ctx context.Context,
		raceID int32,
	) (*models.ImportBatch, error)
	LoadLatestByEventIDAndRaceID(
		ctx context.Context,
		eventID, raceID int32,
	) (*models.ImportBatch, error)
	DeleteByID(ctx context.Context, id int32) error
	DeleteByRaceID(ctx context.Context, raceID int32) error
	Create(ctx context.Context, input *models.ImportBatchSetter) (*models.ImportBatch, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.ImportBatchSetter,
	) (*models.ImportBatch, error)
}

type importBatchesRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &importBatchesRepository{exec: pgbob.New(pool)}
}

func (r *importBatchesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.ImportBatch, error) {
	entity, err := models.ImportBatches.Query(sm.Where(models.ImportBatches.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("import batch %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *importBatchesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.ImportBatches.Delete(dm.Where(models.ImportBatches.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *importBatchesRepository) DeleteByRaceID(ctx context.Context, raceID int32) error {
	_, err := models.ImportBatches.Delete(dm.Where(models.ImportBatches.Columns.RaceID.EQ(psql.Arg(raceID)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *importBatchesRepository) Create(
	ctx context.Context,
	input *models.ImportBatchSetter,
) (*models.ImportBatch, error) {
	return models.ImportBatches.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *importBatchesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.ImportBatchSetter,
) (*models.ImportBatch, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *importBatchesRepository) LoadByEventIDAndRaceID(
	ctx context.Context,
	eventID, raceID int32,
) ([]*models.ImportBatch, error) {
	return models.ImportBatches.Query(
		sm.Where(models.ImportBatches.Columns.EventID.EQ(psql.Arg(eventID))),
		sm.Where(models.ImportBatches.Columns.RaceID.EQ(psql.Arg(raceID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *importBatchesRepository) LoadByRaceID(
	ctx context.Context,
	raceID int32,
) (*models.ImportBatch, error) {
	entity, err := models.ImportBatches.Query(
		sm.Where(models.ImportBatches.Columns.RaceID.EQ(psql.Arg(raceID))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("import batch for race %d: %w", raceID, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *importBatchesRepository) LoadLatestByEventIDAndRaceID(
	ctx context.Context,
	eventID, raceID int32,
) (*models.ImportBatch, error) {
	entity, err := models.ImportBatches.Query(
		sm.Where(models.ImportBatches.Columns.EventID.EQ(psql.Arg(eventID))),
		sm.Where(models.ImportBatches.Columns.RaceID.EQ(psql.Arg(raceID))),
		sm.OrderBy(models.ImportBatches.Columns.ID).Desc(),
		sm.Limit(1),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"import batch for event %d race %d: %w",
			eventID,
			raceID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *importBatchesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
