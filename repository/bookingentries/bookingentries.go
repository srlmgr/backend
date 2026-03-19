// Package bookingentries provides repositories for the booking_entries migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package bookingentries

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

// Repository defines persistence operations for BookingEntry entities.
type Repository interface {
	LoadByID(ctx context.Context, id int32) (*models.BookingEntry, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.BookingEntrySetter) (*models.BookingEntry, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.BookingEntrySetter,
	) (*models.BookingEntry, error)
}

type bookingEntriesRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &bookingEntriesRepository{exec: pgbob.New(pool)}
}

func (r *bookingEntriesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.BookingEntry, error) {
	entity, err := models.BookingEntries.Query(sm.Where(models.BookingEntries.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("booking entry %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *bookingEntriesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.BookingEntries.Delete(dm.Where(models.BookingEntries.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *bookingEntriesRepository) Create(
	ctx context.Context,
	input *models.BookingEntrySetter,
) (*models.BookingEntry, error) {
	return models.BookingEntries.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *bookingEntriesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.BookingEntrySetter,
) (*models.BookingEntry, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *bookingEntriesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
