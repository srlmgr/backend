// Package tracks provides repositories for the tracks migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package tracks

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

// TracksRepository defines persistence operations for Track entities.
type TracksRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.Track, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.TrackSetter) (*models.Track, error)
	Update(ctx context.Context, id int32, input *models.TrackSetter) (*models.Track, error)
}

// TrackLayoutsRepository defines persistence operations for TrackLayout entities.
type TrackLayoutsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.TrackLayout, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.TrackLayoutSetter) (*models.TrackLayout, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.TrackLayoutSetter,
	) (*models.TrackLayout, error)
}

// SimulationTrackLayoutAliasesRepository defines persistence operations for SimulationTrackLayoutAlias entities.
type SimulationTrackLayoutAliasesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SimulationTrackLayoutAlias, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.SimulationTrackLayoutAliasSetter,
	) (*models.SimulationTrackLayoutAlias, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SimulationTrackLayoutAliasSetter,
	) (*models.SimulationTrackLayoutAlias, error)
}

// Repository exposes repositories for the tracks migration group.
type Repository interface {
	Tracks() TracksRepository
	TrackLayouts() TrackLayoutsRepository
	SimulationTrackLayoutAliases() SimulationTrackLayoutAliasesRepository
}

type repository struct {
	tracks                       TracksRepository
	trackLayouts                 TrackLayoutsRepository
	simulationTrackLayoutAliases SimulationTrackLayoutAliasesRepository
}

type (
	tracksRepository                       struct{ exec *pgbob.Executor }
	trackLayoutsRepository                 struct{ exec *pgbob.Executor }
	simulationTrackLayoutAliasesRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		tracks:       &tracksRepository{exec: pgbob.New(pool)},
		trackLayouts: &trackLayoutsRepository{exec: pgbob.New(pool)},
		simulationTrackLayoutAliases: &simulationTrackLayoutAliasesRepository{
			exec: pgbob.New(pool),
		},
	}
}

func (r *repository) Tracks() TracksRepository             { return r.tracks }
func (r *repository) TrackLayouts() TrackLayoutsRepository { return r.trackLayouts }
func (r *repository) SimulationTrackLayoutAliases() SimulationTrackLayoutAliasesRepository {
	return r.simulationTrackLayoutAliases
}

func (r *tracksRepository) LoadByID(ctx context.Context, id int32) (*models.Track, error) {
	entity, err := models.Tracks.Query(sm.Where(models.Tracks.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("track %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *tracksRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Tracks.Delete(dm.Where(models.Tracks.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *tracksRepository) Create(
	ctx context.Context,
	input *models.TrackSetter,
) (*models.Track, error) {
	return models.Tracks.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *tracksRepository) Update(
	ctx context.Context,
	id int32,
	input *models.TrackSetter,
) (*models.Track, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *trackLayoutsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.TrackLayout, error) {
	entity, err := models.TrackLayouts.Query(sm.Where(models.TrackLayouts.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("track layout %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *trackLayoutsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.TrackLayouts.Delete(dm.Where(models.TrackLayouts.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *trackLayoutsRepository) Create(
	ctx context.Context,
	input *models.TrackLayoutSetter,
) (*models.TrackLayout, error) {
	return models.TrackLayouts.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *trackLayoutsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.TrackLayoutSetter,
) (*models.TrackLayout, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *simulationTrackLayoutAliasesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SimulationTrackLayoutAlias, error) {
	entity, err := models.SimulationTrackLayoutAliases.Query(sm.Where(models.SimulationTrackLayoutAliases.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation track layout alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationTrackLayoutAliasesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SimulationTrackLayoutAliases.Delete(dm.Where(models.SimulationTrackLayoutAliases.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationTrackLayoutAliasesRepository) Create(
	ctx context.Context,
	input *models.SimulationTrackLayoutAliasSetter,
) (*models.SimulationTrackLayoutAlias, error) {
	return models.SimulationTrackLayoutAliases.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *simulationTrackLayoutAliasesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SimulationTrackLayoutAliasSetter,
) (*models.SimulationTrackLayoutAlias, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *tracksRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *trackLayoutsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *simulationTrackLayoutAliasesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
