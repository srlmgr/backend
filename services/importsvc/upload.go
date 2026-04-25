package importsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/importer"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

//nolint:whitespace,funlen,gocyclo // editor/linter issue
func (s *service) UploadResultsFile(
	ctx context.Context,
	req *connect.Request[importv1.UploadResultsFileRequest],
) (
	*connect.Response[importv1.UploadResultsFileResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UploadResultsFile")

	gridID := int32(req.Msg.GetRaceGridId())

	if gridID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("race_grid_id is required"))
	}
	if len(req.Msg.GetPayload()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("payload is required"))
	}

	formats, err := conversion.ImportFormatsFromProto(
		[]commonv1.ImportFormat{req.Msg.GetImportFormat()},
	)
	if err != nil || len(formats) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid import_format: %w", err))
	}
	importFormat := formats[0]

	// Load event to get current processing state.
	event, err := s.repo.Events().LoadByGridID(ctx, gridID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	if !slices.Contains([]string{
		conversion.EventProcessingStateDraft,
		conversion.EventProcessingStatePreprocessed,
		conversion.EventProcessingStateMappingError,
	}, event.ProcessingState) {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf(
				"cannot upload results file in current processing state: %s",
				event.ProcessingState,
			),
		)
	}
	ep := processor.NewEventProcInfoCollector(s.repo)
	epi, err := ep.ForEvent(ctx, event.ID)
	if err != nil {
		return nil, err
	}
	importProcessor, simulation, err := s.resolveProcessorForEvent(ctx, epi)
	if err != nil {
		l.Error("failed to resolve import processor", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error,
			"failed to resolve import processor")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	//nolint:lll // readability
	supportedFormats, err := decodeRaceSimImportFormats(simulation.SupportedImportFormats.Val)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	selectedFormat, ok := findRaceSimImportFormat(supportedFormats, importFormat)
	if !ok {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"import format %q is not supported by simulation %q",
				importFormat,
				simulation.Name,
			),
		)
	}

	if !importer.SupportsFormat(importProcessor, importFormat) {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"%w: simulation=%q format=%q",
				importer.ErrUnsupportedFormat,
				simulation.Name,
				importFormat,
			),
		)
	}

	fromState := event.ProcessingState
	toState := conversion.EventProcessingStateRawImported
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}

	var batch *models.ImportBatch
	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		// TODO move to own function
		// - process
		// - resolve
		// - remove data from previous steps
		// - store resultEntries
		input, inpErr := importProcessor.Process(ctx, importFormat, req.Msg.GetPayload())
		if inpErr != nil {
			return fmt.Errorf("process import payload: %w", inpErr)
		}

		entryName := importDataZipEntry(
			input.DataType,
			selectedFormat.AllowMultipleUploads,
		)

		existingBatch, loadErr := s.repo.ImportBatches().LoadByRaceGridID(ctx, gridID)
		if loadErr != nil && !errors.Is(loadErr, rootrepo.ErrNotFound) {
			return fmt.Errorf("load import batch for race grid %d: %w", gridID, loadErr)
		}

		existingPayload := []byte(nil)
		existingMeta := mytypes.ImportBatchMeta{}
		if existingBatch != nil {
			existingPayload = existingBatch.Payload
			existingMeta = existingBatch.MetadataJSON
		}

		zipPayload, zipErr := mergeImportBatchZipPayload(
			existingPayload,
			entryName,
			req.Msg.GetPayload(),
		)
		if zipErr != nil {
			return fmt.Errorf("build import batch zip payload: %w", zipErr)
		}
		meta := mergeImportBatchMetadata(existingMeta, entryName)

		set := &models.ImportBatchSetter{
			RaceGridID:      omit.From(gridID),
			ImportFormat:    omit.From(mytypes.ImportFormat(importFormat)),
			Payload:         omit.From(zipPayload),
			ProcessingState: omit.From(toState),
			MetadataJSON:    omit.From(meta),
			UpdatedBy:       omit.From(execUser),
		}
		var writeErr error
		if existingBatch == nil {
			set.CreatedBy = omit.From(execUser)
			batch, writeErr = s.repo.ImportBatches().Create(ctx, set)
		} else {
			batch, writeErr = s.repo.ImportBatches().Update(ctx, existingBatch.ID, set)
		}
		if writeErr != nil {
			return writeErr
		}

		resolver := importer.NewResolver(
			importer.NewRepositoryEntityResolver(ctx, s.repo, epi, simulation), epi)

		finalInput := input
		if selectedFormat.AllowMultipleUploads {
			merged, mergeErr := buildMergedInputFromZip(
				ctx,
				importProcessor,
				importFormat,
				zipPayload,
				meta,
				epi.Season.IsTeamBased,
			)
			if mergeErr != nil {
				return fmt.Errorf("build merged import input: %w", mergeErr)
			}
			if merged != nil {
				finalInput = merged
			}
		}

		resolved, resolveErr := resolver.ResolveInput(finalInput)
		if resolveErr != nil {
			return fmt.Errorf("resolve import payload: %w", resolveErr)
		}

		if persistErr := s.replaceResultEntriesForBatch(
			ctx,
			batch,
			resolved.Entries,
			execUser,
		); persistErr != nil {
			return persistErr
		}
		if len(resolved.Unmapped) > 0 {
			toState = conversion.EventProcessingStateMappingError
		} else {
			toState = conversion.EventProcessingStatePreprocessed
		}

		// Advance event processing state.
		_, writeErr = s.repo.Events().Update(ctx, event.ID, &models.EventSetter{
			ProcessingState: omit.From(toState),
			UpdatedAt:       omit.From(time.Now()),
			UpdatedBy:       omit.From(execUser),
		})
		if writeErr != nil {
			return writeErr
		}

		// Write audit row.
		_, writeErr = s.repo.EventProcessingAudit().Create(
			ctx,
			&models.EventProcessingAuditSetter{
				EventID:       omit.From(event.ID),
				ImportBatchID: omitnull.From(batch.ID),
				FromState:     omitnull.From(fromState),
				ToState:       omit.From(toState),
				Action:        omit.From("upload_results_file"),
				PayloadJSON:   omit.From(emptyJSON),
				CreatedBy:     omit.From(execUser),
				UpdatedBy:     omit.From(execUser),
			})
		return writeErr
	}); txErr != nil {
		l.Error("failed to upload results file", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to upload results file")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "results file uploaded")
	return connect.NewResponse(&importv1.UploadResultsFileResponse{
		RaceGridId:      uint32(batch.RaceGridID),
		ProcessingState: toState,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) replaceResultEntriesForBatch(
	ctx context.Context,
	batch *models.ImportBatch,
	entries []*models.ResultEntry,
	execUser string,
) error {
	if err := s.repo.ResultEntries().DeleteByRaceGridID(
		ctx, batch.RaceGridID); err != nil {
		return fmt.Errorf("delete result entries for race grid %d: %w",
			batch.RaceGridID, err)
	}
	setters := make([]*models.ResultEntrySetter, len(entries))
	for i, entry := range entries {
		setters[i] = buildResultEntryCreateSetter(batch, entry, execUser)
	}
	if _, createErr := s.repo.ResultEntries().CreateMany(ctx, setters); createErr != nil {
		return fmt.Errorf("create result entries: %w", createErr)
	}

	return nil
}

//nolint:whitespace,funlen,gocyclo // editor/linter issue, many optional fields to set
func buildResultEntryCreateSetter(
	batch *models.ImportBatch,
	entry *models.ResultEntry,
	execUser string,
) *models.ResultEntrySetter {
	setter := &models.ResultEntrySetter{
		RaceGridID:     omit.From(batch.RaceGridID),
		FinishPosition: omit.From(entry.FinishPosition),
		LapsCompleted:  omit.From(entry.LapsCompleted),
		State:          omit.From(entry.State),
		CreatedBy:      omit.From(execUser),
		UpdatedBy:      omit.From(execUser),
	}
	if !entry.StartPosition.IsNull() {
		setter.StartPosition = omitnull.From(entry.StartPosition.GetOr(0))
	}
	if !entry.DriverID.IsNull() {
		setter.DriverID = omitnull.From(entry.DriverID.GetOr(0))
	}
	if !entry.RawDriverName.IsNull() {
		setter.RawDriverName = omitnull.From(entry.RawDriverName.GetOr(""))
	}
	if !entry.TeamID.IsNull() {
		setter.TeamID = omitnull.From(entry.TeamID.MustGet())
	}
	if !entry.TeamDrivers.IsNull() {
		setter.TeamDrivers = omitnull.From(entry.TeamDrivers.MustGet())
	}

	if !entry.RawTeamName.IsNull() {
		setter.RawTeamName = omitnull.From(entry.RawTeamName.GetOr(""))
	}
	if !entry.CarModelID.IsNull() {
		setter.CarModelID = omitnull.From(entry.CarModelID.GetOr(0))
	}
	if !entry.RawCarName.IsNull() {
		setter.RawCarName = omitnull.From(entry.RawCarName.GetOr(""))
	}
	if !entry.CarClassID.IsNull() {
		setter.CarClassID = omitnull.From(entry.CarClassID.GetOr(0))
	}
	if !entry.CarNumber.IsNull() {
		setter.CarNumber = omitnull.From(entry.CarNumber.GetOr(""))
	}
	if !entry.FastestLapTimeMS.IsNull() {
		setter.FastestLapTimeMS = omitnull.From(entry.FastestLapTimeMS.GetOr(0))
	}
	if !entry.QualiLapTimeMS.IsNull() {
		setter.QualiLapTimeMS = omitnull.From(entry.QualiLapTimeMS.GetOr(0))
	}
	if !entry.TotalTimeMS.IsNull() {
		setter.TotalTimeMS = omitnull.From(entry.TotalTimeMS.GetOr(0))
	}
	if !entry.Incidents.IsNull() {
		setter.Incidents = omitnull.From(entry.Incidents.GetOr(0))
	}
	if !entry.AdminNotes.IsNull() {
		setter.AdminNotes = omitnull.From(entry.AdminNotes.GetOr(""))
	}

	return setter
}
