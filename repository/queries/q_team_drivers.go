package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func NewTeamDriverQueries(exec *pgbob.Executor) repository.QueryTeamDriver {
	return &queryTeamDrivers{exec: exec}
}

type queryTeamDrivers struct {
	exec *pgbob.Executor
}

//nolint:whitespace // editor/linter issue
func (r *queryTeamDrivers) FindBySeasonAndDriver(
	ctx context.Context,
	seasonID, driverID int32,
) (*models.TeamDriver, error) {
	query := models.TeamDrivers.Query(
		models.SelectJoins.TeamDrivers.InnerJoin.Team,
		models.SelectWhere.Teams.SeasonID.EQ(seasonID),
		models.SelectWhere.TeamDrivers.DriverID.EQ(driverID),
	)
	entity, err := query.One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"team driver for season %d and driver %d: %w",
			seasonID,
			driverID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *queryTeamDrivers) ResolveTeamDriver(
	ctx context.Context,
	seasonID, driverID int32,
	when time.Time,
) (*models.TeamDriver, error) {
	subQ := psql.Select(
		sm.Columns(models.TeamDrivers.Columns.ID),
		sm.From(models.TeamDrivers.NameExpr()),
		sm.InnerJoin(models.Teams.NameExpr()).
			On(models.Teams.Columns.ID.EQ(models.TeamDrivers.Columns.TeamID)),
		sm.Where(models.Teams.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		sm.Where(models.Teams.Columns.JoinedAt.LTE(psql.Arg(when))),
		sm.Where(
			psql.Or(
				models.Teams.Columns.LeftAt.IsNull(),
				models.Teams.Columns.LeftAt.GT(psql.Arg(when)),
			),
		),
		sm.Where(models.TeamDrivers.Columns.DriverID.EQ(psql.Arg(driverID))),
		sm.Where(models.TeamDrivers.Columns.JoinedAt.LTE(psql.Arg(when))),
		sm.Where(
			psql.Or(
				models.TeamDrivers.Columns.LeftAt.IsNull(),
				models.TeamDrivers.Columns.LeftAt.GT(psql.Arg(when)),
			),
		),
	)
	query := models.TeamDrivers.Query(
		sm.Where(models.TeamDrivers.Columns.ID.In(subQ)),
	)
	entity, err := query.One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"team driver for season %d and driver %d: %w",
			seasonID,
			driverID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *queryTeamDrivers) FindBySeason(
	ctx context.Context,
	seasonID int32,
) ([]*models.TeamDriver, error) {
	query := models.TeamDrivers.Query(
		models.SelectJoins.TeamDrivers.InnerJoin.Team,
		models.SelectWhere.Teams.SeasonID.EQ(seasonID),
	)
	entities, err := query.All(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"team drivers for season %d: %w",
			seasonID,
			repoerrors.ErrNotFound,
		)
	}
	return entities, err
}

func (r *queryTeamDrivers) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
