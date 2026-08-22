//nolint:dupl // cache decorators are intentionally parallel
package cars

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedCarModelsRepository struct {
	CarModelsRepository
	byID *cache.Cache[int32, *models.CarModel]
}

// NewCachedCarModelsRepository wraps repo with a read-through cache keyed by ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedCarModelsRepository(
	repo CarModelsRepository,
	byID *cache.Cache[int32, *models.CarModel],
) CarModelsRepository {
	return &cachedCarModelsRepository{CarModelsRepository: repo, byID: byID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarModel, error) {
	if entity, ok := r.byID.Get(ctx, id); ok {
		return entity, nil
	}

	entity, err := r.CarModelsRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, entity)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelsRepository) Create(
	ctx context.Context,
	input *models.CarModelSetter,
) (*models.CarModel, error) {
	entity, err := r.CarModelsRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, entity.ID, entity, cache.ActionUpdated)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarModelSetter,
) (*models.CarModel, error) {
	entity, err := r.CarModelsRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelsRepository) DeleteByID(
	ctx context.Context,
	id int32,
) error {
	if err := r.CarModelsRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}
