package importsvc

import (
	"context"
	"encoding/json"

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
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) ComputeDriverBookingEntries(
	ctx context.Context,
	req *connect.Request[importv1.ComputeDriverBookingEntriesRequest],
) (
	*connect.Response[importv1.ComputeDriverBookingEntriesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ComputeDriverBookingEntries")

	eventID := int32(req.Msg.GetEventId())

	// Load the event to capture current processing state.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	// Collect result entries across all races for this event.
	races, err := s.repo.Races().LoadByEventID(ctx, eventID)
	if err != nil {
		l.Error("failed to load races", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load races")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	var resultEntries []*models.ResultEntry
	for _, race := range races {
		entries, loadErr := s.repo.ResultEntries().LoadByRaceID(ctx, race.ID)
		if loadErr != nil {
			l.Error("failed to load result entries", log.ErrorField(loadErr))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
			return nil, connect.NewError(s.conversion.MapErrorToRPCCode(loadErr), loadErr)
		}
		resultEntries = append(resultEntries, entries...)
	}

	fromState := event.ProcessingState
	toState := conversion.EventProcessingStateDriverEntriesComputed
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}

	var createdEntries int32

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		// Delete previously computed driver booking entries for idempotency.
		if delErr := s.repo.BookingEntries().DeleteByEventIDAndTargetType(
			ctx, eventID, "driver",
		); delErr != nil {
			return delErr
		}

		// Create one position-based driver booking entry per result entry
		// with a resolved driver.
		for _, entry := range resultEntries {
			if entry.DriverID.IsNull() {
				continue
			}

			_, createErr := s.repo.BookingEntries().Create(
				ctx,
				&models.BookingEntrySetter{
					EventID:             omit.From(eventID),
					SourceResultEntryID: omitnull.From(entry.ID),
					TargetType:          omit.From(mytypes.TargetType("driver")),
					DriverID:            omitnull.From(entry.DriverID.GetOr(0)),
					SourceType:          omit.From(mytypes.SourceType("position")),
					Points:              omit.From(int32(0)),
					Description:         omit.From("driver position booking"),
					IsManual:            omit.From(false),
					MetadataJSON:        omit.From(emptyJSON),
					CreatedBy:           omit.From(execUser),
					UpdatedBy:           omit.From(execUser),
				})
			if createErr != nil {
				return createErr
			}
			createdEntries++
		}

		// Advance event processing state.
		_, updateErr := s.repo.Events().Update(ctx, eventID, &models.EventSetter{
			ProcessingState: omit.From(toState),
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
				Action:      omit.From("compute_driver_booking_entries"),
				PayloadJSON: omit.From(emptyJSON),
				CreatedBy:   omit.From(execUser),
				UpdatedBy:   omit.From(execUser),
			})
		return updateErr
	}); txErr != nil {
		l.Error("failed to compute driver booking entries", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).
			SetStatus(codes.Error, "failed to compute driver booking entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver booking entries computed")
	return connect.NewResponse(&importv1.ComputeDriverBookingEntriesResponse{
		CreatedEntries: createdEntries,
	}), nil
}
