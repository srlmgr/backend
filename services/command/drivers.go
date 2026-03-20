//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"strconv"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type driverRequest interface {
	GetExternalId() uint32
	GetName() string
	GetIsActive() bool
}

type driverSetter = models.DriverSetter

type driverSetterBuilder struct{}

func (b driverSetterBuilder) Build(msg driverRequest) *driverSetter {
	setter := &driverSetter{}

	if externalID := msg.GetExternalId(); externalID != 0 {
		setter.ExternalID = omit.From(strconv.FormatUint(uint64(externalID), 10))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if msg.GetIsActive() {
		setter.IsActive = omit.From(true)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateDriver(
	ctx context.Context,
	req *connect.Request[v1.CreateDriverRequest]) (
	*connect.Response[v1.CreateDriverResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateDriver")
	setter := (driverSetterBuilder{}).Build(req.Msg)

	var newDriver *models.Driver
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newDriver, err = s.repo.Drivers().Drivers().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver created")
	return connect.NewResponse(&v1.CreateDriverResponse{
		Driver: s.conversion.DriverToDriver(newDriver),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateDriver(
	ctx context.Context,
	req *connect.Request[v1.UpdateDriverRequest]) (
	*connect.Response[v1.UpdateDriverResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateDriver")
	setter := (driverSetterBuilder{}).Build(req.Msg)

	var newDriver *models.Driver
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newDriver, err = s.repo.Drivers().Drivers().Update(
			ctx,
			int32(req.Msg.GetDriverId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver updated")
	return connect.NewResponse(&v1.UpdateDriverResponse{
		Driver: s.conversion.DriverToDriver(newDriver),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteDriver(
	ctx context.Context,
	req *connect.Request[v1.DeleteDriverRequest]) (
	*connect.Response[v1.DeleteDriverResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteDriver")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Drivers().Drivers().DeleteByID(
			ctx,
			int32(req.Msg.GetDriverId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver deleted")
	return connect.NewResponse(&v1.DeleteDriverResponse{
		Deleted: true,
	}), nil
}
