//nolint:dupl // some operations are very similar across entities
package query

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

// ListTeams returns a list of teams.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListTeams(
	ctx context.Context,
	req *connect.Request[queryv1.ListTeamsRequest],
) (*connect.Response[queryv1.ListTeamsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListTeams", log.Uint32("season_id", req.Msg.GetSeasonId()))

	teamsRepo := s.repo.Teams().Teams()

	var (
		teamItems []*models.Team
		err       error
	)

	if seasonID := int32(req.Msg.GetSeasonId()); seasonID != 0 {
		teamItems, err = teamsRepo.LoadBySeasonID(ctx, seasonID)
	} else {
		teamItems, err = teamsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load teams", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load teams")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.Team, 0, len(teamItems))
	for _, item := range teamItems {
		if converted := s.conversion.TeamToTeam(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "teams loaded")
	return connect.NewResponse(&queryv1.ListTeamsResponse{Items: items}), nil
}

// GetTeam returns a team by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetTeam(
	ctx context.Context,
	req *connect.Request[queryv1.GetTeamRequest],
) (*connect.Response[queryv1.GetTeamResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetTeam", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Teams().Teams().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load team", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load team")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team loaded")
	return connect.NewResponse(&queryv1.GetTeamResponse{
		Team: s.conversion.TeamToTeam(item),
	}), nil
}

// GetTeamMembers returns all members for a team.
//
//nolint:whitespace,funlen // editor/linter issue
func (s *service) GetTeamMembers(
	ctx context.Context,
	req *connect.Request[queryv1.GetTeamMembersRequest],
) (*connect.Response[queryv1.GetTeamMembersResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetTeamMembers", log.Uint32("id", req.Msg.GetId()))

	teamID := int32(req.Msg.GetId())
	if _, err := s.repo.Teams().Teams().LoadByID(ctx, teamID); err != nil {
		l.Error("failed to load team", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load team")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	teamDrivers, err := s.repo.Teams().TeamDrivers().LoadByTeamID(ctx, teamID)
	if err != nil {
		l.Error("failed to load team members", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load team members")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	driverByID := map[int32]*models.Driver{}
	if len(teamDrivers) > 0 {
		driverIDs := make([]int32, 0, len(teamDrivers))
		for _, teamDriver := range teamDrivers {
			driverIDs = append(driverIDs, teamDriver.DriverID)
		}

		drivers, err := s.repo.Drivers().Drivers().LoadByIDs(ctx, driverIDs)
		if err != nil {
			l.Error("failed to load drivers for team members", log.ErrorField(err))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load drivers")
			return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
		}

		for _, driver := range drivers {
			driverByID[driver.ID] = driver
		}
	}

	members := make([]*commonv1.TeamMember, 0, len(teamDrivers))
	for _, teamDriver := range teamDrivers {
		member := &commonv1.TeamMember{
			Id:       uint32(teamDriver.ID),
			TeamId:   uint32(teamDriver.TeamID),
			Driver:   s.conversion.DriverToDriver(driverByID[teamDriver.DriverID]),
			JoinedAt: timestamppb.New(teamDriver.JoinedAt),
		}
		if leftAt := teamDriver.LeftAt.Ptr(); leftAt != nil {
			member.LeftAt = timestamppb.New(*leftAt)
		}
		members = append(members, member)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team members loaded")
	return connect.NewResponse(&queryv1.GetTeamMembersResponse{Members: members}), nil
}
