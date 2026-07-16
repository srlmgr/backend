// Package cars provides repositories for the cars migration group.
//
//nolint:lll,dupl,whitespace // repository implementations can be verbose
package cars

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
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

// CarManufacturersRepository defines persistence operations for CarManufacturer entities.
type CarManufacturersRepository interface {
	LoadAll(ctx context.Context) ([]*models.CarManufacturer, error)
	LoadByID(ctx context.Context, id int32) (*models.CarManufacturer, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.CarManufacturerSetter,
	) (*models.CarManufacturer, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.CarManufacturerSetter,
	) (*models.CarManufacturer, error)
}

// CarModelsRepository defines persistence operations for CarModel entities.
type CarModelsRepository interface {
	LoadAll(ctx context.Context) ([]*models.CarModel, error)
	LoadByManufacturerID(ctx context.Context, manufacturerID int32) ([]*models.CarModel, error)
	LoadByID(ctx context.Context, id int32) (*models.CarModel, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.CarModelSetter,
	) (*models.CarModel, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.CarModelSetter,
	) (*models.CarModel, error)
}

// CarModelVariantsRepository defines persistence operations for CarModelVariant entities.
type CarModelVariantsRepository interface {
	LoadAll(ctx context.Context) ([]*models.CarModelVariant, error)
	LoadByManufacturerID(
		ctx context.Context,
		manufacturerID int32,
	) ([]*models.CarModelVariant, error)
	LoadByCarClassID(ctx context.Context, classID int32) ([]*models.CarModelVariant, error)
	LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.CarModelVariant, error)
	LoadByModelID(ctx context.Context, modelID int32) ([]*models.CarModelVariant, error)
	LoadByID(ctx context.Context, id int32) (*models.CarModelVariant, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.CarModelVariantSetter,
	) (*models.CarModelVariant, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.CarModelVariantSetter,
	) (*models.CarModelVariant, error)
}

// CarClassesRepository defines persistence operations for CarClass entities.
type CarClassesRepository interface {
	LoadAll(ctx context.Context) ([]*models.CarClass, error)
	LoadByID(ctx context.Context, id int32) (*models.CarClass, error)
	LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.CarClass, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.CarClassSetter) (*models.CarClass, error)
	Update(ctx context.Context, id int32, input *models.CarClassSetter) (*models.CarClass, error)
	AssignCarModelVariant(ctx context.Context, classID, modelVariantID int32) error
	UnassignCarModelVariant(ctx context.Context, classID, modelVariantID int32) error
}

// SimulationCarAliasesRepository defines persistence operations for SimulationCarAlias entities.
type SimulationCarAliasesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SimulationCarAlias, error)

	LoadByVariantID(ctx context.Context, variantID int32) ([]*models.SimulationCarAlias, error)
	LoadBySimulationID(ctx context.Context, simID int32) ([]*models.SimulationCarAlias, error)
	FindBySimID(
		ctx context.Context,
		simID int32,
		aliases ...string,
	) (*models.SimulationCarAlias, error)

	ReplaceForVariantID(
		ctx context.Context,
		variantID int32,
		aliases []*models.SimulationCarAliasSetter,
	) ([]*models.SimulationCarAlias, error)
	DeleteByID(ctx context.Context, id int32) error

	DeleteByVariantID(ctx context.Context, variantID int32) error
	Create(
		ctx context.Context,
		input *models.SimulationCarAliasSetter,
	) (*models.SimulationCarAlias, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.SimulationCarAliasSetter,
	) (*models.SimulationCarAlias, error)
}

// Repository exposes repositories for the cars migration group.
type Repository interface {
	CarManufacturers() CarManufacturersRepository
	CarModels() CarModelsRepository
	CarModelVariants() CarModelVariantsRepository
	CarClasses() CarClassesRepository
	SimulationCarAliases() SimulationCarAliasesRepository
}

type repository struct {
	carManufacturers     CarManufacturersRepository
	carModels            CarModelsRepository
	carModelVariants     CarModelVariantsRepository
	carClasses           CarClassesRepository
	simulationCarAliases SimulationCarAliasesRepository
}

type (
	carManufacturersRepository     struct{ exec *pgbob.Executor }
	carModelsRepository            struct{ exec *pgbob.Executor }
	carModelVariantsRepository     struct{ exec *pgbob.Executor }
	carClassesRepository           struct{ exec *pgbob.Executor }
	simulationCarAliasesRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		carManufacturers:     &carManufacturersRepository{exec: pgbob.New(pool)},
		carModels:            &carModelsRepository{exec: pgbob.New(pool)},
		carModelVariants:     &carModelVariantsRepository{exec: pgbob.New(pool)},
		carClasses:           &carClassesRepository{exec: pgbob.New(pool)},
		simulationCarAliases: &simulationCarAliasesRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) CarManufacturers() CarManufacturersRepository { return r.carManufacturers }
func (r *repository) CarModels() CarModelsRepository               { return r.carModels }
func (r *repository) CarModelVariants() CarModelVariantsRepository { return r.carModelVariants }
func (r *repository) CarClasses() CarClassesRepository             { return r.carClasses }
func (r *repository) SimulationCarAliases() SimulationCarAliasesRepository {
	return r.simulationCarAliases
}

func (r *carManufacturersRepository) LoadAll(
	ctx context.Context,
) ([]*models.CarManufacturer, error) {
	return models.CarManufacturers.Query().All(ctx, r.getExecutor(ctx))
}

func (r *carManufacturersRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarManufacturer, error) {
	entity, err := models.CarManufacturers.Query(sm.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car manufacturer %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carManufacturersRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarManufacturers.Delete(dm.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *carManufacturersRepository) Create(
	ctx context.Context,
	input *models.CarManufacturerSetter,
) (*models.CarManufacturer, error) {
	return models.CarManufacturers.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *carManufacturersRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarManufacturerSetter,
) (*models.CarManufacturer, error) {
	entity, err := models.CarManufacturers.Update(
		input.UpdateMod(),
		um.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car manufacturer %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelsRepository) LoadAll(ctx context.Context) ([]*models.CarModel, error) {
	return models.CarModels.Query().All(ctx, r.getExecutor(ctx))
}

func (r *carModelsRepository) LoadByManufacturerID(
	ctx context.Context,
	manufacturerID int32,
) ([]*models.CarModel, error) {
	return models.CarModels.Query(
		sm.Where(models.CarModels.Columns.ManufacturerID.EQ(psql.Arg(manufacturerID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carModelsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarModel, error) {
	entity, err := models.CarModels.Query(sm.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model v2 %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarModels.Delete(dm.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *carModelsRepository) Create(
	ctx context.Context,
	input *models.CarModelSetter,
) (*models.CarModel, error) {
	return models.CarModels.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *carModelsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarModelSetter,
) (*models.CarModel, error) {
	entity, err := models.CarModels.Update(
		input.UpdateMod(),
		um.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model  %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelVariantsRepository) LoadAll(
	ctx context.Context,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query().All(ctx, r.getExecutor(ctx))
}

// LoadByManufacturerID returns all car model variants belonging to car models v2 of the given manufacturer.
func (r *carModelVariantsRepository) LoadByManufacturerID(
	ctx context.Context,
	manufacturerID int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.CarModelID.In(
			psql.Select(
				sm.Columns(models.CarModels.Columns.ID),
				sm.From(models.CarModels.Name()),
				sm.Where(models.CarModels.Columns.ManufacturerID.EQ(psql.Arg(manufacturerID))),
			),
		)),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carModelVariantsRepository) LoadByCarClassID(
	ctx context.Context,
	classID int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.ID.In(
			psql.Select(
				sm.Columns(models.CarClassesToCarModels.Columns.CarModelVariantID),
				sm.From(models.CarClassesToCarModels.Name()),
				sm.Where(models.CarClassesToCarModels.Columns.CarClassID.EQ(psql.Arg(classID))),
			),
		)),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carModelVariantsRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.InnerJoin(models.SeasonCarModelVariants.Name()).
			On(models.SeasonCarModelVariants.Columns.CarModelVariantID.EQ(models.CarModelVariants.Columns.ID)),
		sm.Where(models.SeasonCarModelVariants.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		sm.OrderBy(models.SeasonCarModelVariants.Columns.Pos).Asc(),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carModelVariantsRepository) LoadByModelID(
	ctx context.Context,
	modelID int32,
) ([]*models.CarModelVariant, error) {
	return models.CarModelVariants.Query(
		sm.Where(models.CarModelVariants.Columns.CarModelID.EQ(psql.Arg(modelID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carModelVariantsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarModelVariant, error) {
	entity, err := models.CarModelVariants.Query(sm.Where(models.CarModelVariants.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model variant %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carModelVariantsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarModelVariants.Delete(
		dm.Where(models.CarModelVariants.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *carModelVariantsRepository) Create(
	ctx context.Context,
	input *models.CarModelVariantSetter,
) (*models.CarModelVariant, error) {
	return models.CarModelVariants.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *carModelVariantsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarModelVariantSetter,
) (*models.CarModelVariant, error) {
	entity, err := models.CarModelVariants.Update(
		input.UpdateMod(),
		um.Where(models.CarModelVariants.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model variant %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carClassesRepository) LoadAll(ctx context.Context) ([]*models.CarClass, error) {
	return models.CarClasses.Query().All(ctx, r.getExecutor(ctx))
}

func (r *carClassesRepository) LoadByID(ctx context.Context, id int32) (*models.CarClass, error) {
	entity, err := models.CarClasses.Query(sm.Where(models.CarClasses.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car class %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carClassesRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.CarClass, error) {
	return models.CarClasses.Query(
		sm.InnerJoin(models.SeasonCarClasses.Name()).
			On(models.SeasonCarClasses.Columns.CarClassID.EQ(models.CarClasses.Columns.ID)),
		sm.Where(models.SeasonCarClasses.Columns.SeasonID.EQ(psql.Arg(seasonID))),
		sm.OrderBy(models.SeasonCarClasses.Columns.Pos).Asc(),
	).All(ctx, bob.Debug(r.getExecutor(ctx)))
}

func (r *carClassesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarClasses.Delete(dm.Where(models.CarClasses.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *carClassesRepository) Create(
	ctx context.Context,
	input *models.CarClassSetter,
) (*models.CarClass, error) {
	return models.CarClasses.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *carClassesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarClassSetter,
) (*models.CarClass, error) {
	entity, err := models.CarClasses.Update(
		input.UpdateMod(),
		um.Where(models.CarClasses.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car class %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) AssignCarModelVariant(
	ctx context.Context,
	classID, modelVariantID int32,
) error {
	if _, checkErr := models.CarClassesToCarModels.Query(
		sm.Where(models.CarClassesToCarModels.Columns.CarClassID.EQ(psql.Arg(classID))),
		sm.Where(
			models.CarClassesToCarModels.Columns.CarModelVariantID.EQ(psql.Arg(modelVariantID)),
		),
	).One(ctx, r.getExecutor(ctx)); checkErr == nil {
		return nil
	}
	_, err := models.CarClassesToCarModels.Insert(&models.CarClassesToCarModelSetter{
		CarClassID:        omit.From(classID),
		CarModelVariantID: omitnull.From(modelVariantID), // TODO: change when mandatory
	}).One(ctx, r.getExecutor(ctx))
	return err
}

//nolint:whitespace // editor/linter issue
func (r *carClassesRepository) UnassignCarModelVariant(
	ctx context.Context,
	classID, modelVariantID int32,
) error {
	_, err := models.CarClassesToCarModels.Delete(
		dm.Where(models.CarClassesToCarModels.Columns.CarClassID.EQ(psql.Arg(classID))),
		dm.Where(
			models.CarClassesToCarModels.Columns.CarModelVariantID.EQ(psql.Arg(modelVariantID)),
		),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationCarAliasesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.Query(sm.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("simulation car alias %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *simulationCarAliasesRepository) LoadBySimulationID(
	ctx context.Context,
	simID int32,
) ([]*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.
		Query(
			sm.Where(
				models.SimulationCarAliases.Columns.SimulationID.EQ(psql.Arg(simID))),
		).
		All(ctx, r.getExecutor(ctx))

	return entity, err
}

func (r *simulationCarAliasesRepository) LoadByVariantID(
	ctx context.Context,
	variantID int32,
) ([]*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.
		Query(
			sm.Where(models.SimulationCarAliases.Columns.CarModelVariantID.EQ(psql.Arg(variantID))),
		).
		All(ctx, r.getExecutor(ctx))

	return entity, err
}

func (r *simulationCarAliasesRepository) FindBySimID(
	ctx context.Context,
	simID int32,
	aliases ...string,
) (*models.SimulationCarAlias, error) {
	entity, err := models.SimulationCarAliases.Query(
		sm.Where(models.SimulationCarAliases.Columns.SimulationID.EQ(psql.Arg(simID))),
		sm.Where(
			models.SimulationCarAliases.Columns.ExternalName.EQ(psql.F("ANY", psql.Arg(aliases))),
		),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"simulation car alias %q for simulation %d: %w",
			aliases,
			simID,
			repoerrors.ErrNotFound,
		)
	}
	return entity, err
}

func (r *simulationCarAliasesRepository) ReplaceForVariantID(
	ctx context.Context,
	variantID int32,
	aliases []*models.SimulationCarAliasSetter,
) ([]*models.SimulationCarAlias, error) {
	if err := r.DeleteByVariantID(ctx, variantID); err != nil {
		return nil, err
	}

	var created []*models.SimulationCarAlias
	for _, alias := range aliases {
		entity, err := models.SimulationCarAliases.Insert(alias).
			One(ctx, r.getExecutor(ctx))
		if err != nil {
			return nil, err
		}
		created = append(created, entity)
	}

	return created, nil
}

func (r *simulationCarAliasesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SimulationCarAliases.Delete(
		dm.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationCarAliasesRepository) DeleteByVariantID(
	ctx context.Context,
	variantID int32,
) error {
	_, err := models.SimulationCarAliases.Delete(
		dm.Where(models.SimulationCarAliases.Columns.CarModelVariantID.EQ(psql.Arg(variantID)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *simulationCarAliasesRepository) Create(
	ctx context.Context,
	input *models.SimulationCarAliasSetter,
) (*models.SimulationCarAlias, error) {
	return models.SimulationCarAliases.Insert(input).One(ctx, r.getExecutor(ctx))
}

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

func (r *carManufacturersRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *carModelsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *carModelVariantsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *carClassesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *simulationCarAliasesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
