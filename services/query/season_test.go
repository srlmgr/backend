//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package query

import (
	"context"
	"errors"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
)

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
	season1 := seedSeason(t, repo, series.ID, "2024 Season")
	season2 := seedSeason(t, repo, series.ID, "2025 Season")

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
	season1 := seedSeason(t, repo, series1.ID, "GT3 2024")
	seedSeason(t, repo, series2.ID, "GTE 2024")

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
}

func TestGetSeasonSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	season := seedSeason(t, repo, series.ID, "2024 Season")

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
