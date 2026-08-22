//nolint:dupl // cache decorators are intentionally parallel
package cars

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedCarManufacturersRepository struct {
	CarManufacturersRepository
	byID *cache.Cache[int32, *models.CarManufacturer]
}

// NewCachedCarManufacturersRepository wraps repo with a read-through cache keyed by ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedCarManufacturersRepository(
	repo CarManufacturersRepository,
	byID *cache.Cache[int32, *models.CarManufacturer],
) CarManufacturersRepository {
	return &cachedCarManufacturersRepository{CarManufacturersRepository: repo, byID: byID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarManufacturersRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.CarManufacturer, error) {
	if entity, ok := r.byID.Get(ctx, id); ok {
		return entity, nil
	}

	entity, err := r.CarManufacturersRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, entity)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarManufacturersRepository) Create(
	ctx context.Context,
	input *models.CarManufacturerSetter,
) (*models.CarManufacturer, error) {
	entity, err := r.CarManufacturersRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, entity.ID, entity, cache.ActionUpdated)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarManufacturersRepository) Update(
	ctx context.Context,
	id int32,
	input *models.CarManufacturerSetter,
) (*models.CarManufacturer, error) {
	entity, err := r.CarManufacturersRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedCarManufacturersRepository) DeleteByID(
	ctx context.Context,
	id int32,
) error {
	if err := r.CarManufacturersRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}
