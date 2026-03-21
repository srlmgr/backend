//nolint:lll // test files can have some duplication and long lines for test data setup
package query

import (
	"context"
	"os"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/testsupport/testdb"
)

const (
	testUserSeed = "seed"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := testdb.InitTestDB()
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	testPool.Close()
	os.Exit(code)
}

func newDBBackedQueryService(t *testing.T) (*service, rootrepo.Repository) {
	t.Helper()
	resetTestTables(t)
	t.Cleanup(func() {
		resetTestTables(t)
	})

	repo := postgresrepo.New(testPool)
	txMgr := rootrepo.NewBobTransactionFromPool(testPool)

	svc := &service{
		logger:     log.New(),
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
	}
	return svc, repo
}

func resetTestTables(t *testing.T) {
	t.Helper()

	if _, err := testPool.Exec(
		context.Background(),
		"TRUNCATE TABLE racing_sims, series, point_systems, seasons, tracks, track_layouts, events, car_manufacturers, car_brands, car_models, drivers RESTART IDENTITY CASCADE",
	); err != nil {
		t.Fatalf("failed to reset test tables: %v", err)
	}
}

//nolint:unparam // caller may use with different parameters in the future
func seedSimulation(t *testing.T, repo rootrepo.Repository, name string) *models.RacingSim {
	t.Helper()

	sim, err := repo.RacingSims().Create(context.Background(), &models.RacingSimSetter{
		Name:                   omit.From(name),
		IsActive:               omit.From(true),
		SupportedImportFormats: omit.From(pq.StringArray{"json"}),
		CreatedBy:              omit.From(testUserSeed),
		UpdatedBy:              omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed simulation %q: %v", name, err)
	}

	return sim
}

func seedSeries(t *testing.T, repo rootrepo.Repository, simID int32, name string) *models.Series {
	t.Helper()

	s, err := repo.Series().Create(context.Background(), &models.SeriesSetter{
		SimulationID: omit.From(simID),
		Name:         omit.From(name),
		CreatedBy:    omit.From(testUserSeed),
		UpdatedBy:    omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed series %q: %v", name, err)
	}

	return s
}

//nolint:whitespace // multiline signature style
func seedSeason(
	t *testing.T,
	repo rootrepo.Repository,
	seriesID int32,
	pointSystemID int32,
	name string,
) *models.Season {
	t.Helper()

	season, err := repo.Seasons().Create(context.Background(), &models.SeasonSetter{
		SeriesID:      omit.From(seriesID),
		PointSystemID: omit.From(pointSystemID),
		Name:          omit.From(name),
		HasTeams:      omit.From(false),
		SkipEvents:    omit.From(int32(0)),
		Status:        omit.From("active"),
		CreatedBy:     omit.From(testUserSeed),
		UpdatedBy:     omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed season %q: %v", name, err)
	}

	return season
}

func seedTrack(t *testing.T, repo rootrepo.Repository, name string) *models.Track {
	t.Helper()

	track, err := repo.Tracks().Tracks().Create(context.Background(), &models.TrackSetter{
		Name:      omit.From(name),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed track %q: %v", name, err)
	}

	return track
}

//nolint:whitespace // multiline signature style
func seedTrackLayout(
	t *testing.T,
	repo rootrepo.Repository,
	trackID int32,
	name string,
) *models.TrackLayout {
	t.Helper()

	layout, err := repo.Tracks().
		TrackLayouts().
		Create(context.Background(), &models.TrackLayoutSetter{
			TrackID:   omit.From(trackID),
			Name:      omit.From(name),
			CreatedBy: omit.From(testUserSeed),
			UpdatedBy: omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed track layout %q: %v", name, err)
	}

	return layout
}
