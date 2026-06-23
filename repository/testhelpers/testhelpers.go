// Package testhelpers provides shared test utilities for repository sub-packages.
//
//nolint:dupl,whitespace,lll // shared seed helpers are intentionally repetitive
package testhelpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	bobtypes "github.com/stephenafamo/bob/types"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
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

func getExecutorFromContext(t *testing.T, ctx context.Context) bob.Executor {
	t.Helper()

	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}

	return getExecutor(t)
}

func emptyJSON(t *testing.T) bobtypes.JSON[json.RawMessage] {
	t.Helper()

	return bobtypes.NewJSON[json.RawMessage]([]byte("{}"))
}

func SeedRacingSim(t *testing.T, name string) *models.RacingSim {
	t.Helper()
	return SeedRacingSimContext(t, context.Background(), name)
}

func SeedRacingSimContext(
	t *testing.T,
	ctx context.Context,
	name string,
) *models.RacingSim {
	t.Helper()

	sim, err := models.RacingSims.Insert(&models.RacingSimSetter{
		Name: omit.From(name),
		SupportedImportFormats: omit.From(func() bobtypes.JSON[json.RawMessage] {
			b, _ := json.Marshal([]mytypes.RaceSimImportFormat{{Format: "json"}})
			return bobtypes.JSON[json.RawMessage]{Val: json.RawMessage(b)}
		}()),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed racing sim %q: %v", name, err)
	}

	return sim
}

func SeedSeries(t *testing.T, simulationID int32, name string) *models.Series {
	t.Helper()
	return SeedSeriesContext(t, context.Background(), simulationID, name)
}

func SeedSeriesContext(
	t *testing.T,
	ctx context.Context,
	simulationID int32,
	name string,
) *models.Series {
	t.Helper()

	series, err := models.Serieses.Insert(&models.SeriesSetter{
		SimulationID: omit.From(simulationID),
		Name:         omit.From(name),
		IsActive:     omit.From(true),
		CreatedBy:    omit.From(TestUserSeed),
		UpdatedBy:    omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed series %q: %v", name, err)
	}

	return series
}

func SeedPointSystem(t *testing.T, name string) *models.PointSystem {
	t.Helper()
	return SeedPointSystemContext(t, context.Background(), name)
}

func SeedPointSystemContext(
	t *testing.T,
	ctx context.Context,
	name string,
) *models.PointSystem {
	t.Helper()

	pointSystem, err := models.PointSystems.Insert(&models.PointSystemSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
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
	return SeedSeasonContext(t, context.Background(), seriesID, pointSystemID, name)
}

func SeedSeasonContext(
	t *testing.T,
	ctx context.Context,
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
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed season %q: %v", name, err)
	}

	return season
}

func SeedCarManufacturer(t *testing.T, name string) *models.CarManufacturer {
	t.Helper()
	return SeedCarManufacturerContext(t, context.Background(), name)
}

func SeedCarManufacturerContext(
	t *testing.T,
	ctx context.Context,
	name string,
) *models.CarManufacturer {
	t.Helper()

	manufacturer, err := models.CarManufacturers.Insert(&models.CarManufacturerSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed car manufacturer %q: %v", name, err)
	}

	return manufacturer
}

func SeedCarBrand(t *testing.T, name string, manufacturerID int32) *models.CarBrand {
	t.Helper()
	return SeedCarBrandContext(t, context.Background(), name, manufacturerID)
}

func SeedCarBrandContext(
	t *testing.T,
	ctx context.Context,
	name string,
	manufacturerID int32,
) *models.CarBrand {
	t.Helper()

	brand, err := models.CarBrands.Insert(&models.CarBrandSetter{
		ManufacturerID: omit.From(manufacturerID),
		Name:           omit.From(name),
		IsActive:       omit.From(true),
		CreatedBy:      omit.From(TestUserSeed),
		UpdatedBy:      omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed car brand %q: %v", name, err)
	}

	return brand
}

func SeedCarModel(t *testing.T, name string, brandID int32) *models.CarModel {
	t.Helper()
	return SeedCarModelContext(t, context.Background(), name, brandID)
}

func SeedCarModelContext(
	t *testing.T,
	ctx context.Context,
	name string,
	brandID int32,
) *models.CarModel {
	t.Helper()

	model, err := models.CarModels.Insert(&models.CarModelSetter{
		BrandID:   omit.From(brandID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed car model %q: %v", name, err)
	}

	return model
}

func SeedSeasonCar(
	t *testing.T,
	seasonID int32,
	carModelID int32,
	pos int32,
) *models.SeasonCarModel {
	t.Helper()
	return SeedSeasonCarContext(t, context.Background(), seasonID, carModelID, pos)
}

func SeedSeasonCarContext(
	t *testing.T,
	ctx context.Context,
	seasonID int32,
	carModelID int32,
	pos int32,
) *models.SeasonCarModel {
	t.Helper()
	return SeedSeasonCarModelContext(t, ctx, seasonID, carModelID, pos)
}

func SeedSeasonCarModel(
	t *testing.T,
	seasonID int32,
	carModelID int32,
	pos int32,
) *models.SeasonCarModel {
	t.Helper()
	return SeedSeasonCarModelContext(t, context.Background(), seasonID, carModelID, pos)
}

func SeedSeasonCarModelContext(
	t *testing.T,
	ctx context.Context,
	seasonID int32,
	carModelID int32,
	pos int32,
) *models.SeasonCarModel {
	t.Helper()

	seasonCarModel, err := models.SeasonCarModels.Insert(&models.SeasonCarModelSetter{
		SeasonID:   omit.From(seasonID),
		CarModelID: omit.From(carModelID),
		Pos:        omit.From(pos),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf(
			"failed to seed season car model for season %d and car model %d: %v",
			seasonID,
			carModelID,
			err,
		)
	}

	return seasonCarModel
}

func SeedSeasonCarClass(
	t *testing.T,
	seasonID int32,
	carClassID int32,
	pos int32,
) *models.SeasonCarClass {
	t.Helper()
	return SeedSeasonCarClassContext(t, context.Background(), seasonID, carClassID, pos)
}

func SeedSeasonCarClassContext(
	t *testing.T,
	ctx context.Context,
	seasonID int32,
	carClassID int32,
	pos int32,
) *models.SeasonCarClass {
	t.Helper()

	seasonCarClass, err := models.SeasonCarClasses.Insert(&models.SeasonCarClassSetter{
		SeasonID:   omit.From(seasonID),
		CarClassID: omit.From(carClassID),
		Pos:        omit.From(pos),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf(
			"failed to seed season car class for season %d and car class %d: %v",
			seasonID,
			carClassID,
			err,
		)
	}

	return seasonCarClass
}

func SeedTrack(t *testing.T, name string) *models.Track {
	t.Helper()
	return SeedTrackContext(t, context.Background(), name)
}

func SeedTrackContext(
	t *testing.T,
	ctx context.Context,
	name string,
) *models.Track {
	t.Helper()

	track, err := models.Tracks.Insert(&models.TrackSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed track %q: %v", name, err)
	}

	return track
}

func SeedDriver(t *testing.T, name, externalID string) *models.Driver {
	t.Helper()
	return SeedDriverContext(t, context.Background(), name, externalID)
}

func SeedDriverContext(
	t *testing.T,
	ctx context.Context,
	name, externalID string,
) *models.Driver {
	t.Helper()
	setter := &models.DriverSetter{
		ExternalID: omit.From(externalID),
		Name:       omit.From(name),
		IsActive:   omit.From(true),
		CreatedBy:  omit.From(TestUserSeed),
		UpdatedBy:  omit.From(TestUserSeed),
	}
	driver, err := models.Drivers.Insert(setter).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed driver %q: %v", name, err)
	}

	return driver
}

func SeedSeasonDriver(
	t *testing.T,
	carNumber string,
	driverID, seasonID, carModelID int32,
	joinedAt time.Time,
	leftAt *time.Time,
) *models.SeasonDriver {
	t.Helper()
	return SeedSeasonDriverContext(
		t,
		context.Background(),
		carNumber,
		driverID,
		seasonID,
		carModelID,
		joinedAt,
		leftAt,
	)
}

func SeedSeasonDriverContext(
	t *testing.T,
	ctx context.Context,
	carNumber string,
	driverID, seasonID, carModelID int32,
	joinedAt time.Time, leftAt *time.Time,
) *models.SeasonDriver {
	t.Helper()
	setter := &models.SeasonDriverSetter{
		CarNumber:  omit.From(carNumber),
		DriverID:   omit.From(driverID),
		SeasonID:   omit.From(seasonID),
		CarModelID: omit.From(carModelID),
		JoinedAt:   omit.From(joinedAt),
		CreatedBy:  omit.From(TestUserSeed),
		UpdatedBy:  omit.From(TestUserSeed),
	}
	if leftAt != nil {
		setter.LeftAt = omitnull.From(*leftAt)
	}
	seasonDriver, err := models.SeasonDrivers.Insert(setter).
		One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed season driver %q: %v", carNumber, err)
	}

	return seasonDriver
}

func SeedTeamSimple(
	t *testing.T,
	seasonID int32,
	name string,
	joinedAt time.Time,
	leftAt *time.Time,
) *models.Team {
	t.Helper()
	return SeedTeamSimpleContext(t, context.Background(), seasonID, name, joinedAt, leftAt)
}

func SeedTeamSimpleContext(
	t *testing.T,
	ctx context.Context,
	seasonID int32,
	name string,
	joinedAt time.Time,
	leftAt *time.Time,
) *models.Team {
	t.Helper()

	team, err := models.Teams.Insert(&models.TeamSetter{
		SeasonID:  omit.From(seasonID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		JoinedAt:  omit.From(joinedAt),
		LeftAt:    omitnull.FromPtr(leftAt),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed team %q: %v", name, err)
	}

	return team
}

func SeedTeamDriver(
	t *testing.T,
	teamID, driverID int32,
	joinedAt time.Time,
	leftAt *time.Time,
) *models.TeamDriver {
	t.Helper()
	return SeedTeamDriverContext(t, context.Background(), teamID, driverID, joinedAt, leftAt)
}

func SeedTeamDriverContext(
	t *testing.T,
	ctx context.Context,
	teamID, driverID int32, joinedAt time.Time, leftAt *time.Time,
) *models.TeamDriver {
	t.Helper()
	setter := &models.TeamDriverSetter{
		TeamID:    omit.From(teamID),
		DriverID:  omit.From(driverID),
		JoinedAt:  omit.From(joinedAt),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}
	if leftAt != nil {
		setter.LeftAt = omitnull.From(*leftAt)
	}
	teamDriver, err := models.TeamDrivers.Insert(setter).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed team member %q: %v", driverID, err)
	}

	return teamDriver
}

func SeedTrackLayout(t *testing.T, trackID int32, name string) *models.TrackLayout {
	t.Helper()
	return SeedTrackLayoutContext(t, context.Background(), trackID, name)
}

func SeedTrackLayoutContext(
	t *testing.T,
	ctx context.Context,
	trackID int32,
	name string,
) *models.TrackLayout {
	t.Helper()

	layout, err := models.TrackLayouts.Insert(&models.TrackLayoutSetter{
		TrackID:   omit.From(trackID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(TestUserSeed),
		UpdatedBy: omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed track layout %q: %v", name, err)
	}

	return layout
}

func SeedEvent(
	t *testing.T,
	seasonID int32,
	trackLayoutID int32,
	name string,
	sequenceNo int32,
) *models.Event {
	t.Helper()
	return SeedEventContext(t, context.Background(),
		seasonID, trackLayoutID, name, sequenceNo)
}

func SeedEventContext(
	t *testing.T,
	ctx context.Context,
	seasonID int32,
	trackLayoutID int32,
	name string,
	sequenceNo int32,
) *models.Event {
	t.Helper()

	event, err := models.Events.Insert(&models.EventSetter{
		SeasonID:      omit.From(seasonID),
		TrackLayoutID: omit.From(trackLayoutID),
		Name:          omit.From(name),
		SequenceNo:    omit.From(sequenceNo),
		EventDate:     omit.From(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		CreatedBy:     omit.From(TestUserSeed),
		UpdatedBy:     omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed event %q: %v", name, err)
	}

	return event
}

func SeedRace(
	t *testing.T,
	eventID int32,
	name string,
	sessionType string,
	sequenceNo int32,
) *models.Race {
	t.Helper()
	return SeedRaceContext(t, context.Background(), eventID, name, sessionType, sequenceNo)
}

func SeedRaceContext(
	t *testing.T,
	ctx context.Context,
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
	}).One(ctx, getExecutorFromContext(t, ctx))
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
	return SeedRaceGridContext(
		t,
		context.Background(),
		raceID,
		name,
		sessionType,
		sequenceNo)
}

func SeedRaceGridContext(
	t *testing.T,
	ctx context.Context,
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
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed race grid %q: %v", name, err)
	}

	return grid
}

func SeedResultEntry(
	t *testing.T,
	raceGridID int32,
	driverName string,
	finishPosition int32,
) *models.ResultEntry {
	t.Helper()
	return SeedResultEntryContext(
		t,
		context.Background(),
		raceGridID,
		driverName,
		finishPosition,
	)
}

func SeedResultEntryContext(
	t *testing.T,
	ctx context.Context,
	raceGridID int32,
	driverName string,
	finishPosition int32,
) *models.ResultEntry {
	t.Helper()

	entry, err := models.ResultEntries.Insert(&models.ResultEntrySetter{
		RaceGridID:     omit.From(raceGridID),
		RawDriverName:  omitnull.From(driverName),
		FinishPosition: omit.From(finishPosition),
		LapsCompleted:  omit.From(int32(0)),
		State:          omit.From("normal"),
		CreatedBy:      omit.From(TestUserSeed),
		UpdatedBy:      omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed result entry %q: %v", driverName, err)
	}

	return entry
}

func SeedBookingEntry(
	t *testing.T,
	eventID int32,
	raceID int32,
	raceGridID int32,
	description string,
	points int32,
) *models.BookingEntry {
	t.Helper()
	return SeedBookingEntryContext(
		t,
		context.Background(),
		eventID,
		raceID,
		raceGridID,
		description,
		points,
	)
}

func SeedBookingEntryContext(
	t *testing.T,
	ctx context.Context,
	eventID int32,
	raceID int32,
	raceGridID int32,
	description string,
	points int32,
) *models.BookingEntry {
	t.Helper()

	entry, err := models.BookingEntries.Insert(&models.BookingEntrySetter{
		EventID:      omit.From(eventID),
		RaceID:       omitnull.From(raceID),
		RaceGridID:   omitnull.From(raceGridID),
		TargetType:   omit.From(mytypes.TargetType("driver")),
		SourceType:   omit.From(mytypes.SourceType("finish_pos")),
		Points:       omit.From(points),
		Description:  omit.From(description),
		IsManual:     omit.From(false),
		MetadataJSON: omit.From(emptyJSON(t)),
		CreatedBy:    omit.From(TestUserSeed),
		UpdatedBy:    omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed booking entry %q: %v", description, err)
	}

	return entry
}

func SeedImportBatch(
	t *testing.T,
	gridID int32,
	sourceFilename string,
) *models.ImportBatch {
	t.Helper()
	return SeedImportBatchContext(t, context.Background(), gridID, sourceFilename)
}

func SeedImportBatchContext(
	t *testing.T,
	ctx context.Context,
	gridID int32,
	sourceFilename string,
) *models.ImportBatch {
	t.Helper()

	batch, err := models.ImportBatches.Insert(&models.ImportBatchSetter{
		RaceGridID:      omit.From(gridID),
		ImportFormat:    omit.From(mytypes.ImportFormat("json")),
		Payload:         omit.From([]byte(`{"entries":[]}`)),
		SourceFilename:  omitnull.From(sourceFilename),
		ProcessingState: omit.From("raw_imported"),
		MetadataJSON:    omit.From(mytypes.ImportBatchMeta{}),
		CreatedBy:       omit.From(TestUserSeed),
		UpdatedBy:       omit.From(TestUserSeed),
	}).One(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to seed import batch %q: %v", sourceFilename, err)
	}

	return batch
}
