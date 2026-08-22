package drivers

import (
	"context"

	"github.com/samber/lo"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

// cachedRepository decorates a Repository with a read-through cache for LoadByID.
type cachedDriversRepository struct {
	DriversRepository
	byID *cache.Cache[int32, *models.Driver]
}

// NewCached wraps repo with a read-through cache keyed by driver ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedDriversRepository(
	repo DriversRepository,
	byID *cache.Cache[int32, *models.Driver],
) DriversRepository {
	return &cachedDriversRepository{DriversRepository: repo, byID: byID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedDriversRepository) LoadByID(
	ctx context.Context, id int32,
) (*models.Driver, error) {
	if v, ok := r.byID.Get(ctx, id); ok {
		return v, nil
	}

	v, err := r.DriversRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, v)
	return v, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedDriversRepository) LoadByIDs(
	ctx context.Context, ids []int32,
) ([]*models.Driver, error) {
	cachedIDs := r.byID.Keys()
	nonCachedIDs, _ := lo.Difference(ids, cachedIDs)

	cachedReturn := func() []*models.Driver {
		return lo.Map(ids, func(id int32, _ int) *models.Driver {
			v, _ := r.byID.Get(ctx, id)
			return v
		})
	}
	// read remaining IDs from the underlying repository and populate the cache
	if len(nonCachedIDs) > 0 {
		v, err := r.DriversRepository.LoadByIDs(ctx, nonCachedIDs)
		if err != nil {
			return nil, err
		}
		for _, d := range v {
			r.byID.Set(ctx, d.ID, d)
		}
	}

	return cachedReturn(), nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedDriversRepository) Update(
	ctx context.Context,
	id int32,
	input *models.DriverSetter,
) (*models.Driver, error) {
	v, err := r.DriversRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	// remove from caches rather than Set to avoid a stale-write race
	r.byID.DeleteAndPublish(ctx, id)
	return v, nil
}

func (r *cachedDriversRepository) DeleteByID(ctx context.Context, id int32) error {
	if err := r.DriversRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedDriversRepository) Create(
	ctx context.Context,
	input *models.DriverSetter,
) (*models.Driver, error) {
	v, err := r.DriversRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, v.ID, v, cache.ActionUpdated)
	return v, nil
}
