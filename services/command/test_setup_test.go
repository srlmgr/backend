//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"os"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/testsupport/testdb"
)

const (
	testUserSeed   = "seed"
	testUserTester = "tester"
	testUserEditor = "editor"

	txFailedErrMsg = "transaction failed"
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

type txManagerStub struct {
	runInTx func(
		ctx context.Context,
		fn func(ctx context.Context) error,
	) error
}

//nolint:whitespace // multiline signature style
func (t txManagerStub) RunInTx(
	ctx context.Context,
	fn func(ctx context.Context) error,
) (
	err error,
) {
	if t.runInTx != nil {
		return t.runInTx(ctx, fn)
	}
	return fn(ctx)
}

//nolint:whitespace // multiline signature style
func newTestService(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
) (
	svc *service,
) {
	svc = &service{
		logger:     log.New(),
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
	}

	return svc
}

func newDBBackedTestService(t *testing.T) (*service, rootrepo.Repository) {
	t.Helper()
	resetTestTables(t)
	t.Cleanup(func() {
		resetTestTables(t)
	})

	repo := postgresrepo.New(testPool)
	txMgr := rootrepo.NewBobTransactionFromPool(testPool)

	return newTestService(repo, txMgr), repo
}

func resetTestTables(t *testing.T) {
	t.Helper()

	err := testdb.ClearAllTables(testPool)
	if err != nil {
		t.Fatalf("failed to reset test tables: %v", err)
	}
}

//nolint:whitespace // multiline signature style
func seedSimulation(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	sim *models.RacingSim,
) {
	t.Helper()

	var err error
	sim, err = repo.RacingSims().Create(context.Background(), &models.RacingSimSetter{
		Name:                   omit.From(name),
		IsActive:               omit.From(true),
		SupportedImportFormats: omit.From(pq.StringArray{conversion.ImportFormatJSON}),
		CreatedBy:              omit.From(testUserSeed),
		UpdatedBy:              omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed simulation %q: %v", name, err)
	}

	return sim
}

//nolint:whitespace // multiline signature style
func seedPointSystem(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	ps *models.PointSystem,
) {
	t.Helper()

	var err error
	ps, err = repo.PointSystems().
		PointSystems().
		Create(context.Background(), &models.PointSystemSetter{
			Name:      omit.From(name),
			CreatedBy: omit.From(testUserSeed),
			UpdatedBy: omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed point system %q: %v", name, err)
	}

	return ps
}

//nolint:whitespace // multiline signature style
func seedCarManufacturer(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	cm *models.CarManufacturer,
) {
	t.Helper()

	var err error
	cm, err = repo.Cars().
		CarManufacturers().
		Create(context.Background(), &models.CarManufacturerSetter{
			Name:      omit.From(name),
			IsActive:  omit.From(true),
			CreatedBy: omit.From(testUserSeed),
			UpdatedBy: omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed car manufacturer %q: %v", name, err)
	}

	return cm
}

//nolint:whitespace // multiline signature style
func seedCarBrand(
	t *testing.T,
	repo rootrepo.Repository,
	manufacturerID int32,
	name string,
) (
	cb *models.CarBrand,
) {
	t.Helper()

	var err error
	cb, err = repo.Cars().CarBrands().Create(context.Background(), &models.CarBrandSetter{
		ManufacturerID: omit.From(manufacturerID),
		Name:           omit.From(name),
		IsActive:       omit.From(true),
		CreatedBy:      omit.From(testUserSeed),
		UpdatedBy:      omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car brand %q: %v", name, err)
	}

	return cb
}

//nolint:whitespace // multiline signature style
func seedCarModel(
	t *testing.T,
	repo rootrepo.Repository,
	brandID int32,
	name string,
) (
	cmod *models.CarModel,
) {
	t.Helper()

	var err error
	cmod, err = repo.Cars().CarModels().Create(context.Background(), &models.CarModelSetter{
		BrandID:   omit.From(brandID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car model %q: %v", name, err)
	}

	return cmod
}

//nolint:whitespace // multiline signature style
func seedTrack(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	track *models.Track,
) {
	t.Helper()

	var err error
	track, err = repo.Tracks().Tracks().Create(context.Background(), &models.TrackSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
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
) (
	layout *models.TrackLayout,
) {
	t.Helper()

	var err error
	layout, err = repo.Tracks().
		TrackLayouts().
		Create(context.Background(), &models.TrackLayoutSetter{
			TrackID:   omit.From(trackID),
			Name:      omit.From(name),
			IsActive:  omit.From(true),
			CreatedBy: omit.From(testUserSeed),
			UpdatedBy: omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed track layout %q: %v", name, err)
	}

	return layout
}

//nolint:whitespace // multiline signature style
func seedSeason(
	t *testing.T,
	repo rootrepo.Repository,
	seriesID int32,
	pointSystemID int32,
	name string,
) (
	season *models.Season,
) {
	t.Helper()

	var err error
	season, err = repo.Seasons().Create(context.Background(), &models.SeasonSetter{
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

//nolint:whitespace // multiline signature style
func seedDriver(
	t *testing.T,
	repo rootrepo.Repository,
	externalID string,
	name string,
) (
	driver *models.Driver,
) {
	t.Helper()

	var err error
	driver, err = repo.Drivers().Drivers().Create(context.Background(), &models.DriverSetter{
		ExternalID: omit.From(externalID),
		Name:       omit.From(name),
		IsActive:   omit.From(true),
		CreatedBy:  omit.From(testUserSeed),
		UpdatedBy:  omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed driver %q: %v", name, err)
	}

	return driver
}

//nolint:whitespace // multiline signature style
func seedTeam(
	t *testing.T,
	repo rootrepo.Repository,
	seasonID int32,
	name string,
) (
	team *models.Team,
) {
	t.Helper()

	var err error
	team, err = repo.Teams().Teams().Create(context.Background(), &models.TeamSetter{
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

//nolint:whitespace // multiline signature style
func seedImportBatch(
	t *testing.T,
	repo rootrepo.Repository,
	gridID int32,
) (
	batch *models.ImportBatch,
) {
	t.Helper()

	var err error
	batch, err = repo.ImportBatches().Create(context.Background(), &models.ImportBatchSetter{
		RaceGridID:      omit.From(gridID),
		ImportFormat:    omit.From(mytypes.ImportFormat(conversion.ImportFormatCSV)),
		Payload:         omit.From([]byte("{}")),
		ProcessingState: omit.From(conversion.EventProcessingStateRawImported),
		CreatedBy:       omit.From(testUserSeed),
		UpdatedBy:       omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed import batch: %v", err)
	}

	return batch
}

//nolint:whitespace,unparam // multiline signature style
func seedRaceGrid(
	t *testing.T,
	repo rootrepo.Repository,
	raceID int32,
	name, sessionType string,
	sequenceNo int32,
) *models.RaceGrid {
	t.Helper()
	raceGrid, err := repo.Races().RaceGrids().Create(context.Background(), &models.RaceGridSetter{
		RaceID:      omit.From(raceID),
		Name:        omit.From(name),
		SessionType: omit.From(sessionType),
		SequenceNo:  omit.From(sequenceNo),
		CreatedBy:   omit.From(testUserSeed),
		UpdatedBy:   omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed race grid %q: %v", name, err)
	}
	return raceGrid
}

//nolint:whitespace // multiline signature style
func seedResultEntry(
	t *testing.T,
	repo rootrepo.Repository,
	raceGridID int32,
	driverName string,
	finishingPosition int32,
) (
	entry *models.ResultEntry,
) {
	t.Helper()

	var err error
	entry, err = repo.ResultEntries().Create(context.Background(), &models.ResultEntrySetter{
		RaceGridID:     omit.From(raceGridID),
		RawDriverName:  omitnull.From(driverName),
		FinishPosition: omit.From(finishingPosition),
		LapsCompleted:  omit.From(int32(0)),
		State:          omit.From(conversion.ResultStateNormal),
		CreatedBy:      omit.From(testUserSeed),
		UpdatedBy:      omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed result entry: %v", err)
	}

	return entry
}
