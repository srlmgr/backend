package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func NewCarClassQueries(exec *pgbob.Executor) repository.QueryCarClass {
	return &queryCarClasses{exec: exec}
}

type queryCarClasses struct {
	exec *pgbob.Executor
}

//nolint:whitespace // editor/linter issue
func (r *queryCarClasses) FindBySeasonAndCarModel(
	ctx context.Context,
	seasonID, carModelID int32,
) (*models.CarClass, error) {
	subQuery := psql.Select(
		sm.Columns(models.CarClassesToCarModels.Columns.CarClassID),
		sm.From(models.CarClassesToCarModels.Name()),
		sm.Where(models.CarClassesToCarModels.Columns.CarModelID.EQ(psql.Arg(carModelID))),
	)
	query := models.CarClasses.Query(
		models.SelectJoins.CarClasses.InnerJoin.SeasonCarClasses,
		models.SelectWhere.SeasonCarClasses.SeasonID.EQ(seasonID),
		// models.SelectWhere.CarClasses.ID.In(subQuery),
		sm.Where(models.CarClasses.Columns.ID.In(subQuery)),
	)

	entity, err := query.One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"car class for season %d and car model %d: %w",
			seasonID,
			carModelID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *queryCarClasses) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
