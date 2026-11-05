//nolint:dupl // some operations are very similar across entities
package query

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

// ListEvents returns a list of events.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListEvents(
	ctx context.Context,
	req *queryv1.ListEventsRequest,
) (*queryv1.ListEventsResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListEvents", log.Uint32("season_id", req.GetSeasonId()))

	eventsRepo := s.repo.Events()

	var (
		eventItems []*models.Event
		err        error
	)

	if seasonID := int32(req.GetSeasonId()); seasonID != 0 {
		eventItems, err = eventsRepo.LoadBySeasonID(ctx, seasonID)
	} else {
		eventItems, err = eventsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load events", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load events")
		return nil, s.conversion.ToConnectError(err)
	}

	items := make([]*commonv1.Event, 0, len(eventItems))
	for _, item := range eventItems {
		if converted := s.conversion.EventToEvent(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "events loaded")
	return &queryv1.ListEventsResponse{Items: items}, nil
}

// GetEvent returns an event by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetEvent(
	ctx context.Context,
	req *queryv1.GetEventRequest,
) (*queryv1.GetEventResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetEvent", log.Uint32("id", req.GetId()))

	item, err := s.repo.Events().LoadByID(ctx, int32(req.GetId()))
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, s.conversion.ToConnectError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "event loaded")
	return &queryv1.GetEventResponse{
		Event: s.conversion.EventToEvent(item),
	}, nil
}
