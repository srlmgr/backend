package resultentries

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

// cachedRepository decorates a Repository with a read-through cache for LoadByID.
type cachedRepository struct {
	Repository
	byIDType *cache.Cache[cache.CompositeKey[int32], []*models.ResultEntry]
}

// NewCached wraps repo with a read-through cache keyed by season ID.
//
//nolint:whitespace // editor/linter issue
func NewCached(
	repo Repository,
	byIDType *cache.Cache[cache.CompositeKey[int32],
		[]*models.ResultEntry],
) Repository {
	return &cachedRepository{Repository: repo, byIDType: byIDType}
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.ResultEntry, error) {
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
func (r *cachedRepository) LoadByRaceGridID(
	ctx context.Context,
	raceGridID int32,
) ([]*models.ResultEntry, error) {
	searchKey := cache.NewCompositeKey(cache.IDRaceGrid, raceGridID)
	if entries, ok := r.byIDType.Get(ctx, searchKey); ok {
		return entries, nil
	}
	entries, err := r.Repository.LoadByRaceGridID(ctx, raceGridID)
	if err != nil {
		return nil, err
	}
	r.byIDType.Set(ctx, searchKey, entries)
	return entries, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) DeleteByTeamID(ctx context.Context, teamID int32) (
	int64, error,
) {
	num, err := r.Repository.DeleteByTeamID(ctx, teamID)
	if num > 0 {
		// we don't collect the affected events, instead we flush the cache
		r.byIDType.Flush()
	}
	return num, err
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) DeleteByRaceGridID(
	ctx context.Context,
	gridID int32,
) error {
	err := r.Repository.DeleteByRaceGridID(ctx, gridID)
	if err == nil {
		// we don't collect the affected events, instead we flush the cache
		r.byIDType.Flush()
	}
	return err
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) DeleteByRaceGridIDs(
	ctx context.Context,
	gridIDs []int32,
) error {
	err := r.Repository.DeleteByRaceGridIDs(ctx, gridIDs)
	if err == nil {
		// we don't collect the affected events, instead we flush the cache
		r.byIDType.Flush()
	}
	return err
}

//nolint:whitespace // editor/linter issue
func (r *cachedRepository) Update(
	ctx context.Context,
	id int32,
	input *models.ResultEntrySetter,
) (*models.ResultEntry, error) {
	entity, err := r.Repository.Update(ctx, id, input)
	if err == nil {
		// we don't collect the affected events, instead we flush the cache
		r.byIDType.Flush()
	}
	return entity, err
}
