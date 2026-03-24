// Package cars provides repositories for the cars migration group.
//
//nolint:lll,dupl,whitespace // repository implementations can be verbose
package cars

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

// CarBrandsRepository defines persistence operations for CarBrand entities.
type CarBrandsRepository interface {
	LoadAll(ctx context.Context) ([]*models.CarBrand, error)
	LoadByManufacturerID(ctx context.Context, manufacturerID int32) ([]*models.CarBrand, error)
	LoadByID(ctx context.Context, id int32) (*models.CarBrand, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.CarBrandSetter) (*models.CarBrand, error)
	Update(ctx context.Context, id int32, input *models.CarBrandSetter) (*models.CarBrand, error)
}

// CarModelsRepository defines persistence operations for CarModel entities.
type CarModelsRepository interface {
	LoadAll(ctx context.Context) ([]*models.CarModel, error)
	LoadByManufacturerID(ctx context.Context, manufacturerID int32) ([]*models.CarModel, error)
	LoadByID(ctx context.Context, id int32) (*models.CarModel, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.CarModelSetter) (*models.CarModel, error)
	Update(ctx context.Context, id int32, input *models.CarModelSetter) (*models.CarModel, error)
}

// SimulationCarAliasesRepository defines persistence operations for SimulationCarAlias entities.
type SimulationCarAliasesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.SimulationCarAlias, error)
	LoadBySimulationID(ctx context.Context, simID int32) ([]*models.SimulationCarAlias, error)
	FindBySimID(
		ctx context.Context,
		simID int32,
		aliases ...string,
	) (*models.SimulationCarAlias, error)
	DeleteByID(ctx context.Context, id int32) error
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
	CarBrands() CarBrandsRepository
	CarModels() CarModelsRepository
	SimulationCarAliases() SimulationCarAliasesRepository
}

type repository struct {
	carManufacturers     CarManufacturersRepository
	carBrands            CarBrandsRepository
	carModels            CarModelsRepository
	simulationCarAliases SimulationCarAliasesRepository
}

type (
	carManufacturersRepository     struct{ exec *pgbob.Executor }
	carBrandsRepository            struct{ exec *pgbob.Executor }
	carModelsRepository            struct{ exec *pgbob.Executor }
	simulationCarAliasesRepository struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		carManufacturers:     &carManufacturersRepository{exec: pgbob.New(pool)},
		carBrands:            &carBrandsRepository{exec: pgbob.New(pool)},
		carModels:            &carModelsRepository{exec: pgbob.New(pool)},
		simulationCarAliases: &simulationCarAliasesRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) CarManufacturers() CarManufacturersRepository { return r.carManufacturers }
func (r *repository) CarBrands() CarBrandsRepository               { return r.carBrands }
func (r *repository) CarModels() CarModelsRepository               { return r.carModels }
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
	return models.CarManufacturers.Update(
		input.UpdateMod(),
		um.Where(models.CarManufacturers.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
}

func (r *carBrandsRepository) LoadAll(ctx context.Context) ([]*models.CarBrand, error) {
	return models.CarBrands.Query().All(ctx, r.getExecutor(ctx))
}

func (r *carBrandsRepository) LoadByManufacturerID(
	ctx context.Context,
	manufacturerID int32,
) ([]*models.CarBrand, error) {
	return models.CarBrands.Query(
		sm.Where(models.CarBrands.Columns.ManufacturerID.EQ(psql.Arg(manufacturerID))),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carBrandsRepository) LoadByID(ctx context.Context, id int32) (*models.CarBrand, error) {
	entity, err := models.CarBrands.Query(sm.Where(models.CarBrands.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car brand %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *carBrandsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.CarBrands.Delete(dm.Where(models.CarBrands.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *carBrandsRepository) Create(
	ctx context.Context,
	input *models.CarBrandSetter,
) (*models.CarBrand, error) {
	return models.CarBrands.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *carBrandsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarBrandSetter,
) (*models.CarBrand, error) {
	return models.CarBrands.Update(
		input.UpdateMod(),
		um.Where(models.CarBrands.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
}

func (r *carModelsRepository) LoadAll(ctx context.Context) ([]*models.CarModel, error) {
	return models.CarModels.Query().All(ctx, r.getExecutor(ctx))
}

// LoadByManufacturerID returns all car models belonging to brands of the given manufacturer.
// It uses a subquery to filter by brand_id IN (SELECT id FROM car_brands WHERE manufacturer_id = ?).
func (r *carModelsRepository) LoadByManufacturerID(
	ctx context.Context,
	manufacturerID int32,
) ([]*models.CarModel, error) {
	return models.CarModels.Query(
		sm.Where(models.CarModels.Columns.BrandID.In(
			psql.Select(
				sm.Columns(models.CarBrands.Columns.ID),
				sm.From(models.CarBrands.Name()),
				sm.Where(models.CarBrands.Columns.ManufacturerID.EQ(psql.Arg(manufacturerID))),
			),
		)),
	).All(ctx, r.getExecutor(ctx))
}

func (r *carModelsRepository) LoadByID(ctx context.Context, id int32) (*models.CarModel, error) {
	entity, err := models.CarModels.Query(sm.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("car model %d: %w", id, repoerrors.ErrNotFound)
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
	return models.CarModels.Update(
		input.UpdateMod(),
		um.Where(models.CarModels.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
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

func (r *simulationCarAliasesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.SimulationCarAliases.Delete(dm.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id)))).
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
	return models.SimulationCarAliases.Update(
		input.UpdateMod(),
		um.Where(models.SimulationCarAliases.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
}

func (r *carManufacturersRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *carBrandsRepository) getExecutor(ctx context.Context) bob.Executor {
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

func (r *simulationCarAliasesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
