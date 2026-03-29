// Package testhelpers provides shared test utilities for repository sub-packages.
//
//nolint:dupl,whitespace // shared seed helpers are intentionally repetitive
package testhelpers

import (
	"context"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/testsupport/testdb"
)

const TestUserSeed = "seed"

var TestPool *pgxpool.Pool

// InitTestPool initializes the test database pool.
// It must be called from TestMain in each test package.
func InitTestPool() (*pgxpool.Pool, error) {
	return testdb.InitTestDB()
}

// ResetTestTables clears all tables in the test database. Call in test setup/cleanup.
func ResetTestTables(t *testing.T) {
	t.Helper()

	if err := testdb.ClearAllTables(TestPool); err != nil {
		t.Fatalf("failed to reset test tables: %v", err)
	}
}

func getExecutor(t *testing.T) *pgbob.Executor {
	t.Helper()

	if TestPool == nil {
		t.Fatal("test pool is not initialized")
	}

	return pgbob.New(TestPool)
}

func SeedRacingSim(t *testing.T, name string) *models.RacingSim {
	t.Helper()

	sim, err := models.RacingSims.Insert(&models.RacingSimSetter{
		Name:                   omit.From(name),
		SupportedImportFormats: omit.From(pq.StringArray{"json"}),
		IsActive:               omit.From(true),
		CreatedBy:              omit.From(TestUserSeed),
		UpdatedBy:              omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed racing sim %q: %v", name, err)
	}

	return sim
}

func SeedSeries(t *testing.T, simulationID int32, name string) *models.Series {
	t.Helper()

	series, err := models.Serieses.Insert(&models.SeriesSetter{
		SimulationID: omit.From(simulationID),
		Name:         omit.From(name),
		IsActive:     omit.From(true),
		CreatedBy:    omit.From(TestUserSeed),
		UpdatedBy:    omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed series %q: %v", name, err)
	}

	return series
}

func SeedPointSystem(t *testing.T, name string) *models.PointSystem {
	t.Helper()

	pointSystem, err := models.PointSystems.Insert(&models.PointSystemSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed point system %q: %v", name, err)
	}

	return pointSystem
}

func SeedSeason(
	t *testing.T,
	seriesID int32,
	pointSystemID int32,
	name string,
) *models.Season {
	t.Helper()

	season, err := models.Seasons.Insert(&models.SeasonSetter{
		SeriesID:      omit.From(seriesID),
		PointSystemID: omit.From(pointSystemID),
		Name:          omit.From(name),
		HasTeams:      omit.From(false),
		SkipEvents:    omit.From(int32(0)),
		Status:        omit.From("active"),
		CreatedBy:     omit.From(TestUserSeed),
		UpdatedBy:     omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed season %q: %v", name, err)
	}

	return season
}

func SeedTrack(t *testing.T, name string) *models.Track {
	t.Helper()

	track, err := models.Tracks.Insert(&models.TrackSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed track %q: %v", name, err)
	}

	return track
}

func SeedTrackLayout(t *testing.T, trackID int32, name string) *models.TrackLayout {
	t.Helper()

	layout, err := models.TrackLayouts.Insert(&models.TrackLayoutSetter{
		TrackID:   omit.From(trackID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed track layout %q: %v", name, err)
	}

	return layout
}

func SeedRace(
	t *testing.T,
	eventID int32,
	name string,
	sessionType string,
	sequenceNo int32,
) *models.Race {
	t.Helper()

	race, err := models.Races.Insert(&models.RaceSetter{
		EventID:     omit.From(eventID),
		Name:        omit.From(name),
		SessionType: omit.From(sessionType),
		SequenceNo:  omit.From(sequenceNo),
		CreatedBy:   omit.From(TestUserSeed),
		UpdatedBy:   omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed race %q: %v", name, err)
	}

	return race
}

func SeedRaceGrid(
	t *testing.T,
	raceID int32,
	name string,
	sessionType string,
	sequenceNo int32,
) *models.RaceGrid {
	t.Helper()

	grid, err := models.RaceGrids.Insert(&models.RaceGridSetter{
		RaceID:      omit.From(raceID),
		Name:        omit.From(name),
		SessionType: omit.From(sessionType),
		SequenceNo:  omit.From(sequenceNo),
		CreatedBy:   omit.From(TestUserSeed),
		UpdatedBy:   omit.From(TestUserSeed),
	}).One(context.Background(), getExecutor(t))
	if err != nil {
		t.Fatalf("failed to seed race grid %q: %v", name, err)
	}

	return grid
}
