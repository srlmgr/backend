package bookings

import (
	"context"
	"errors"
	"fmt"
	"slices"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
)

var errInvalidBookingSelector = fmt.Errorf("invalid booking selector")

// GetBookingEntries returns booking entries for the selected scope.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetBookingEntries(
	ctx context.Context,
	req *connect.Request[queryv1.GetBookingEntriesRequest],
) (*connect.Response[queryv1.GetBookingEntriesResponse], error) {
	l := s.logger.WithCtx(ctx)

	bookingItems, err := s.loadBookingEntriesByScope(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, errInvalidBookingSelector) {
			l.Error("invalid booking selector")
			trace.SpanFromContext(ctx).SetStatus(
				codes.Error, "invalid booking selector")
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		l.Error("failed to load booking entries", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load booking entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	response, err := s.buildResponse(ctx, bookingItems)
	if err != nil {
		l.Error("failed to build booking entries response", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to build booking entries response")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "booking entries loaded")
	return connect.NewResponse(response), nil
}

func (s *service) loadBookingEntriesByScope(
	ctx context.Context,
	msg *queryv1.GetBookingEntriesRequest,
) ([]*models.BookingEntry, error) {
	//nolint:exhaustive // by design
	switch msg.WhichScope() {
	case queryv1.GetBookingEntriesRequest_EventId_case:
		return s.repo.BookingEntries().LoadByEventID(ctx, int32(msg.GetEventId()))
	case queryv1.GetBookingEntriesRequest_RaceId_case:
		return s.repo.BookingEntries().LoadByRaceID(ctx, int32(msg.GetRaceId()))
	case queryv1.GetBookingEntriesRequest_GridId_case:
		return s.repo.BookingEntries().LoadByRaceGridID(ctx, int32(msg.GetGridId()))
	default:
		return nil, errInvalidBookingSelector
	}
}

//nolint:whitespace // editor/linter issue
func (s *service) buildResponse(
	ctx context.Context,
	bookingItems []*models.BookingEntry,
) (*queryv1.GetBookingEntriesResponse, error) {
	items := make([]*commonv1.BookingEntry, 0, len(bookingItems))
	driverIDSet := make(map[int32]struct{})
	teamIDSet := make(map[int32]struct{})

	for _, item := range bookingItems {
		if item == nil {
			continue
		}

		converted := bookingEntryToProto(item)
		items = append(items, converted)

		if driverID := item.DriverID.GetOrZero(); driverID != 0 {
			driverIDSet[driverID] = struct{}{}
		}
		if teamID := item.TeamID.GetOrZero(); teamID != 0 {
			teamIDSet[teamID] = struct{}{}
		}
	}

	drivers, err := s.loadDrivers(ctx, keysSorted(driverIDSet))
	if err != nil {
		return nil, err
	}

	teams, err := s.loadTeams(ctx, keysSorted(teamIDSet))
	if err != nil {
		return nil, err
	}

	return &queryv1.GetBookingEntriesResponse{
		Items:   items,
		Drivers: drivers,
		Teams:   teams,
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) loadDrivers(
	ctx context.Context, driverIDs []int32,
) ([]*commonv1.Driver, error) {
	if len(driverIDs) == 0 {
		return nil, nil
	}

	dbDrivers, err := s.repo.Drivers().Drivers().LoadByIDs(ctx, driverIDs)
	if err != nil {
		return nil, err
	}

	drivers := make([]*commonv1.Driver, 0, len(dbDrivers))
	for _, item := range dbDrivers {
		if converted := s.conversion.DriverToDriver(item); converted != nil {
			drivers = append(drivers, converted)
		}
	}

	return drivers, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) loadTeams(
	ctx context.Context, teamIDs []int32,
) ([]*commonv1.Team, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}

	teams := make([]*commonv1.Team, 0, len(teamIDs))
	for _, teamID := range teamIDs {
		item, err := s.repo.Teams().Teams().LoadByID(ctx, teamID)
		if err != nil {
			return nil, err
		}

		if converted := s.conversion.TeamToTeam(item); converted != nil {
			teams = append(teams, converted)
		}
	}

	return teams, nil
}

func bookingEntryToProto(item *models.BookingEntry) *commonv1.BookingEntry {
	entry := &commonv1.BookingEntry{
		Id:          uint32(item.ID),
		EventId:     uint32(item.EventID),
		TargetType:  toBookingTargetType(item.TargetType),
		SourceType:  toBookingSourceType(item.SourceType),
		Points:      item.Points,
		Description: item.Description,
	}

	if driverID := item.DriverID.GetOrZero(); driverID != 0 {
		entry.TargetId = uint32(driverID)
	}
	if teamID := item.TeamID.GetOrZero(); teamID != 0 {
		entry.TargetId = uint32(teamID)
	}

	return entry
}

func toBookingTargetType(targetType mytypes.TargetType) commonv1.BookingTargetType {
	switch string(targetType) {
	case "driver":
		return commonv1.BookingTargetType_BOOKING_TARGET_TYPE_DRIVER
	case "team":
		return commonv1.BookingTargetType_BOOKING_TARGET_TYPE_TEAM
	default:
		return commonv1.BookingTargetType_BOOKING_TARGET_TYPE_UNSPECIFIED
	}
}

func toBookingSourceType(sourceType mytypes.SourceType) commonv1.BookingSourceType {
	switch string(sourceType) {
	case "manual_adjustment":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_MANUAL_ADJUSTMENT
	case "finish_pos":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_FINISH_POS
	case "qualification_pos":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_QUALIFICATION_POS
	case "least_incidents":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_LEAST_INCIDENTS
	case "fastest_lap":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_FASTEST_LAP
	case "top_n_finishers":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_TOP_N_FINISHERS
	case "incidents_exceeded":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_INCIDENTS_EXCEEDED
	case "penalty_points":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_PENALTY_POINTS
	case "team_contribution":
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_TEAM_CONTRIBUTION
	default:
		return commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_UNSPECIFIED
	}
}

func keysSorted(input map[int32]struct{}) []int32 {
	ids := make([]int32, 0, len(input))
	for id := range input {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
