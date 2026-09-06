package service

import (
	"context"

	"github.com/samber/lo"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/html/server/model"
	gs "github.com/srlmgr/backend/service"
	"github.com/srlmgr/backend/service/summary"
)

//nolint:whitespace,funlen //editor/linter issue
func (s *serviceImpl) GetResultsOverview(
	ctx context.Context,
	seasonID, classID int,
) (
	*model.SeasonResultsOverviewContainer, error,
) {
	overview, err := s.svc.SummaryService().GetSeasonSummary(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	seasonContainer, err := s.GetSeasonList(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	events, err := s.r.Events().LoadBySeasonID(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}
	ret := &model.SeasonResultsOverviewContainer{
		ServiceData:      overview,
		SeasonsContainer: seasonContainer,
		Events: lo.Map(events, func(e *dbModels.Event, _ int) *model.Event {
			return &model.Event{
				ID:   int(e.ID),
				Name: e.Name,
				Date: e.EventDate,
			}
		}),
	}

	if overview.Season.IsMulticlass {
		carClasses, ccErr := s.r.Cars().CarClasses().LoadBySeasonID(
			ctx, overview.Season.ID,
		)
		if ccErr != nil {
			return nil, ccErr
		}
		ret.CarClasses = lo.Map(carClasses,
			func(cc *dbModels.CarClass, _ int) *model.CarClass {
				return &model.CarClass{
					ID:   int(cc.ID),
					Name: cc.Name,
				}
			})
		filterByClassID := func(se *summary.SummaryEntry, _ int) bool {
			return se.CarClassID == classID
		}
		//nolint:lll // readability
		for i := range overview.Events {
			overview.Events[i].Primary = lo.Filter(overview.Events[i].Primary, filterByClassID)
			overview.Events[i].Secondary = lo.Filter(overview.Events[i].Secondary, filterByClassID)
		}
		overview.PrimarySummaries = lo.Filter(overview.PrimarySummaries, filterByClassID)
		overview.SecondarySummaries = lo.Filter(overview.SecondarySummaries, filterByClassID)

	}

	entryResolver, err := newSeasonEntryComposer(s.r, ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}
	teamLookup := entryResolver.CreateTeamLookup(ctx)

	if overview.Season.IsTeamBased {
		ret.PrimaryLookup = teamLookup
		driverLookup := entryResolver.CreateDriverLookupByTeams(ctx)
		ret.SecondaryLookup = driverLookup
	} else {
		ids := collectPrimaryIDs(overview)
		driverLookup := entryResolver.CreateDriverLookupByIDs(ctx, ids)
		ret.PrimaryLookup = driverLookup
		ret.SecondaryLookup = teamLookup
	}

	ret.PrimaryMatrixLookup = createPrimaryMatrixLookup(overview)
	ret.SecondaryMatrixLookup = createSecondaryMatrixLookup(overview)
	return ret, nil
}

//nolint:whitespace //editor/linter issue
func createPrimaryMatrixLookup(
	o *gs.SummaryContainer,
) map[model.IDTuple]int {
	lookup := make(map[model.IDTuple]int)
	for _, s := range o.Events {
		for _, e := range s.Primary {
			lookup[model.IDTuple{
				ReferenceID: e.ReferenceID, SubID: s.EventID,
			}] = e.Points.TotalPoints
		}
	}
	return lookup
}

//nolint:whitespace //editor/linter issue
func createSecondaryMatrixLookup(
	o *gs.SummaryContainer,
) map[model.IDTuple]int {
	lookup := make(map[model.IDTuple]int)
	for _, s := range o.Events {
		for _, e := range s.Secondary {
			lookup[model.IDTuple{
				ReferenceID: e.ReferenceID, SubID: s.EventID,
			}] = e.Points.TotalPoints
		}
	}
	return lookup
}

func collectPrimaryIDs(o *gs.SummaryContainer) []int32 {
	work := make(map[int32]struct{})
	for _, s := range o.Events {
		for _, e := range s.Primary {
			work[int32(e.ReferenceID)] = struct{}{}
		}
	}

	ret := make([]int32, 0, len(work))
	for id := range work {
		ret = append(ret, id)
	}
	return ret
}
