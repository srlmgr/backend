// Package races provides repositories for the races migration group.
//
//nolint:lll,dupl,whitespace // repository implementations can be verbose
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
	"github.com/stephenafamo/bob/dialect/psql/um"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// RacesRepository defines persistence operations for Race entities.
type RacesRepository interface {
	LoadAll(ctx context.Context) ([]*models.Race, error)
	LoadByEventID(ctx context.Context, eventID int32) ([]*models.Race, error)
	LoadByID(ctx context.Context, id int32) (*models.Race, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.RaceSetter) (*models.Race, error)
	Update(ctx context.Context, id int32, input *models.RaceSetter) (*models.Race, error)
}

// RaceGridsRepository defines persistence operations for RaceGrid entities.
type RaceGridsRepository interface {
	LoadAll(ctx context.Context) ([]*models.RaceGrid, error)
	LoadByRaceID(ctx context.Context, raceID int32) ([]*models.RaceGrid, error)
	LoadByID(ctx context.Context, id int32) (*models.RaceGrid, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.RaceGridSetter) (*models.RaceGrid, error)
	Update(ctx context.Context, id int32, input *models.RaceGridSetter) (*models.RaceGrid, error)
}

// Repository exposes repositories for the races migration group.
type Repository interface {
	Races() RacesRepository
	RaceGrids() RaceGridsRepository
}

type repository struct {
	races     RacesRepository
	raceGrids RaceGridsRepository
}

type (
	racesRepository     struct{ exec *pgbob.Executor }
	raceGridsRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		races:     &racesRepository{exec: pgbob.New(pool)},
		raceGrids: &raceGridsRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) Races() RacesRepository         { return r.races }
func (r *repository) RaceGrids() RaceGridsRepository { return r.raceGrids }

func (r *racesRepository) LoadAll(ctx context.Context) ([]*models.Race, error) {
	return models.Races.Query().All(ctx, r.getExecutor(ctx))
}

func (r *racesRepository) LoadByEventID(
	ctx context.Context,
	eventID int32,
) ([]*models.Race, error) {
	return models.Races.Query(
		sm.Where(models.Races.Columns.EventID.EQ(psql.Arg(eventID))),
	).All(ctx, r.getExecutor(ctx))
}

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
	entity, err := models.Races.Update(
		input.UpdateMod(),
		um.Where(models.Races.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("race %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *racesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *raceGridsRepository) LoadAll(ctx context.Context) ([]*models.RaceGrid, error) {
	return models.RaceGrids.Query().All(ctx, r.getExecutor(ctx))
}

func (r *raceGridsRepository) LoadByRaceID(
	ctx context.Context,
	raceID int32,
) ([]*models.RaceGrid, error) {
	return models.RaceGrids.Query(
		sm.Where(models.RaceGrids.Columns.RaceID.EQ(psql.Arg(raceID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *raceGridsRepository) LoadByID(ctx context.Context, id int32) (*models.RaceGrid, error) {
	entity, err := models.RaceGrids.Query(sm.Where(models.RaceGrids.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("race grid %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *raceGridsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.RaceGrids.Delete(dm.Where(models.RaceGrids.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *raceGridsRepository) Create(
	ctx context.Context,
	input *models.RaceGridSetter,
) (*models.RaceGrid, error) {
	return models.RaceGrids.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *raceGridsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.RaceGridSetter,
) (*models.RaceGrid, error) {
	entity, err := models.RaceGrids.Update(
		input.UpdateMod(),
		um.Where(models.RaceGrids.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("race grid %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *raceGridsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
