//nolint:dupl // some operations are very similar across entities
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

// ListPointSystems returns a list of all point systems with reconstructed
// nested structure.
//
// Similar to GetPointSystem, loads all PointSystem records with related
// PointRule rows, and uses conversion service to rebuild nested
// PointRaceSettings for each item.
// See conversion/pointsystem.go for metadata decoding details.
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
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load point systems",
		)
		return nil, connect.NewError(
			s.conversion.MapErrorToRPCCode(err), err,
		)
	}
	trace.SpanFromContext(ctx).SetStatus(
		codes.Ok, "point systems loaded",
	)

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

// GetPointSystem returns a point system by ID with reconstructed nested
// race settings.
//
// Retrieves a PointSystem from database with related PointRule rows, then
// uses conversion service to rebuild original nested message structure:
//   - point_system row → base PointSystem fields
//   - guest_points + race_distance_pct → PointEligibility
//   - point_rules rows (grouped by race_no) → PointRaceSettings with Policies
//
// The metadata_json field in each point_rules row is decoded to recover
// race-setting name and policy configuration, enabling full round-trip of
// nested proto structure.
// See conversion/pointsystem.go for metadata encoding/decoding details.
//
// Returns connect.CodeNotFound if point system ID does not exist.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetPointSystem(
	ctx context.Context,
	req *connect.Request[queryv1.GetPointSystemRequest],
) (*connect.Response[queryv1.GetPointSystemResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetPointSystem", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.PointSystems().PointSystems().LoadByID(
		ctx, int32(req.Msg.GetId()),
	)
	if err != nil {
		l.Error("failed to load point system", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load point system",
		)
		return nil, connect.NewError(
			s.conversion.MapErrorToRPCCode(err), err,
		)
	}
	trace.SpanFromContext(ctx).SetStatus(
		codes.Ok, "point system loaded",
	)

	return connect.NewResponse(&queryv1.GetPointSystemResponse{
		PointSystem: s.conversion.PointSystemToPointSystem(item),
	}), nil
}
