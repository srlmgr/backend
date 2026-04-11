package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/stephenafamo/bob"

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

func (r *queryTeamDrivers) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
