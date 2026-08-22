package query

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace // editor/linter issue
func (s *service) GetDriverStandings(
	ctx context.Context,
	req *connect.Request[queryv1.GetDriverStandingsRequest],
) (*connect.Response[queryv1.GetDriverStandingsResponse], error) {
	l := s.logger.WithCtx(ctx)
	eventID := int32(req.Msg.GetEventId())
	l.Debug("GetDriverStandings", log.Int32("event_id", eventID))
	items := make([]*commonv1.DriverStanding, 0)
	standingsData, err := s.svc.StandingsService().GetEventStandings(
		ctx, int(eventID),
		conversion.SkipModeFromProto(req.Msg.GetSkipMode()),
	)
	if err != nil {
		return nil, err
	}
	driverIDSet := make(map[int32]struct{})
	for i := range standingsData.Primary {
		computedStanding := standingsData.Primary[i]
		driverIDSet[int32(computedStanding.ReferenceID)] = struct{}{}
		x := s.conversion.ServiceStandingsToProto(computedStanding)

		items = append(items, &commonv1.DriverStanding{
			DriverId: uint32(computedStanding.ReferenceID),
			EventId:  uint32(eventID),
			Data:     x.Data,
		})
	}

	drivers, err := s.loadDrivers(ctx, lo.Keys(driverIDSet))
	if err != nil {
		l.Error("failed to load drivers", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load drivers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver standings computed")
	return connect.NewResponse(&queryv1.GetDriverStandingsResponse{
		Standings: items,
		Drivers:   drivers,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) loadDrivers(
	ctx context.Context,
	driverIDs []int32,
) ([]*commonv1.Driver, error) {
	driverItems, err := s.repo.Drivers().Drivers().LoadByIDs(ctx, driverIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*commonv1.Driver, 0, len(driverItems))
	for _, item := range driverItems {
		if converted := s.conversion.DriverToDriver(item); converted != nil {
			items = append(items, converted)
		}
	}

	return items, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) GetTeamStandings(
	context.Context,
	*connect.Request[queryv1.GetTeamStandingsRequest],
) (*connect.Response[queryv1.GetTeamStandingsResponse], error) {
	return connect.NewResponse(&queryv1.GetTeamStandingsResponse{
		Standings: []*commonv1.TeamStanding{},
	}), nil
}
