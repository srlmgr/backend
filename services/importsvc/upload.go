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

//nolint:whitespace,funlen // editor/linter issue
func (s *service) UploadResultsFile(
	ctx context.Context,
	req *connect.Request[importv1.UploadResultsFileRequest],
) (
	*connect.Response[importv1.UploadResultsFileResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UploadResultsFile")

	eventID := int32(req.Msg.GetEventId())
	raceID := int32(req.Msg.GetRaceId())

	if eventID == 0 || raceID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("event_id and race_id are required"))
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
	if race.EventID != eventID {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("race %d does not belong to event %d", raceID, eventID))
	}

	// Load event to get current processing state.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
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
		var createErr error
		batch, createErr = s.repo.ImportBatches().Create(ctx, &models.ImportBatchSetter{
			EventID:         omit.From(eventID),
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
		ImportBatchId:   uint32(batch.ID),
		ProcessingState: toState,
	}), nil
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
