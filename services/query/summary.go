package query

import (
	"context"
	"fmt"
	"slices"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/importsvc/points"
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
			connect.CodeInvalidArgument, errInvalidSummarySelector)
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
			codes.Error, "failed to enrich summary response")
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
	bookings, err := s.repo.BookingEntries().LoadByEventID(ctx, int32(eventID))
	if err != nil {
		s.logger.WithCtx(ctx).Error(
			"failed to load booking entries", log.ErrorField(err))
		return nil, err
	}
	return s.createSummaryBy(ctx, req.Msg.Selector, bookings), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) getSummaryByRaceID(
	ctx context.Context,
	req *connect.Request[queryv1.GetSummaryRequest],
) ([]*commonv1.Summary, error) {
	raceID := req.Msg.GetSelector().GetRaceId()
	bookings, err := s.repo.BookingEntries().LoadByRaceID(ctx, int32(raceID))
	if err != nil {
		s.logger.WithCtx(ctx).Error("failed to load booking entries", log.ErrorField(err))
		return nil, err
	}
	return s.createSummaryBy(ctx, req.Msg.Selector, bookings), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) createSummaryBy(
	ctx context.Context,
	sel *commonv1.SummarySelector,
	bookings []*models.BookingEntry,
) []*commonv1.Summary {
	attrs := []attribute.KeyValue{}
	var summaries []*commonv1.Summary
	var work map[int32][]*models.BookingEntry
	if len(bookings) == 0 {
		return summaries
	}
	event, err := s.repo.Events().LoadByID(ctx, bookings[0].EventID)
	if err != nil {
		return []*commonv1.Summary{}
	}

	season, err := s.repo.Seasons().LoadByID(ctx, event.SeasonID)
	if err != nil {
		return []*commonv1.Summary{}
	}

	// s.repo.Seasons().
	filteredBy := func(tt mytypes.TargetType) []*models.BookingEntry {
		return lo.Filter(bookings, func(item *models.BookingEntry, _ int) bool {
			return item.TargetType == tt
		})
	}
	switch sel.Type {
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_DRIVER:
		attrs = append(attrs, attribute.String("summary_target_type", "driver"))
		work = lo.GroupBy(
			filteredBy(mytypes.TargetType("driver")),
			func(item *models.BookingEntry) int32 {
				return item.DriverID.GetOrZero()
			})
		if !season.IsTeamBased {
			summaries = s.summaryByPrimary(work)
		} else {
			summaries = s.summaryBySecondary(work)
		}

	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_TEAM:
		attrs = append(attrs, attribute.String("summary_target_type", "team"))
		work = lo.GroupBy(
			filteredBy(mytypes.TargetType("team")),
			func(item *models.BookingEntry) int32 {
				return item.TeamID.GetOrZero()
			})
		if season.IsTeamBased {
			summaries = s.summaryByPrimary(work)
		} else {
			summaries = s.summaryBySecondary(work)
		}
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_UNSPECIFIED:
		return summaries
	}
	spanCtx, span := s.tracer.Start(ctx, "create summary", trace.WithAttributes(attrs...))
	defer span.End()
	_ = spanCtx

	return summaries
}

//nolint:whitespace // editor/linter issue
func (s *service) pointCat(
	cond func(points.PointPolicyType) bool,
) func(*models.BookingEntry) int32 {
	return func(item *models.BookingEntry) int32 {
		var p points.PointPolicyType
		if err := p.UnmarshalText([]byte(item.SourceType)); err != nil {
			return 0
		}
		if cond(p) {
			return item.Points
		}
		return 0
	}
}

//nolint:whitespace // editor/linter issue
func (s *service) summaryByPrimary(
	work map[int32][]*models.BookingEntry,
) []*commonv1.Summary {
	summaries := make([]*commonv1.Summary, 0)
	for k, v := range work {
		rawPoints := lo.SumBy(v, s.pointCat(func(p points.PointPolicyType) bool {
			return p == points.PointsPolicyFinishPos
		}))

		bonusPoints := lo.SumBy(v, s.pointCat(func(p points.PointPolicyType) bool {
			return slices.Contains([]points.PointPolicyType{
				points.PointsPolicyFastestLap,
				points.PointsPolicyQualificationPos,
				points.PointsPolicyTopNFinishers,
				points.PointsPolicyLeastIncidents,
			}, p)
		}))
		// note: penaltyPoints are negative in bookings!
		penaltyPoints := lo.SumBy(v, s.pointCat(func(p points.PointPolicyType) bool {
			return slices.Contains([]points.PointPolicyType{
				points.PointsPolicyPenalty,
				points.PointsPolicyIncidentsExceeded,
			}, p)
		}))
		localSums := &commonv1.Summary{
			ReferenceId:   uint32(k),
			Points:        rawPoints,
			BonusPoints:   bonusPoints,
			PenaltyPoints: -penaltyPoints,
			TotalPoints:   rawPoints + bonusPoints + penaltyPoints,
		}
		summaries = append(summaries, localSums)

	}
	slices.SortFunc(summaries, func(a, b *commonv1.Summary) int {
		return int(b.TotalPoints - a.TotalPoints)
	})
	return summaries
}

//nolint:whitespace // editor/linter issue
func (s *service) summaryBySecondary(
	work map[int32][]*models.BookingEntry,
) []*commonv1.Summary {
	summaries := make([]*commonv1.Summary, 0)
	for k, v := range work {
		rawPoints := lo.SumBy(v, func(item *models.BookingEntry) int32 {
			if item.Points > 0 {
				return item.Points
			}
			return 0
		})

		localSums := &commonv1.Summary{
			ReferenceId:   uint32(k),
			Points:        rawPoints,
			BonusPoints:   0,
			PenaltyPoints: 0,
			TotalPoints:   rawPoints,
		}
		summaries = append(summaries, localSums)

	}
	slices.SortFunc(summaries, func(a, b *commonv1.Summary) int {
		return int(b.TotalPoints - a.TotalPoints)
	})
	return summaries
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
