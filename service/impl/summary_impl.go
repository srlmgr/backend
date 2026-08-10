package impl

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/grpc/services/importsvc/points"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
	"github.com/srlmgr/backend/service/summary"
)

type (
	summaryImpl struct {
		r      repository.Repository
		logger *log.Logger
		tracer trace.Tracer
	}
)

var _ service.SummaryService = (*summaryImpl)(nil)

//nolint:whitespace //editor/linter issue
func newSummaryImpl(
	r repository.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
) *summaryImpl {
	return &summaryImpl{
		r:      r,
		logger: logger,
		tracer: tracer,
	}
}

//nolint:whitespace,funlen //editor/linter issue
func (s *summaryImpl) GetSeasonSummary(
	ctx context.Context,
	seasonID int,
) (*service.SummaryContainer, error) {
	season, err := s.r.Seasons().LoadByID(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}

	bookings, err := s.r.BookingEntries().LoadBySeasonID(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}
	ret := &service.SummaryContainer{
		Season: season,
	}
	// collect events
	events, err := s.r.Events().LoadBySeasonID(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}
	primSumLookup := make(map[int32]*summary.SummaryEntry)
	secondSumLookup := make(map[int32]*summary.SummaryEntry)
	for _, event := range events {
		eventBookings := lo.Filter(bookings,
			func(item *models.BookingEntry, _ int) bool {
				return item.EventID == event.ID
			})
		prim, sec := s.createSummaryBy(ctx, eventBookings)
		eventSummary := &summary.EventSummary{
			EventID:   int(event.ID),
			Primary:   prim,
			Secondary: sec,
		}
		ret.Events = append(ret.Events, eventSummary)

		s.updateSums(primSumLookup, prim)
		s.updateSums(secondSumLookup, sec)
	}
	sorter := func(a, b *summary.SummaryEntry) int {
		return b.Points.TotalPoints - a.Points.TotalPoints
	}
	vals := slices.Values(lo.Values(primSumLookup))
	ret.PrimarySummaries = slices.SortedFunc(vals, sorter)

	vals = slices.Values(lo.Values(secondSumLookup))
	ret.SecondarySummaries = slices.SortedFunc(vals, sorter)

	return ret, nil
}

//nolint:whitespace //editor/linter issue
func (s *summaryImpl) updateSums(
	lookup map[int32]*summary.SummaryEntry,
	sums []*summary.SummaryEntry,
) {
	for _, s := range sums {
		work, ok := lookup[int32(s.ReferenceID)]
		if !ok {
			startPoints := summary.PointSummary{}
			work = &summary.SummaryEntry{
				ReferenceID: s.ReferenceID,
				CarClassID:  s.CarClassID,
				Points:      startPoints,
			}
			lookup[int32(s.ReferenceID)] = work
		}
		work.AddFrom(&s.Points)
	}
}

//nolint:whitespace //editor/linter issue
func (s *summaryImpl) GetEventSummary(
	ctx context.Context,
	eventID int,
) (*summary.EventSummary, error) {
	return nil, nil
}

//nolint:whitespace // editor/linter issue
func (s *summaryImpl) createSummaryBy(
	ctx context.Context,
	bookings []*models.BookingEntry,
) (primary, secondary []*summary.SummaryEntry) {
	if len(bookings) == 0 {
		return nil, nil
	}
	event, err := s.r.Events().LoadByID(ctx, bookings[0].EventID)
	if err != nil {
		return nil, nil
	}

	season, err := s.r.Seasons().LoadByID(ctx, event.SeasonID)
	if err != nil {
		return nil, nil
	}

	// s.repo.Seasons().
	filteredBy := func(tt mytypes.TargetType) []*models.BookingEntry {
		return lo.Filter(bookings, func(item *models.BookingEntry, _ int) bool {
			return item.TargetType == tt
		})
	}
	driverBookings := lo.GroupBy(
		filteredBy(mytypes.TargetType("driver")),
		func(item *models.BookingEntry) int32 {
			return item.DriverID.GetOrZero()
		},
	)
	teamBookings := lo.GroupBy(
		filteredBy(mytypes.TargetType("team")),
		func(item *models.BookingEntry) int32 {
			return item.TeamID.GetOrZero()
		},
	)
	if season.IsTeamBased {
		primary = s.summaryByPrimary(teamBookings)
		secondary = s.summaryBySecondary(driverBookings)
	} else {
		primary = s.summaryByPrimary(driverBookings)
		secondary = s.summaryBySecondary(teamBookings)
	}
	return primary, secondary
}

//nolint:whitespace // editor/linter issue
func (s *summaryImpl) pointCat(
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
func (s *summaryImpl) summaryByPrimary(
	work map[int32][]*models.BookingEntry,
) []*summary.SummaryEntry {
	summaries := make([]*summary.SummaryEntry, 0)
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
		localSums := &summary.SummaryEntry{
			ReferenceID: int(k),
			CarClassID:  int(v[0].CarClassID.GetOrZero()),
			Points: summary.PointSummary{
				RawPoints:     int(rawPoints),
				BonusPoints:   int(bonusPoints),
				PenaltyPoints: -int(penaltyPoints),
				TotalPoints:   int(rawPoints + bonusPoints + penaltyPoints),
			},
		}
		summaries = append(summaries, localSums)

	}
	slices.SortFunc(summaries, func(a, b *summary.SummaryEntry) int {
		return b.Points.TotalPoints - a.Points.TotalPoints
	})
	return summaries
}

//nolint:whitespace // editor/linter issue
func (s *summaryImpl) summaryBySecondary(
	work map[int32][]*models.BookingEntry,
) []*summary.SummaryEntry {
	summaries := make([]*summary.SummaryEntry, 0)
	for k, v := range work {
		rawPoints := lo.SumBy(v, func(item *models.BookingEntry) int32 {
			if item.Points > 0 {
				return item.Points
			}
			return 0
		})

		localSums := &summary.SummaryEntry{
			ReferenceID: int(k),
			CarClassID:  int(v[0].CarClassID.GetOrZero()),
			Points: summary.PointSummary{
				RawPoints:     int(rawPoints),
				BonusPoints:   0,
				PenaltyPoints: 0,
				TotalPoints:   int(rawPoints),
			},
		}
		summaries = append(summaries, localSums)

	}
	slices.SortFunc(summaries, func(a, b *summary.SummaryEntry) int {
		return b.Points.TotalPoints - a.Points.TotalPoints
	})
	return summaries
}
