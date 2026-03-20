package query

import (
	"context"
	"errors"
	"testing"
	"time"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
)

//nolint:whitespace // editor/linter issue
func seedEvent(
	t *testing.T,
	repo rootrepo.Repository,
	seasonID, trackLayoutID int32,
	name string,
) *models.Event {
	t.Helper()
	event, err := repo.Events().Create(context.Background(), &models.EventSetter{
		SeasonID:      omit.From(seasonID),
		TrackLayoutID: omit.From(trackLayoutID),
		Name:          omit.From(name),
		EventDate:     omit.From(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		CreatedBy:     omit.From(testUserSeed),
		UpdatedBy:     omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed event %q: %v", name, err)
	}
	return event
}

func TestListEventsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListEvents(
		context.Background(),
		connect.NewRequest(&queryv1.ListEventsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListEventsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout1 := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	layout2 := seedTrackLayout(t, repo, track.ID, "Short Circuit")
	event1 := seedEvent(t, repo, season.ID, layout1.ID, "Round 1")
	event2 := seedEvent(t, repo, season.ID, layout2.ID, "Round 2")

	resp, err := svc.ListEvents(
		context.Background(),
		connect.NewRequest(&queryv1.ListEventsRequest{}),
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
	}

	if !ids[uint32(event1.ID)] {
		t.Errorf("event1 (id=%d) not found in response", event1.ID)
	}
	if !ids[uint32(event2.ID)] {
		t.Errorf("event2 (id=%d) not found in response", event2.ID)
	}
}

func TestListEventsBySeasonID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season1 := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	season2 := seedSeason(t, repo, series.ID, pointSystem.ID, "2025 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event1 := seedEvent(t, repo, season1.ID, layout.ID, "Round 1")
	seedEvent(t, repo, season2.ID, layout.ID, "Round 1 S2")

	resp, err := svc.ListEvents(
		context.Background(),
		connect.NewRequest(&queryv1.ListEventsRequest{
			SeasonId: uint32(season1.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetSeasonId() != uint32(season1.ID) {
		t.Errorf("expected season_id %d, got %d", season1.ID, items[0].GetSeasonId())
	}
	if items[0].GetId() != uint32(event1.ID) {
		t.Errorf("expected id %d, got %d", event1.ID, items[0].GetId())
	}
}

func TestGetEventSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1")

	resp, err := svc.GetEvent(
		context.Background(),
		connect.NewRequest(&queryv1.GetEventRequest{
			Id: uint32(event.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := resp.Msg.GetEvent()
	if got.GetId() != uint32(event.ID) {
		t.Errorf("expected id %d, got %d", event.ID, got.GetId())
	}
	if got.GetSeasonId() != uint32(season.ID) {
		t.Errorf("expected season_id %d, got %d", season.ID, got.GetSeasonId())
	}
	if got.GetName() != "Round 1" {
		t.Errorf("expected name %q, got %q", "Round 1", got.GetName())
	}
}

func TestGetEventNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetEvent(
		context.Background(),
		connect.NewRequest(&queryv1.GetEventRequest{
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
