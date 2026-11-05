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

// ListSeries returns a list of series.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListSeries(
	ctx context.Context,
	req *queryv1.ListSeriesRequest,
) (*queryv1.ListSeriesResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListSeries", log.Uint32("simulation_id", req.GetSimulationId()))

	seriesRepo := s.repo.Series()

	var (
		seriesItems []*models.Series
		err         error
	)

	if simulationID := int32(req.GetSimulationId()); simulationID != 0 {
		seriesItems, err = seriesRepo.LoadBySimulationID(ctx, simulationID)
	} else {
		seriesItems, err = seriesRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load series", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load series")
		return nil, s.conversion.ToConnectError(err)
	}

	items := make([]*commonv1.Series, 0, len(seriesItems))
	for _, item := range seriesItems {
		if converted := s.conversion.SeriesToSeries(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "series loaded")
	return &queryv1.ListSeriesResponse{Items: items}, nil
}

// GetSeries returns a series by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetSeries(
	ctx context.Context,
	req *queryv1.GetSeriesRequest,
) (*queryv1.GetSeriesResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetSeries", log.Uint32("id", req.GetId()))

	item, err := s.repo.Series().LoadByID(ctx, int32(req.GetId()))
	if err != nil {
		l.Error("failed to load series", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load series")
		return nil, s.conversion.ToConnectError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "series loaded")
	return &queryv1.GetSeriesResponse{
		Series: s.conversion.SeriesToSeries(item),
	}, nil
}
