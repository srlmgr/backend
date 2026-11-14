package importsvc

import (
	"context"
	"errors"
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
)

//nolint:whitespace // editor/linter issue
func (s *service) GetPreprocessPreview(
	ctx context.Context,
	req *importv1.GetPreprocessPreviewRequest,
) (*importv1.GetPreprocessPreviewResponse, error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetPreprocessPreview")

	gridID := int32(req.GetRaceGridId())
	if gridID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, "race_grid_id is required")
	}

	// Load result entries for the race.
	entries, err := s.repo.ResultEntries().LoadByRaceGridID(ctx, gridID)
	if err != nil {
		l.Error("failed to load result entries", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
		return nil, s.conversion.ToConnectError(err)
	}

	var unresolvedMappings []*commonv1.UnresolvedMapping
	batch, err := s.repo.ImportBatches().LoadByRaceGridID(ctx, gridID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) || hasMappingErrors(entries) {
			l.Error("failed to load import batch", log.ErrorField(err))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load import batch")
			return nil, s.conversion.ToConnectError(err)
		}
	} else {
		unresolvedMappings, err = s.previewMappings(ctx, gridID, batch, entries)
		if err != nil {
			l.Error("failed to preview mappings", log.ErrorField(err))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to preview mappings")
			return nil, s.conversion.ToConnectError(err)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "preprocess preview loaded")
	return &importv1.GetPreprocessPreviewResponse{
		Rows:               s.resultEntryRows(entries),
		UnresolvedMappings: unresolvedMappings,
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) previewMappings(
	ctx context.Context,
	gridID int32,
	batch *models.ImportBatch,
	entries []*models.ResultEntry,
) ([]*commonv1.UnresolvedMapping, error) {
	event, err := s.repo.Events().LoadByGridID(ctx, gridID)
	if err != nil {
		return nil, fmt.Errorf("load event: %w", err)
	}
	epi, err := processor.NewEventProcInfoCollector(s.repo).ForEvent(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("load event processing information: %w", err)
	}
	input, resolver, err := s.mappingInput(ctx, epi, batch, gridID)
	if err != nil {
		return nil, err
	}
	input, err = parsedInputWithResultEntries(input, entries, epi.Season.IsTeamBased)
	if err != nil {
		return nil, err
	}
	resolved, err := resolver.ResolveInput(input)
	if err != nil {
		return nil, fmt.Errorf("resolve preview mappings: %w", err)
	}
	for _, mapping := range resolved.Unmapped {
		if mapping.ResultIndex == nil {
			continue
		}
		index := int(mapping.GetResultIndex())
		if index < 0 || index >= len(entries) {
			return nil, fmt.Errorf("unresolved mapping has invalid result index %d", index)
		}
		entryID := uint32(entries[index].ID)
		mapping.ResultEntryId = &entryID
	}
	scopeMappingsToGrid(resolved.Unmapped, uint32(gridID))
	return resolved.Unmapped, nil
}

func hasMappingErrors(entries []*models.ResultEntry) bool {
	for _, entry := range entries {
		if entry != nil && entry.State == conversion.ResultStateMappingError {
			return true
		}
	}
	return false
}

//nolint:whitespace // editor/linter issue
func (s *service) resultEntryRows(
	entries []*models.ResultEntry,
) []*commonv1.ResultEntry {
	rows := make([]*commonv1.ResultEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, s.conversion.ResultEntryToResultEntry(entry))
	}
	return rows
}
