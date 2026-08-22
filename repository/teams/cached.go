package teams

import (
	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/db/models"
)

type cachedRepository struct {
	teams       TeamsRepository
	teamDrivers TeamDriversRepository
}

// Caches contains the read-through caches used by Repository.
type Caches struct {
	Teams       *cache.Cache[int32, *models.Team]
	SeasonTeams *cache.Cache[int32, []*models.Team]
	TeamDrivers *cache.Cache[int32, []*models.TeamDriver]
}

// NewCached wraps repo with cached child repositories.
func NewCached(repo Repository, caches *Caches) Repository {
	return &cachedRepository{
		teams:       NewCachedTeamsRepository(repo.Teams(), caches.Teams, caches.SeasonTeams),
		teamDrivers: NewCachedTeamDriversRepository(repo.TeamDrivers(), caches.TeamDrivers),
	}
}

func (r *cachedRepository) Teams() TeamsRepository { return r.teams }

func (r *cachedRepository) TeamDrivers() TeamDriversRepository { return r.teamDrivers }
