package query

import (
	"context"
	"errors"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
)

//nolint:whitespace // editor/linter issue
func seedTeam(
	t *testing.T,
	repo rootrepo.Repository,
	seasonID int32,
	name string,
) *models.Team {
	t.Helper()
	team, err := repo.Teams().Teams().Create(context.Background(), &models.TeamSetter{
		SeasonID:  omit.From(seasonID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed team %q: %v", name, err)
	}
	return team
}

func TestListTeamsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListTeams(
		context.Background(),
		connect.NewRequest(&queryv1.ListTeamsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListTeamsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")

	alpha := seedTeam(t, repo, season.ID, "Alpha Team")
	beta := seedTeam(t, repo, season.ID, "Beta Team")

	resp, err := svc.ListTeams(
		context.Background(),
		connect.NewRequest(&queryv1.ListTeamsRequest{}),
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

	if !ids[uint32(alpha.ID)] {
		t.Errorf("alpha team (id=%d) not found in response", alpha.ID)
	}
	if !ids[uint32(beta.ID)] {
		t.Errorf("beta team (id=%d) not found in response", beta.ID)
	}
}

func TestListTeamsBySeasonID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season1 := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	season2 := seedSeason(t, repo, series.ID, pointSystem.ID, "2025 Season")

	team1 := seedTeam(t, repo, season1.ID, "Team Alpha")
	seedTeam(t, repo, season2.ID, "Team Beta")

	resp, err := svc.ListTeams(
		context.Background(),
		connect.NewRequest(&queryv1.ListTeamsRequest{
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
	if items[0].GetId() != uint32(team1.ID) {
		t.Errorf("expected id %d, got %d", team1.ID, items[0].GetId())
	}
}

func TestGetTeamSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	team := seedTeam(t, repo, season.ID, "Test Team")

	resp, err := svc.GetTeam(
		context.Background(),
		connect.NewRequest(&queryv1.GetTeamRequest{
			Id: uint32(team.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := resp.Msg.GetTeam()
	if got.GetId() != uint32(team.ID) {
		t.Errorf("expected id %d, got %d", team.ID, got.GetId())
	}
	if got.GetSeasonId() != uint32(season.ID) {
		t.Errorf("expected season_id %d, got %d", season.ID, got.GetSeasonId())
	}
	if got.GetName() != "Test Team" {
		t.Errorf("expected name %q, got %q", "Test Team", got.GetName())
	}
	if !got.GetIsActive() {
		t.Errorf("expected is_active true, got false")
	}
}

func TestGetTeamNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetTeam(
		context.Background(),
		connect.NewRequest(&queryv1.GetTeamRequest{
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
