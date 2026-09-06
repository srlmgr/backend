package impl

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
	"github.com/srlmgr/backend/service/standings"
)

type (
	standingsImpl struct {
		r      repository.Repository
		logger *log.Logger
		tracer trace.Tracer
	}
)

var _ service.StandingsService = (*standingsImpl)(nil)

//nolint:whitespace //editor/linter issue
func newStandingsImpl(
	r repository.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
) *standingsImpl {
	return &standingsImpl{
		r:      r,
		logger: logger,
		tracer: tracer,
	}
}

//nolint:whitespace //editor/linter issue
func (s *standingsImpl) GetSeasonStandings(
	ctx context.Context,
	seasonID int,
	skipMode standings.SkipModeType) (
	*service.StandingsContainer, error,
) {
	l := s.logger.WithCtx(ctx)

	l.Debug("GetStandings for season", log.Int("season_id", seasonID))

	// Load the event to capture current processing state.
	events, err := s.r.Events().LoadBySeasonID(ctx, int32(seasonID))
	if err != nil {
		l.Error("failed to load events", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load events")
		return nil, err
	}
	if len(events) == 0 {
		l.Error("no events found for season", log.Int("season_id", seasonID))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "no events found for season")
		return nil, err
	}
	event := events[len(events)-1]
	return s.GetEventStandings(ctx, int(event.ID), skipMode)
}

//nolint:whitespace,funlen //editor/linter issue
func (s *standingsImpl) GetEventStandings(
	ctx context.Context,
	eventID int,
	skipMode standings.SkipModeType) (
	*service.StandingsContainer, error,
) {
	l := s.logger.WithCtx(ctx)

	l.Debug("GetStandings for event", log.Int("event_id", eventID))

	// Load the event to capture current processing state.
	event, err := s.r.Events().LoadByID(ctx, int32(eventID))
	if err != nil {
		return nil, err
	}

	bookingEntries, err := doInSpan(ctx, l, s.tracer, "booking entries",
		func(ctx context.Context) ([]*models.BookingEntry, error) {
			return s.r.BookingEntries().LoadBySeasonID(ctx, event.SeasonID)
		})
	if err != nil {
		return nil, err
	}

	resultEntries, err := doInSpan(ctx, l, s.tracer, "result entries",
		func(ctx context.Context) ([]*models.ResultEntry, error) {
			return s.r.ResultEntries().LoadBySeasonID(ctx, event.SeasonID)
		})
	if err != nil {
		return nil, err
	}

	var epi *processor.EventProcInfo
	epi, err = doInSpan(ctx, l, s.tracer, "event processing info",
		func(ctx context.Context) (*processor.EventProcInfo, error) {
			ep := processor.NewEventProcInfoCollector(s.r)
			return ep.ForEvent(ctx, event.ID)
		})
	if err != nil {
		return nil, err
	}

	allEvents, err := doInSpan(ctx, l, s.tracer, "season events",
		func(ctx context.Context) ([]*models.Event, error) {
			return s.r.Events().LoadBySeasonID(ctx, event.SeasonID)
		})
	if err != nil {
		return nil, err
	}

	useEvents := lo.Filter(allEvents, func(item *models.Event, _ int) bool {
		return item.SequenceNo <= event.SequenceNo
	})
	useEventIDs := lo.Map(useEvents, func(item *models.Event, _ int) int32 {
		return item.ID
	})

	ret, err := doInSpan(ctx, l, s.tracer, "compute standings",
		func(ctx context.Context) (*service.StandingsContainer, error) {
			sp := &standingProc{
				epi:            epi,
				eventIDs:       useEventIDs,
				bookingEntries: bookingEntries,
				resultEntries:  resultEntries,
				skipMode:       skipMode,
			}
			primary, secondary := sp.computeStandings()
			ret := &service.StandingsContainer{
				Season:    epi.Season,
				Primary:   primary,
				Secondary: secondary,
			}
			return ret, nil
		})
	return ret, err
}

type standingProc struct {
	epi            *processor.EventProcInfo
	eventIDs       []int32
	bookingEntries []*models.BookingEntry
	resultEntries  []*models.ResultEntry
	skipMode       standings.SkipModeType
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computeStandings() (
	primary []*service.Standing,
	secondary []*service.Standing,
) {
	if sp.epi.Season.IsTeamBased {
		primary = sp.computePrimaryFromTeamBookings()

		secondary = sp.computePrimaryFromDriverBookings()

	} else {
		primary = sp.computePrimaryFromDriverBookings()

		secondary = sp.computeSecondaryFromTeamContribution()

	}
	return primary, secondary
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computePrimaryFromTeamBookings() (
	ret []*service.Standing,
) {
	ret = sp.computePrimaryByParam(func(be *models.BookingEntry) bool {
		return be.TargetType == "team"
	},
		func(be *models.BookingEntry) int32 {
			return be.TeamID.GetOrZero()
		},
		func(re *models.ResultEntry) int32 {
			return re.TeamID.GetOrZero()
		},
		service.StandingsTypeTeam)
	return ret
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computePrimaryFromDriverBookings() (
	ret []*service.Standing,
) {
	ret = sp.computePrimaryByParam(
		func(be *models.BookingEntry) bool {
			return be.TargetType == "driver"
		},
		func(be *models.BookingEntry) int32 {
			return be.DriverID.GetOrZero()
		},
		func(re *models.ResultEntry) int32 {
			return re.DriverID.GetOrZero()
		},
		service.StandingsTypeDriver,
	)
	return ret
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computePrimaryByParam(
	bookingFilter func(*models.BookingEntry) bool,
	refIDBookingFunc func(*models.BookingEntry) int32,
	refIDResultFunc func(*models.ResultEntry) int32,
	standingsType service.StandingsType,
) (
	ret []*service.Standing,
) {
	bookingsByClass := lo.GroupBy(sp.bookingEntries,
		func(item *models.BookingEntry) int32 { return item.CarClassID.GetOrZero() })

	resultsByClass := sp.recomputeFinishPosByClassAndGrid()
	ret = make([]*service.Standing, 0)
	for classID := range bookingsByClass {
		classBookings := lo.Filter(bookingsByClass[classID],
			func(item *models.BookingEntry, _ int) bool {
				return bookingFilter(item)
			})
		classResults := resultsByClass[classID]
		comp := standings.NewComputeStandings()
		computedStandings := comp.Compute(&standings.ComputeStandingsInput{
			EventIDs: sp.eventIDs,
			Bookings: classBookings,
			Participations: standings.ParticipationsFromResultEntries(
				classResults,
				refIDResultFunc,
			),
			NumTotalEvents: len(sp.eventIDs),
			NumSkip:        int(sp.epi.Season.SkipEvents),
			SkipMode:       sp.skipMode,
			ReferenceID:    refIDBookingFunc,
		})
		tmp := sp.convertToStandings(
			standingsType,
			classID,
			computedStandings,
		)
		ret = append(ret, tmp...)
	}
	return ret
}

// recomputes the finish position by classID as the finish pos is the overall race pos
//
//nolint:lll // readability
func (sp *standingProc) recomputeFinishPosByClassAndGrid() map[int32][]*models.ResultEntry {
	ret := make(map[int32][]*models.ResultEntry)
	// first: split by classID
	resultsByClass := lo.GroupBy(sp.resultEntries,
		func(item *models.ResultEntry) int32 { return item.CarClassID.GetOrZero() })
	for classID := range resultsByClass {
		classResults := resultsByClass[classID]
		// second: split by race grid
		byRaceGrid := lo.GroupBy(classResults,
			func(item *models.ResultEntry) int32 { return item.RaceGridID })
		allResultsForClass := make([]*models.ResultEntry, 0)
		for _, raceGridResults := range byRaceGrid {
			// sort by finish position
			slices.SortStableFunc(raceGridResults, func(a, b *models.ResultEntry) int {
				return cmp.Compare(a.FinishPosition, b.FinishPosition)
			})
			// in the sorted result reset the finish pos by class/race grid
			// and add the sorted results to the combined class results
			for pos, result := range raceGridResults {
				result.FinishPosition = int32(pos + 1)
				allResultsForClass = append(allResultsForClass, result)
			}
		}
		ret[classID] = allResultsForClass
	}
	return ret
}

//nolint:whitespace,funlen // editor/linter issue
func (sp *standingProc) computeSecondaryFromTeamContribution() (
	ret []*service.Standing,
) {
	bookingsByEventID := make(map[int32][]*models.BookingEntry)
	for _, booking := range sp.bookingEntries {
		if booking == nil {
			continue
		}
		bookingsByEventID[booking.EventID] = append(
			bookingsByEventID[booking.EventID],
			booking,
		)
	}
	// contains standing data per team.
	// gets updated per event
	workStandings := make(map[int32]*service.Standing)
	for _, eventID := range sp.eventIDs {
		eventBookings := bookingsByEventID[eventID]
		currentTeamContribution := sp.aggregateTeamBookings(eventBookings)
		for _, contribution := range currentTeamContribution {
			if contribution == nil {
				continue
			}
			current, ok := workStandings[int32(contribution.ReferenceID)]
			if !ok {
				workStandings[int32(contribution.ReferenceID)] = contribution
			} else {
				current.Data.TotalPoints += contribution.Data.TotalPoints
				current.Data.PrevPosition = current.Data.Position
			}
		}
		orderedReferenceIDs := slices.Collect(maps.Keys(workStandings))
		slices.SortFunc(orderedReferenceIDs, func(a, b int32) int {
			left := workStandings[a]
			right := workStandings[b]
			diff := cmp.Compare(right.Data.TotalPoints, left.Data.TotalPoints)
			return diff
		})
		for pos, referenceID := range orderedReferenceIDs {
			current := workStandings[referenceID]
			current.Data.Position = int32(pos + 1)
			current.EventID = int(eventID)
		}
	}
	// sort the return results by position
	ret = lo.Values(workStandings)
	slices.SortFunc(ret, func(a, b *service.Standing) int {
		return cmp.Compare(a.Data.Position, b.Data.Position)
	})
	return ret
}

//nolint:whitespace,funlen // editor/linter issue
func (sp *standingProc) aggregateTeamBookings(
	eventBookings []*models.BookingEntry) (
	ret []*service.Standing,
) {
	ret = make([]*service.Standing, 0)
	bookingsByClass := lo.GroupBy(eventBookings,
		func(item *models.BookingEntry) int32 { return item.CarClassID.GetOrZero() })
	for classID := range bookingsByClass {
		classBookings := lo.Filter(bookingsByClass[classID],
			func(item *models.BookingEntry, _ int) bool {
				return item.TargetType == "team" && item.SourceType == "team_contribution"
			})
		teamMap := lo.Reduce(
			classBookings,
			func(
				acc map[int32]*models.BookingEntry,
				e *models.BookingEntry,
				_ int,
			) map[int32]*models.BookingEntry {
				if current := acc[e.TeamID.GetOrZero()]; current != nil {
					current.Points += e.Points
				} else {
					current = &models.BookingEntry{
						TeamID: e.TeamID,
						Points: e.Points,
					}
					acc[e.TeamID.GetOrZero()] = current
				}
				return acc
			},
			make(map[int32]*models.BookingEntry),
		)
		raw := lo.Values(teamMap)
		slices.SortStableFunc(raw, func(a, b *models.BookingEntry) int {
			return int(b.Points - a.Points)
		})

		x := lo.Map(raw, func(be *models.BookingEntry, _ int) *service.Standing {
			return &service.Standing{
				Type:        service.StandingsTypeTeam,
				ReferenceID: int(be.TeamID.GetOrZero()),
				EventID:     int(be.EventID),
				CarClassID:  int(be.CarClassID.GetOrZero()),
				Data: &standings.StandingData{
					TotalPoints: be.Points,
				},
			}
		})
		ret = append(ret, x...)

	}
	return ret
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) convertToStandings(
	standingType service.StandingsType,
	carClassID int32,
	computedStandings []*standings.ComputedStanding) (
	ret []*service.Standing,
) {
	ret = make([]*service.Standing, 0, len(computedStandings))
	for _, computedStanding := range computedStandings {
		ret = append(ret, &service.Standing{
			Type:        standingType,
			ReferenceID: int(computedStanding.ReferenceID),
			EventID:     int(sp.epi.Event.ID),
			CarClassID:  int(carClassID),
			Data:        computedStanding.StandingData,
			// DroppedEventIds: toUint32Slice(computedStanding.SkipEventIDs),
		})
	}
	return ret
}

//nolint:whitespace //editor/linter issue
func doInSpan[T any](
	ctx context.Context,
	l *log.Logger,
	tracer trace.Tracer,
	spanName string,
	f func(ctx context.Context) (T, error),
) (T, error) {
	spanCtx, span := tracer.Start(ctx, fmt.Sprintf("collect  %s", spanName))
	defer span.End()
	ret, err := f(spanCtx)
	if err != nil {

		errText := fmt.Sprintf("failed to load %s", spanName)
		l.Error(errText, log.ErrorField(err))
		span.SetStatus(codes.Error, errText)
		return ret, err
	}
	return ret, nil
}
