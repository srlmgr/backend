package standings

import (
	"context"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/service"
)

//nolint:whitespace // editor/linter issue
func (s *standingsService) GetStandings(
	ctx context.Context,
	req *connect.Request[queryv1.GetStandingsRequest],
) (*connect.Response[queryv1.GetStandingsResponse], error) {
	l := s.logger.WithCtx(ctx)
	eventID := int32(req.Msg.GetEventId())
	l.Debug("GetStandings for event",
		log.Int32("event_id", eventID),
		log.Int("skip_mode", int(req.Msg.GetSkipMode())),
	)
	standingsData, err := s.svc.StandingsService().GetEventStandings(
		ctx, int(eventID),
		conversion.SkipModeFromProto(req.Msg.GetSkipMode()),
	)
	if err != nil {
		return nil, err
	}

	primary := lo.Map(standingsData.Primary,
		func(st *service.Standing, _ int) *queryv1.Standing {
			return s.conversion.ServiceStandingsToProto(st)
		})
	secondary := lo.Map(standingsData.Secondary,
		func(st *service.Standing, _ int) *queryv1.Standing {
			return s.conversion.ServiceStandingsToProto(st)
		})

	resp := &queryv1.GetStandingsResponse{
		EventId:            req.Msg.GetEventId(),
		PrimaryStandings:   primary,
		SecondaryStandings: secondary,
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "standings loaded")
	return connect.NewResponse(resp), nil
}
