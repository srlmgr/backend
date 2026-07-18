//nolint:dupl // some operations are very similar across entities
package query

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

// ListCarModels returns a list of car model v2 records,
// optionally filtered by manufacturer ID.
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
		items []*models.CarModel
		err   error
	)

	if manufacturerID := int32(req.Msg.GetManufacturerId()); manufacturerID != 0 {
		items, err = modelsRepo.LoadByManufacturerID(ctx, manufacturerID)
	} else {
		items, err = modelsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load car model v2 records", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load car model v2 records")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	out := make([]*commonv1.CarModel, 0, len(items))
	for _, item := range items {
		if converted := s.conversion.CarModelToCarModel(item); converted != nil {
			out = append(out, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model v2 records loaded")
	return connect.NewResponse(&queryv1.ListCarModelsResponse{Items: out}), nil
}

// GetCarModel returns a car model v2 by ID.
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
		l.Error("failed to load car model v2", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car model v2")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model v2 loaded")
	return connect.NewResponse(&queryv1.GetCarModelResponse{
		CarModel: s.conversion.CarModelToCarModel(item),
	}), nil
}

// ListCarModelVariants returns a list of car model variants,
// optionally filtered by model ID.
//
//nolint:whitespace,lll // editor/linter issue
func (s *service) ListCarModelVariants(
	ctx context.Context,
	req *connect.Request[queryv1.ListCarModelVariantsRequest],
) (*connect.Response[queryv1.ListCarModelVariantsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListCarModelVariants", log.Uint32("model_id", req.Msg.GetModelId()))

	variantsRepo := s.repo.Cars().CarModelVariants()

	var (
		items []*models.CarModelVariant
		err   error
	)

	if modelID := int32(req.Msg.GetModelId()); modelID != 0 {
		items, err = variantsRepo.LoadByModelID(ctx, modelID)
	} else {
		items, err = variantsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load car model variants", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load car model variants")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	out := make([]*commonv1.CarModelVariant, 0, len(items))
	for _, item := range items {
		if converted := s.conversion.CarModelVariantToCarModelVariant(item); converted != nil {
			out = append(out, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model variants loaded")
	return connect.NewResponse(&queryv1.ListCarModelVariantsResponse{Items: out}), nil
}

// GetCarModelVariant returns a car model variant by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetCarModelVariant(
	ctx context.Context,
	req *connect.Request[queryv1.GetCarModelVariantRequest],
) (*connect.Response[queryv1.GetCarModelVariantResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetCarModelVariant", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Cars().CarModelVariants().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load car model variant", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car model variant")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	aliases, aliasErr := s.repo.Cars().SimulationCarAliases().LoadByVariantID(
		ctx,
		int32(req.Msg.GetId()),
	)
	if aliasErr != nil {
		l.Error("failed to load simulation car aliases", log.ErrorField(aliasErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load simulation car aliases")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(aliasErr), aliasErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model variant loaded")
	return connect.NewResponse(&queryv1.GetCarModelVariantResponse{
		CarModelVariant:   s.conversion.CarModelVariantToCarModelVariant(item),
		SimulationAliases: s.conversion.SimulationCarAliasToProto(aliases),
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

// ListCarClassModelVariants returns a list of car model variants
// for a given car class ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListCarClassModelVariants(
	ctx context.Context,
	req *connect.Request[queryv1.ListCarClassModelVariantsRequest],
) (*connect.Response[queryv1.ListCarClassModelVariantsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListCarClassModelVariants", log.Uint32("classID", req.Msg.GetClassId()))

	carModelVariants, err := s.repo.Cars().CarModelVariants().LoadByCarClassID(
		ctx, int32(req.Msg.GetClassId()))
	if err != nil {
		l.Error("failed to load car model variants for car class", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to load car model variants for car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(
		codes.Ok, "car model variants loaded for car class")
	return connect.NewResponse(&queryv1.ListCarClassModelVariantsResponse{
		Items: lo.Map(
			carModelVariants,
			func(m *models.CarModelVariant, _ int) *commonv1.CarModelVariant {
				return s.conversion.CarModelVariantToCarModelVariant(m)
			},
		),
	}), nil
}
