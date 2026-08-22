package cars

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

// SimulationCarAliasesRepository defines persistence operations
// for SimulationCarAlias entities.
type SimulationCarAliasesRepository interface {
	LoadByID(context.Context, int32) (*models.SimulationCarAlias, error)
	LoadByVariantID(context.Context, int32) ([]*models.SimulationCarAlias, error)
	LoadBySimulationID(context.Context, int32) ([]*models.SimulationCarAlias, error)
	FindBySimID(context.Context, int32, ...string) (*models.SimulationCarAlias, error)
	ReplaceForVariantID(
		context.Context,
		int32,
		[]*models.SimulationCarAliasSetter,
	) ([]*models.SimulationCarAlias, error)
	DeleteByID(context.Context, int32) error
	DeleteByVariantID(context.Context, int32) error
	Create(
		context.Context, *models.SimulationCarAliasSetter,
	) (*models.SimulationCarAlias, error)
	Update(
		context.Context,
		int32,
		*models.SimulationCarAliasSetter,
	) (*models.SimulationCarAlias, error)
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.Query(
		sm.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation car alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) LoadByVariantID(
	ctx context.Context,
	id int32,
) ([]*models.SimulationCarAlias, error) {
	return models.SimulationCarAliases.Query(
		sm.Where(models.SimulationCarAliases.Columns.CarModelVariantID.EQ(psql.Arg(id))),
	).All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) LoadBySimulationID(
	ctx context.Context,
	id int32,
) ([]*models.SimulationCarAlias, error) {
	return models.SimulationCarAliases.Query(
		sm.Where(models.SimulationCarAliases.Columns.SimulationID.EQ(psql.Arg(id))),
	).All(ctx, r.getExecutor(ctx))
}

//nolint:whitespace,lll // editor/linter issue, readability
func (r *simulationCarAliasesRepository) FindBySimID(
	ctx context.Context,
	id int32,
	aliases ...string,
) (*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.Query(
		sm.Where(models.SimulationCarAliases.Columns.SimulationID.EQ(psql.Arg(id))),
		sm.Where(
			models.SimulationCarAliases.Columns.ExternalName.EQ(psql.F("ANY", psql.Arg(aliases))),
		),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"simulation car alias %q for simulation %d: %w",
			aliases,
			id,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) ReplaceForVariantID(
	ctx context.Context,
	id int32,
	aliases []*models.SimulationCarAliasSetter,
) ([]*models.SimulationCarAlias, error) {
	if err := r.DeleteByVariantID(ctx, id); err != nil {
		return nil, err
	}
	created := make([]*models.SimulationCarAlias, 0, len(aliases))
	for _, alias := range aliases {
		entity, err := models.SimulationCarAliases.Insert(alias).One(ctx, r.getExecutor(ctx))
		if err != nil {
			return nil, err
		}
		created = append(created, entity)
	}
	return created, nil
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) DeleteByID(
	ctx context.Context, id int32,
) error {
	_, err := models.SimulationCarAliases.Delete(
		dm.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) DeleteByVariantID(
	ctx context.Context, id int32,
) error {
	_, err := models.SimulationCarAliases.Delete(
		dm.Where(models.SimulationCarAliases.Columns.CarModelVariantID.EQ(psql.Arg(id))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) Create(
	ctx context.Context,
	input *models.SimulationCarAliasSetter,
) (*models.SimulationCarAlias, error) {
	return models.SimulationCarAliases.Insert(input).One(ctx, r.getExecutor(ctx))
}

//nolint:whitespace // editor/linter issue
func (r *simulationCarAliasesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SimulationCarAliasSetter,
) (*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.Update(
		input.UpdateMod(),
		um.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation car alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationCarAliasesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
