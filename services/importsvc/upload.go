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

	if !slices.ContainsFunc(
		func() []mytypes.RaceSimImportFormat {
			var f []mytypes.RaceSimImportFormat
			_ = json.Unmarshal(simulation.SupportedImportFormats.Val, &f)
			return f
		}(),
		func(f mytypes.RaceSimImportFormat) bool { return string(f.Format) == importFormat },
	) {
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
		if cleanupErr := s.cleanupExistingImportBatch(ctx, gridID); cleanupErr != nil {
			l.Error("failed to cleanup existing import batch", log.ErrorField(cleanupErr))
			trace.SpanFromContext(ctx).
				SetStatus(codes.Error, "failed to cleanup existing import batch")
			return cleanupErr
		}

		// TODO: support Upsert for ImportBatch
		var createErr error
		batch, createErr = s.repo.ImportBatches().Create(ctx, &models.ImportBatchSetter{
			RaceGridID:      omit.From(gridID),
			ImportFormat:    omit.From(mytypes.ImportFormat(importFormat)),
			Payload:         omit.From(req.Msg.GetPayload()),
			ProcessingState: omit.From(toState),
			MetadataJSON:    omit.From(mytypes.ImportBatchMeta{}),
			CreatedBy:       omit.From(execUser),
			UpdatedBy:       omit.From(execUser),
		})
		if createErr != nil {
			return createErr
		}

		// TODO move to own function
		// - process
		// - resolve
		// - remove data from previous steps
		// - store resultEntries
		input, inpErr := importProcessor.Process(ctx, importFormat, req.Msg.GetPayload())
		if inpErr != nil {
			return fmt.Errorf("process import payload: %w", inpErr)
		}

		resolver := importer.NewResolver(
			importer.NewRepositoryEntityResolver(ctx, s.repo, epi, simulation), epi)
		resolved, resolveErr := resolver.ResolveInput(input)
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
		_, createErr = s.repo.Events().Update(ctx, event.ID, &models.EventSetter{
			ProcessingState: omit.From(toState),
			UpdatedAt:       omit.From(time.Now()),
			UpdatedBy:       omit.From(execUser),
		})
		if createErr != nil {
			return createErr
		}

		// Write audit row.
		_, createErr = s.repo.EventProcessingAudit().Create(
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
		return createErr
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

func (s *service) cleanupExistingImportBatch(ctx context.Context, gridID int32) error {
	importBatch, err := s.repo.ImportBatches().LoadByRaceGridID(ctx, gridID)
	if err == nil && importBatch != nil {
		if err := s.repo.EventProcessingAudit().DeleteByImportBatchID(
			ctx, importBatch.ID); err != nil {
			return fmt.Errorf("delete existing event processing audits: %w", err)
		}
	}
	if err := s.repo.ResultEntries().DeleteByRaceGridID(ctx, gridID); err != nil {
		return fmt.Errorf("delete existing result entries: %w", err)
	}
	if err := s.repo.ImportBatches().DeleteByRaceGridID(ctx, gridID); err != nil {
		return fmt.Errorf("delete existing import batch: %w", err)
	}
	return nil
}

func (s *service) replaceResultEntriesForBatch(
	ctx context.Context,
	batch *models.ImportBatch,
	entries []*models.ResultEntry,
	execUser string,
) error {
	// TODO: change to RaceGridID when that is supported in ImportBatch
	existing, err := s.repo.ResultEntries().LoadByRaceGridID(ctx, batch.RaceGridID)
	if err != nil {
		return fmt.Errorf("load result entries for race grid %d: %w", batch.RaceGridID, err)
	}

	for _, item := range existing {
		if deleteErr := s.repo.ResultEntries().DeleteByID(ctx, item.ID); deleteErr != nil {
			return fmt.Errorf("delete result entry %d: %w", item.ID, deleteErr)
		}
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
