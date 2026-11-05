package query

import (
	"context"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect/v2"

	"github.com/srlmgr/backend/grpc/services/conversion"
)

func TestResolveSeasonTarget(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, pointSystem.ID, layout.ID, "Round 1", 4)
	race := seedRace(t, repo, event.ID, "Feature Race", conversion.RaceSessionTypeRace, 2)
	raceGrid := seedRaceGrid(
		t,
		repo,
		race.ID,
		"Starting Grid",
		conversion.RaceSessionTypeRace,
		3,
	)

	resp, err := svc.ResolveSeasonTarget(
		context.Background(),
		&queryv1.ResolveSeasonTargetRequest{
			SeasonId:   uint32(season.ID),
			EventNo:    4,
			RaceNo:     2,
			RaceGridNo: 3,
		},
	)
	if err != nil {
		t.Fatalf("ResolveSeasonTarget() failed: %v", err)
	}
	matches := resp.GetEventId() == uint32(event.ID) &&
		resp.GetRaceId() == uint32(race.ID) &&
		resp.GetRaceGridId() == uint32(raceGrid.ID)
	if !matches {
		t.Errorf(
			"ResolveSeasonTarget() = event=%d race=%d grid=%d, want event=%d race=%d grid=%d",
			resp.GetEventId(),
			resp.GetRaceId(),
			resp.GetRaceGridId(),
			event.ID,
			race.ID,
			raceGrid.ID,
		)
	}
}

func TestResolveSeasonTargetRequiresSeasonID(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.ResolveSeasonTarget(
		context.Background(),
		&queryv1.ResolveSeasonTargetRequest{},
	)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf(
			"ResolveSeasonTarget() error code = %v, want %v",
			connect.CodeOf(err),
			connect.CodeInvalidArgument,
		)
	}
}
