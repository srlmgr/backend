package standings

import (
	"cmp"
	"context"
	"maps"
	"slices"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) GetStandings(
	ctx context.Context,
	req *connect.Request[queryv1.GetStandingsRequest],
) (*connect.Response[queryv1.GetStandingsResponse], error) {
	l := s.logger.WithCtx(ctx)
	eventID := int32(req.Msg.GetEventId())
	l.Debug("GetStandings", log.Int32("event_id", eventID))

	// Load the event to capture current processing state.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	spanCtx, spanBooking := s.tracer.Start(ctx, "collect booking entries")
	defer spanBooking.End()
	bookingEntries, err := s.repo.BookingEntries().LoadBySeasonID(spanCtx, event.SeasonID)
	if err != nil {
		l.Error("failed to load booking entries", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load booking entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	spanBooking.End()
	spanCtx, spanResultEntries := s.tracer.Start(ctx, "collect result entries")
	defer spanResultEntries.End()
	resultEntries, err := s.repo.ResultEntries().LoadBySeasonID(spanCtx, event.SeasonID)
	if err != nil {
		l.Error("failed to load result entries", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	spanResultEntries.End()

	var epi *processor.EventProcInfo
	ep := processor.NewEventProcInfoCollector(s.repo)
	epi, err = ep.ForEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	allEvents, err := s.repo.Events().LoadBySeasonID(ctx, event.SeasonID)
	if err != nil {
		l.Error("failed to load all events", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load all events")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	useEvents := lo.Filter(allEvents, func(item *models.Event, _ int) bool {
		return item.SequenceNo <= event.SequenceNo
	})
	useEventIDs := lo.Map(useEvents, func(item *models.Event, _ int) int32 {
		return item.ID
	})
	sp := standingProc{
		epi:            epi,
		eventIDs:       useEventIDs,
		bookingEntries: bookingEntries,
		resultEntries:  resultEntries,
	}
	primary, secondary := sp.computeStandings()

	resp := &queryv1.GetStandingsResponse{
		EventId:            req.Msg.GetEventId(),
		PrimaryStandings:   primary,
		SecondaryStandings: secondary,
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "standings loaded")
	return connect.NewResponse(resp), nil
}

type standingProc struct {
	epi            *processor.EventProcInfo
	eventIDs       []int32
	bookingEntries []*models.BookingEntry
	resultEntries  []*models.ResultEntry
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computeStandings() (
	primary []*queryv1.Standing,
	secondary []*queryv1.Standing,
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
	ret []*queryv1.Standing,
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
		queryv1.StandingsType_STANDINGS_TYPE_TEAM)
	return ret
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computePrimaryFromDriverBookings() (
	ret []*queryv1.Standing,
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
		queryv1.StandingsType_STANDINGS_TYPE_DRIVER)
	return ret
}

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computePrimaryByParam(
	bookingFilter func(*models.BookingEntry) bool,
	refIDBookingFunc func(*models.BookingEntry) int32,
	refIDResultFunc func(*models.ResultEntry) int32,
	standingsType queryv1.StandingsType,
) (
	ret []*queryv1.Standing,
) {
	bookingsByClass := lo.GroupBy(sp.bookingEntries,
		func(item *models.BookingEntry) int32 { return item.CarClassID.GetOrZero() })

	resultsByClass := sp.recomputeFinishPosByClassAndGrid()
	ret = make([]*queryv1.Standing, 0)
	for classID := range bookingsByClass {
		classBookings := lo.Filter(bookingsByClass[classID],
			func(item *models.BookingEntry, _ int) bool {
				return bookingFilter(item)
			})
		classResults := resultsByClass[classID]
		comp := NewComputeStandings()
		computedStandings := comp.Compute(&ComputeStandingsInput{
			EventIDs: sp.eventIDs,
			Bookings: classBookings,
			Participations: ParticipationsFromResultEntries(
				classResults,
				refIDResultFunc,
			),
			// NumTotalEvents: int(epi.NumEvents),
			// NumSkip:        int(epi.NumSkips),
			SkipMode:    SkipModeNever,
			ReferenceID: refIDBookingFunc,
		})
		tmp := sp.convertToStandings(
			standingsType,
			classID,
			computedStandings)
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

//nolint:whitespace // editor/linter issue
func (sp *standingProc) computeSecondaryFromTeamContribution() (
	ret []*queryv1.Standing,
) {
	bookingsByEventID := make(map[int32][]*models.BookingEntry)
	for _, booking := range sp.bookingEntries {
		if booking == nil {
			continue
		}
		bookingsByEventID[booking.EventID] = append(
			bookingsByEventID[booking.EventID],
			booking)
	}
	// contains standing data per team.
	// gets updated per event
	workStandings := make(map[int32]*queryv1.Standing)
	for _, eventID := range sp.eventIDs {
		eventBookings := bookingsByEventID[eventID]
		currentTeamContribution := sp.aggregateTeamBookings(eventBookings)
		for _, contribution := range currentTeamContribution {
			if contribution == nil {
				continue
			}
			current, ok := workStandings[int32(contribution.ReferenceId)]
			if !ok {
				workStandings[int32(contribution.ReferenceId)] = contribution
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
			current.EventId = uint32(eventID)
		}
	}
	// sort the return results by position
	ret = lo.Values(workStandings)
	slices.SortFunc(ret, func(a, b *queryv1.Standing) int {
		return cmp.Compare(a.Data.Position, b.Data.Position)
	})
	return ret
}

//nolint:whitespace,funlen // editor/linter issue
func (sp *standingProc) aggregateTeamBookings(
	eventBookings []*models.BookingEntry) (
	ret []*queryv1.Standing,
) {
	ret = make([]*queryv1.Standing, 0)
	bookingsByClass := lo.GroupBy(eventBookings,
		func(item *models.BookingEntry) int32 { return item.CarClassID.GetOrZero() })
	for classID := range bookingsByClass {
		classBookings := lo.Filter(bookingsByClass[classID],
			func(item *models.BookingEntry, _ int) bool {
				return item.TargetType == "team" && item.SourceType == "team_contribution"
			})
		teamMap := lo.Reduce(classBookings,
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

		x := lo.Map(raw, func(be *models.BookingEntry, _ int) *queryv1.Standing {
			return &queryv1.Standing{
				StandingsType: queryv1.StandingsType_STANDINGS_TYPE_TEAM,
				ReferenceId:   uint32(be.TeamID.GetOrZero()),
				EventId:       uint32(be.EventID),
				CarClassId:    uint32(classID),
				Data: &commonv1.StandingData{
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
	standingType queryv1.StandingsType,
	carClassID int32,
	computedStandings []*ComputedStanding) (
	ret []*queryv1.Standing,
) {
	ret = make([]*queryv1.Standing, 0, len(computedStandings))
	for _, computedStanding := range computedStandings {
		ret = append(ret, &queryv1.Standing{
			StandingsType: standingType,
			ReferenceId:   uint32(computedStanding.ReferenceID),
			EventId:       uint32(sp.epi.Event.ID),
			CarClassId:    uint32(carClassID),
			Data:          computedStanding.StandingData,
			// DroppedEventIds: toUint32Slice(computedStanding.SkipEventIDs),
		})
	}
	return ret
}
