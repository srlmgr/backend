//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type resultEntryRequest interface {
	GetRaceId() uint32
	GetDriverId() uint32
	GetCarModelId() uint32
	GetFinishingPosition() int32
	GetCompletedLaps() int32
	GetFastestLapTimeMs() int32
	GetIncidents() int32
	GetState() commonv1.ResultState
	GetAdminNotes() string
}

type resultEntrySetter = models.ResultEntrySetter

type resultEntrySetterBuilder struct{}

func (b resultEntrySetterBuilder) Build(msg resultEntryRequest) *resultEntrySetter {
	setter := &resultEntrySetter{}

	if raceID := msg.GetRaceId(); raceID != 0 {
		setter.RaceID = omit.From(int32(raceID))
	}

	if driverID := msg.GetDriverId(); driverID != 0 {
		setter.DriverID = omitnull.From(int32(driverID))
	}

	if carModelID := msg.GetCarModelId(); carModelID != 0 {
		setter.CarModelID = omitnull.From(int32(carModelID))
	}

	if pos := msg.GetFinishingPosition(); pos != 0 {
		setter.FinishingPosition = omit.From(pos)
	}

	if laps := msg.GetCompletedLaps(); laps != 0 {
		setter.CompletedLaps = omit.From(laps)
	}

	if lapTimeMs := msg.GetFastestLapTimeMs(); lapTimeMs != 0 {
		setter.FastestLapTimeMS = omitnull.From(lapTimeMs)
	}

	if incidents := msg.GetIncidents(); incidents != 0 {
		setter.Incidents = omitnull.From(incidents)
	}

	if state := msg.GetState(); state != commonv1.ResultState_RESULT_STATE_UNSPECIFIED {
		setter.State = omit.From(resultStateToString(state))
	}

	if notes := msg.GetAdminNotes(); notes != "" {
		setter.AdminNotes = omitnull.From(notes)
	}

	return setter
}

func resultStateToString(state commonv1.ResultState) string {
	switch state {
	case commonv1.ResultState_RESULT_STATE_NORMAL:
		return "normal"
	case commonv1.ResultState_RESULT_STATE_DQ:
		return "dq"
	default:
		return ""
	}
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateResultEntry(
	ctx context.Context,
	req *connect.Request[v1.CreateResultEntryRequest]) (
	*connect.Response[v1.CreateResultEntryResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateResultEntry")
	setter := (resultEntrySetterBuilder{}).Build(req.Msg)

	var newEntry *models.ResultEntry
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newEntry, err = s.repo.ResultEntries().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create result entry", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create result entry")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "result entry created")
	return connect.NewResponse(&v1.CreateResultEntryResponse{
		ResultEntry: s.conversion.ResultEntryToResultEntry(newEntry),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateResultEntry(
	ctx context.Context,
	req *connect.Request[v1.UpdateResultEntryRequest]) (
	*connect.Response[v1.UpdateResultEntryResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateResultEntry")
	setter := (resultEntrySetterBuilder{}).Build(req.Msg)

	var newEntry *models.ResultEntry
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newEntry, err = s.repo.ResultEntries().Update(
			ctx,
			int32(req.Msg.GetResultEntryId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update result entry", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update result entry")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "result entry updated")
	return connect.NewResponse(&v1.UpdateResultEntryResponse{
		ResultEntry: s.conversion.ResultEntryToResultEntry(newEntry),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteResultEntry(
	ctx context.Context,
	req *connect.Request[v1.DeleteResultEntryRequest]) (
	*connect.Response[v1.DeleteResultEntryResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteResultEntry")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.ResultEntries().DeleteByID(
			ctx,
			int32(req.Msg.GetResultEntryId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete result entry", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete result entry")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "result entry deleted")
	return connect.NewResponse(&v1.DeleteResultEntryResponse{
		Deleted: true,
	}), nil
}
