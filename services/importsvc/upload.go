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

	raceID := int32(req.Msg.GetRaceId())

	if raceID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("race_id is required"))
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

	// Validate race belongs to event.
	race, err := s.repo.Races().LoadByID(ctx, raceID)
	if err != nil {
		l.Error("failed to load race", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	eventID := race.EventID

	// Load event to get current processing state.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
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

	importProcessor, simulation, err := s.resolveProcessorForEvent(ctx, event)
	if err != nil {
		l.Error("failed to resolve import processor", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error,
			"failed to resolve import processor")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	if !slices.Contains(simulation.SupportedImportFormats, importFormat) {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"import format %q is not supported by simulation %q",
				importFormat,
				simulation.Name,
			),
		)
	}

	if !processor.SupportsFormat(importProcessor, importFormat) {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"%w: simulation=%q format=%q",
				processor.ErrUnsupportedFormat,
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
		if cleanupErr := s.cleanupExistingImportBatch(ctx, raceID); cleanupErr != nil {
			l.Error("failed to cleanup existing import batch", log.ErrorField(cleanupErr))
			trace.SpanFromContext(ctx).
				SetStatus(codes.Error, "failed to cleanup existing import batch")
			return cleanupErr
		}

		// TODO: support Upsert for ImportBatch
		var createErr error
		batch, createErr = s.repo.ImportBatches().Create(ctx, &models.ImportBatchSetter{
			RaceID:          omit.From(raceID),
			ImportFormat:    omit.From(mytypes.ImportFormat(importFormat)),
			Payload:         omit.From(req.Msg.GetPayload()),
			ProcessingState: omit.From(toState),
			MetadataJSON:    omit.From(emptyJSON),
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

		resolver := processor.NewResolver(
			processor.NewRepositoryEntityResolver(ctx, s.repo, simulation))
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
		_, createErr = s.repo.Events().Update(ctx, eventID, &models.EventSetter{
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
				EventID:       omit.From(eventID),
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
		RaceId:          uint32(batch.RaceID),
		ProcessingState: toState,
	}), nil
}

func (s *service) cleanupExistingImportBatch(ctx context.Context, raceID int32) error {
	importBatch, err := s.repo.ImportBatches().LoadByRaceID(ctx, raceID)
	if err == nil && importBatch != nil {
		if err := s.repo.EventProcessingAudit().DeleteByImportBatchID(
			ctx, importBatch.ID); err != nil {
			return fmt.Errorf("delete existing event processing audits: %w", err)
		}
	}
	if err := s.repo.ResultEntries().DeleteByRaceID(ctx, raceID); err != nil {
		return fmt.Errorf("delete existing result entries: %w", err)
	}
	if err := s.repo.ImportBatches().DeleteByRaceID(ctx, raceID); err != nil {
		return fmt.Errorf("delete existing import batch: %w", err)
	}
	return nil
}

//nolint:whitespace // editor/linter issue
func (s *service) resolveProcessorForEvent(
	ctx context.Context,
	event *models.Event,
) (processor.ProcessImport, *models.RacingSim, error) {
	season, err := s.repo.Seasons().LoadByID(ctx, event.SeasonID)
	if err != nil {
		return nil, nil, err
	}

	series, err := s.repo.Series().LoadByID(ctx, season.SeriesID)
	if err != nil {
		return nil, nil, err
	}

	simulation, err := s.repo.RacingSims().LoadByID(ctx, series.SimulationID)
	if err != nil {
		return nil, nil, err
	}

	importProcessor, err := s.processor.Get(simulation.Name)
	if err != nil {
		return nil, nil, err
	}

	return importProcessor, simulation, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) replaceResultEntriesForBatch(
	ctx context.Context,
	batch *models.ImportBatch,
	entries []*models.ResultEntry,
	execUser string,
) error {
	existing, err := s.repo.ResultEntries().LoadByRaceID(ctx, batch.RaceID)
	if err != nil {
		return fmt.Errorf("load result entries for race %d: %w", batch.RaceID, err)
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

//nolint:whitespace // editor/linter issue
func buildResultEntryCreateSetter(
	batch *models.ImportBatch,
	entry *models.ResultEntry,
	execUser string,
) *models.ResultEntrySetter {
	setter := &models.ResultEntrySetter{
		RaceID:            omit.From(batch.RaceID),
		DriverName:        omit.From(entry.DriverName),
		FinishingPosition: omit.From(entry.FinishingPosition),
		CompletedLaps:     omit.From(entry.CompletedLaps),
		State:             omit.From(entry.State),
		CreatedBy:         omit.From(execUser),
		UpdatedBy:         omit.From(execUser),
	}

	if !entry.DriverID.IsNull() {
		setter.DriverID = omitnull.From(entry.DriverID.GetOr(0))
	}
	if !entry.CarModelID.IsNull() {
		setter.CarModelID = omitnull.From(entry.CarModelID.GetOr(0))
	}
	if !entry.CarName.IsNull() {
		setter.CarName = omitnull.From(entry.CarName.GetOr(""))
	}
	if !entry.FastestLapTimeMS.IsNull() {
		setter.FastestLapTimeMS = omitnull.From(entry.FastestLapTimeMS.GetOr(0))
	}
	if !entry.Incidents.IsNull() {
		setter.Incidents = omitnull.From(entry.Incidents.GetOr(0))
	}
	if !entry.SourceRowNumber.IsNull() {
		setter.SourceRowNumber = omitnull.From(entry.SourceRowNumber.GetOr(0))
	}
	if !entry.AdminNotes.IsNull() {
		setter.AdminNotes = omitnull.From(entry.AdminNotes.GetOr(""))
	}

	return setter
}
