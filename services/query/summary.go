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
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/importsvc/points"
)

var errInvalidSummarySelector = fmt.Errorf("invalid summary selector")

// GetSummary returns a summary by ID provided in selector.
//
//nolint:whitespace // editor/linter issue
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
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidSummarySelector)
	}
	if err != nil {
		l.Error("failed to get summary", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to get summary")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "summary loaded")
	return connect.NewResponse(&queryv1.GetSummaryResponse{
		Summaries: summaries,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) getSummaryByEventID(
	ctx context.Context,
	req *connect.Request[queryv1.GetSummaryRequest],
) ([]*commonv1.Summary, error) {
	eventID := req.Msg.GetSelector().GetEventId()
	bookings, err := s.repo.BookingEntries().LoadByEventID(ctx, int32(eventID))
	if err != nil {
		s.logger.WithCtx(ctx).Error("failed to load booking entries", log.ErrorField(err))
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
	summaries := make([]*commonv1.Summary, 0)
	var work map[int32][]*models.BookingEntry
	switch sel.Type {
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_DRIVER:
		attrs = append(attrs, attribute.String("summary_target_type", "driver"))
		work = lo.GroupBy(bookings, func(item *models.BookingEntry) int32 {
			return item.DriverID.GetOrZero()
		})
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_TEAM:
		attrs = append(attrs, attribute.String("summary_target_type", "team"))
		work = lo.GroupBy(bookings, func(item *models.BookingEntry) int32 {
			return item.TeamID.GetOrZero()
		})
	case commonv1.SummaryTargetType_SUMMARY_TARGET_TYPE_UNSPECIFIED:
		return summaries
	}
	spanCtx, span := s.tracer.Start(ctx, "create summary", trace.WithAttributes(attrs...))
	defer span.End()
	_ = spanCtx

	pointCat := func(
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
	for k, v := range work {
		rawPoints := lo.SumBy(v, pointCat(func(p points.PointPolicyType) bool {
			return p == points.PointsPolicyFinishPos
		}))

		bonusPoints := lo.SumBy(v, pointCat(func(p points.PointPolicyType) bool {
			return slices.Contains([]points.PointPolicyType{
				points.PointsPolicyFastestLap,
				points.PointsPolicyQualificationPos,
				points.PointsPolicyTopNFinishers,
				points.PointsPolicyLeastIncidents,
				points.PointsPolicyIncidentsExceeded,
			}, p)
		}))

		localSums := &commonv1.Summary{
			ReferenceId: uint32(k),
			Points:      rawPoints,
			BonusPoints: bonusPoints,
			TotalPoints: rawPoints + bonusPoints,
		}
		summaries = append(summaries, localSums)

	}
	slices.SortFunc(summaries, func(a, b *commonv1.Summary) int {
		return int(b.TotalPoints - a.TotalPoints)
	})
	return summaries
}
