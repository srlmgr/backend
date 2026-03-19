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

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// EventsRepository defines persistence operations for Event entities.
type EventsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.Event, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.EventSetter) (*models.Event, error)
	Update(ctx context.Context, id int32, input *models.EventSetter) (*models.Event, error)
}

// Repository exposes repositories for the events migration group.
type Repository interface {
	Events() EventsRepository
}

type (
	repository       struct{ events EventsRepository }
	eventsRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{events: &eventsRepository{exec: pgbob.New(pool)}}
}

func (r *repository) Events() EventsRepository { return r.events }

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
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *eventsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
