//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type eventRequest interface {
	GetSeasonId() uint32
	GetTrackLayoutId() uint32
	GetName() string
	GetEventDate() *timestamppb.Timestamp
	GetStatus() string
	GetProcessingState() string
}

type eventSetter = models.EventSetter

type eventSetterBuilder struct{}

func (b eventSetterBuilder) Build(msg eventRequest) *eventSetter {
	setter := &eventSetter{}

	if seasonID := msg.GetSeasonId(); seasonID != 0 {
		setter.SeasonID = omit.From(int32(seasonID))
	}

	if trackLayoutID := msg.GetTrackLayoutId(); trackLayoutID != 0 {
		setter.TrackLayoutID = omit.From(int32(trackLayoutID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if eventDate := msg.GetEventDate(); eventDate != nil {
		setter.EventDate = omit.From(eventDate.AsTime())
	}

	if status := msg.GetStatus(); status != "" {
		setter.Status = omit.From(status)
	}

	if processingState := msg.GetProcessingState(); processingState != "" {
		setter.ProcessingState = omit.From(processingState)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateEvent(
	ctx context.Context,
	req *connect.Request[v1.CreateEventRequest]) (
	*connect.Response[v1.CreateEventResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateEvent")
	setter := (eventSetterBuilder{}).Build(req.Msg)

	var newEvent *models.Event
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newEvent, err = s.repo.Events().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create event", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "event created")
	return connect.NewResponse(&v1.CreateEventResponse{
		Event: s.conversion.EventToEvent(newEvent),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateEvent(
	ctx context.Context,
	req *connect.Request[v1.UpdateEventRequest]) (
	*connect.Response[v1.UpdateEventResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateEvent")
	setter := (eventSetterBuilder{}).Build(req.Msg)

	var newEvent *models.Event
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newEvent, err = s.repo.Events().Update(
			ctx,
			int32(req.Msg.GetEventId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update event", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "event updated")
	return connect.NewResponse(&v1.UpdateEventResponse{
		Event: s.conversion.EventToEvent(newEvent),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteEvent(
	ctx context.Context,
	req *connect.Request[v1.DeleteEventRequest]) (
	*connect.Response[v1.DeleteEventResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteEvent")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Events().DeleteByID(
			ctx,
			int32(req.Msg.GetEventId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete event", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "event deleted")
	return connect.NewResponse(&v1.DeleteEventResponse{
		Deleted: true,
	}), nil
}
