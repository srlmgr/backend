package queries

import (
	"context"
	"errors"
	"testing"

	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/testhelpers"
)

//nolint:funlen // exercises each supported resolution depth against one fixture
func Test_queryResolver_ResolveSeasonTarget(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	track := testhelpers.SeedTrack(t, "Track A")
	layout := testhelpers.SeedTrackLayout(t, track.ID, "Layout A")
	event := testhelpers.SeedEvent(t, season.ID, pointSystem.ID, layout.ID, "Event A", 4)
	race := testhelpers.SeedRace(t, event.ID, "Race A", "race", 2)
	raceGrid := testhelpers.SeedRaceGrid(t, race.ID, "Grid A", "race", 3)

	resolver := repo.QueryResolver()
	resolve := func(eventNo, raceNo, raceGridNo int32) (*rootrepo.SeasonTarget, error) {
		ctx := context.Background()
		switch {
		case raceNo == 0:
			return resolver.ResolveEvent(ctx, season.ID, eventNo)
		case raceGridNo == 0:
			return resolver.ResolveRace(ctx, season.ID, eventNo, raceNo)
		default:
			return resolver.ResolveRaceGrid(ctx, season.ID, eventNo, raceNo, raceGridNo)
		}
	}

	tests := []struct {
		name       string
		eventNo    int32
		raceNo     int32
		raceGridNo int32
		wantEvent  int32
		wantRace   int32
		wantGrid   int32
		wantErr    bool
	}{
		{
			name:      "event",
			eventNo:   4,
			wantEvent: event.ID,
		},
		{
			name:      "event and race",
			eventNo:   4,
			raceNo:    2,
			wantEvent: event.ID,
			wantRace:  race.ID,
		},
		{
			name:       "event, race, and race grid",
			eventNo:    4,
			raceNo:     2,
			raceGridNo: 3,
			wantEvent:  event.ID,
			wantRace:   race.ID,
			wantGrid:   raceGrid.ID,
		},
		{
			name:    "event in other season is not found",
			eventNo: 5,
			wantErr: true,
		},
		{
			name:    "unknown event",
			eventNo: 99,
			wantErr: true,
		},
		{
			name:    "unknown race",
			eventNo: 4,
			raceNo:  99,
			wantErr: true,
		},
		{
			name:       "unknown race grid",
			eventNo:    4,
			raceNo:     2,
			raceGridNo: 99,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolve(tt.eventNo, tt.raceNo, tt.raceGridNo)
			if tt.wantErr {
				if !errors.Is(err, repoerrors.ErrNotFound) {
					t.Fatalf("ResolveSeasonTarget() error = %v, want ErrNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSeasonTarget() failed: %v", err)
			}
			matches := got.EventID == tt.wantEvent &&
				got.RaceID == tt.wantRace &&
				got.RaceGridID == tt.wantGrid
			if !matches {
				t.Errorf(
					"ResolveSeasonTarget() = %+v, want event=%d race=%d grid=%d",
					got,
					tt.wantEvent,
					tt.wantRace,
					tt.wantGrid,
				)
			}
		})
	}
}
