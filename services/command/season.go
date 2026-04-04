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

type seasonRequest interface {
	GetName() string
	GetPointSystemId() uint32
	GetHasTeams() bool
	GetIsTeamBased() bool
	GetIsMulticlass() bool
	GetSkipEvents() int32
	GetTeamPointsTopN() int32
	GetStatus() string
}

type seasonSetter = models.SeasonSetter

type seasonSetterBuilder struct{}

func (b seasonSetterBuilder) Build(msg seasonRequest) *seasonSetter {
	setter := &seasonSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if pointSystemID := msg.GetPointSystemId(); pointSystemID != 0 {
		setter.PointSystemID = omit.From(int32(pointSystemID))
	}

	setter.HasTeams = omit.From(msg.GetHasTeams())
	setter.IsTeamBased = omit.From(msg.GetIsTeamBased())
	setter.IsMulticlass = omit.From(msg.GetIsMulticlass())

	if skipEvents := msg.GetSkipEvents(); skipEvents != 0 {
		setter.SkipEvents = omit.From(skipEvents)
	}

	if teamPointsTopN := msg.GetTeamPointsTopN(); teamPointsTopN != 0 {
		setter.TeamPointsTopN = omitnull.From(teamPointsTopN)
	}

	if status := msg.GetStatus(); status != "" {
		setter.Status = omit.From(status)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateSeason(
	ctx context.Context,
	req *connect.Request[v1.CreateSeasonRequest]) (
	*connect.Response[v1.CreateSeasonResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateSeason")
	setter := (seasonSetterBuilder{}).Build(req.Msg)

	var newSeason *models.Season
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.SeriesID = omit.From(int32(req.Msg.GetSeriesId()))
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSeason, err = s.repo.Seasons().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season created")
	return connect.NewResponse(&v1.CreateSeasonResponse{
		Season: s.conversion.SeasonToSeason(newSeason),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateSeason(
	ctx context.Context,
	req *connect.Request[v1.UpdateSeasonRequest]) (
	*connect.Response[v1.UpdateSeasonResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateSeason")
	setter := (seasonSetterBuilder{}).Build(req.Msg)

	var newSeason *models.Season
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSeason, err = s.repo.Seasons().Update(
			ctx,
			int32(req.Msg.GetSeasonId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season updated")
	return connect.NewResponse(&v1.UpdateSeasonResponse{
		Season: s.conversion.SeasonToSeason(newSeason),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteSeason(
	ctx context.Context,
	req *connect.Request[v1.DeleteSeasonRequest]) (
	*connect.Response[v1.DeleteSeasonResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteSeason")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Seasons().DeleteByID(
			ctx,
			int32(req.Msg.GetSeasonId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season deleted")
	return connect.NewResponse(&v1.DeleteSeasonResponse{
		Deleted: true,
	}), nil
}
