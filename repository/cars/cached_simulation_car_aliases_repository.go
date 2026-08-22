package cars

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedSimulationCarAliasesRepository struct {
	SimulationCarAliasesRepository
	byID *cache.Cache[int32, *models.SimulationCarAlias]
}

// NewCachedSimulationCarAliasesRepository wraps repo with a read-through cache
// keyed by ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedSimulationCarAliasesRepository(
	repo SimulationCarAliasesRepository,
	byID *cache.Cache[int32, *models.SimulationCarAlias],
) SimulationCarAliasesRepository {
	return &cachedSimulationCarAliasesRepository{
		SimulationCarAliasesRepository: repo,
		byID:                           byID,
	}
}

//nolint:whitespace // editor/linter issue
func (r *cachedSimulationCarAliasesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.SimulationCarAlias, error) {
	if entity, ok := r.byID.Get(ctx, id); ok {
		return entity, nil
	}

	entity, err := r.SimulationCarAliasesRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, entity)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSimulationCarAliasesRepository) Create(
	ctx context.Context,
	input *models.SimulationCarAliasSetter,
) (*models.SimulationCarAlias, error) {
	entity, err := r.SimulationCarAliasesRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, entity.ID, entity, cache.ActionUpdated)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSimulationCarAliasesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SimulationCarAliasSetter,
) (*models.SimulationCarAlias, error) {
	entity, err := r.SimulationCarAliasesRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSimulationCarAliasesRepository) DeleteByID(
	ctx context.Context,
	id int32,
) error {
	if err := r.SimulationCarAliasesRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSimulationCarAliasesRepository) DeleteByVariantID(
	ctx context.Context,
	variantID int32,
) error {
	if err := r.SimulationCarAliasesRepository.DeleteByVariantID(
		ctx,
		variantID,
	); err != nil {
		return err
	}
	r.byID.Flush()
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSimulationCarAliasesRepository) ReplaceForVariantID(
	ctx context.Context,
	variantID int32,
	aliases []*models.SimulationCarAliasSetter,
) ([]*models.SimulationCarAlias, error) {
	entities, err := r.SimulationCarAliasesRepository.ReplaceForVariantID(
		ctx,
		variantID,
		aliases,
	)
	if err != nil {
		return nil, err
	}
	r.byID.Flush()
	return entities, nil
}
