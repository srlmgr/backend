package query

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

// ListPointSystems returns a list of all point systems.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListPointSystems(
	ctx context.Context,
	req *connect.Request[queryv1.ListPointSystemsRequest],
) (*connect.Response[queryv1.ListPointSystemsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListPointSystems")

	pointSystems, err := s.repo.PointSystems().PointSystems().LoadAll(ctx)
	if err != nil {
		l.Error("failed to load point systems", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load point systems")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point systems loaded")

	items := make([]*commonv1.PointSystem, 0, len(pointSystems))
	for _, item := range pointSystems {
		if converted := s.conversion.PointSystemToPointSystem(item); converted != nil {
			items = append(items, converted)
		}
	}

	return connect.NewResponse(&queryv1.ListPointSystemsResponse{
		Items: items,
	}), nil
}

// GetPointSystem returns a point system by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetPointSystem(
	ctx context.Context,
	req *connect.Request[queryv1.GetPointSystemRequest],
) (*connect.Response[queryv1.GetPointSystemResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetPointSystem", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.PointSystems().PointSystems().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load point system", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system loaded")

	return connect.NewResponse(&queryv1.GetPointSystemResponse{
		PointSystem: s.conversion.PointSystemToPointSystem(item),
	}), nil
}
