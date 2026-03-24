// Package events provides repositories for the events migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package events

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

// Repository defines persistence operations for Event entities.
type Repository interface {
	LoadAll(ctx context.Context) ([]*models.Event, error)
	LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.Event, error)
	LoadByID(ctx context.Context, id int32) (*models.Event, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.EventSetter) (*models.Event, error)
	Update(ctx context.Context, id int32, input *models.EventSetter) (*models.Event, error)
}

type eventsRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &eventsRepository{exec: pgbob.New(pool)}
}

func (r *eventsRepository) LoadAll(ctx context.Context) ([]*models.Event, error) {
	return models.Events.Query().All(ctx, r.getExecutor(ctx))
}

func (r *eventsRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.Event, error) {
	return models.Events.Query(
		sm.Where(models.Events.Columns.SeasonID.EQ(psql.Arg(seasonID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *eventsRepository) LoadByID(ctx context.Context, id int32) (*models.Event, error) {
	entity, err := models.Events.Query(sm.Where(models.Events.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("event %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *eventsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Events.Delete(dm.Where(models.Events.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *eventsRepository) Create(
	ctx context.Context,
	input *models.EventSetter,
) (*models.Event, error) {
	return models.Events.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *eventsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.EventSetter,
) (*models.Event, error) {
	return models.Events.Update(
		input.UpdateMod(),
		um.Where(models.Events.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
}

func (r *eventsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
