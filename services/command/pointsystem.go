//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type pointSystemRequest interface {
	GetName() string
	GetDescription() string
}

type pointSystemSetter = models.PointSystemSetter

type pointSystemSetterBuilder struct{}

func (b pointSystemSetterBuilder) Build(msg pointSystemRequest) *pointSystemSetter {
	setter := &pointSystemSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if description := msg.GetDescription(); description != "" {
		setter.Description = omitnull.From(description)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreatePointSystem(
	ctx context.Context,
	req *connect.Request[v1.CreatePointSystemRequest]) (
	*connect.Response[v1.CreatePointSystemResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreatePointSystem")
	setter := (pointSystemSetterBuilder{}).Build(req.Msg)

	var newPointSystem *models.PointSystem
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newPointSystem, err = s.repo.PointSystems().PointSystems().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create point system", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system created")
	return connect.NewResponse(&v1.CreatePointSystemResponse{
		PointSystem: s.conversion.PointSystemToPointSystem(newPointSystem),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdatePointSystem(
	ctx context.Context,
	req *connect.Request[v1.UpdatePointSystemRequest]) (
	*connect.Response[v1.UpdatePointSystemResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdatePointSystem")
	setter := (pointSystemSetterBuilder{}).Build(req.Msg)

	var newPointSystem *models.PointSystem
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newPointSystem, err = s.repo.PointSystems().PointSystems().Update(
			ctx,
			int32(req.Msg.GetPointSystemId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update point system", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system updated")
	return connect.NewResponse(&v1.UpdatePointSystemResponse{
		PointSystem: s.conversion.PointSystemToPointSystem(newPointSystem),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeletePointSystem(
	ctx context.Context,
	req *connect.Request[v1.DeletePointSystemRequest]) (
	*connect.Response[v1.DeletePointSystemResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeletePointSystem")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.PointSystems().PointSystems().DeleteByID(
			ctx,
			int32(req.Msg.GetPointSystemId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete point system", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system deleted")
	return connect.NewResponse(&v1.DeletePointSystemResponse{
		Deleted: true,
	}), nil
}
