//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"fmt"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
)

type raceRequest interface {
	GetEventId() uint32
	GetName() string
	GetSessionType() commonv1.RaceSessionType
	GetSequenceNo() int32
}

type raceSetter = models.RaceSetter

type raceSetterBuilder struct{}

func (b raceSetterBuilder) Build(msg raceRequest) (*raceSetter, error) {
	setter := &raceSetter{}

	if eventID := msg.GetEventId(); eventID != 0 {
		setter.EventID = omit.From(int32(eventID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	//nolint:lll // readability
	if st := msg.GetSessionType(); st != commonv1.RaceSessionType_RACE_SESSION_TYPE_UNSPECIFIED {
		dbStr, err := raceSessionTypeToString(st)
		if err != nil {
			return nil, err
		}
		setter.SessionType = omit.From(dbStr)
	}

	if seqNo := msg.GetSequenceNo(); seqNo != 0 {
		setter.SequenceNo = omit.From(seqNo)
	}

	return setter, nil
}

func raceSessionTypeToString(st commonv1.RaceSessionType) (string, error) {
	//nolint:exhaustive // we want to return an error for unsupported types
	switch st {
	case commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING:
		return conversion.RaceSessionTypeQualifying, nil
	case commonv1.RaceSessionType_RACE_SESSION_TYPE_HEAT:
		return conversion.RaceSessionTypeHeat, nil
	case commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE:
		return conversion.RaceSessionTypeRace, nil
	default:
		return "", fmt.Errorf("unsupported session type: %s", st.String())
	}
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateRace(
	ctx context.Context,
	req *connect.Request[v1.CreateRaceRequest]) (
	*connect.Response[v1.CreateRaceResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateRace")

	setter, err := (raceSetterBuilder{}).Build(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var newRace *models.Race
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newRace, err = s.repo.Races().Races().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create race", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create race")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "race created")
	return connect.NewResponse(&v1.CreateRaceResponse{
		Race: s.conversion.RaceToRace(newRace),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateRace(
	ctx context.Context,
	req *connect.Request[v1.UpdateRaceRequest]) (
	*connect.Response[v1.UpdateRaceResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateRace")

	setter, err := (raceSetterBuilder{}).Build(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var newRace *models.Race
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newRace, err = s.repo.Races().Races().Update(
			ctx,
			int32(req.Msg.GetRaceId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update race", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update race")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "race updated")
	return connect.NewResponse(&v1.UpdateRaceResponse{
		Race: s.conversion.RaceToRace(newRace),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteRace(
	ctx context.Context,
	req *connect.Request[v1.DeleteRaceRequest]) (
	*connect.Response[v1.DeleteRaceResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteRace")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Races().Races().DeleteByID(
			ctx,
			int32(req.Msg.GetRaceId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete race", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete race")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "race deleted")
	return connect.NewResponse(&v1.DeleteRaceResponse{
		Deleted: true,
	}), nil
}
