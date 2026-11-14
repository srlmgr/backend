//nolint:lll,whitespace // integration fixtures and assertions
package importsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"google.golang.org/protobuf/proto"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/testhelpers"
)

type mappingTestProcessor struct{}

func (mappingTestProcessor) SupportedFormats() []string { return []string{"json"} }

func (mappingTestProcessor) Process(
	_ context.Context, _ importer.ImportFormat, payload any,
) (*importer.ParsedImportPayload, error) {
	rawPayload, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("expected byte payload, got %T", payload)
	}
	var input importer.ParsedImportPayload
	if err := json.Unmarshal(rawPayload, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

//nolint:funlen,gocyclo // exercises the complete upload, preview, and retry workflow
func TestMappingRPCWorkflow(t *testing.T) {
	if os.Getenv("TESTDB_URL") != "" {
		t.Skip("requires an isolated testcontainers database")
	}
	pool := testhelpers.NewTestPool(t, nil)
	repo := postgresrepo.New(pool)
	svc := &service{
		repo:       repo,
		txMgr:      repository.NewBobTransactionFromPool(pool),
		logger:     log.New(),
		conversion: conversion.New(),
		processor: importer.NewFactory(
			map[string]importer.ProcessImport{"mapping-test": mappingTestProcessor{}},
		),
	}
	ctx := context.Background()
	sim := testhelpers.SeedRacingSim(t, "mapping-test")
	series := testhelpers.SeedSeries(t, sim.ID, "series")
	points := testhelpers.SeedPointSystem(t, "points")
	season := testhelpers.SeedSeason(t, series.ID, points.ID, "season")
	track := testhelpers.SeedTrack(t, "track")
	layout := testhelpers.SeedTrackLayout(t, track.ID, "layout")
	event := testhelpers.SeedEvent(t, season.ID, points.ID, layout.ID, "event", 1)
	race := testhelpers.SeedRace(t, event.ID, "race", "race", 1)
	grid := testhelpers.SeedRaceGrid(t, race.ID, "grid", "race", 1)
	emptyGrid := testhelpers.SeedRaceGrid(t, race.ID, "empty grid", "race", 2)
	emptyPreview, err := svc.GetPreprocessPreview(ctx, &importv1.GetPreprocessPreviewRequest{
		RaceGridId: uint32(emptyGrid.ID),
	})
	if err != nil || len(emptyPreview.GetRows()) != 0 ||
		len(emptyPreview.GetUnresolvedMappings()) != 0 {
		t.Fatalf("expected empty preview before import: %v, %v", emptyPreview, err)
	}
	orphan := testhelpers.SeedResultEntry(t, emptyGrid.ID, "Orphan", 1)
	if _, updateErr := repo.ResultEntries().Update(ctx, orphan.ID, &models.ResultEntrySetter{
		State: omit.From(conversion.ResultStateMappingError),
	}); updateErr != nil {
		t.Fatal(updateErr)
	}
	if _, previewErr := svc.GetPreprocessPreview(ctx, &importv1.GetPreprocessPreviewRequest{
		RaceGridId: uint32(emptyGrid.ID),
	}); previewErr == nil {
		t.Fatal("expected an error for marked rows with no import batch")
	}
	input := &importer.ParsedImportPayload{
		Session: importer.SessionInfo{Track: "unknown track"},
		Results: []*importer.ResultRow{
			{
				FinPos:    1,
				DriverID:  "driver-id",
				Name:      "Driver",
				CarID:     "car-id",
				Car:       "Car",
				CarNumber: "12",
			},
		},
	}
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	uploaded, err := svc.UploadResultsFile(ctx, &importv1.UploadResultsFileRequest{
		RaceGridId: uint32(
			grid.ID,
		),
		ImportFormat: commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		Payload:      payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.ProcessingState != conversion.EventProcessingStateMappingError {
		t.Fatalf("unexpected upload state: %v", uploaded)
	}
	preview, err := svc.GetPreprocessPreview(
		ctx,
		&importv1.GetPreprocessPreviewRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertMappingResponse(t, preview.Rows, preview.UnresolvedMappings, uint32(grid.ID), 3)
	retry, err := svc.ResolveMappings(
		ctx,
		&importv1.ResolveMappingsRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertMappingResponse(t, retry.Rows, retry.UnresolvedMappings, uint32(grid.ID), 3)
	if !proto.Equal(preview.Rows[0], retry.Rows[0]) {
		t.Fatalf("preview and retry rows differ: %v vs %v", preview.Rows[0], retry.Rows[0])
	}

	driver := testhelpers.SeedDriver(t, "Driver", "external")
	preview, err = svc.GetPreprocessPreview(
		ctx,
		&importv1.GetPreprocessPreviewRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertMappingResponse(t, preview.Rows, preview.UnresolvedMappings, uint32(grid.ID), 3)
	if got := preview.UnresolvedMappings[0].GetSeasonDriverNotActive(); got == nil ||
		got.DriverId != uint32(driver.ID) {
		t.Fatalf("preview did not use live mapping: %v", preview.UnresolvedMappings[0])
	}
	stored, err := repo.ResultEntries().LoadByID(ctx, int32(preview.Rows[0].Id))
	if err != nil || !stored.DriverID.IsNull() ||
		stored.State != conversion.ResultStateMappingError {
		t.Fatalf("preview changed stored row: %v, %v", stored, err)
	}

	manufacturer := testhelpers.SeedCarManufacturer(t, "manufacturer")
	carModel := testhelpers.SeedCarModelV2(t, "model", manufacturer.ID)
	variant := testhelpers.SeedCarModelVariant(t, "variant", carModel.ID)
	testhelpers.SeedSeasonDriver(
		t,
		"12",
		driver.ID,
		season.ID,
		variant.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	_, err = repo.Cars().SimulationCarAliases().Create(ctx, &models.SimulationCarAliasSetter{
		SimulationID: omit.From(
			sim.ID,
		),
		CarModelVariantID: omit.From(variant.ID),
		ExternalName:      omit.From("car-id"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = models.SeasonCarModelVariants.Insert(&models.SeasonCarModelVariantSetter{
		SeasonID: omit.From(
			season.ID,
		),
		CarModelVariantID: omit.From(variant.ID),
		Pos:               omit.From(int32(1)),
	}).One(ctx, pgbob.New(pool))
	if err != nil {
		t.Fatal(err)
	}
	preview, err = svc.GetPreprocessPreview(
		ctx,
		&importv1.GetPreprocessPreviewRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.UnresolvedMappings) != 1 ||
		preview.Rows[0].State != commonv1.ResultEntryState_RESULT_ENTRY_STATE_MAPPING_ERROR {
		t.Fatalf("expected live track error and persisted row marker: %v", preview)
	}
	retry, err = svc.ResolveMappings(
		ctx,
		&importv1.ResolveMappingsRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(retry.UnresolvedMappings) != 1 ||
		retry.Rows[0].State != commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL ||
		retry.Rows[0].DriverId != uint32(driver.ID) ||
		retry.Rows[0].CarModelVariantId != uint32(variant.ID) ||
		retry.UnresolvedMappings[0].GetRaceGridId() != uint32(grid.ID) {
		t.Fatalf("unexpected track-only retry: %v", retry)
	}
	stored, err = repo.ResultEntries().LoadByID(ctx, int32(retry.Rows[0].Id))
	if err != nil || stored.State != conversion.ResultStateNormal ||
		stored.DriverID.GetOr(0) != driver.ID ||
		stored.CarModelVariantID.GetOr(0) != variant.ID {
		t.Fatalf("retry was not persisted: %v, %v", stored, err)
	}
	_, err = repo.Tracks().
		SimulationTrackLayoutAliases().
		Create(ctx, &models.SimulationTrackLayoutAliasSetter{
			SimulationID:  omit.From(sim.ID),
			TrackLayoutID: omit.From(layout.ID),
			ExternalName:  omit.From("unknown track"),
		})
	if err != nil {
		t.Fatal(err)
	}
	editedDriver := testhelpers.SeedDriver(t, "Edited Driver", "external-edited")
	testhelpers.SeedSeasonDriver(
		t,
		"12",
		editedDriver.ID,
		season.ID,
		variant.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if _, updateErr := repo.ResultEntries().Update(
		ctx,
		int32(retry.Rows[0].Id),
		&models.ResultEntrySetter{
			RawDriverName: omitnull.From("Edited Driver"),
			State:         omit.From(conversion.ResultStateMappingError),
		},
	); updateErr != nil {
		t.Fatal(updateErr)
	}
	preview, err = svc.GetPreprocessPreview(
		ctx,
		&importv1.GetPreprocessPreviewRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil || len(preview.GetUnresolvedMappings()) != 0 ||
		len(preview.GetRows()) != 1 || preview.Rows[0].DriverId != uint32(driver.ID) {
		t.Fatalf("preview did not use the edited result entry: %v, %v", preview, err)
	}
	stored, err = repo.ResultEntries().LoadByID(ctx, int32(retry.Rows[0].Id))
	if err != nil || stored.State != conversion.ResultStateMappingError ||
		stored.DriverID.GetOr(0) != driver.ID {
		t.Fatalf("preview persisted the resolved mapping: %v, %v", stored, err)
	}
	retry, err = svc.ResolveMappings(
		ctx,
		&importv1.ResolveMappingsRequest{RaceGridId: uint32(grid.ID)},
	)
	if err != nil || len(retry.GetUnresolvedMappings()) != 0 {
		t.Fatalf("expected all mappings resolved: %v, %v", retry, err)
	}
	if retry.Rows[0].DriverId != uint32(editedDriver.ID) {
		t.Fatalf("ResolveMappings ignored the edited result entry: %v", retry.Rows[0])
	}
}

func assertMappingResponse(
	t *testing.T,
	rows []*commonv1.ResultEntry,
	mappings []*commonv1.UnresolvedMapping,
	gridID uint32,
	count int,
) {
	t.Helper()
	if len(rows) != 1 ||
		rows[0].State != commonv1.ResultEntryState_RESULT_ENTRY_STATE_MAPPING_ERROR ||
		len(mappings) != count {
		t.Fatalf("unexpected rows or mappings: %v, %v", rows, mappings)
	}
	for _, mapping := range mappings {
		if mapping.GetRaceGridId() != gridID || mapping.Reason == nil {
			t.Fatalf("missing diagnostic context: %v", mapping)
		}
		if mapping.ResultIndex != nil &&
			(mapping.GetResultIndex() != 0 || mapping.GetResultEntryId() != rows[0].Id) {
			t.Fatalf("bad row correlation: %v", mapping)
		}
	}
	track := mappings[len(mappings)-1]
	if track.GetSourceEntityNotFound().
		GetEntityType() !=
		commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_TRACK ||
		track.ResultEntryId != nil {
		t.Fatalf("missing track diagnostic: %v", track)
	}
}
