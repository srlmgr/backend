package command

import (
	"context"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace // editor/linter issue
func (s *service) CreateSimulation(
	ctx context.Context,
	req *connect.Request[v1.CreateSimulationRequest]) (
	*connect.Response[v1.CreateSimulationResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateSimulation")
	var newSim *models.RacingSim
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		newSim, err = s.repo.RacingSims().Create(
			ctx,
			&models.RacingSimSetter{
				Name:                   omit.From(req.Msg.GetName()),
				IsActive:               omit.From(true),
				SupportedImportFormats: omit.From(pq.StringArray(req.Msg.SupportedFormats)),
			})
		return err
	}); txErr != nil {
		l.Error("failed to create simulation", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create simulation")
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation created")
	return connect.NewResponse(&v1.CreateSimulationResponse{
		Simulation: &commonv1.Simulation{
			Id:               uint32(newSim.ID),
			Name:             newSim.Name,
			SupportedFormats: newSim.SupportedImportFormats,
		},
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateSimulation(
	ctx context.Context,
	req *connect.Request[v1.UpdateSimulationRequest]) (
	*connect.Response[v1.UpdateSimulationResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateSimulation")
	var newSim *models.RacingSim
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		newSim, err = s.repo.RacingSims().Update(
			ctx,
			int32(req.Msg.GetSimulationId()),
			&models.RacingSimSetter{
				Name:                   omit.From(req.Msg.GetName()),
				IsActive:               omit.From(true),
				SupportedImportFormats: omit.From(pq.StringArray(req.Msg.SupportedFormats)),
			})
		return err
	}); txErr != nil {
		l.Error("failed to update simulation", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update simulation")
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation updated")
	return connect.NewResponse(&v1.UpdateSimulationResponse{
		Simulation: &commonv1.Simulation{
			Id:   uint32(newSim.ID),
			Name: newSim.Name,
		},
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteSimulation(
	ctx context.Context,
	req *connect.Request[v1.DeleteSimulationRequest]) (
	*connect.Response[v1.DeleteSimulationResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteSimulation")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.RacingSims().DeleteByID(
			ctx,
			int32(req.Msg.GetSimulationId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete simulation", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete simulation")
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation deleted")
	return connect.NewResponse(&v1.DeleteSimulationResponse{
		Deleted: true,
	}), nil
}
