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

func NewSeasonDriverQueries(exec *pgbob.Executor) repository.QuerySeasonDriver {
	return &querySeasonDrivers{exec: exec}
}

type querySeasonDrivers struct {
	exec *pgbob.Executor
}

//nolint:whitespace // editor/linter issue
func (r *querySeasonDrivers) ResolveSeasonDriver(
	ctx context.Context,
	seasonID, driverID int32,
	when time.Time,
) (*models.SeasonDriver, error) {
	query := models.SeasonDrivers.Query(
		sm.Where(models.SeasonDrivers.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		sm.Where(models.SeasonDrivers.Columns.DriverID.EQ(psql.Arg(driverID))),
		sm.Where(models.SeasonDrivers.Columns.JoinedAt.LTE(psql.Arg(when))),
		sm.Where(
			psql.Or(
				models.SeasonDrivers.Columns.LeftAt.IsNull(),
				models.SeasonDrivers.Columns.LeftAt.GT(psql.Arg(when)),
			),
		),
	)
	entity, err := query.One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"season driver for season %d and driver %d: %w",
			seasonID,
			driverID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *querySeasonDrivers) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
