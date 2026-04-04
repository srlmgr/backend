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
		Event               *models.Event
		Races               []*models.Race
		Grids               []*models.RaceGrid
		Season              *models.Season
		PointSystem         *models.PointSystem
		PointSystemSettings *points.PointSystemSettings
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

//nolint:whitespace //editor/linter issue
func (e *EventProcInfoCollector) ForEvent(ctx context.Context, eventID int32) (
	*EventProcInfo, error,
) {
	event, err := e.repos.Events().LoadByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	season, err := e.repos.Seasons().LoadByID(ctx, event.SeasonID)
	if err != nil {
		return nil, err
	}

	pointSystem, err := e.repos.PointSystems().PointSystems().LoadByID(
		ctx, season.PointSystemID)
	if err != nil {
		return nil, err
	}
	// TODO: remove dummy when fully loaded by repo
	pointSystemSettings := e.fakePointSystemSettings(pointSystem, season)

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
		Event:               event,
		Races:               races,
		Grids:               grids,
		Season:              season,
		PointSystem:         pointSystem,
		PointSystemSettings: pointSystemSettings,
	}, nil
}

//nolint:whitespace //editor/linter issue
func (e *EventProcInfoCollector) fakePointSystemSettings(
	ps *models.PointSystem,
	season *models.Season,
) *points.PointSystemSettings {
	switch ps.Name {
	case "Standard", "VRPC":
		return e.StandardPointSystemSettings(season)
	case "VRGES":
		return e.VRGESPointSystemSettings()
	default:
		return e.StandardPointSystemSettings(season)
	}
}

//nolint:whitespace,funlen // tmp method will be replaced by database values later
func (e *EventProcInfoCollector) StandardPointSystemSettings(
	season *models.Season,
) *points.PointSystemSettings {
	settings := &points.PointSystemSettings{
		Eligibility: points.EligibilitySettings{
			RaceDistPct: 0.75,
		},
		Races: []points.RaceSettings{
			{
				Policies: []points.PointPolicyType{
					points.PointsPolicyFinishPos,
					points.PointsPolicyQualificationPos,
					points.PointsPolicyLeastIncidents,
					points.PointsPolicyFastestLap,
				},
				AwardSettings: []points.RankedPolicySettings{
					{
						Points: map[points.PointPolicyType][]points.PointType{
							points.PointsPolicyFinishPos: {
								100,
								95,
								92,
								90,
								88,
								86,
								84,
								82,
								80,
								78,
								76,
								74,
								72,
								70,
								68,
								66,
								64,
								62,
								60,
								58,
								56,
								54,
								52,
								50,
								48,
								46,
								44,
								42,
								40,
							},
							points.PointsPolicyQualificationPos: {3, 2, 1},
							points.PointsPolicyLeastIncidents:   {3, 2, 1},
							points.PointsPolicyFastestLap:       {1},
						},
					},
				},
				PenaltySettings: []points.PointPenaltySettings{
					{Arguments: map[points.PointPolicyType]any{}},
				},
			},
			{
				Policies: []points.PointPolicyType{
					points.PointsPolicyFinishPos,
					points.PointsPolicyLeastIncidents,
					points.PointsPolicyFastestLap,
				},
				AwardSettings: []points.RankedPolicySettings{
					{
						Points: map[points.PointPolicyType][]points.PointType{
							points.PointsPolicyFinishPos: {
								100,
								95,
								92,
								90,
								88,
								86,
								84,
								82,
								80,
								78,
								76,
								74,
								72,
								70,
								68,
								66,
								64,
								62,
								60,
								58,
								56,
								54,
								52,
								50,
								48,
								46,
								44,
								42,
								40,
							},

							points.PointsPolicyLeastIncidents: {3, 2, 1},
							points.PointsPolicyFastestLap:     {1},
						},
					},
				},
				PenaltySettings: []points.PointPenaltySettings{
					{Arguments: map[points.PointPolicyType]any{}},
				},
			},
		},
	}
	return settings
}

//nolint:lll,funlen // tmp method will be replaced by database values later
func (e *EventProcInfoCollector) VRGESPointSystemSettings() *points.PointSystemSettings {
	settings := &points.PointSystemSettings{
		Eligibility: points.EligibilitySettings{
			RaceDistPct: 0.75,
		},
		Races: []points.RaceSettings{
			{
				Policies: []points.PointPolicyType{
					points.PointsPolicyFinishPos,
					points.PointsPolicyLeastIncidents,
					points.PointsPolicyIncidentsExceeded,
				},
				AwardSettings: []points.RankedPolicySettings{
					{
						Points: map[points.PointPolicyType][]points.PointType{
							points.PointsPolicyFinishPos: {
								50,
								45,
								41,
								38,
								36,
								34,
								32,
								30,
								28,
								26,
								25,
								24,
								23,
								22,
								21,
								20,
								19,
								18,
								17,
								16,
								15,
								14,
								13,
								12,
								11,
								10,
								9,
								8,
								7,
								6,
								5,
								4,
								3,
								2,
								1,
							},
							points.PointsPolicyLeastIncidents: {3, 2, 1},
						},
					},
				},
				PenaltySettings: []points.PointPenaltySettings{
					{
						Arguments: map[points.PointPolicyType]any{
							points.PointsPolicyIncidentsExceeded: points.ThresholdPenaltySettings{
								Threshold:  3,
								PenaltyPct: 0.1,
							},
						},
					},
				},
			},
		},
	}
	return settings
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
		return race.SequenceNo - 1, grid.SequenceNo - 1, nil
	}
}
