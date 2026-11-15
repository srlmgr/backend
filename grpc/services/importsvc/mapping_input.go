package importsvc

import (
	"context"
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
)

//nolint:whitespace // editor/linter issue
func (s *service) mappingInput(
	ctx context.Context,
	epi *processor.EventProcInfo,
	batch *models.ImportBatch,
	gridID int32,
) (*importer.ParsedImportPayload, *importer.Resolver, error) {
	importProcessor, simulation, err := s.resolveProcessorForEvent(ctx, epi)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve import processor: %w", err)
	}
	input, err := reprocessMappingInput(
		ctx, importProcessor, simulation, batch, epi, gridID,
	)
	if err != nil {
		return nil, nil, err
	}
	return input, importer.NewResolver(
		importer.NewRepositoryEntityResolver(ctx, s.repo, epi, simulation), epi,
	), nil
}

//nolint:whitespace // editor/linter issue
func reprocessMappingInput(
	ctx context.Context,
	importProcessor importer.ProcessImport,
	simulation *models.RacingSim,
	batch *models.ImportBatch,
	epi *processor.EventProcInfo,
	gridID int32,
) (*importer.ParsedImportPayload, error) {
	format := string(batch.ImportFormat)
	if !importer.SupportsFormat(importProcessor, format) {
		return nil, fmt.Errorf(
			"%w: simulation=%q format=%q",
			importer.ErrUnsupportedFormat, simulation.Name, format,
		)
	}
	payload, err := selectImportPayloadForProcessing(batch.Payload, batch.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("select import payload: %w", err)
	}
	ctx = importer.WithoutLapDataCollection(ctx)
	inputs, err := processImportPayload(ctx, importProcessor, format, payload)
	if err != nil {
		return nil, fmt.Errorf("process import payload: %w", err)
	}
	allowMultipleUploads, err := supportsMultipleMappingUploads(simulation, format)
	if err != nil {
		return nil, err
	}
	if allowMultipleUploads && len(inputs) == 1 {
		merged, mergeErr := mergeMappingUploads(ctx, importProcessor, format, batch, epi)
		if mergeErr != nil {
			return nil, mergeErr
		}
		if merged != nil {
			return merged, nil
		}
	}
	input, err := selectInputForGrid(epi, inputs, gridID)
	if err != nil {
		return nil, fmt.Errorf("select input for race grid %d: %w", gridID, err)
	}
	return input, nil
}

//nolint:whitespace // editor/linter issue
func supportsMultipleMappingUploads(
	simulation *models.RacingSim,
	format string,
) (bool, error) {
	formats, err := decodeRaceSimImportFormats(simulation.SupportedImportFormats.Val)
	if err != nil {
		return false, err
	}
	selected, ok := findRaceSimImportFormat(formats, format)
	if !ok {
		return false, fmt.Errorf(
			"%w: simulation=%q format=%q",
			importer.ErrUnsupportedFormat,
			simulation.Name,
			format,
		)
	}
	return selected.AllowMultipleUploads, nil
}

//nolint:whitespace // editor/linter issue
func mergeMappingUploads(
	ctx context.Context,
	importProcessor importer.ProcessImport,
	format string,
	batch *models.ImportBatch,
	epi *processor.EventProcInfo,
) (*importer.ParsedImportPayload, error) {
	merged, err := buildMergedInputFromZip(
		ctx, importProcessor, format, batch.Payload, batch.MetadataJSON,
		epi.Season.IsTeamBased,
	)
	if err != nil {
		return nil, fmt.Errorf("build merged import input: %w", err)
	}
	return merged, nil
}

func scopeMappingsToGrid(mappings []*commonv1.UnresolvedMapping, gridID uint32) {
	for _, mapping := range mappings {
		mapping.RaceGridId = &gridID
	}
}
