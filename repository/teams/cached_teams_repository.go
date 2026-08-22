package teams

import (
	"context"

	"github.com/samber/lo"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedTeamsRepository struct {
	TeamsRepository
	byID       *cache.Cache[int32, *models.Team]
	bySeasonID *cache.Cache[int32, []*models.Team]
}

// NewCachedTeamsRepository wraps repo with a read-through cache keyed by ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedTeamsRepository(
	repo TeamsRepository,
	byID *cache.Cache[int32, *models.Team],
	bySeasonID *cache.Cache[int32, []*models.Team],
) TeamsRepository {
	return &cachedTeamsRepository{
		TeamsRepository: repo,
		byID:            byID,
		bySeasonID:      bySeasonID,
	}
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamsRepository) LoadByID(
	ctx context.Context, id int32,
) (*models.Team, error) {
	if entity, ok := r.byID.Get(ctx, id); ok {
		return entity, nil
	}

	entity, err := r.TeamsRepository.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.byID.Set(ctx, id, entity)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamsRepository) LoadBySeasonID(
	ctx context.Context, seasonID int32,
) ([]*models.Team, error) {
	if entities, ok := r.bySeasonID.Get(ctx, seasonID); ok {
		return entities, nil
	}

	entities, err := r.TeamsRepository.LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	r.bySeasonID.Set(ctx, seasonID, entities)
	lo.ForEach(entities, func(item *models.Team, _ int) {
		r.byID.Set(ctx, item.ID, item)
	})
	return entities, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamsRepository) Create(
	ctx context.Context,
	input *models.TeamSetter,
) (*models.Team, error) {
	entity, err := r.TeamsRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byID.SetAndPublish(ctx, entity.ID, entity, cache.ActionUpdated)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.TeamSetter,
) (*models.Team, error) {
	entity, err := r.TeamsRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamsRepository) DeleteByID(
	ctx context.Context, id int32,
) error {
	if err := r.TeamsRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	r.byID.DeleteAndPublish(ctx, id)
	return nil
}
