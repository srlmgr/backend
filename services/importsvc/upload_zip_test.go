//nolint:lll // test
package importsvc

import (
	"archive/zip"
	"bytes"
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
		{Format: mytypes.ImportFormat("csv"), AllowMultipleUploads: "true"},
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
	if !isMultiUploadEnabled(formats[0].AllowMultipleUploads) {
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
