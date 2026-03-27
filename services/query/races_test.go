package query

import (
	"context"
	"errors"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:whitespace // multiline signature style
func seedRace(
	t *testing.T,
	repo rootrepo.Repository,
	eventID int32,
	name, sessionType string,
	sequenceNo int32,
) *models.Race {
	t.Helper()
	race, err := repo.Races().Races().Create(context.Background(), &models.RaceSetter{
		EventID:     omit.From(eventID),
		Name:        omit.From(name),
		SessionType: omit.From(sessionType),
		SequenceNo:  omit.From(sequenceNo),
		CreatedBy:   omit.From(testUserSeed),
		UpdatedBy:   omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed race %q: %v", name, err)
	}
	return race
}

func TestListRacesEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListRaces(
		context.Background(),
		connect.NewRequest(&queryv1.ListRacesRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

//nolint:lll // readability
func TestListRacesReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1")
	race1 := seedRace(t, repo, event.ID, "Qualifying", conversion.RaceSessionTypeQualifying, 1)
	race2 := seedRace(t, repo, event.ID, "Feature Race", conversion.RaceSessionTypeRace, 2)

	resp, err := svc.ListRaces(
		context.Background(),
		connect.NewRequest(&queryv1.ListRacesRequest{}),
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

	if !ids[uint32(race1.ID)] {
		t.Errorf("race1 (id=%d) not found in response", race1.ID)
	}
	if !ids[uint32(race2.ID)] {
		t.Errorf("race2 (id=%d) not found in response", race2.ID)
	}
}

//nolint:lll // readability
func TestListRacesByEventID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event1 := seedEvent(t, repo, season.ID, layout.ID, "Round 1")
	event2 := seedEvent(t, repo, season.ID, layout.ID, "Round 2")
	race1 := seedRace(t, repo, event1.ID, "Feature Race", conversion.RaceSessionTypeRace, 1)
	seedRace(t, repo, event2.ID, "Feature Race", conversion.RaceSessionTypeRace, 1)

	resp, err := svc.ListRaces(
		context.Background(),
		connect.NewRequest(&queryv1.ListRacesRequest{
			EventId: uint32(event1.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetEventId() != uint32(event1.ID) {
		t.Errorf("expected event_id %d, got %d", event1.ID, items[0].GetEventId())
	}
	if items[0].GetId() != uint32(race1.ID) {
		t.Errorf("expected id %d, got %d", race1.ID, items[0].GetId())
	}
}

func TestGetRaceSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1")
	race := seedRace(t, repo, event.ID, "Feature Race", conversion.RaceSessionTypeRace, 1)

	resp, err := svc.GetRace(
		context.Background(),
		connect.NewRequest(&queryv1.GetRaceRequest{
			Id: uint32(race.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := resp.Msg.GetRace()
	if got.GetId() != uint32(race.ID) {
		t.Errorf("expected id %d, got %d", race.ID, got.GetId())
	}
	if got.GetEventId() != uint32(event.ID) {
		t.Errorf("expected event_id %d, got %d", event.ID, got.GetEventId())
	}
	if got.GetName() != "Feature Race" {
		t.Errorf("expected name %q, got %q", "Feature Race", got.GetName())
	}
	if got.GetSessionType() != commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE {
		t.Errorf("expected session_type %v, got %v",
			commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE, got.GetSessionType())
	}
	if got.GetSequenceNo() != 1 {
		t.Errorf("expected sequence_no 1, got %d", got.GetSequenceNo())
	}
}

func TestGetRaceNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetRace(
		context.Background(),
		connect.NewRequest(&queryv1.GetRaceRequest{
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
