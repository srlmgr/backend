//nolint:lll,whitespace // test fixtures
package importsvc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stephenafamo/bob/types"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
)

func TestReprocessMappingInputMergesMultipleUploads(t *testing.T) {
	race := &importer.ParsedImportPayload{
		Session: importer.SessionInfo{Track: "race track"},
		Results: []*importer.ResultRow{{DriverID: "a", Name: "Driver", Car: "Car"}},
	}
	quali := &importer.ParsedImportPayload{
		Session: importer.SessionInfo{Track: "quali track"},
		Results: []*importer.ResultRow{{DriverID: "a", StartPos: 2, QualiLapTime: 12345}},
	}
	raceJSON := marshalMappingInput(t, race)
	qualiJSON := marshalMappingInput(t, quali)
	payload, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, raceJSON)
	if err != nil {
		t.Fatal(err)
	}
	payload, err = mergeImportBatchZipPayload(payload, importer.ImportDataQuali, qualiJSON)
	if err != nil {
		t.Fatal(err)
	}
	batch := &models.ImportBatch{
		ImportFormat: "json",
		Payload:      payload,
		MetadataJSON: mytypes.ImportBatchMeta{
			Race:  importer.ImportDataRace,
			Quali: importer.ImportDataQuali,
		},
	}
	simulation := &models.RacingSim{
		Name: "test",
	}
	simulation.SupportedImportFormats = types.JSON[json.RawMessage]{
		Val: marshalMappingInput(
			t,
			[]mytypes.RaceSimImportFormat{{Format: "json", AllowMultipleUploads: true}},
		),
	}
	result, err := reprocessMappingInput(context.Background(), mappingTestProcessor{}, simulation,
		batch, &processor.EventProcInfo{Season: &models.Season{}}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.Track != "race track" || len(result.Results) != 1 ||
		result.Results[0].Name != "Driver" || result.Results[0].StartPos != 2 ||
		result.Results[0].QualiLapTime != 12345 {
		t.Fatalf("unexpected merged input: %+v", result)
	}
}

func marshalMappingInput(t *testing.T, input any) []byte {
	t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

type multiMappingTestProcessor struct {
	mappingTestProcessor
}

func (multiMappingTestProcessor) ProcessMultiRace(
	_ context.Context, _ importer.ImportFormat, _ any,
) ([]*importer.ParsedImportPayload, error) {
	return []*importer.ParsedImportPayload{
		{RaceSequenceNo: 1, Session: importer.SessionInfo{Track: "first"}},
		{RaceSequenceNo: 2, Session: importer.SessionInfo{Track: "second"}},
	}, nil
}

func TestReprocessMappingInputSelectsHeat(t *testing.T) {
	payload, err := mergeImportBatchZipPayload(nil, importer.ImportDataAll, []byte("{}"))
	if err != nil {
		t.Fatal(err)
	}
	batch := &models.ImportBatch{ImportFormat: "json", Payload: payload}
	simulation := &models.RacingSim{
		Name: "test", SupportedImportFormats: types.JSON[json.RawMessage]{
			Val: json.RawMessage(`[{"format":"json"}]`),
		},
	}
	epi := &processor.EventProcInfo{
		Season: &models.Season{},
		Races:  []*models.Race{{ID: 1, SequenceNo: 1}, {ID: 2, SequenceNo: 2}},
		Grids:  []*models.RaceGrid{{ID: 10, RaceID: 1}, {ID: 20, RaceID: 2}},
	}
	result, err := reprocessMappingInput(context.Background(), multiMappingTestProcessor{},
		simulation, batch, epi, 20)
	if err != nil || result.Session.Track != "second" {
		t.Fatalf("unexpected selected heat: %v, %v", result, err)
	}
}

func TestReprocessMappingInputRejectsCorruptBatch(t *testing.T) {
	batch := &models.ImportBatch{ImportFormat: "json", Payload: []byte("not a zip")}
	_, err := reprocessMappingInput(context.Background(), mappingTestProcessor{},
		&models.RacingSim{Name: "test"}, batch,
		&processor.EventProcInfo{Season: &models.Season{}}, 7)
	if err == nil {
		t.Fatal("expected error for invalid stored batch")
	}
}
