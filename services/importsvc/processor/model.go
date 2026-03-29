package processor

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/importsvc/points"
)

type (
	EventProcInfo struct {
		Event *models.Event
		Races []*models.Race
		Grids []*models.RaceGrid
	}
	EventProcInfoCollector struct {
		repos repository.Repository
	}
)

var (
	ErrRaceNotFound = fmt.Errorf("race not found")
	ErrGridNotFound = fmt.Errorf("grid not found")
)

func NewEventProcInfoCollector(repos repository.Repository) *EventProcInfoCollector {
	return &EventProcInfoCollector{
		repos: repos,
	}
}

func (e *EventProcInfoCollector) ForEvent(ctx context.Context, eventID int32) (
	*EventProcInfo, error,
) {
	event, err := e.repos.Events().LoadByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	races, err := e.repos.Races().Races().LoadByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	grids := make([]*models.RaceGrid, 0)
	for rIdx := range races {
		raceGrids, err := e.repos.Races().RaceGrids().LoadByRaceID(ctx, races[rIdx].ID)
		if err != nil {
			return nil, err
		}
		grids = append(grids, raceGrids...)
	}
	return &EventProcInfo{
		Event: event,
		Races: races,
		Grids: grids,
	}, nil
}

func (epi *EventProcInfo) ResolverFunc(ctx context.Context) points.ResolveGridID {
	return func(gridID int32) (raceNo, gridNo int32, err error) {
		grid, ok := lo.Find(epi.Grids, func(item *models.RaceGrid) bool {
			return item.ID == gridID
		})
		if !ok {
			return 0, 0, ErrGridNotFound
		}
		race, ok := lo.Find(epi.Races, func(item *models.Race) bool {
			return item.ID == grid.RaceID
		})
		if !ok {
			return 0, 0, ErrRaceNotFound
		}
		return race.SequenceNo, grid.SequenceNo, nil
	}
}
