//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
)

type simulationRequest interface {
	GetName() string
	GetIsActive() bool
	GetSupportedFormats() []commonv1.ImportFormat
}

type simSetter = models.RacingSimSetter

type racingSimSetterBuilder struct{}

func (r racingSimSetterBuilder) Build(msg simulationRequest) (*simSetter, error) {
	setter := &simSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if msg.GetIsActive() {
		setter.IsActive = omit.From(true)
	}

	if formats := msg.GetSupportedFormats(); len(formats) > 0 {
		supportedFormats, err := conversion.ImportFormatsFromProto(formats)
		if err != nil {
			return nil, err
		}
		setter.SupportedImportFormats = omit.From(pq.StringArray(supportedFormats))
	}

	return setter, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateSimulation(
	ctx context.Context,
	req *connect.Request[v1.CreateSimulationRequest]) (
	*connect.Response[v1.CreateSimulationResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateSimulation")
	setter, err := (racingSimSetterBuilder{}).Build(req.Msg)
	if err != nil {
		l.Error("invalid simulation supported formats", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "invalid simulation supported formats")
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var newSim *models.RacingSim
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSim, err = s.repo.RacingSims().Create(
			ctx,
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to create simulation", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create simulation")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation created")
	return connect.NewResponse(&v1.CreateSimulationResponse{
		Simulation: s.conversion.RacingSimToSimulation(newSim),
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
	setter, err := (racingSimSetterBuilder{}).Build(req.Msg)
	if err != nil {
		l.Error("invalid simulation supported formats", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "invalid simulation supported formats")
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var newSim *models.RacingSim
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSim, err = s.repo.RacingSims().Update(
			ctx,
			int32(req.Msg.GetSimulationId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update simulation", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update simulation")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation updated")
	return connect.NewResponse(&v1.UpdateSimulationResponse{
		Simulation: s.conversion.RacingSimToSimulation(newSim),
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
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation deleted")
	return connect.NewResponse(&v1.DeleteSimulationResponse{
		Deleted: true,
	}), nil
}
