// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"
	"sort"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

// ListSeasonTeams returns season team data with related team and car data.
//
//nolint:whitespace,funlen // editor/linter issue
func (s *service) ListSeasonTeams(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonTeamsRequest],
) (*connect.Response[queryv1.ListSeasonTeamsResponse], error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.Msg.GetSeasonId())
	l.Debug("ListSeasonTeams", log.Int32("season_id", seasonID))

	seasonTeams, err := s.repo.Teams().Teams().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season teams", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season teams")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	sort.Slice(seasonTeams, func(i, j int) bool {
		return seasonTeams[i].ID < seasonTeams[j].ID
	})

	carModelIDSet := make(map[int32]struct{}, len(seasonTeams))
	seasonTeamProto := make([]*commonv1.Team, 0, len(seasonTeams))
	for _, item := range seasonTeams {
		if carModelID := item.CarModelID.Ptr(); carModelID != nil {
			carModelIDSet[*carModelID] = struct{}{}
		}

		if converted := s.conversion.TeamToTeam(item); converted != nil {
			seasonTeamProto = append(seasonTeamProto, converted)
		}
	}

	carData, loadCarDataErr := s.loadCarDataForSeasonDrivers(
		ctx, mapKeysSorted(carModelIDSet))
	if loadCarDataErr != nil {
		l.Error("failed to load car data", log.ErrorField(loadCarDataErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car data")
		return nil, connect.NewError(
			s.conversion.MapErrorToRPCCode(loadCarDataErr),
			loadCarDataErr,
		)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season teams loaded")
	return connect.NewResponse(&queryv1.ListSeasonTeamsResponse{
		Items: []*queryv1.SeasonTeamContainer{{
			Teams:   seasonTeamProto,
			CarData: carData,
		}},
	}), nil
}
