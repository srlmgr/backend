//nolint:dupl // cache decorators are intentionally parallel
package cars

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedCarModelVariantsRepository struct {
	CarModelVariantsRepository
	byID *cache.Cache[int32, *models.CarModelVariant]
}

// NewCachedCarModelVariantsRepository wraps repo with a read-through cache keyed by ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedCarModelVariantsRepository(
	repo CarModelVariantsRepository,
	byID *cache.Cache[int32, *models.CarModelVariant],
) CarModelVariantsRepository {
	return &cachedCarModelVariantsRepository{CarModelVariantsRepository: repo, byID: byID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelVariantsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarModelVariant, error) {
	if entity, ok := r.byID.Get(ctx, id); ok {
		return entity, nil
	}

	entity, err := r.CarModelVariantsRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, entity)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelVariantsRepository) Create(
	ctx context.Context,
	input *models.CarModelVariantSetter,
) (*models.CarModelVariant, error) {
	entity, err := r.CarModelVariantsRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, entity.ID, entity, cache.ActionUpdated)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelVariantsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarModelVariantSetter,
) (*models.CarModelVariant, error) {
	entity, err := r.CarModelVariantsRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarModelVariantsRepository) DeleteByID(
	ctx context.Context,
	id int32,
) error {
	if err := r.CarModelVariantsRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}
