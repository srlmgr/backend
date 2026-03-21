//nolint:dupl // some operations are very similar across entities
package query

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

// ListRaces returns a list of races.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListRaces(
	ctx context.Context,
	req *connect.Request[queryv1.ListRacesRequest],
) (*connect.Response[queryv1.ListRacesResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListRaces", log.Uint32("event_id", req.Msg.GetEventId()))

	racesRepo := s.repo.Races()

	var (
		raceItems []*models.Race
		err       error
	)

	if eventID := int32(req.Msg.GetEventId()); eventID != 0 {
		raceItems, err = racesRepo.LoadByEventID(ctx, eventID)
	} else {
		raceItems, err = racesRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load races", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load races")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.Race, 0, len(raceItems))
	for _, item := range raceItems {
		if converted := s.conversion.RaceToRace(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "races loaded")
	return connect.NewResponse(&queryv1.ListRacesResponse{Items: items}), nil
}

// GetRace returns a race by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetRace(
	ctx context.Context,
	req *connect.Request[queryv1.GetRaceRequest],
) (*connect.Response[queryv1.GetRaceResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetRace", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Races().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load race", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "race loaded")
	return connect.NewResponse(&queryv1.GetRaceResponse{
		Race: s.conversion.RaceToRace(item),
	}), nil
}
