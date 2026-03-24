// Package teams provides repositories for the teams migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package teams

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

// TeamsRepository defines persistence operations for Team entities.
type TeamsRepository interface {
	LoadAll(ctx context.Context) ([]*models.Team, error)
	LoadByID(ctx context.Context, id int32) (*models.Team, error)
	LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.Team, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.TeamSetter) (*models.Team, error)
	Update(ctx context.Context, id int32, input *models.TeamSetter) (*models.Team, error)
}

// TeamDriversRepository defines persistence operations for TeamDriver entities.
type TeamDriversRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.TeamDriver, error)
	LoadByTeamID(ctx context.Context, teamID int32) ([]*models.TeamDriver, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.TeamDriverSetter) (*models.TeamDriver, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.TeamDriverSetter,
	) (*models.TeamDriver, error)
}

// Repository exposes repositories for the teams migration group.
type Repository interface {
	Teams() TeamsRepository
	TeamDrivers() TeamDriversRepository
}

type repository struct {
	teams       TeamsRepository
	teamDrivers TeamDriversRepository
}

type (
	teamsRepository       struct{ exec *pgbob.Executor }
	teamDriversRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		teams:       &teamsRepository{exec: pgbob.New(pool)},
		teamDrivers: &teamDriversRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) Teams() TeamsRepository             { return r.teams }
func (r *repository) TeamDrivers() TeamDriversRepository { return r.teamDrivers }

func (r *teamsRepository) LoadAll(ctx context.Context) ([]*models.Team, error) {
	return models.Teams.Query().All(ctx, r.getExecutor(ctx))
}

func (r *teamsRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.Team, error) {
	return models.Teams.Query(
		sm.Where(models.Teams.Columns.SeasonID.EQ(psql.Arg(seasonID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *teamsRepository) LoadByID(ctx context.Context, id int32) (*models.Team, error) {
	entity, err := models.Teams.Query(sm.Where(models.Teams.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("team %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *teamsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.Teams.Delete(dm.Where(models.Teams.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *teamsRepository) Create(
	ctx context.Context,
	input *models.TeamSetter,
) (*models.Team, error) {
	return models.Teams.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *teamsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.TeamSetter,
) (*models.Team, error) {
	entity, err := models.Teams.Update(
		input.UpdateMod(),
		um.Where(models.Teams.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("team %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *teamDriversRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.TeamDriver, error) {
	entity, err := models.TeamDrivers.Query(sm.Where(models.TeamDrivers.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("team driver %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *teamDriversRepository) LoadByTeamID(
	ctx context.Context,
	teamID int32,
) ([]*models.TeamDriver, error) {
	return models.TeamDrivers.Query(
		sm.Where(models.TeamDrivers.Columns.TeamID.EQ(psql.Arg(teamID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *teamDriversRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.TeamDrivers.Delete(dm.Where(models.TeamDrivers.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *teamDriversRepository) Create(
	ctx context.Context,
	input *models.TeamDriverSetter,
) (*models.TeamDriver, error) {
	return models.TeamDrivers.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *teamDriversRepository) Update(
	ctx context.Context,
	id int32,
	input *models.TeamDriverSetter,
) (*models.TeamDriver, error) {
	entity, err := models.TeamDrivers.Update(
		input.UpdateMod(),
		um.Where(models.TeamDrivers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("team driver %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *teamsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *teamDriversRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
