package standings

import (
	"context"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
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
	req *queryv1.GetStandingsRequest,
) (*queryv1.GetStandingsResponse, error) {
	l := s.logger.WithCtx(ctx)
	eventID := int32(req.GetEventId())
	l.Debug("GetStandings for event",
		log.Int32("event_id", eventID),
		log.Int("skip_mode", int(req.GetSkipMode())),
	)
	standingsData, err := s.svc.StandingsService().GetEventStandings(
		ctx, int(eventID),
		conversion.SkipModeFromProto(req.GetSkipMode()),
	)
	if err != nil {
		return nil, s.conversion.ToConnectError(err)
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
		EventId:            req.GetEventId(),
		PrimaryStandings:   primary,
		SecondaryStandings: secondary,
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "standings loaded")
	return resp, nil
}
