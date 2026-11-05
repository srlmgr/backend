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

// ListRaces returns a list of races.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListRaces(
	ctx context.Context,
	req *queryv1.ListRacesRequest,
) (*queryv1.ListRacesResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListRaces", log.Uint32("event_id", req.GetEventId()))

	racesRepo := s.repo.Races().Races()

	var (
		raceItems []*models.Race
		err       error
	)

	if eventID := int32(req.GetEventId()); eventID != 0 {
		raceItems, err = racesRepo.LoadByEventID(ctx, eventID)
	} else {
		raceItems, err = racesRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load races", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load races")
		return nil, s.conversion.ToConnectError(err)
	}

	items := make([]*commonv1.Race, 0, len(raceItems))
	for _, item := range raceItems {
		if converted := s.conversion.RaceToRace(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "races loaded")
	return &queryv1.ListRacesResponse{Items: items}, nil
}

// GetRace returns a race by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetRace(
	ctx context.Context,
	req *queryv1.GetRaceRequest,
) (*queryv1.GetRaceResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetRace", log.Uint32("id", req.GetId()))

	item, err := s.repo.Races().Races().LoadByID(ctx, int32(req.GetId()))
	if err != nil {
		l.Error("failed to load race", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race")
		return nil, s.conversion.ToConnectError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "race loaded")
	return &queryv1.GetRaceResponse{
		Race: s.conversion.RaceToRace(item),
	}, nil
}

// ListRaceGrids returns a list of race grids.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListRaceGrids(
	ctx context.Context,
	req *queryv1.ListRaceGridsRequest,
) (*queryv1.ListRaceGridsResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListRaceGrids", log.Uint32("race_id", req.GetRaceId()))

	raceGridsRepo := s.repo.Races().RaceGrids()

	var (
		raceGridItems []*models.RaceGrid
		err           error
	)

	if raceID := int32(req.GetRaceId()); raceID != 0 {
		raceGridItems, err = raceGridsRepo.LoadByRaceID(ctx, raceID)
	} else {
		raceGridItems, err = raceGridsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load race grids", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race grids")
		return nil, s.conversion.ToConnectError(err)
	}

	items := make([]*commonv1.RaceGrid, 0, len(raceGridItems))
	for _, item := range raceGridItems {
		if converted := s.conversion.RaceGridToRaceGrid(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "race grids loaded")
	return &queryv1.ListRaceGridsResponse{Items: items}, nil
}
