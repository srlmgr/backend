//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type teamRequest interface {
	GetSeasonId() uint32
	GetName() string
	GetIsActive() bool
}

type teamSetter = models.TeamSetter

type teamSetterBuilder struct{}

func (b teamSetterBuilder) Build(msg teamRequest) *teamSetter {
	setter := &teamSetter{}

	if seasonID := msg.GetSeasonId(); seasonID != 0 {
		setter.SeasonID = omit.From(int32(seasonID))
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
func (s *service) CreateTeam(
	ctx context.Context,
	req *connect.Request[v1.CreateTeamRequest]) (
	*connect.Response[v1.CreateTeamResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateTeam")
	setter := (teamSetterBuilder{}).Build(req.Msg)

	var newTeam *models.Team
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTeam, err = s.repo.Teams().Teams().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create team", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create team")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team created")
	return connect.NewResponse(&v1.CreateTeamResponse{
		Team: s.conversion.TeamToTeam(newTeam),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateTeam(
	ctx context.Context,
	req *connect.Request[v1.UpdateTeamRequest]) (
	*connect.Response[v1.UpdateTeamResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateTeam")
	setter := (teamSetterBuilder{}).Build(req.Msg)

	var newTeam *models.Team
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTeam, err = s.repo.Teams().Teams().Update(
			ctx,
			int32(req.Msg.GetTeamId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update team", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update team")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team updated")
	return connect.NewResponse(&v1.UpdateTeamResponse{
		Team: s.conversion.TeamToTeam(newTeam),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteTeam(
	ctx context.Context,
	req *connect.Request[v1.DeleteTeamRequest]) (
	*connect.Response[v1.DeleteTeamResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteTeam")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Teams().Teams().DeleteByID(
			ctx,
			int32(req.Msg.GetTeamId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete team", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete team")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team deleted")
	return connect.NewResponse(&v1.DeleteTeamResponse{
		Deleted: true,
	}), nil
}
