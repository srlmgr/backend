package seasons

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

// cachedRepository decorates a Repository with a read-through cache for LoadByID.
type cachedRepository struct {
	Repository
	byID *cache.Cache[int32, *models.Season]
}

// NewCached wraps repo with a read-through cache keyed by season ID.
func NewCached(repo Repository, byID *cache.Cache[int32, *models.Season]) Repository {
	return &cachedRepository{Repository: repo, byID: byID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) LoadByID(
	ctx context.Context, id int32,
) (*models.Season, error) {
	if v, ok := r.byID.Get(ctx, id); ok {
		return v, nil
	}

	v, err := r.Repository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, v)
	return v, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeasonSetter,
) (*models.Season, error) {
	v, err := r.Repository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	// remove from caches rather than Set to avoid a stale-write race
	r.byID.DeleteAndPublish(ctx, id)
	return v, nil
}

func (r *cachedRepository) DeleteByID(ctx context.Context, id int32) error {
	if err := r.Repository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}
