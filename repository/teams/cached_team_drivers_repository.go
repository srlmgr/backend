package teams

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedTeamDriversRepository struct {
	TeamDriversRepository
	byTeamID *cache.Cache[int32, []*models.TeamDriver]
}

// NewCachedTeamDriversRepository wraps repo with a read-through cache keyed by team ID.
//
//nolint:whitespace // editor/linter issue
func NewCachedTeamDriversRepository(
	repo TeamDriversRepository,
	byTeamID *cache.Cache[int32, []*models.TeamDriver],
) TeamDriversRepository {
	return &cachedTeamDriversRepository{TeamDriversRepository: repo, byTeamID: byTeamID}
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamDriversRepository) LoadByTeamID(
	ctx context.Context,
	teamID int32,
) ([]*models.TeamDriver, error) {
	if entities, ok := r.byTeamID.Get(ctx, teamID); ok {
		return entities, nil
	}

	entities, err := r.TeamDriversRepository.LoadByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	r.byTeamID.Set(ctx, teamID, entities)
	return entities, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamDriversRepository) Create(
	ctx context.Context,
	input *models.TeamDriverSetter,
) (*models.TeamDriver, error) {
	entity, err := r.TeamDriversRepository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	r.byTeamID.DeleteAndPublish(ctx, entity.TeamID)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamDriversRepository) Update(
	ctx context.Context,
	id int32,
	input *models.TeamDriverSetter,
) (*models.TeamDriver, error) {
	entity, err := r.TeamDriversRepository.Update(ctx, id, input)
	if err != nil {
		return nil, err
	}
	r.byTeamID.DeleteAndPublish(ctx, entity.TeamID)
	return entity, nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamDriversRepository) DeleteByID(
	ctx context.Context, id int32,
) error {
	existing, loadErr := r.LoadByID(ctx, id)
	if err := r.TeamDriversRepository.DeleteByID(ctx, id); err != nil {
		return err
	}
	if loadErr == nil {
		r.byTeamID.DeleteAndPublish(ctx, existing.TeamID)
	}
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *cachedTeamDriversRepository) DeleteByTeamID(
	ctx context.Context,
	teamID int32,
) error {
	if err := r.TeamDriversRepository.DeleteByTeamID(ctx, teamID); err != nil {
		return err
	}
	r.byTeamID.DeleteAndPublish(ctx, teamID)
	return nil
}
