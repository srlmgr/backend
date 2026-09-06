package bookingentries

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

// cachedRepository decorates a Repository with a read-through cache for LoadByID.
type cachedRepository struct {
	Repository
	byIDType *cache.Cache[cache.CompositeKey[int32], []*models.BookingEntry]
}

// NewCached wraps repo with a read-through cache keyed by season or event ID.
//
//nolint:whitespace // editor/linter issue
func NewCached(
	repo Repository,
	byIDType *cache.Cache[cache.CompositeKey[int32],
		[]*models.BookingEntry],
) Repository {
	return &cachedRepository{Repository: repo, byIDType: byIDType}
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.BookingEntry, error) {
	searchKey := cache.NewCompositeKey(cache.IDSeason, seasonID)
	if entries, ok := r.byIDType.Get(ctx, searchKey); ok {
		return entries, nil
	}
	entries, err := r.Repository.LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	r.byIDType.Set(ctx, searchKey, entries)
	return entries, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) LoadByEventID(
	ctx context.Context, eventID int32,
) ([]*models.BookingEntry, error) {
	searchKey := cache.NewCompositeKey(cache.IDEvent, eventID)
	if entries, ok := r.byIDType.Get(ctx, searchKey); ok {
		return entries, nil
	}
	entries, err := r.Repository.LoadByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	r.byIDType.Set(ctx, searchKey, entries)
	return entries, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) Create(
	ctx context.Context,
	input *models.BookingEntrySetter,
) (*models.BookingEntry, error) {
	entity, err := r.Repository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byIDType.DeleteAndPublish(ctx, cache.NewCompositeKey(cache.IDEvent, entity.EventID))
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) CreateMany(
	ctx context.Context,
	input []*models.BookingEntrySetter,
) ([]*models.BookingEntry, error) {
	entities, err := r.Repository.CreateMany(ctx, input)
	if err != nil {
		return nil, err
	}
	invalidated := make(map[int32]struct{})
	for _, entity := range entities {
		if _, ok := invalidated[entity.EventID]; ok {
			continue
		}
		invalidated[entity.EventID] = struct{}{}
		r.byIDType.DeleteAndPublish(
			ctx,
			cache.NewCompositeKey(cache.IDEvent, entity.EventID),
		)
	}
	return entities, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) Update(
	ctx context.Context,
	id int32,
	input *models.BookingEntrySetter,
) (*models.BookingEntry, error) {
	entity, err := r.Repository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byIDType.DeleteAndPublish(ctx, cache.NewCompositeKey(cache.IDEvent, entity.EventID))
	return entity, nil
}

func (r *cachedRepository) DeleteByID(ctx context.Context, id int32) error {
	// look up the event ID beforehand since it is not derivable after deletion
	existing, loadErr := r.LoadByID(ctx, id)
	if err := r.Repository.DeleteByID(ctx, id); err != nil {
		return err
	}
	if loadErr == nil {
		r.byIDType.DeleteAndPublish(
			ctx,
			cache.NewCompositeKey(cache.IDEvent, existing.EventID),
		)
	}
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) DeleteNonManualByEvent(
	ctx context.Context, eventID int32,
) error {
	if err := r.Repository.DeleteNonManualByEvent(ctx, eventID); err != nil {
		return err
	}
	r.byIDType.DeleteAndPublish(ctx, cache.NewCompositeKey(cache.IDEvent, eventID))
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) CleanupByEventID(
	ctx context.Context, eventID int32, includeManuals bool,
) error {
	if err := r.Repository.CleanupByEventID(ctx, eventID, includeManuals); err != nil {
		return err
	}
	r.byIDType.DeleteAndPublish(ctx, cache.NewCompositeKey(cache.IDEvent, eventID))
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) CleanupByRaceIDs(
	ctx context.Context, raceIDs []int32, includeManuals bool,
) error {
	if err := r.Repository.CleanupByRaceIDs(ctx, raceIDs, includeManuals); err != nil {
		return err
	}
	// race IDs don't map to specific season/event cache keys, so flush the whole cache
	r.byIDType.Flush()
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) CleanupByGridIDs(
	ctx context.Context, gridIDs []int32, includeManuals bool,
) error {
	if err := r.Repository.CleanupByGridIDs(ctx, gridIDs, includeManuals); err != nil {
		return err
	}
	// grid IDs don't map to specific season/event cache keys, so flush the whole cache
	r.byIDType.Flush()
	return nil
}
