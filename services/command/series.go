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

type seriesRequest interface {
	GetSimulationId() uint32
	GetName() string
	GetDescription() string
	GetIsActive() bool
}

type seriesSetter = models.SeriesSetter

type seriesSetterBuilder struct{}

func (b seriesSetterBuilder) Build(msg seriesRequest) *seriesSetter {
	setter := &seriesSetter{}

	if simulationID := msg.GetSimulationId(); simulationID != 0 {
		setter.SimulationID = omit.From(int32(simulationID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if description := msg.GetDescription(); description != "" {
		setter.Description = omitnull.From(description)
	}

	if msg.GetIsActive() {
		setter.IsActive = omit.From(true)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateSeries(
	ctx context.Context,
	req *connect.Request[v1.CreateSeriesRequest]) (
	*connect.Response[v1.CreateSeriesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateSeries")
	setter := (seriesSetterBuilder{}).Build(req.Msg)

	var newSeries *models.Series
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSeries, err = s.repo.Series().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create series", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create series")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "series created")
	return connect.NewResponse(&v1.CreateSeriesResponse{
		Series: s.conversion.SeriesToSeries(newSeries),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateSeries(
	ctx context.Context,
	req *connect.Request[v1.UpdateSeriesRequest]) (
	*connect.Response[v1.UpdateSeriesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateSeries")
	setter := (seriesSetterBuilder{}).Build(req.Msg)

	var newSeries *models.Series
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSeries, err = s.repo.Series().Update(
			ctx,
			int32(req.Msg.GetSeriesId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update series", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update series")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "series updated")
	return connect.NewResponse(&v1.UpdateSeriesResponse{
		Series: s.conversion.SeriesToSeries(newSeries),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteSeries(
	ctx context.Context,
	req *connect.Request[v1.DeleteSeriesRequest]) (
	*connect.Response[v1.DeleteSeriesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteSeries")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Series().DeleteByID(
			ctx,
			int32(req.Msg.GetSeriesId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete series", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete series")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "series deleted")
	return connect.NewResponse(&v1.DeleteSeriesResponse{
		Deleted: true,
	}), nil
}
