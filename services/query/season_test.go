//nolint:lll,funlen,whitespace // readability in testcode
package query

import (
	"context"
	"errors"
	"testing"
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
)

func seedSeasonWithDates(
	t *testing.T,
	repo rootrepo.Repository,
	seriesID int32,
	pointSystemID int32,
	name string,
	startsAt time.Time,
	endsAt time.Time,
) *models.Season {
	t.Helper()

	season, err := repo.Seasons().Create(context.Background(), &models.SeasonSetter{
		SeriesID:      omit.From(seriesID),
		PointSystemID: omit.From(pointSystemID),
		Name:          omit.From(name),
		HasTeams:      omit.From(false),
		SkipEvents:    omit.From(int32(0)),
		Status:        omit.From("active"),
		StartsAt:      omitnull.From(startsAt),
		EndsAt:        omitnull.From(endsAt),
		CreatedBy:     omit.From(testUserSeed),
		UpdatedBy:     omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed season %q with dates: %v", name, err)
	}

	return season
}

func TestListSeasonsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListSeasons(
		context.Background(),
		connect.NewRequest(&queryv1.ListSeasonsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListSeasonsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season1Starts := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	season1Ends := time.Date(2024, 11, 30, 0, 0, 0, 0, time.UTC)
	season2Starts := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	season2Ends := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	season1 := seedSeasonWithDates(
		t,
		repo,
		series.ID,
		pointSystem.ID,
		"2024 Season",
		season1Starts,
		season1Ends,
	)
	season2 := seedSeasonWithDates(
		t,
		repo,
		series.ID,
		pointSystem.ID,
		"2025 Season",
		season2Starts,
		season2Ends,
	)

	resp, err := svc.ListSeasons(
		context.Background(),
		connect.NewRequest(&queryv1.ListSeasonsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
		if item.GetSeriesId() != uint32(series.ID) {
			t.Errorf("expected series_id %d, got %d", series.ID, item.GetSeriesId())
		}
	}
	byID := make(map[uint32]*commonv1.Season, len(items))
	for _, item := range items {
		byID[item.GetId()] = item
	}
	if got := byID[uint32(season1.ID)].GetStartsAt(); got == nil ||
		!got.AsTime().Equal(season1Starts) {
		t.Fatalf("unexpected starts_at for season1: %+v", got)
	}
	if got := byID[uint32(season2.ID)].GetEndsAt(); got == nil || !got.AsTime().Equal(season2Ends) {
		t.Fatalf("unexpected ends_at for season2: %+v", got)
	}

	if !ids[uint32(season1.ID)] {
		t.Errorf("season1 (id=%d) not found in response", season1.ID)
	}
	if !ids[uint32(season2.ID)] {
		t.Errorf("season2 (id=%d) not found in response", season2.ID)
	}
}

func TestListSeasonsBySeriesID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series1 := seedSeries(t, repo, sim.ID, "GT3 Series")
	series2 := seedSeries(t, repo, sim.ID, "GTE Series")
	pointSystem := seedPointSystem(t, repo, "Series Points")
	season1 := seedSeasonWithDates(
		t,
		repo,
		series1.ID,
		pointSystem.ID,
		"GT3 2024",
		time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC),
	)
	seedSeason(t, repo, series2.ID, pointSystem.ID, "GTE 2024")

	resp, err := svc.ListSeasons(
		context.Background(),
		connect.NewRequest(&queryv1.ListSeasonsRequest{
			SeriesId: uint32(series1.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetSeriesId() != uint32(series1.ID) {
		t.Errorf("expected series_id %d, got %d", series1.ID, items[0].GetSeriesId())
	}
	if items[0].GetId() != uint32(season1.ID) {
		t.Errorf("expected id %d, got %d", season1.ID, items[0].GetId())
	}
	if got := items[0].GetStartsAt(); got == nil ||
		!got.AsTime().Equal(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected starts_at in filtered list: %+v", got)
	}
}

func TestGetSeasonSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeasonWithDates(
		t,
		repo,
		series.ID,
		pointSystem.ID,
		"2024 Season",
		time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 11, 30, 0, 0, 0, 0, time.UTC),
	)

	resp, err := svc.GetSeason(
		context.Background(),
		connect.NewRequest(&queryv1.GetSeasonRequest{
			Id: uint32(season.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := resp.Msg.GetSeason()
	if got.GetId() != uint32(season.ID) {
		t.Errorf("expected id %d, got %d", season.ID, got.GetId())
	}
	if got.GetSeriesId() != uint32(series.ID) {
		t.Errorf("expected series_id %d, got %d", series.ID, got.GetSeriesId())
	}
	if got.GetName() != "2024 Season" {
		t.Errorf("expected name %q, got %q", "2024 Season", got.GetName())
	}
	if got.GetStartsAt() == nil ||
		!got.GetStartsAt().AsTime().Equal(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected starts_at: %+v", got.GetStartsAt())
	}
	if got.GetEndsAt() == nil ||
		!got.GetEndsAt().AsTime().Equal(time.Date(2024, 11, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected ends_at: %+v", got.GetEndsAt())
	}
}

func TestGetSeasonNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetSeason(
		context.Background(),
		connect.NewRequest(&queryv1.GetSeasonRequest{
			Id: 99999,
		}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}
