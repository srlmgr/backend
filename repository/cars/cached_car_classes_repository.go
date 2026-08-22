//nolint:dupl // cache decorators are intentionally parallel
package cars

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedCarClassesRepository struct {
	CarClassesRepository
	byID *cache.Cache[int32, *models.CarClass]
}

// NewCachedCarClassesRepository wraps repo with a read-through cache keyed by ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedCarClassesRepository(
	repo CarClassesRepository,
	byID *cache.Cache[int32, *models.CarClass],
) CarClassesRepository {
	return &cachedCarClassesRepository{CarClassesRepository: repo, byID: byID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarClassesRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarClass, error) {
	if entity, ok := r.byID.Get(ctx, id); ok {
		return entity, nil
	}

	entity, err := r.CarClassesRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, entity)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarClassesRepository) Create(
	ctx context.Context,
	input *models.CarClassSetter,
) (*models.CarClass, error) {
	entity, err := r.CarClassesRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, entity.ID, entity, cache.ActionUpdated)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarClassesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarClassSetter,
) (*models.CarClass, error) {
	entity, err := r.CarClassesRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarClassesRepository) DeleteByID(
	ctx context.Context,
	id int32,
) error {
	if err := r.CarClassesRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}
