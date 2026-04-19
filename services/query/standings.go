package query

import (
	"context"
	"errors"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/importsvc/processor"
	"github.com/srlmgr/backend/services/query/standings"
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) GetDriverStandings(
	ctx context.Context,
	req *connect.Request[queryv1.GetDriverStandingsRequest],
) (*connect.Response[queryv1.GetDriverStandingsResponse], error) {
	l := s.logger.WithCtx(ctx)
	eventID := int32(req.Msg.GetEventId())
	l.Debug("GetDriverStandings", log.Int32("event_id", eventID))
	items := make([]*commonv1.DriverStanding, 0)
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

	driverBookings := lo.Filter(bookingEntries,
		func(item *models.BookingEntry, _ int) bool {
			return item.TargetType == "driver"
		})

	comp := standings.NewComputeStandings()
	computedStandings := comp.Compute(&standings.ComputeStandingsInput{
		EventIDs: useEventIDs,
		Bookings: driverBookings,
		Participations: standings.ParticipationsFromResultEntries(
			resultEntries,
			func(entry *models.ResultEntry) int32 {
				return entry.DriverID.GetOrZero()
			},
		),
		// NumTotalEvents: int(epi.NumEvents),
		// NumSkip:        int(epi.NumSkips),
		SkipMode: standings.SkipModeNever,
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})
	for i := range computedStandings {
		computedStanding := computedStandings[i]
		items = append(items, &commonv1.DriverStanding{
			DriverId:        uint32(computedStanding.ReferenceID),
			EventId:         uint32(eventID),
			Data:            computedStanding.StandingData,
			DroppedEventIds: toUint32Slice(computedStanding.SkipEventIDs),
		})
	}
	// TODO:remove
	_ = event
	_ = epi
	_ = bookingEntries

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver standings computed")
	return connect.NewResponse(&queryv1.GetDriverStandingsResponse{Standings: items}), nil
}

func toUint32Slice(items []int32) []uint32 {
	result := make([]uint32, 0, len(items))
	for _, item := range items {
		result = append(result, uint32(item))
	}

	return result
}

//nolint:whitespace // editor/linter issue
func (s *service) GetTeamStandings(
	context.Context,
	*connect.Request[queryv1.GetTeamStandingsRequest],
) (*connect.Response[queryv1.GetTeamStandingsResponse], error) {
	return nil, connect.NewError(
		connect.CodeUnimplemented,
		errors.New("backend.query.v1.QueryService.GetTeamStandings is not implemented"),
	)
}
