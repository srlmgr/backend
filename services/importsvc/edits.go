package importsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) ApplyResultEdits(
	ctx context.Context,
	req *connect.Request[importv1.ApplyResultEditsRequest],
) (
	*connect.Response[importv1.ApplyResultEditsResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ApplyResultEdits")

	gridID := int32(req.Msg.GetRaceGridId())

	if gridID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("race_id is required"))
	}
	if len(req.Msg.GetEditedRows()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("edited_rows must not be empty"))
	}

	race, err := s.repo.Races().Races().LoadByID(ctx, gridID)
	if err != nil {
		l.Error("failed to load race", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	eventID := race.EventID

	// Resolve the import batch for the race.
	batch, err := s.repo.ImportBatches().LoadByRaceGridID(ctx, gridID)
	if err != nil {
		l.Error("failed to load import batch", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load import batch")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	fromState := batch.ProcessingState
	toState := conversion.EventProcessingStatePreprocessed
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}

	var updatedRows int32

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		for _, row := range req.Msg.GetEditedRows() {
			if row.GetId() == 0 {
				return connect.NewError(connect.CodeInvalidArgument,
					errors.New("result entry id is required in edited_rows"))
			}

			// Verify the entry belongs to the correct race.
			existing, loadErr := s.repo.ResultEntries().LoadByID(ctx, int32(row.GetId()))
			if loadErr != nil {
				return loadErr
			}
			if existing.RaceGridID != gridID {
				return connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("result entry %d does not belong to race %d",
						row.GetId(), gridID))
			}

			setter := buildResultEntrySetterFromProto(row, execUser)
			if _, updateErr := s.repo.ResultEntries().
				Update(ctx, int32(row.GetId()), setter); updateErr != nil {
				return updateErr
			}
			updatedRows++
		}

		// Advance batch state.
		_, updateErr := s.repo.ImportBatches().Update(
			ctx,
			batch.ID,
			&models.ImportBatchSetter{
				ProcessingState: omit.From(toState),
				ProcessedAt:     omitnull.From(time.Now()),
				UpdatedAt:       omit.From(time.Now()),
				UpdatedBy:       omit.From(execUser),
			})
		if updateErr != nil {
			return updateErr
		}

		// Advance event state.
		_, updateErr = s.repo.Events().Update(
			ctx,
			eventID,
			&models.EventSetter{
				ProcessingState: omit.From(toState),
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
				EventID:       omit.From(eventID),
				ImportBatchID: omitnull.From(batch.ID),
				FromState:     omitnull.From(fromState),
				ToState:       omit.From(toState),
				Action:        omit.From("apply_result_edits"),
				PayloadJSON:   omit.From(emptyJSON),
				CreatedBy:     omit.From(execUser),
				UpdatedBy:     omit.From(execUser),
			})
		return updateErr
	}); txErr != nil {
		l.Error("failed to apply result edits", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to apply result edits")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "result edits applied")
	return connect.NewResponse(&importv1.ApplyResultEditsResponse{
		UpdatedRows: updatedRows,
	}), nil
}

//nolint:whitespace,lll // editor/linter issue
func buildResultEntrySetterFromProto(
	row *commonv1.ResultEntry,
	execUser string,
) *models.ResultEntrySetter {
	setter := &models.ResultEntrySetter{
		UpdatedAt: omit.From(time.Now()),
		UpdatedBy: omit.From(execUser),
	}

	if driverID := row.GetDriverId(); driverID != 0 {
		setter.DriverID = omitnull.From(int32(driverID))
	}

	if carModelID := row.GetCarModelId(); carModelID != 0 {
		setter.CarModelID = omitnull.From(int32(carModelID))
	}

	if pos := row.GetFinishingPosition(); pos != 0 {
		setter.FinishPosition = omit.From(pos)
	}

	if laps := row.GetCompletedLaps(); laps != 0 {
		setter.LapsCompleted = omit.From(laps)
	}

	if lapTimeMs := row.GetFastestLapTimeMs(); lapTimeMs != 0 {
		setter.FastestLapTimeMS = omitnull.From(lapTimeMs)
	}

	if incidents := row.GetIncidents(); incidents != 0 {
		setter.Incidents = omitnull.From(incidents)
	}

	if state := row.GetState(); state != commonv1.ResultEntryState_RESULT_ENTRY_STATE_UNSPECIFIED {
		setter.State = omit.From(resultStateToStr(state))
	}

	if notes := row.GetAdminNotes(); notes != "" {
		setter.AdminNotes = omitnull.From(notes)
	}

	return setter
}

//nolint:exhaustive // by design
func resultStateToStr(state commonv1.ResultEntryState) string {
	switch state {
	case commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL:
		return conversion.ResultStateNormal
	case commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ:
		return conversion.ResultStateDQ
	default:
		return ""
	}
}
