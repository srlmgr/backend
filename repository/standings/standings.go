// Package standings provides repositories for the standings migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package standings

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

// SeasonDriverStandingsRepository defines persistence operations for SeasonDriverStanding entities.
type SeasonDriverStandingsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SeasonDriverStanding, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.SeasonDriverStandingSetter,
	) (*models.SeasonDriverStanding, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SeasonDriverStandingSetter,
	) (*models.SeasonDriverStanding, error)
}

// SeasonTeamStandingsRepository defines persistence operations for SeasonTeamStanding entities.
type SeasonTeamStandingsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SeasonTeamStanding, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.SeasonTeamStandingSetter,
	) (*models.SeasonTeamStanding, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SeasonTeamStandingSetter,
	) (*models.SeasonTeamStanding, error)
}

// EventDriverStandingsRepository defines persistence operations for EventDriverStanding entities.
type EventDriverStandingsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.EventDriverStanding, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.EventDriverStandingSetter,
	) (*models.EventDriverStanding, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.EventDriverStandingSetter,
	) (*models.EventDriverStanding, error)
}

// EventTeamStandingsRepository defines persistence operations for EventTeamStanding entities.
type EventTeamStandingsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.EventTeamStanding, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.EventTeamStandingSetter,
	) (*models.EventTeamStanding, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.EventTeamStandingSetter,
	) (*models.EventTeamStanding, error)
}

// Repository exposes repositories for the standings migration group.
type Repository interface {
	SeasonDriverStandings() SeasonDriverStandingsRepository
	SeasonTeamStandings() SeasonTeamStandingsRepository
	EventDriverStandings() EventDriverStandingsRepository
	EventTeamStandings() EventTeamStandingsRepository
}

type repository struct {
	seasonDriverStandings SeasonDriverStandingsRepository
	seasonTeamStandings   SeasonTeamStandingsRepository
	eventDriverStandings  EventDriverStandingsRepository
	eventTeamStandings    EventTeamStandingsRepository
}

type (
	seasonDriverStandingsRepository struct{ exec *pgbob.Executor }
	seasonTeamStandingsRepository   struct{ exec *pgbob.Executor }
	eventDriverStandingsRepository  struct{ exec *pgbob.Executor }
	eventTeamStandingsRepository    struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		seasonDriverStandings: &seasonDriverStandingsRepository{exec: pgbob.New(pool)},
		seasonTeamStandings:   &seasonTeamStandingsRepository{exec: pgbob.New(pool)},
		eventDriverStandings:  &eventDriverStandingsRepository{exec: pgbob.New(pool)},
		eventTeamStandings:    &eventTeamStandingsRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) SeasonDriverStandings() SeasonDriverStandingsRepository {
	return r.seasonDriverStandings
}

func (r *repository) SeasonTeamStandings() SeasonTeamStandingsRepository {
	return r.seasonTeamStandings
}

func (r *repository) EventDriverStandings() EventDriverStandingsRepository {
	return r.eventDriverStandings
}

func (r *repository) EventTeamStandings() EventTeamStandingsRepository { return r.eventTeamStandings }

func (r *seasonDriverStandingsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SeasonDriverStanding, error) {
	entity, err := models.SeasonDriverStandings.Query(sm.Where(models.SeasonDriverStandings.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("season driver standing %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *seasonDriverStandingsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SeasonDriverStandings.Delete(dm.Where(models.SeasonDriverStandings.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonDriverStandingsRepository) Create(
	ctx context.Context,
	input *models.SeasonDriverStandingSetter,
) (*models.SeasonDriverStanding, error) {
	return models.SeasonDriverStandings.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *seasonDriverStandingsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeasonDriverStandingSetter,
) (*models.SeasonDriverStanding, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *seasonTeamStandingsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SeasonTeamStanding, error) {
	entity, err := models.SeasonTeamStandings.Query(sm.Where(models.SeasonTeamStandings.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("season team standing %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *seasonTeamStandingsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SeasonTeamStandings.Delete(dm.Where(models.SeasonTeamStandings.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *seasonTeamStandingsRepository) Create(
	ctx context.Context,
	input *models.SeasonTeamStandingSetter,
) (*models.SeasonTeamStanding, error) {
	return models.SeasonTeamStandings.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *seasonTeamStandingsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeasonTeamStandingSetter,
) (*models.SeasonTeamStanding, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *eventDriverStandingsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.EventDriverStanding, error) {
	entity, err := models.EventDriverStandings.Query(sm.Where(models.EventDriverStandings.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("event driver standing %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *eventDriverStandingsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.EventDriverStandings.Delete(dm.Where(models.EventDriverStandings.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *eventDriverStandingsRepository) Create(
	ctx context.Context,
	input *models.EventDriverStandingSetter,
) (*models.EventDriverStanding, error) {
	return models.EventDriverStandings.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *eventDriverStandingsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.EventDriverStandingSetter,
) (*models.EventDriverStanding, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *eventTeamStandingsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.EventTeamStanding, error) {
	entity, err := models.EventTeamStandings.Query(sm.Where(models.EventTeamStandings.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("event team standing %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *eventTeamStandingsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.EventTeamStandings.Delete(dm.Where(models.EventTeamStandings.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *eventTeamStandingsRepository) Create(
	ctx context.Context,
	input *models.EventTeamStandingSetter,
) (*models.EventTeamStanding, error) {
	return models.EventTeamStandings.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *eventTeamStandingsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.EventTeamStandingSetter,
) (*models.EventTeamStanding, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *seasonDriverStandingsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *seasonTeamStandingsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *eventDriverStandingsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *eventTeamStandingsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
