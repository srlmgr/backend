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
	"github.com/stephenafamo/bob/dialect/psql/um"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// Repository defines persistence operations for BookingEntry entities.
type Repository interface {
	LoadByID(ctx context.Context, id int32) (*models.BookingEntry, error)
	LoadByEventID(ctx context.Context, eventID int32) ([]*models.BookingEntry, error)
	DeleteByID(ctx context.Context, id int32) error
	DeleteByEventIDAndTargetType(ctx context.Context, eventID int32, targetType string) error
	DeleteByEventIDAndSourceType(ctx context.Context, eventID int32, sourceType string) error
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
	return models.BookingEntries.Update(
		input.UpdateMod(),
		um.Where(models.BookingEntries.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
}

func (r *bookingEntriesRepository) LoadByEventID(
	ctx context.Context,
	eventID int32,
) ([]*models.BookingEntry, error) {
	return models.BookingEntries.Query(
		sm.Where(models.BookingEntries.Columns.EventID.EQ(psql.Arg(eventID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *bookingEntriesRepository) DeleteByEventIDAndTargetType(
	ctx context.Context,
	eventID int32,
	targetType string,
) error {
	_, err := models.BookingEntries.Delete(
		dm.Where(models.BookingEntries.Columns.EventID.EQ(psql.Arg(eventID))),
		dm.Where(models.BookingEntries.Columns.TargetType.EQ(psql.Arg(targetType))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *bookingEntriesRepository) DeleteByEventIDAndSourceType(
	ctx context.Context,
	eventID int32,
	sourceType string,
) error {
	_, err := models.BookingEntries.Delete(
		dm.Where(models.BookingEntries.Columns.EventID.EQ(psql.Arg(eventID))),
		dm.Where(models.BookingEntries.Columns.SourceType.EQ(psql.Arg(sourceType))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *bookingEntriesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
