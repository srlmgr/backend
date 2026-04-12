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

// ListCarManufacturers returns a list of all car manufacturers.
//
//nolint:whitespace,lll // editor/linter issue, readability
func (s *service) ListCarManufacturers(
	ctx context.Context,
	req *connect.Request[queryv1.ListCarManufacturersRequest],
) (*connect.Response[queryv1.ListCarManufacturersResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListCarManufacturers")

	manufacturers, err := s.repo.Cars().CarManufacturers().LoadAll(ctx)
	if err != nil {
		l.Error("failed to load car manufacturers", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car manufacturers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.CarManufacturer, 0, len(manufacturers))
	for _, item := range manufacturers {
		if converted := s.conversion.CarManufacturerToCarManufacturer(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car manufacturers loaded")
	return connect.NewResponse(&queryv1.ListCarManufacturersResponse{Items: items}), nil
}

// GetCarManufacturer returns a car manufacturer by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetCarManufacturer(
	ctx context.Context,
	req *connect.Request[queryv1.GetCarManufacturerRequest],
) (*connect.Response[queryv1.GetCarManufacturerResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetCarManufacturer", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Cars().CarManufacturers().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load car manufacturer", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car manufacturer")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car manufacturer loaded")
	return connect.NewResponse(&queryv1.GetCarManufacturerResponse{
		CarManufacturer: s.conversion.CarManufacturerToCarManufacturer(item),
	}), nil
}

// ListCarBrands returns a list of car brands, optionally filtered by manufacturer ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListCarBrands(
	ctx context.Context,
	req *connect.Request[queryv1.ListCarBrandsRequest],
) (*connect.Response[queryv1.ListCarBrandsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListCarBrands", log.Uint32("manufacturer_id", req.Msg.GetManufacturerId()))

	brandsRepo := s.repo.Cars().CarBrands()

	var (
		brands []*models.CarBrand
		err    error
	)

	if manufacturerID := int32(req.Msg.GetManufacturerId()); manufacturerID != 0 {
		brands, err = brandsRepo.LoadByManufacturerID(ctx, manufacturerID)
	} else {
		brands, err = brandsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load car brands", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car brands")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.CarBrand, 0, len(brands))
	for _, item := range brands {
		if converted := s.conversion.CarBrandToCarBrand(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car brands loaded")
	return connect.NewResponse(&queryv1.ListCarBrandsResponse{Items: items}), nil
}

// GetCarBrand returns a car brand by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetCarBrand(
	ctx context.Context,
	req *connect.Request[queryv1.GetCarBrandRequest],
) (*connect.Response[queryv1.GetCarBrandResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetCarBrand", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Cars().CarBrands().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load car brand", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car brand")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car brand loaded")
	return connect.NewResponse(&queryv1.GetCarBrandResponse{
		CarBrand: s.conversion.CarBrandToCarBrand(item),
	}), nil
}

// ListCarModels returns a list of car models, optionally filtered by manufacturer ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListCarModels(
	ctx context.Context,
	req *connect.Request[queryv1.ListCarModelsRequest],
) (*connect.Response[queryv1.ListCarModelsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListCarModels", log.Uint32("manufacturer_id", req.Msg.GetManufacturerId()))

	modelsRepo := s.repo.Cars().CarModels()

	var (
		carModels []*models.CarModel
		err       error
	)

	if manufacturerID := int32(req.Msg.GetManufacturerId()); manufacturerID != 0 {
		carModels, err = modelsRepo.LoadByManufacturerID(ctx, manufacturerID)
	} else {
		carModels, err = modelsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load car models", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car models")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.CarModel, 0, len(carModels))
	for _, item := range carModels {
		if converted := s.conversion.CarModelToCarModel(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car models loaded")
	return connect.NewResponse(&queryv1.ListCarModelsResponse{Items: items}), nil
}

// GetCarModel returns a car model by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetCarModel(
	ctx context.Context,
	req *connect.Request[queryv1.GetCarModelRequest],
) (*connect.Response[queryv1.GetCarModelResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetCarModel", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Cars().CarModels().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load car model", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car model")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model loaded")
	return connect.NewResponse(&queryv1.GetCarModelResponse{
		CarModel: s.conversion.CarModelToCarModel(item),
	}), nil
}

// ListCarClasses returns a list of all car classes.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListCarClasses(
	ctx context.Context,
	req *connect.Request[queryv1.ListCarClassesRequest],
) (*connect.Response[queryv1.ListCarClassesResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListCarClasses")

	carClasses, err := s.repo.Cars().CarClasses().LoadAll(ctx)
	if err != nil {
		l.Error("failed to load car classes", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car classes")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.CarClass, 0, len(carClasses))
	for _, item := range carClasses {
		if converted := s.conversion.CarClassToCarClass(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car classes loaded")
	return connect.NewResponse(&queryv1.ListCarClassesResponse{Items: items}), nil
}

// GetCarClass returns a car class by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetCarClass(
	ctx context.Context,
	req *connect.Request[queryv1.GetCarClassRequest],
) (*connect.Response[queryv1.GetCarClassResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetCarClass", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Cars().CarClasses().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load car class", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car class loaded")
	return connect.NewResponse(&queryv1.GetCarClassResponse{
		CarClass: s.conversion.CarClassToCarClass(item),
	}), nil
}
