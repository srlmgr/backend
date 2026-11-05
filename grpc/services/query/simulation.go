package query

import (
	"context"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

// ListSimulations returns a list of all simulations.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListSimulations(
	ctx context.Context,
	req *queryv1.ListSimulationsRequest,
) (*queryv1.ListSimulationsResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListSimulations")

	sims, err := s.repo.RacingSims().LoadAll(ctx)
	if err != nil {
		l.Error("failed to load simulation", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load simulations")
		return nil, s.conversion.ToConnectError(err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulations loaded")

	return &queryv1.ListSimulationsResponse{
		Items: s.conversion.RacingSimsToSimulations(sims),
	}, nil
}

// GetSimulation returns a simulation by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetSimulation(
	ctx context.Context,
	req *queryv1.GetSimulationRequest,
) (*queryv1.GetSimulationResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetSimulation", log.Uint32("id", req.Id))

	sim, err := s.repo.RacingSims().LoadByID(ctx, int32(req.Id))
	if err != nil {
		l.Error("failed to load simulation", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load simulation")
		return nil, s.conversion.ToConnectError(err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation loaded")
	return &queryv1.GetSimulationResponse{
		Simulation: s.conversion.RacingSimToSimulation(sim),
	}, nil
}
