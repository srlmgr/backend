// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

//
//nolint:whitespace,dupl // editor/linter issue
func (s *service) ListSeasonCarModels(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonCarModelsRequest],
) (*connect.Response[queryv1.ListSeasonCarModelsResponse], error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.Msg.GetSeasonId())
	l.Debug("ListSeasonCarModels", log.Int32("season_id", seasonID))

	seasonCarModels, err := s.repo.Cars().CarModels().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season car models", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season car models")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season car models loaded")
	return connect.NewResponse(&queryv1.ListSeasonCarModelsResponse{
		Items: lo.Map(seasonCarModels,
			func(item *models.CarModel, _ int) *commonv1.CarModel {
				return s.conversion.CarModelToCarModel(item)
			}),
	}), nil
}

//nolint:whitespace,dupl // editor/linter issue
func (s *service) ListSeasonCarClasses(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonCarClassesRequest],
) (*connect.Response[queryv1.ListSeasonCarClassesResponse], error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.Msg.GetSeasonId())
	l.Debug("ListSeasonCarClasses", log.Int32("season_id", seasonID))

	seasonCarClasses, err := s.repo.Cars().CarClasses().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season car classes", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season car classes")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season car classes loaded")
	return connect.NewResponse(&queryv1.ListSeasonCarClassesResponse{
		Items: lo.Map(seasonCarClasses,
			func(item *models.CarClass, _ int) *commonv1.CarClass {
				return s.conversion.CarClassToCarClass(item)
			}),
	}), nil
}
