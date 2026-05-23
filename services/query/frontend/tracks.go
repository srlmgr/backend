// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

// ListTrackLayouts returns track/layout pairs, optionally filtered by simulation.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListTrackLayouts(
	ctx context.Context,
	req *connect.Request[queryv1.FrontendServiceListTrackLayoutsRequest],
) (*connect.Response[queryv1.FrontendServiceListTrackLayoutsResponse], error) {
	l := s.logger.WithCtx(ctx)
	simulationID := int32(req.Msg.GetSimulationId())
	l.Debug("ListTrackLayouts", log.Int32("simulation_id", simulationID))

	queries := s.repo.Queries().QueryTrackLayouts()

	trackLayouts, err := func() ([]*queryv1.TrackLayoutContainer, error) {
		if simulationID != 0 {
			items, loadErr := queries.ForSimulationID(ctx, simulationID)
			if loadErr != nil {
				return nil, loadErr
			}
			return s.toTrackLayoutContainers(items), nil
		}

		items, loadErr := queries.GetAll(ctx)
		if loadErr != nil {
			return nil, loadErr
		}
		return s.toTrackLayoutContainers(items), nil
	}()
	if err != nil {
		l.Error("failed to load track layouts", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load track layouts")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track layouts loaded")
	return connect.NewResponse(&queryv1.FrontendServiceListTrackLayoutsResponse{
		Items: trackLayouts,
	}), nil
}

func (s *service) toTrackLayoutContainers(
	items []*rootrepo.TrackLayoutContainer,
) []*queryv1.TrackLayoutContainer {
	out := make([]*queryv1.TrackLayoutContainer, 0, len(items))
	for _, item := range items {
		out = append(out, &queryv1.TrackLayoutContainer{
			Track:       s.conversion.TrackToTrack(item.Track),
			TrackLayout: s.conversion.TrackLayoutToTrackLayout(item.TrackLayout),
		})
	}

	return out
}
