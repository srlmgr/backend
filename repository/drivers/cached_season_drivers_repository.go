package drivers

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

// cachedSeasonDriversRepository decorates a Repository with a read-through cache
// for LoadBySeasonID.
type cachedSeasonDriversRepository struct {
	SeasonDriversRepository
	bySeasonID *cache.Cache[int32, []*models.SeasonDriver]
}

// NewCachedSeasonDriversRepository wraps repo with a read-through cache
// keyed by season ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedSeasonDriversRepository(
	repo SeasonDriversRepository,
	bySeasonID *cache.Cache[int32, []*models.SeasonDriver],
) SeasonDriversRepository {
	return &cachedSeasonDriversRepository{
		SeasonDriversRepository: repo, bySeasonID: bySeasonID,
	}
}

//nolint:whitespace // editor/linter issue
func (r *cachedSeasonDriversRepository) LoadBySeasonID(
	ctx context.Context, seasonID int32,
) ([]*models.SeasonDriver, error) {
	if entries, ok := r.bySeasonID.Get(ctx, seasonID); ok {
		return entries, nil
	}
	entries, err := r.SeasonDriversRepository.LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	r.bySeasonID.Set(ctx, seasonID, entries)
	return entries, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSeasonDriversRepository) Create(
	ctx context.Context,
	input *models.SeasonDriverSetter,
) (*models.SeasonDriver, error) {
	entity, err := r.SeasonDriversRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.bySeasonID.DeleteAndPublish(ctx, entity.SeasonID)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedSeasonDriversRepository) Update(
	ctx context.Context,
	id int32,
	input *models.SeasonDriverSetter,
) (*models.SeasonDriver, error) {
	entity, err := r.SeasonDriversRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.bySeasonID.DeleteAndPublish(ctx, entity.SeasonID)
	return entity, nil
}

func (r *cachedSeasonDriversRepository) DeleteByID(
	ctx context.Context, id int32,
) error {
	// look up the season ID beforehand since it is not derivable after deletion
	existing, loadErr := r.LoadByID(ctx, id)
	if err := r.SeasonDriversRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	if loadErr == nil {
		r.bySeasonID.DeleteAndPublish(ctx, existing.SeasonID)
	}
	return nil
}
