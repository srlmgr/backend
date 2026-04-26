package importsvc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/samber/lo"
	"github.com/stephenafamo/bob/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/services/conversion"
)

type (
	cleanupFunc func(ctx context.Context) error
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) CleanupProcessingData(
	ctx context.Context,
	req *connect.Request[importv1.CleanupProcessingDataRequest]) (
	*connect.Response[importv1.CleanupProcessingDataResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CleanupProcessingData")

	var msg *importv1.CleanupProcessingDataRequest
	if req != nil {
		msg = req.Msg
	}

	var cleanup cleanupFunc
	var event *models.Event
	execUser := s.execUser(ctx)
	switch {
	case msg.CleanupTarget.HasEventId():
		eventID := int32(msg.CleanupTarget.GetEventId())
		if eventID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target event_id must not be zero"))
		}
		event = s.resolveEventByID(eventID)
		if event == nil {
			return nil, connect.NewError(connect.CodeNotFound,
				errors.New("event not found"))
		}
		cleanup = s.cleanupByEvent(eventID, msg.IncludeManualEdits)

	case msg.CleanupTarget.HasRaceId():
		raceID := int32(msg.CleanupTarget.GetRaceId())
		if raceID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target race_id must not be zero"))
		}
		event = s.resolveEventByRaceID(raceID)
		if event == nil {
			return nil, connect.NewError(connect.CodeNotFound,
				errors.New("event not found"))
		}
		cleanup = s.cleanupByRace(raceID, msg.IncludeManualEdits)

	case msg.CleanupTarget.HasRaceGridId():
		raceGridID := int32(msg.CleanupTarget.GetRaceGridId())
		if raceGridID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target race_grid_id must not be zero"))
		}
		event = s.resolveEventByGridID(raceGridID)
		if event == nil {
			return nil, connect.NewError(connect.CodeNotFound,
				errors.New("event not found"))
		}
		cleanup = s.cleanupByGrid(raceGridID, msg.IncludeManualEdits)

	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("target scope event_id, race_id, or race_grid_id is required"))
	}

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		err := cleanup(ctx)
		if err != nil {
			return err
		}
		_, updateErr := s.repo.Events().Update(ctx, event.ID, &models.EventSetter{
			ProcessingState: omit.From(conversion.EventProcessingStateRawImported),

			UpdatedAt: omit.From(time.Now()),
			UpdatedBy: omit.From(execUser),
		})
		if updateErr != nil {
			return updateErr
		}
		emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}

		// Write audit row.
		_, updateErr = s.repo.EventProcessingAudit().Create(
			ctx,
			&models.EventProcessingAuditSetter{
				EventID:     omit.From(event.ID),
				FromState:   omitnull.From(event.ProcessingState),
				ToState:     omit.From(conversion.EventProcessingStateRawImported),
				Action:      omit.From("cleanup_processing_data"),
				PayloadJSON: omit.From(emptyJSON),
				CreatedBy:   omit.From(execUser),
				UpdatedBy:   omit.From(execUser),
			})
		return updateErr
	}); txErr != nil {
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "cleanup processing data failed")
		return nil, s.toRPCError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "cleanup processing data completed")
	return connect.NewResponse(&importv1.CleanupProcessingDataResponse{}), nil
}

func (s *service) resolveEventByID(eventID int32) *models.Event {
	event, err := s.repo.Events().LoadByID(context.Background(), eventID)
	if err != nil {
		return nil
	}
	return event
}

func (s *service) resolveEventByRaceID(raceID int32) *models.Event {
	event, err := s.repo.Events().LoadByRaceID(context.Background(), raceID)
	if err != nil {
		return nil
	}
	return event
}

func (s *service) resolveEventByGridID(gridID int32) *models.Event {
	event, err := s.repo.Events().LoadByGridID(context.Background(), gridID)
	if err != nil {
		return nil
	}
	return event
}

func (s *service) cleanupByEvent(eventID int32, includeManuals bool) cleanupFunc {
	return func(ctx context.Context) error {
		if _, err := s.repo.Events().LoadByID(ctx, eventID); err != nil {
			return err
		}
		raceGrids, err := s.repo.Races().RaceGrids().LoadByEventID(ctx, eventID)
		if err != nil {
			return err
		}
		err = s.repo.BookingEntries().CleanupByEventID(ctx, eventID, includeManuals)
		if err != nil {
			return err
		}
		raceGridIDs := lo.Map(raceGrids,
			func(rg *models.RaceGrid, _ int) int32 { return rg.ID })
		err = s.repo.ResultEntries().DeleteByRaceGridIDs(ctx, raceGridIDs)
		if err != nil {
			return err
		}
		return nil
	}
}

func (s *service) cleanupByRace(raceID int32, includeManuals bool) cleanupFunc {
	return func(ctx context.Context) error {
		raceGrids, err := s.repo.Races().RaceGrids().LoadByRaceID(ctx, raceID)
		if err != nil {
			return err
		}
		err = s.repo.BookingEntries().CleanupByRaceIDs(ctx, []int32{raceID}, includeManuals)
		if err != nil {
			return err
		}
		raceGridIDs := lo.Map(raceGrids,
			func(rg *models.RaceGrid, _ int) int32 { return rg.ID })
		err = s.repo.ResultEntries().DeleteByRaceGridIDs(ctx, raceGridIDs)
		if err != nil {
			return err
		}
		return nil
	}
}

func (s *service) cleanupByGrid(gridID int32, includeManuals bool) cleanupFunc {
	return func(ctx context.Context) error {
		err := s.repo.BookingEntries().CleanupByGridIDs(
			ctx, []int32{gridID}, includeManuals)
		if err != nil {
			return err
		}

		err = s.repo.ResultEntries().DeleteByRaceGridIDs(ctx, []int32{gridID})
		if err != nil {
			return err
		}
		return nil
	}
}
