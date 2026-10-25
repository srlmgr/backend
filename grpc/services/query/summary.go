package query

import (
	"context"
	"fmt"
	"slices"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/service/summary"
)

var errInvalidSummarySelector = fmt.Errorf("invalid summary selector")

// GetSummary returns a summary by ID provided in selector.
//
//nolint:whitespace,funlen // editor/linter issue
func (s *service) GetSummary(
	ctx context.Context,
	req *connect.Request[queryv1.GetSummaryRequest],
) (*connect.Response[queryv1.GetSummaryResponse], error) {
	l := s.logger.WithCtx(ctx)
	var summaries []*commonv1.Summary
	var err error
	switch req.Msg.Selector.Scope.(type) {
	case *commonv1.SummarySelector_EventId:
		l.Debug("GetSummary", log.Uint32("eventID", req.Msg.Selector.GetEventId()))
		summaries, err = s.getSummaryByEventID(ctx, req)

	case *commonv1.SummarySelector_RaceId:
		l.Debug("GetSummary", log.Uint32("raceID", req.Msg.Selector.GetRaceId()))
		summaries, err = s.getSummaryByRaceID(ctx, req)

	default:
		l.Error("invalid summary selector")
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "invalid summary selector")
		return nil, connect.NewError(
			connect.CodeInvalidArgument, errInvalidSummarySelector,
		)
	}
	if err != nil {
		l.Error("failed to get summary", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to get summary")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &queryv1.GetSummaryResponse{Summaries: summaries}
	switch req.Msg.GetSelector().GetType() {
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_DRIVER:
		response.Drivers, err = s.loadSummaryDrivers(ctx, summaries)
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_TEAM:
		response.Teams, err = s.loadSummaryTeams(ctx, summaries)
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_UNSPECIFIED:
		// Keep drivers and teams empty when no target type was selected.
	}
	if err != nil {
		l.Error("failed to enrich summary response", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to enrich summary response",
		)
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "summary loaded")
	return connect.NewResponse(response), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) getSummaryByEventID(
	ctx context.Context,
	req *connect.Request[queryv1.GetSummaryRequest],
) ([]*commonv1.Summary, error) {
	eventID := req.Msg.GetSelector().GetEventId()
	res, err := s.svc.SummaryService().GetEventSummary(ctx, int(eventID))
	if err != nil {
		s.logger.WithCtx(ctx).Error("failed to get event summary", log.ErrorField(err))
		return nil, err
	}

	return s.convertByRequest(req, res.Summary), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) getSummaryByRaceID(
	ctx context.Context,
	req *connect.Request[queryv1.GetSummaryRequest],
) ([]*commonv1.Summary, error) {
	raceID := req.Msg.GetSelector().GetRaceId()
	res, err := s.svc.SummaryService().GetRaceSummary(ctx, int(raceID))
	if err != nil {
		s.logger.WithCtx(ctx).Error("failed to get race summary", log.ErrorField(err))
		return nil, err
	}

	return s.convertByRequest(req, res.Summary), nil
}

// maps driver/team to primary/secondary and converts inp to message format
//
//nolint:whitespace // editor/linter issue
func (s *service) convertByRequest(
	req *connect.Request[queryv1.GetSummaryRequest],
	inp *summary.BaseSummary,
) []*commonv1.Summary {
	if inp == nil {
		return nil
	}
	if inp.Season == nil {
		return nil
	}
	convert := func(items []*summary.SummaryEntry) []*commonv1.Summary {
		return lo.Map(items, func(item *summary.SummaryEntry, _ int) *commonv1.Summary {
			return &commonv1.Summary{
				ReferenceId:   uint32(item.ReferenceID),
				CarClassId:    uint32(item.CarClassID),
				Points:        int32(item.Points.RawPoints),
				BonusPoints:   int32(item.Points.BonusPoints),
				PenaltyPoints: int32(item.Points.PenaltyPoints),
				TotalPoints:   int32(item.Points.TotalPoints),
			}
		})
	}
	switch req.Msg.GetSelector().GetType() {
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_DRIVER:
		if inp.Season.IsTeamBased {
			return convert(inp.Secondary)
		}
		return convert(inp.Primary)
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_TEAM:
		if inp.Season.IsTeamBased {
			return convert(inp.Primary)
		}
		return convert(inp.Secondary)
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_UNSPECIFIED:
		return nil
	default:
		return nil
	}
}

//nolint:whitespace // editor/linter issue
func (s *service) loadSummaryDrivers(
	ctx context.Context,
	summaries []*commonv1.Summary,
) ([]*commonv1.Driver, error) {
	driverIDs := summaryReferenceIDs(summaries)
	if len(driverIDs) == 0 {
		return nil, nil
	}

	driverItems, err := s.repo.Drivers().Drivers().LoadByIDs(ctx, driverIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*commonv1.Driver, 0, len(driverItems))
	for _, item := range driverItems {
		if converted := s.conversion.DriverToDriver(item); converted != nil {
			items = append(items, converted)
		}
	}

	return items, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) loadSummaryTeams(
	ctx context.Context,
	summaries []*commonv1.Summary,
) ([]*commonv1.Team, error) {
	teamIDs := summaryReferenceIDs(summaries)
	if len(teamIDs) == 0 {
		return nil, nil
	}

	items := make([]*commonv1.Team, 0, len(teamIDs))
	for _, teamID := range teamIDs {
		item, err := s.repo.Teams().Teams().LoadByID(ctx, teamID)
		if err != nil {
			return nil, err
		}

		if converted := s.conversion.TeamToTeam(item); converted != nil {
			items = append(items, converted)
		}
	}

	return items, nil
}

func summaryReferenceIDs(summaries []*commonv1.Summary) []int32 {
	idSet := make(map[int32]struct{}, len(summaries))
	for _, summary := range summaries {
		if summary == nil {
			continue
		}

		referenceID := int32(summary.GetReferenceId())
		if referenceID == 0 {
			continue
		}

		idSet[referenceID] = struct{}{}
	}

	ids := make([]int32, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	return ids
}
