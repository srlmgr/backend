package importsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/services/importsvc/importer"
)

//nolint:whitespace // editor/linter issue
func decodeRaceSimImportFormats(raw json.RawMessage) (
	[]mytypes.RaceSimImportFormat, error,
) {
	var formats []mytypes.RaceSimImportFormat
	if len(raw) == 0 {
		return formats, nil
	}
	if err := json.Unmarshal(raw, &formats); err != nil {
		return nil, fmt.Errorf("decode supported import formats: %w", err)
	}

	return formats, nil
}

//nolint:whitespace // editor/linter issue
func findRaceSimImportFormat(
	formats []mytypes.RaceSimImportFormat,
	importFormat string,
) (mytypes.RaceSimImportFormat, bool) {
	for i := range formats {
		if string(formats[i].Format) == importFormat {
			return formats[i], true
		}
	}

	return mytypes.RaceSimImportFormat{}, false
}

func importDataZipEntry(dataType importer.ImportData, multiUpload bool) string {
	if !multiUpload {
		return importer.ImportDataAll
	}

	switch dataType {
	case importer.ImportDataRace, importer.ImportDataQuali, importer.ImportDataAll:
		return dataType
	default:
		return importer.ImportDataAll
	}
}

//nolint:whitespace,nestif,funlen // editor/linter issue
func mergeImportBatchZipPayload(
	existingZip []byte,
	entryName string,
	payload []byte,
) ([]byte, error) {
	entryName = strings.TrimSpace(entryName)
	if entryName == "" {
		return nil, fmt.Errorf("zip entry name is required")
	}

	entries := map[string][]byte{}
	if len(existingZip) > 0 {
		existingReader, err := zip.NewReader(
			bytes.NewReader(existingZip), int64(len(existingZip)))
		if err != nil {
			return nil, fmt.Errorf("read existing import batch zip payload: %w", err)
		}

		for _, file := range existingReader.File {
			reader, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("open zip entry %q: %w", file.Name, err)
			}

			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read zip entry %q: %w", file.Name, readErr)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("close zip entry %q: %w", file.Name, closeErr)
			}

			entries[file.Name] = data
		}
	}

	entries[entryName] = payload

	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for name, data := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("create zip entry %q: %w", name, err)
		}
		if _, err := entry.Write(data); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write zip entry %q: %w", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip payload: %w", err)
	}

	return out.Bytes(), nil
}

//nolint:whitespace // editor/linter issue
func mergeImportBatchMetadata(
	meta mytypes.ImportBatchMeta,
	entryName string,
) mytypes.ImportBatchMeta {
	switch entryName {
	case importer.ImportDataRace:
		meta.Race = entryName
	case importer.ImportDataQuali:
		meta.Quali = entryName
	case importer.ImportDataAll:
		meta.All = entryName
	}

	return meta
}

//nolint:whitespace,funlen // editor/linter issue
func selectImportPayloadForProcessing(
	zipPayload []byte,
	meta mytypes.ImportBatchMeta,
) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipPayload), int64(len(zipPayload)))
	if err != nil {
		return nil, fmt.Errorf("read import batch zip payload: %w", err)
	}

	byName := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %q: %w", file.Name, err)
		}
		data, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read zip entry %q: %w", file.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close zip entry %q: %w", file.Name, closeErr)
		}
		byName[file.Name] = data
	}

	for _, name := range []string{meta.All, meta.Race, meta.Quali} {
		if name == "" {
			continue
		}
		if payload, ok := byName[name]; ok {
			return payload, nil
		}
	}

	for _, fallback := range []string{
		importer.ImportDataAll, importer.ImportDataRace, importer.ImportDataQuali,
	} {
		if payload, ok := byName[fallback]; ok {
			return payload, nil
		}
	}

	if len(reader.File) == 1 {
		return byName[reader.File[0].Name], nil
	}

	return nil, fmt.Errorf("zip payload does not contain a known import entry")
}

// extractNamedZipEntry returns the bytes for a named entry in the zip,
// or nil if not present.
func extractNamedZipEntry(zipPayload []byte, entryName string) ([]byte, error) {
	if entryName == "" {
		return nil, nil
	}
	reader, err := zip.NewReader(bytes.NewReader(zipPayload), int64(len(zipPayload)))
	if err != nil {
		return nil, fmt.Errorf("read zip payload: %w", err)
	}
	for _, file := range reader.File {
		if file.Name != entryName {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %q: %w", entryName, err)
		}
		data, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read zip entry %q: %w", entryName, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close zip entry %q: %w", entryName, closeErr)
		}
		return data, nil
	}
	return nil, nil
}

// mergeResultRows copies quali-specific fields (StartPos, QualiLapTime) from quali
// rows into matching race rows. Matching is by TeamID when isTeamBased is true,
// otherwise by DriverID.
//
//nolint:whitespace // editor/linter issue
func mergeResultRows(
	isTeamBased bool,
	raceRows, qualiRows []*importer.ResultRow,
) []*importer.ResultRow {
	index := make(map[string]*importer.ResultRow, len(qualiRows))
	for _, q := range qualiRows {
		key := q.DriverID
		if isTeamBased {
			key = q.TeamID
		}
		if key != "" {
			index[key] = q
		}
	}
	for _, r := range raceRows {
		key := r.DriverID
		if isTeamBased {
			key = r.TeamID
		}
		if q, ok := index[key]; ok {
			r.StartPos = q.StartPos
			r.QualiLapTime = q.QualiLapTime
		}
	}
	return raceRows
}

// buildMergedInputFromZip extracts the race and quali payloads from the merged zip,
// processes each with the import processor, and merges the results into a single
// ParsedImportPayload. If only one entry is present it is returned as-is.
// Returns nil when neither race nor quali entry exists in the zip.
//
//nolint:whitespace // editor/linter issue
func buildMergedInputFromZip(
	ctx context.Context,
	importProcessor importer.ProcessImport,
	importFormat importer.ImportFormat,
	zipPayload []byte,
	meta mytypes.ImportBatchMeta,
	isTeamBased bool,
) (*importer.ParsedImportPayload, error) {
	raceBytes, err := extractNamedZipEntry(zipPayload, meta.Race)
	if err != nil {
		return nil, fmt.Errorf("extract race zip entry: %w", err)
	}
	qualiBytes, err := extractNamedZipEntry(zipPayload, meta.Quali)
	if err != nil {
		return nil, fmt.Errorf("extract quali zip entry: %w", err)
	}

	if raceBytes == nil && qualiBytes == nil {
		return nil, nil
	}
	if raceBytes == nil {
		return importProcessor.Process(ctx, importFormat, qualiBytes)
	}
	if qualiBytes == nil {
		return importProcessor.Process(ctx, importFormat, raceBytes)
	}

	raceInput, err := importProcessor.Process(ctx, importFormat, raceBytes)
	if err != nil {
		return nil, fmt.Errorf("process race payload: %w", err)
	}
	qualiInput, err := importProcessor.Process(ctx, importFormat, qualiBytes)
	if err != nil {
		return nil, fmt.Errorf("process quali payload: %w", err)
	}

	raceInput.Results = mergeResultRows(isTeamBased, raceInput.Results, qualiInput.Results)
	return raceInput, nil
}
