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

//nolint:whitespace // editor/linter issue
func NewSeasonCarModelVariantQueries(
	exec *pgbob.Executor,
) repository.QuerySeasonCarModelVariant {
	return &querySeasonCarModelVariants{exec: exec}
}

type querySeasonCarModelVariants struct {
	exec *pgbob.Executor
}

//nolint:whitespace // editor/linter issue
func (r *querySeasonCarModelVariants) FindBySeasonAndCarModelVariant(
	ctx context.Context,
	seasonID, carModelVariantID int32,
) (*models.SeasonCarModelVariant, error) {
	entity, err := models.SeasonCarModelVariants.Query(
		sm.Where(models.SeasonCarModelVariants.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		sm.Where(
			models.SeasonCarModelVariants.Columns.CarModelVariantID.EQ(
				psql.Arg(carModelVariantID),
			),
		),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"car model variant %d for season %d: %w",
			carModelVariantID,
			seasonID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *querySeasonCarModelVariants) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
