//nolint:dupl // some operations are very similar across entities
package query

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

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
