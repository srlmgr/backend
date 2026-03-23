package importsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) FinalizeEventProcessing(
	ctx context.Context,
	req *connect.Request[importv1.FinalizeEventProcessingRequest],
) (
	*connect.Response[importv1.FinalizeEventProcessingResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("FinalizeEventProcessing")

	eventID := int32(req.Msg.GetEventId())

	// Load event and verify it can be finalized.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	if event.ProcessingState == conversion.EventProcessingStateFinalized {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("event is already finalized"))
	}
	if event.ProcessingState == conversion.EventProcessingStateDraft ||
		event.ProcessingState == conversion.EventProcessingStateRawImported ||
		event.ProcessingState == conversion.EventProcessingStatePreprocessed ||
		event.ProcessingState == conversion.EventProcessingStateDriverEntriesComputed {
		//nolint:lll // readability
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf(
				"event processing state %q is not ready for finalization: expected team_entries_computed",
				event.ProcessingState,
			),
		)
	}

	fromState := event.ProcessingState
	toState := conversion.EventProcessingStateFinalized
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}
	finalizedAt := time.Now()

	// Resolve the latest import batch across all races for the event.
	races, err := s.repo.Races().LoadByEventID(ctx, eventID)
	if err != nil {
		l.Error("failed to load races", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load races")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		// Finalize the import batch for each race.
		for _, race := range races {
			batch, loadErr := s.repo.ImportBatches().LoadByRaceID(ctx, race.ID)
			if loadErr != nil {
				// If no batch exists for this race, skip it.
				continue
			}
			_, updateErr := s.repo.ImportBatches().Update(
				ctx,
				batch.ID,
				&models.ImportBatchSetter{
					ProcessingState: omit.From(toState),
					UpdatedAt:       omit.From(time.Now()),
					UpdatedBy:       omit.From(execUser),
				})
			if updateErr != nil {
				return updateErr
			}
		}

		// Finalize the event.
		_, updateErr := s.repo.Events().Update(ctx, eventID, &models.EventSetter{
			ProcessingState: omit.From(toState),
			FinalizedAt:     omitnull.From(finalizedAt),
			UpdatedAt:       omit.From(time.Now()),
			UpdatedBy:       omit.From(execUser),
		})
		if updateErr != nil {
			return updateErr
		}

		// Write audit row.
		_, updateErr = s.repo.EventProcessingAudit().Create(
			ctx,
			&models.EventProcessingAuditSetter{
				EventID:     omit.From(eventID),
				FromState:   omitnull.From(fromState),
				ToState:     omit.From(toState),
				Action:      omit.From("finalize_event_processing"),
				PayloadJSON: omit.From(emptyJSON),
				CreatedBy:   omit.From(execUser),
				UpdatedBy:   omit.From(execUser),
			})
		return updateErr
	}); txErr != nil {
		l.Error("failed to finalize event processing", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to finalize event processing")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "event processing finalized")
	return connect.NewResponse(&importv1.FinalizeEventProcessingResponse{
		ProcessingState: toState,
	}), nil
}
