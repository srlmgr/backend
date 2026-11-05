// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace,dupl // editor/linter issue
func (s *service) ListSeasonCarModelVariants(
	ctx context.Context,
	req *queryv1.ListSeasonCarModelVariantsRequest,
) (*queryv1.ListSeasonCarModelVariantsResponse, error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.GetSeasonId())
	l.Debug("ListSeasonCarModelVariants", log.Int32("season_id", seasonID))

	seasonCarModelVariants, err := s.repo.Cars().CarModelVariants().LoadBySeasonID(
		ctx, seasonID,
	)
	if err != nil {
		l.Error("failed to load season car model variants", log.ErrorField(err))
		trace.SpanFromContext(ctx).
			SetStatus(codes.Error, "failed to load season car model variants")
		return nil, s.conversion.ToConnectError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season car model variants loaded")
	return &queryv1.ListSeasonCarModelVariantsResponse{
		Items: lo.Map(seasonCarModelVariants,
			func(item *models.CarModelVariant, _ int) *commonv1.CarModelVariant {
				return s.conversion.CarModelVariantToCarModelVariant(item)
			}),
	}, nil
}

//nolint:whitespace,dupl // editor/linter issue
func (s *service) ListSeasonCarClasses(
	ctx context.Context,
	req *queryv1.ListSeasonCarClassesRequest,
) (*queryv1.ListSeasonCarClassesResponse, error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.GetSeasonId())
	l.Debug("ListSeasonCarClasses", log.Int32("season_id", seasonID))

	seasonCarClasses, err := s.repo.Cars().CarClasses().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season car classes", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season car classes")
		return nil, s.conversion.ToConnectError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season car classes loaded")
	return &queryv1.ListSeasonCarClassesResponse{
		Items: lo.Map(seasonCarClasses,
			func(item *models.CarClass, _ int) *commonv1.CarClass {
				return s.conversion.CarClassToCarClass(item)
			}),
	}, nil
}
