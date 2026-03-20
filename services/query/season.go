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

// ListSeasons returns a list of seasons.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListSeasons(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonsRequest],
) (*connect.Response[queryv1.ListSeasonsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListSeasons", log.Uint32("series_id", req.Msg.GetSeriesId()))

	seasonsRepo := s.repo.Seasons()

	var (
		seasonItems []*models.Season
		err         error
	)

	if seriesID := int32(req.Msg.GetSeriesId()); seriesID != 0 {
		seasonItems, err = seasonsRepo.LoadBySeriesID(ctx, seriesID)
	} else {
		seasonItems, err = seasonsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load seasons", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load seasons")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.Season, 0, len(seasonItems))
	for _, item := range seasonItems {
		if converted := s.conversion.SeasonToSeason(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "seasons loaded")
	return connect.NewResponse(&queryv1.ListSeasonsResponse{Items: items}), nil
}

// GetSeason returns a season by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetSeason(
	ctx context.Context,
	req *connect.Request[queryv1.GetSeasonRequest],
) (*connect.Response[queryv1.GetSeasonResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetSeason", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Seasons().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load season", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season loaded")
	return connect.NewResponse(&queryv1.GetSeasonResponse{
		Season: s.conversion.SeasonToSeason(item),
	}), nil
}
