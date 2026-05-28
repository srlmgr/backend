//nolint:lll,whitespace // repository implementations can be verbose
package drivers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// SimulationDriverAliasesRepository defines persistence operations for SimulationDriverAlias entities.
type SimulationDriverAliasesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SimulationDriverAlias, error)
	LoadBySimulationID(ctx context.Context, simID int32) ([]*models.SimulationDriverAlias, error)
	GetDriverAliases(ctx context.Context, driverID int32) ([]*models.SimulationDriverAlias, error)
	FindBySimID(
		ctx context.Context,
		simID int32,
		aliases ...string,
	) (*models.SimulationDriverAlias, error)
	DeleteByID(ctx context.Context, id int32) error
	DeleteByDriverID(ctx context.Context, driverID int32) error
	Create(
		ctx context.Context,
		input *models.SimulationDriverAliasSetter,
	) (*models.SimulationDriverAlias, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SimulationDriverAliasSetter,
	) (*models.SimulationDriverAlias, error)
}

func (r *simulationDriverAliasesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.Query(sm.Where(models.SimulationDriverAliases.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation driver alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationDriverAliasesRepository) LoadBySimulationID(
	ctx context.Context,
	simID int32,
) ([]*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.
		Query(
			sm.Where(
				models.SimulationDriverAliases.Columns.SimulationID.EQ(psql.Arg(simID)))).
		All(ctx, r.getExecutor(ctx))

	return entity, err
}

func (r *simulationDriverAliasesRepository) GetDriverAliases(
	ctx context.Context,
	driverID int32,
) ([]*models.SimulationDriverAlias, error) {
	return models.SimulationDriverAliases.Query(
		sm.Where(models.SimulationDriverAliases.Columns.DriverID.EQ(psql.Arg(driverID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *simulationDriverAliasesRepository) FindBySimID(
	ctx context.Context,
	simID int32,
	aliases ...string,
) (*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.Query(
		sm.Where(models.SimulationDriverAliases.Columns.SimulationID.EQ(psql.Arg(simID))),
		sm.Where(
			models.SimulationDriverAliases.Columns.SimulationDriverID.EQ(
				psql.F("ANY", psql.Arg(aliases)),
			),
		),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"simulation driver alias %q for simulation %d: %w",
			aliases,
			simID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *simulationDriverAliasesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SimulationDriverAliases.Delete(dm.Where(models.SimulationDriverAliases.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationDriverAliasesRepository) DeleteByDriverID(
	ctx context.Context,
	driverID int32,
) error {
	_, err := models.SimulationDriverAliases.Delete(dm.Where(models.SimulationDriverAliases.Columns.DriverID.EQ(psql.Arg(driverID)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationDriverAliasesRepository) Create(
	ctx context.Context,
	input *models.SimulationDriverAliasSetter,
) (*models.SimulationDriverAlias, error) {
	return models.SimulationDriverAliases.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *simulationDriverAliasesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SimulationDriverAliasSetter,
) (*models.SimulationDriverAlias, error) {
	entity, err := models.SimulationDriverAliases.Update(
		input.UpdateMod(),
		um.Where(models.SimulationDriverAliases.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation driver alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationDriverAliasesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
