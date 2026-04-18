package importsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
)

//nolint:whitespace // editor/linter issue
func (s *service) AddPenalty(
	ctx context.Context,
	req *connect.Request[importv1.AddPenaltyRequest],
) (*connect.Response[importv1.AddPenaltyResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("AddPenalty")

	var msg *importv1.AddPenaltyRequest
	if req != nil {
		msg = req.Msg
	}

	setter, err := s.buildPenaltyBookingEntry(ctx, msg)
	if err != nil {
		l.Error("failed to build penalty booking entry", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to build penalty booking entry")
		return nil, s.toRPCError(err)
	}

	if _, err := s.repo.BookingEntries().Create(ctx, setter); err != nil {
		l.Error("failed to create penalty booking entry", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to create penalty booking entry")
		return nil, s.toRPCError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "penalty added")
	return connect.NewResponse(&importv1.AddPenaltyResponse{}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeletePenalty(
	ctx context.Context,
	req *connect.Request[importv1.DeletePenaltyRequest],
) (*connect.Response[importv1.DeletePenaltyResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeletePenalty")

	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("request is required"))
	}

	penaltyID := int32(req.Msg.GetPenaltyId())
	if penaltyID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("penalty_id is required"))
	}

	if err := s.withTx(ctx, func(txCtx context.Context) error {
		entry, err := s.repo.BookingEntries().LoadByID(txCtx, penaltyID)
		if err != nil {
			return err
		}

		if entry.SourceType != mytypes.SourceType("penalty_points") {
			return connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("booking entry %d is not a penalty entry", penaltyID))
		}

		return s.repo.BookingEntries().DeleteByID(txCtx, penaltyID)
	}); err != nil {
		l.Error("failed to delete penalty booking entry", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to delete penalty booking entry")
		return nil, s.toRPCError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "penalty deleted")
	return connect.NewResponse(&importv1.DeletePenaltyResponse{}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) buildPenaltyBookingEntry(
	ctx context.Context,
	req *importv1.AddPenaltyRequest,
) (*models.BookingEntrySetter, error) {
	if req == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("request is required"))
	}

	target := req.GetTarget()
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("target is required"))
	}

	penaltyPoints := req.GetPenaltyPoints()
	if penaltyPoints == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("penalty_points must not be zero"))
	}
	if penaltyPoints > 0 {
		penaltyPoints = -penaltyPoints
	}

	resolvedTarget, err := s.resolvePenaltyTarget(ctx, target)
	if err != nil {
		return nil, err
	}

	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}
	execUser := s.execUser(ctx)

	return &models.BookingEntrySetter{
		EventID:      omit.From(resolvedTarget.eventID),
		RaceID:       resolvedTarget.raceID,
		RaceGridID:   resolvedTarget.raceGridID,
		TargetType:   omit.From(resolvedTarget.targetType),
		DriverID:     resolvedTarget.driverID,
		TeamID:       resolvedTarget.teamID,
		SourceType:   omit.From(mytypes.SourceType("penalty_points")),
		Points:       omit.From(penaltyPoints),
		Description:  omit.From(req.GetReason()),
		IsManual:     omit.From(true),
		MetadataJSON: omit.From(emptyJSON),
		CreatedBy:    omit.From(execUser),
		UpdatedBy:    omit.From(execUser),
	}, nil
}

type penaltyTargetData struct {
	eventID    int32
	raceID     omitnull.Val[int32]
	raceGridID omitnull.Val[int32]
	targetType mytypes.TargetType
	driverID   omitnull.Val[int32]
	teamID     omitnull.Val[int32]
}

//nolint:whitespace,gocyclo,funlen // editor/linter issue, complex logic
func (s *service) resolvePenaltyTarget(
	ctx context.Context,
	target *importv1.PenaltyTarget,
) (*penaltyTargetData, error) {
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("target is required"))
	}

	data := &penaltyTargetData{}

	switch {
	case target.HasDriverId():
		if target.GetDriverId() == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target driver_id must not be zero"))
		}
		data.targetType = mytypes.TargetType("driver")
		data.driverID = omitnull.From(int32(target.GetDriverId()))
	case target.HasTeamId():
		if target.GetTeamId() == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target team_id must not be zero"))
		}
		data.targetType = mytypes.TargetType("team")
		data.teamID = omitnull.From(int32(target.GetTeamId()))
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("target driver_id or team_id is required"))
	}

	switch {
	case target.HasEventId():
		eventID := int32(target.GetEventId())
		if eventID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target event_id must not be zero"))
		}
		if _, err := s.repo.Events().LoadByID(ctx, eventID); err != nil {
			return nil, err
		}
		data.eventID = eventID
	case target.HasRaceId():
		raceID := int32(target.GetRaceId())
		if raceID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target race_id must not be zero"))
		}
		race, err := s.repo.Races().Races().LoadByID(ctx, raceID)
		if err != nil {
			return nil, err
		}
		data.eventID = race.EventID
		data.raceID = omitnull.From(raceID)
	case target.HasRaceGridId():
		raceGridID := int32(target.GetRaceGridId())
		if raceGridID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("target race_grid_id must not be zero"))
		}
		grid, err := s.repo.Races().RaceGrids().LoadByID(ctx, raceGridID)
		if err != nil {
			return nil, err
		}
		race, err := s.repo.Races().Races().LoadByID(ctx, grid.RaceID)
		if err != nil {
			return nil, err
		}
		data.eventID = race.EventID
		data.raceID = omitnull.From(grid.RaceID)
		data.raceGridID = omitnull.From(raceGridID)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("target scope event_id, race_id, or race_grid_id is required"))
	}

	if data.eventID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("target scope resolved to an empty event_id"))
	}

	return data, nil
}

func (s *service) toRPCError(err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr
	}

	return connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
}
