//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
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
	req *v1.CreateTeamRequest,
) (*v1.CreateTeamResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateTeam")
	setter := (teamSetterBuilder{}).Build(req)
	if carModelVariantID := int32(req.GetCarModelVariantId()); carModelVariantID > 0 {
		setter.CarModelVariantID = omitnull.From(carModelVariantID)
	}
	if carNumber := req.GetCarNumber(); carNumber != "" {
		setter.CarNumber = omitnull.From(carNumber)
	}
	if req.HasJoinedAt() {
		setter.JoinedAt = omit.From(req.GetJoinedAt().AsTime())
	}
	if req.HasLeftAt() {
		setter.LeftAt = omitnull.From(req.GetLeftAt().AsTime())
	}

	var newTeam *models.Team
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTeam, err = s.repo.Teams().Teams().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create team", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create team")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team created")
	return &v1.CreateTeamResponse{
		Team: s.conversion.TeamToTeam(newTeam),
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateTeam(
	ctx context.Context,
	req *v1.UpdateTeamRequest,
) (*v1.UpdateTeamResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateTeam")
	setter := (teamSetterBuilder{}).Build(req)
	if carModelVariantID := int32(req.GetCarModelVariantId()); carModelVariantID > 0 {
		setter.CarModelVariantID = omitnull.From(carModelVariantID)
	}
	if carNumber := req.GetCarNumber(); carNumber != "" {
		setter.CarNumber = omitnull.From(carNumber)
	}
	if req.HasJoinedAt() {
		setter.JoinedAt = omit.From(req.GetJoinedAt().AsTime())
	}
	if req.HasLeftAt() {
		setter.LeftAt = omitnull.From(req.GetLeftAt().AsTime())
	}

	var newTeam *models.Team
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTeam, err = s.repo.Teams().Teams().Update(
			ctx,
			int32(req.GetTeamId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update team", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update team")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team updated")
	return &v1.UpdateTeamResponse{
		Team: s.conversion.TeamToTeam(newTeam),
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteTeam(
	ctx context.Context,
	req *v1.DeleteTeamRequest,
) (*v1.DeleteTeamResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteTeam")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		if req.Cascade {
			_, err = s.repo.BookingEntries().DeleteByTeamID(
				ctx,
				int32(req.GetTeamId()),
			)
			if err != nil {
				return err
			}
			_, err = s.repo.ResultEntries().DeleteByTeamID(
				ctx,
				int32(req.GetTeamId()),
			)
			if err != nil {
				return err
			}
			err = s.repo.Teams().TeamDrivers().DeleteByTeamID(
				ctx,
				int32(req.GetTeamId()),
			)
			if err != nil {
				return err
			}
		}
		err = s.repo.Teams().Teams().DeleteByID(
			ctx,
			int32(req.GetTeamId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete team", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete team")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team deleted")
	return &v1.DeleteTeamResponse{
		Deleted: true,
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) SetTeamMembers(
	ctx context.Context,
	req *v1.SetTeamMembersRequest,
) (*v1.SetTeamMembersResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetTeamMembers")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Teams().TeamDrivers().DeleteByTeamID(
			ctx,
			int32(req.GetTeamId()),
		)
		if err != nil {
			return err
		}
		for _, member := range req.GetMembers() {
			tdSetter := &models.TeamDriverSetter{
				TeamID:   omit.From(int32(req.GetTeamId())),
				DriverID: omit.From(int32(member.GetDriverId())),
			}
			if member.HasJoinedAt() {
				tdSetter.JoinedAt = omit.From(member.GetJoinedAt().AsTime())
			} else {
				tdSetter.JoinedAt = omit.From(time.Now())
			}
			if member.HasLeftAt() {
				tdSetter.LeftAt = omitnull.From(member.GetLeftAt().AsTime())
			}
			_, err = s.repo.Teams().TeamDrivers().Create(ctx, tdSetter)
			if err != nil {
				return err
			}
		}
		return nil
	}); txErr != nil {
		l.Error("failed to set team members", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to set team members")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team members set")
	return &v1.SetTeamMembersResponse{}, nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) AddTeamMember(
	ctx context.Context,
	req *v1.AddTeamMemberRequest,
) (*v1.AddTeamMemberResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("AddTeamMember")
	var currentMembers []*models.TeamDriver
	var err error
	currentMembers, err = s.repo.Teams().TeamDrivers().LoadByTeamID(
		ctx,
		int32(req.GetTeamId()),
	)
	if err != nil {
		l.Error("failed to load current team members", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load current team members",
		)
		return nil, s.conversion.ToConnectError(err)
	}
	_ = currentMembers // TODO: maybe check if data for overlap existing member

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter := &models.TeamDriverSetter{
			TeamID:   omit.From(int32(req.GetTeamId())),
			DriverID: omit.From(int32(req.GetDriverId())),
		}
		if req.HasJoinedAt() {
			setter.JoinedAt = omit.From(req.GetJoinedAt().AsTime())
		} else {
			setter.JoinedAt = omit.From(time.Now())
		}
		if req.HasLeftAt() {
			setter.LeftAt = omitnull.From(req.GetLeftAt().AsTime())
		}
		_, err = s.repo.Teams().TeamDrivers().Create(
			ctx, setter,
		)
		if err != nil {
			return err
		}

		return nil
	}); txErr != nil {
		l.Error("failed to add team member", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to add team member")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "added team member")
	return &v1.AddTeamMemberResponse{}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) RemoveTeamMember(
	ctx context.Context,
	req *v1.RemoveTeamMemberRequest,
) (*v1.RemoveTeamMemberResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("RemoveTeamMember")

	var err error
	_, err = s.repo.Teams().TeamDrivers().LoadByID(
		ctx,
		int32(req.GetId()),
	)
	if err != nil {
		l.Error("failed to load current team member entry", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load current team member entry",
		)
		return nil, s.conversion.ToConnectError(err)
	}

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		_, err = s.repo.Teams().TeamDrivers().Update(
			ctx,
			int32(req.GetId()),
			&models.TeamDriverSetter{
				LeftAt:    omitnull.From(time.Now()),
				UpdatedAt: omit.From(time.Now()),
				UpdatedBy: omit.From(s.execUser(ctx)),
			},
		)
		if err != nil {
			return err
		}

		return nil
	}); txErr != nil {
		l.Error("failed to remove team member", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to remove team member")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "removed team member")
	return &v1.RemoveTeamMemberResponse{}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteTeamMember(
	ctx context.Context,
	req *v1.DeleteTeamMemberRequest,
) (*v1.DeleteTeamMemberResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteTeamMember")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		return s.repo.Teams().TeamDrivers().DeleteByID(
			ctx,
			int32(req.GetId()),
		)
	}); txErr != nil {
		l.Error("failed to delete team member", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete team member")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team member deleted")
	return &v1.DeleteTeamMemberResponse{}, nil
}
