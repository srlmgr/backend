package events

import (
	"context"
	"errors"
	"testing"

	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/testhelpers"
)

func TestRepositoryCreateLoadByIDLoadAll(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	track := testhelpers.SeedTrack(t, "Track A")
	trackLayout := testhelpers.SeedTrackLayout(t, track.ID, "Layout A")

	created := seedEvent(t, repo, season.ID, trackLayout.ID, "Round 1")

	loaded, err := repo.LoadByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("LoadByID returned error: %v", err)
	}
	if loaded.ID != created.ID {
		t.Fatalf("unexpected event id: got %d want %d", loaded.ID, created.ID)
	}
	if loaded.Name != "Round 1" {
		t.Fatalf("unexpected event name: got %q want %q", loaded.Name, "Round 1")
	}

	all, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("unexpected number of events: got %d want 1", len(all))
	}
}

func TestRepositoryLoadBySeasonID(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	seasonA := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	seasonB := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season B")
	track := testhelpers.SeedTrack(t, "Track A")
	trackLayout := testhelpers.SeedTrackLayout(t, track.ID, "Layout A")

	eventA := seedEvent(t, repo, seasonA.ID, trackLayout.ID, "Round A")
	_ = seedEvent(t, repo, seasonB.ID, trackLayout.ID, "Round B")

	bySeason, err := repo.LoadBySeasonID(context.Background(), seasonA.ID)
	if err != nil {
		t.Fatalf("LoadBySeasonID returned error: %v", err)
	}
	if len(bySeason) != 1 {
		t.Fatalf("unexpected season event count: got %d want 1", len(bySeason))
	}
	if bySeason[0].ID != eventA.ID {
		t.Fatalf("unexpected season event id: got %d want %d", bySeason[0].ID, eventA.ID)
	}
}

func TestRepositoryLoadByGridID(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	track := testhelpers.SeedTrack(t, "Track A")
	trackLayout := testhelpers.SeedTrackLayout(t, track.ID, "Layout A")
	event := seedEvent(t, repo, season.ID, trackLayout.ID, "Round 1")
	race := testhelpers.SeedRace(t, event.ID, "Race A", "R", 1)
	grid := testhelpers.SeedRaceGrid(t, race.ID, "Grid A", "R", 1)

	loaded, err := repo.LoadByGridID(context.Background(), grid.ID)
	if err != nil {
		t.Fatalf("LoadByGridID returned error: %v", err)
	}
	if loaded.ID != event.ID {
		t.Fatalf("unexpected event id for grid: got %d want %d", loaded.ID, event.ID)
	}
}

func TestRepositoryLoadByIDNotFound(t *testing.T) {
	repo := newDBBackedRepository(t)

	_, err := repo.LoadByID(context.Background(), 99999)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepositoryUpdate(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	track := testhelpers.SeedTrack(t, "Track A")
	trackLayout := testhelpers.SeedTrackLayout(t, track.ID, "Layout A")
	event := seedEvent(t, repo, season.ID, trackLayout.ID, "Round 1")

	updated, err := repo.Update(context.Background(), event.ID, &models.EventSetter{
		Name:      omit.From("Round 1 Updated"),
		UpdatedBy: omit.From("editor"),
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Name != "Round 1 Updated" {
		t.Fatalf("unexpected updated name: got %q want %q", updated.Name, "Round 1 Updated")
	}

	_, err = repo.Update(context.Background(), 99999, &models.EventSetter{
		Name: omit.From("missing"),
	})
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing update, got %v", err)
	}
}

func TestRepositoryDeleteByID(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	track := testhelpers.SeedTrack(t, "Track A")
	trackLayout := testhelpers.SeedTrackLayout(t, track.ID, "Layout A")
	event := seedEvent(t, repo, season.ID, trackLayout.ID, "Round 1")

	if err := repo.DeleteByID(context.Background(), event.ID); err != nil {
		t.Fatalf("DeleteByID returned error: %v", err)
	}

	_, err := repo.LoadByID(context.Background(), event.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
