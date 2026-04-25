//nolint:lll,dupl,funlen // test
package importsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/services/importsvc/importer"
)

func TestMergeImportBatchZipPayloadCreatesZip(t *testing.T) {
	t.Parallel()

	zipPayload, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, []byte("race-data"))
	if err != nil {
		t.Fatalf("mergeImportBatchZipPayload returned unexpected error: %v", err)
	}

	entries := readZipEntries(t, zipPayload)
	expected := map[string]string{importer.ImportDataRace: "race-data"}
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("unexpected zip entries: got %#v want %#v", entries, expected)
	}
}

func TestMergeImportBatchZipPayloadReplacesEntryAndKeepsOthers(t *testing.T) {
	t.Parallel()

	firstZip, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, []byte("race-v1"))
	if err != nil {
		t.Fatalf("first merge failed: %v", err)
	}
	withQuali, err := mergeImportBatchZipPayload(
		firstZip,
		importer.ImportDataQuali,
		[]byte("quali-v1"),
	)
	if err != nil {
		t.Fatalf("second merge failed: %v", err)
	}
	updated, err := mergeImportBatchZipPayload(
		withQuali,
		importer.ImportDataRace,
		[]byte("race-v2"),
	)
	if err != nil {
		t.Fatalf("third merge failed: %v", err)
	}

	entries := readZipEntries(t, updated)
	expected := map[string]string{
		importer.ImportDataRace:  "race-v2",
		importer.ImportDataQuali: "quali-v1",
	}
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("unexpected zip entries: got %#v want %#v", entries, expected)
	}
}

func TestMergeImportBatchZipPayloadInvalidExistingZip(t *testing.T) {
	t.Parallel()

	_, err := mergeImportBatchZipPayload(
		[]byte("not-a-zip"),
		importer.ImportDataAll,
		[]byte("payload"),
	)
	if err == nil {
		t.Fatal("expected error for invalid zip payload")
	}
}

func TestMergeImportBatchMetadata(t *testing.T) {
	t.Parallel()

	meta := mytypes.ImportBatchMeta{}
	meta = mergeImportBatchMetadata(meta, importer.ImportDataRace)
	meta = mergeImportBatchMetadata(meta, importer.ImportDataQuali)
	meta = mergeImportBatchMetadata(meta, importer.ImportDataAll)

	if meta.Race != importer.ImportDataRace {
		t.Fatalf("unexpected race metadata entry: %q", meta.Race)
	}
	if meta.Quali != importer.ImportDataQuali {
		t.Fatalf("unexpected quali metadata entry: %q", meta.Quali)
	}
	if meta.All != importer.ImportDataAll {
		t.Fatalf("unexpected all metadata entry: %q", meta.All)
	}
}

func TestImportDataZipEntry(t *testing.T) {
	t.Parallel()

	if got := importDataZipEntry(importer.ImportDataQuali, false); got != importer.ImportDataAll {
		t.Fatalf("unexpected single-upload entry: got %q want %q", got, importer.ImportDataAll)
	}
	if got := importDataZipEntry(importer.ImportDataQuali, true); got != importer.ImportDataQuali {
		t.Fatalf("unexpected multi-upload entry: got %q want %q", got, importer.ImportDataQuali)
	}
	if got := importDataZipEntry("", true); got != importer.ImportDataAll {
		t.Fatalf("unexpected fallback entry: got %q want %q", got, importer.ImportDataAll)
	}
}

func TestDecodeRaceSimImportFormats(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal([]mytypes.RaceSimImportFormat{
		{Format: mytypes.ImportFormat("csv"), AllowMultipleUploads: true},
	})
	if err != nil {
		t.Fatalf("marshal test json failed: %v", err)
	}

	formats, err := decodeRaceSimImportFormats(raw)
	if err != nil {
		t.Fatalf("decodeRaceSimImportFormats returned unexpected error: %v", err)
	}
	if len(formats) != 1 {
		t.Fatalf("unexpected format count: got %d want %d", len(formats), 1)
	}
	if !formats[0].AllowMultipleUploads {
		t.Fatal("expected multi-upload flag to be enabled")
	}
}

func TestSelectImportPayloadForProcessing(t *testing.T) {
	t.Parallel()

	zipPayload, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, []byte("race-data"))
	if err != nil {
		t.Fatalf("prepare race zip payload failed: %v", err)
	}
	zipPayload, err = mergeImportBatchZipPayload(
		zipPayload,
		importer.ImportDataQuali,
		[]byte("quali-data"),
	)
	if err != nil {
		t.Fatalf("prepare quali zip payload failed: %v", err)
	}

	payload, err := selectImportPayloadForProcessing(
		zipPayload,
		mytypes.ImportBatchMeta{Quali: importer.ImportDataQuali},
	)
	if err != nil {
		t.Fatalf("selectImportPayloadForProcessing returned unexpected error: %v", err)
	}
	if string(payload) != "quali-data" {
		t.Fatalf("unexpected selected payload: got %q want %q", string(payload), "quali-data")
	}
}

func TestSelectImportPayloadForProcessingFallbackAll(t *testing.T) {
	t.Parallel()

	zipPayload, err := mergeImportBatchZipPayload(nil, importer.ImportDataAll, []byte("all-data"))
	if err != nil {
		t.Fatalf("prepare all zip payload failed: %v", err)
	}

	payload, err := selectImportPayloadForProcessing(zipPayload, mytypes.ImportBatchMeta{})
	if err != nil {
		t.Fatalf("selectImportPayloadForProcessing returned unexpected error: %v", err)
	}
	if string(payload) != "all-data" {
		t.Fatalf("unexpected selected payload: got %q want %q", string(payload), "all-data")
	}
}

func readZipEntries(t *testing.T, payload []byte) map[string]string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("failed to read zip payload: %v", err)
	}

	entries := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("failed to open zip entry %q: %v", file.Name, err)
		}
		var data bytes.Buffer
		if _, err := data.ReadFrom(rc); err != nil {
			_ = rc.Close()
			t.Fatalf("failed to read zip entry %q: %v", file.Name, err)
		}
		if err := rc.Close(); err != nil {
			t.Fatalf("failed to close zip entry %q: %v", file.Name, err)
		}

		entries[file.Name] = data.String()
	}

	return entries
}

// stubProcessor is a minimal ProcessImport that echoes back payloads as ResultRows.
// The payload is expected to be a JSON-encoded []*importer.ResultRow.
type stubProcessor struct {
	dataType importer.ImportData
}

//nolint:whitespace,errcheck // editor/linter issue
func (s *stubProcessor) Process(
	_ context.Context,
	_ importer.ImportFormat,
	payload any,
) (*importer.ParsedImportPayload, error) {
	var rows []*importer.ResultRow
	if err := json.Unmarshal(payload.([]byte), &rows); err != nil {
		return nil, err
	}
	return &importer.ParsedImportPayload{DataType: s.dataType, Results: rows}, nil
}

func TestExtractNamedZipEntryFound(t *testing.T) {
	t.Parallel()

	zip1, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, []byte("race-bytes"))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data, err := extractNamedZipEntry(zip1, importer.ImportDataRace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "race-bytes" {
		t.Fatalf("got %q, want %q", string(data), "race-bytes")
	}
}

func TestExtractNamedZipEntryNotFound(t *testing.T) {
	t.Parallel()

	zip1, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, []byte("race-bytes"))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data, err := extractNamedZipEntry(zip1, importer.ImportDataQuali)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil for missing entry, got %q", string(data))
	}
}

func TestExtractNamedZipEntryEmptyName(t *testing.T) {
	t.Parallel()

	data, err := extractNamedZipEntry([]byte("irrelevant"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Fatal("expected nil for empty entry name")
	}
}

func TestMergeResultRowsByDriverID(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{
		{DriverID: "driver1", FinPos: 1, StartPos: 0, QualiLapTime: 0},
		{DriverID: "driver2", FinPos: 2, StartPos: 0, QualiLapTime: 0},
	}
	qualiRows := []*importer.ResultRow{
		{DriverID: "driver2", StartPos: 1, QualiLapTime: 90000},
		{DriverID: "driver1", StartPos: 2, QualiLapTime: 91000},
	}

	result := mergeResultRows(false, raceRows, qualiRows)

	if result[0].StartPos != 2 || result[0].QualiLapTime != 91000 {
		t.Errorf("driver1: got StartPos=%d QualiLapTime=%d, want 2 91000",
			result[0].StartPos, result[0].QualiLapTime)
	}
	if result[1].StartPos != 1 || result[1].QualiLapTime != 90000 {
		t.Errorf("driver2: got StartPos=%d QualiLapTime=%d, want 1 90000",
			result[1].StartPos, result[1].QualiLapTime)
	}
}

func TestMergeResultRowsByTeamID(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{
		{TeamID: "teamA", FinPos: 1, StartPos: 0, QualiLapTime: 0},
		{TeamID: "teamB", FinPos: 2, StartPos: 0, QualiLapTime: 0},
	}
	qualiRows := []*importer.ResultRow{
		{TeamID: "teamB", StartPos: 1, QualiLapTime: 88000},
		{TeamID: "teamA", StartPos: 3, QualiLapTime: 87500},
	}

	result := mergeResultRows(true, raceRows, qualiRows)

	if result[0].StartPos != 3 || result[0].QualiLapTime != 87500 {
		t.Errorf("teamA: got StartPos=%d QualiLapTime=%d, want 3 87500",
			result[0].StartPos, result[0].QualiLapTime)
	}
	if result[1].StartPos != 1 || result[1].QualiLapTime != 88000 {
		t.Errorf("teamB: got StartPos=%d QualiLapTime=%d, want 1 88000",
			result[1].StartPos, result[1].QualiLapTime)
	}
}

func TestMergeResultRowsNoMatchingQuali(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{
		{DriverID: "driver1", FinPos: 1, StartPos: 5, QualiLapTime: 99999},
	}
	qualiRows := []*importer.ResultRow{
		{DriverID: "driver99", StartPos: 1, QualiLapTime: 80000},
	}

	result := mergeResultRows(false, raceRows, qualiRows)

	if result[0].StartPos != 5 || result[0].QualiLapTime != 99999 {
		t.Errorf("unmatched row should be unchanged: got StartPos=%d QualiLapTime=%d",
			result[0].StartPos, result[0].QualiLapTime)
	}
}

func TestBuildMergedInputFromZipBothEntries(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{
		{DriverID: "d1", FinPos: 1},
		{DriverID: "d2", FinPos: 2},
	}
	qualiRows := []*importer.ResultRow{
		{DriverID: "d1", StartPos: 2, QualiLapTime: 91000},
		{DriverID: "d2", StartPos: 1, QualiLapTime: 90000},
	}

	raceJSON, _ := json.Marshal(raceRows)
	qualiJSON, _ := json.Marshal(qualiRows)

	zipPayload, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, raceJSON)
	if err != nil {
		t.Fatalf("build race zip: %v", err)
	}
	zipPayload, err = mergeImportBatchZipPayload(zipPayload, importer.ImportDataQuali, qualiJSON)
	if err != nil {
		t.Fatalf("build quali zip: %v", err)
	}

	meta := mytypes.ImportBatchMeta{Race: importer.ImportDataRace, Quali: importer.ImportDataQuali}
	proc := &stubProcessor{dataType: importer.ImportDataRace}

	result, err := buildMergedInputFromZip(
		context.Background(),
		proc,
		"csv",
		zipPayload,
		meta,
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected merged result, got nil")
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}

	for _, r := range result.Results {
		switch r.DriverID {
		case "d1":
			if r.StartPos != 2 || r.QualiLapTime != 91000 {
				t.Errorf(
					"d1: got StartPos=%d QualiLapTime=%d, want 2 91000",
					r.StartPos,
					r.QualiLapTime,
				)
			}
		case "d2":
			if r.StartPos != 1 || r.QualiLapTime != 90000 {
				t.Errorf(
					"d2: got StartPos=%d QualiLapTime=%d, want 1 90000",
					r.StartPos,
					r.QualiLapTime,
				)
			}
		default:
			t.Errorf("unexpected driverID %q", r.DriverID)
		}
	}
}

func TestBuildMergedInputFromZipOnlyRace(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{{DriverID: "d1", FinPos: 1}}
	raceJSON, _ := json.Marshal(raceRows)

	zipPayload, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, raceJSON)
	if err != nil {
		t.Fatalf("build race zip: %v", err)
	}

	meta := mytypes.ImportBatchMeta{Race: importer.ImportDataRace}
	proc := &stubProcessor{dataType: importer.ImportDataRace}

	result, err := buildMergedInputFromZip(
		context.Background(),
		proc,
		"csv",
		zipPayload,
		meta,
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %v", result)
	}
}

func TestBuildMergedInputFromZipNoEntries(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{{DriverID: "d1", FinPos: 1}}
	raceJSON, _ := json.Marshal(raceRows)

	zipPayload, err := mergeImportBatchZipPayload(nil, importer.ImportDataRace, raceJSON)
	if err != nil {
		t.Fatalf("build race zip: %v", err)
	}

	// meta references entries not present in the zip
	meta := mytypes.ImportBatchMeta{}
	proc := &stubProcessor{dataType: importer.ImportDataRace}

	result, err := buildMergedInputFromZip(
		context.Background(),
		proc,
		"csv",
		zipPayload,
		meta,
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil when no entries in meta, got %v", result)
	}
}

func TestBuildMergedInputFromZipTeamBased(t *testing.T) {
	t.Parallel()

	raceRows := []*importer.ResultRow{
		{TeamID: "tA", FinPos: 1},
	}
	qualiRows := []*importer.ResultRow{
		{TeamID: "tA", StartPos: 3, QualiLapTime: 85000},
	}

	raceJSON, _ := json.Marshal(raceRows)
	qualiJSON, _ := json.Marshal(qualiRows)

	zipPayload, _ := mergeImportBatchZipPayload(nil, importer.ImportDataRace, raceJSON)
	zipPayload, _ = mergeImportBatchZipPayload(zipPayload, importer.ImportDataQuali, qualiJSON)

	meta := mytypes.ImportBatchMeta{Race: importer.ImportDataRace, Quali: importer.ImportDataQuali}
	proc := &stubProcessor{dataType: importer.ImportDataRace}

	result, err := buildMergedInputFromZip(
		context.Background(),
		proc,
		"csv",
		zipPayload,
		meta,
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || len(result.Results) == 0 {
		t.Fatal("expected results")
	}
	if result.Results[0].StartPos != 3 || result.Results[0].QualiLapTime != 85000 {
		t.Errorf("tA: got StartPos=%d QualiLapTime=%d, want 3 85000",
			result.Results[0].StartPos, result.Results[0].QualiLapTime)
	}
}
