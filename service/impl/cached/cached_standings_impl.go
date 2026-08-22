package cached

import (
	"context"

	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/service"
	"github.com/srlmgr/backend/service/standings"
)

type (
	CacheStandingsKey struct {
		SeasonID int
		EventID  int
		SkipMode standings.SkipModeType
	}
	cachedStandingsImpl struct {
		service.StandingsService
		byRequest *cache.Cache[CacheStandingsKey, *service.StandingsContainer]
	}
)

//nolint:whitespace // editor/linter issue
func NewCachedStandings(
	base service.StandingsService,
	byRequest *cache.Cache[CacheStandingsKey, *service.StandingsContainer],
) service.StandingsService {
	return &cachedStandingsImpl{
		StandingsService: base,
		byRequest:        byRequest,
	}
}

//nolint:whitespace // editor/linter issue
func (s *cachedStandingsImpl) GetSeasonStandings(
	ctx context.Context,
	seasonID int,
	skipMode standings.SkipModeType,
) (*service.StandingsContainer, error) {
	key := CacheStandingsKey{SeasonID: seasonID, SkipMode: skipMode, EventID: 0}
	if v, ok := s.byRequest.Get(ctx, key); ok {
		return v, nil
	}
	v, err := s.StandingsService.GetSeasonStandings(ctx, seasonID, skipMode)
	if err != nil {
		return nil, err
	}
	s.byRequest.Set(ctx, key, v)
	return v, nil
}

//nolint:whitespace // editor/linter issue
func (s *cachedStandingsImpl) GetEventStandings(
	ctx context.Context,
	eventID int,
	skipMode standings.SkipModeType,
) (*service.StandingsContainer, error) {
	key := CacheStandingsKey{EventID: eventID, SkipMode: skipMode, SeasonID: 0}
	if v, ok := s.byRequest.Get(ctx, key); ok {
		return v, nil
	}
	v, err := s.StandingsService.GetEventStandings(ctx, eventID, skipMode)
	if err != nil {
		return nil, err
	}
	s.byRequest.Set(ctx, key, v)
	return v, nil
}
