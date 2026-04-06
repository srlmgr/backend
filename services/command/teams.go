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

//nolint:whitespace // editor/linter issue
func (s *service) SetTeamMembers(
	ctx context.Context,
	req *connect.Request[v1.SetTeamMembersRequest]) (
	*connect.Response[v1.SetTeamMembersResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetTeamMembers")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Teams().TeamDrivers().DeleteByTeamID(
			ctx,
			int32(req.Msg.GetTeamId()),
		)
		if err != nil {
			return err
		}
		for _, driverID := range req.Msg.GetDriverIds() {
			_, err = s.repo.Teams().TeamDrivers().Create(ctx, &models.TeamDriverSetter{
				TeamID:   omit.From(int32(req.Msg.GetTeamId())),
				DriverID: omit.From(int32(driverID)),
			})
			if err != nil {
				return err
			}
		}
		return nil
	}); txErr != nil {
		l.Error("failed to set team members", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to set team members")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team members set")
	return connect.NewResponse(&v1.SetTeamMembersResponse{}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) AddTeamMember(
	ctx context.Context,
	req *connect.Request[v1.AddTeamMemberRequest]) (
	*connect.Response[v1.AddTeamMemberResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("AddTeamMember")
	var currentMembers []*models.TeamDriver
	var err error
	currentMembers, err = s.repo.Teams().TeamDrivers().LoadByTeamID(
		ctx,
		int32(req.Msg.GetTeamId()),
	)
	if err != nil {
		l.Error("failed to load current team members", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load current team members")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	for _, member := range currentMembers {
		if member.DriverID == int32(req.Msg.GetDriverId()) {
			l.Debug("driver is already a team member, skipping")
			trace.SpanFromContext(ctx).SetStatus(
				codes.Ok, "driver is already a team member, skipping")
			return connect.NewResponse(&v1.AddTeamMemberResponse{}), nil
		}
	}
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		_, err = s.repo.Teams().TeamDrivers().Create(
			ctx,
			&models.TeamDriverSetter{
				TeamID:   omit.From(int32(req.Msg.GetTeamId())),
				DriverID: omit.From(int32(req.Msg.GetDriverId())),
			},
		)
		if err != nil {
			return err
		}

		return nil
	}); txErr != nil {
		l.Error("failed to add team member", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to add team member")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "added team member")
	return connect.NewResponse(&v1.AddTeamMemberResponse{}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) RemoveTeamMember(
	ctx context.Context,
	req *connect.Request[v1.RemoveTeamMemberRequest]) (
	*connect.Response[v1.RemoveTeamMemberResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("RemoveTeamMember")
	var currentMembers []*models.TeamDriver
	var err error
	currentMembers, err = s.repo.Teams().TeamDrivers().LoadByTeamID(
		ctx,
		int32(req.Msg.GetTeamId()),
	)
	if err != nil {
		l.Error("failed to load current team members", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load current team members")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	for _, member := range currentMembers {
		if member.DriverID == int32(req.Msg.GetDriverId()) {
			l.Debug("driver is not a team member, skipping")
			trace.SpanFromContext(ctx).SetStatus(
				codes.Ok, "driver is not a team member, skipping")
			return connect.NewResponse(&v1.RemoveTeamMemberResponse{}), nil
		}
	}
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		_, err = s.repo.Teams().TeamDrivers().Update(
			ctx,
			int32(req.Msg.GetTeamId()),
			&models.TeamDriverSetter{
				DriverID: omit.From(int32(req.Msg.GetDriverId())),
				LeftAt:   omitnull.From(time.Now()),
			},
		)
		if err != nil {
			return err
		}

		return nil
	}); txErr != nil {
		l.Error("failed to remove team member", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to remove team member")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "removed team member")
	return connect.NewResponse(&v1.RemoveTeamMemberResponse{}), nil
}
