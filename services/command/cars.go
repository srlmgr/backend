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

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type carManufacturerRequest interface {
	GetName() string
}

type carBrandRequest interface {
	GetManufacturerId() uint32
	GetName() string
}

type carModelRequest interface {
	GetBrandId() uint32
	GetName() string
}

type carManufacturerSetter = models.CarManufacturerSetter

type carManufacturerSetterBuilder struct{}

//nolint:whitespace // multiline signature style
func (b carManufacturerSetterBuilder) Build(
	msg carManufacturerRequest,
) *carManufacturerSetter {
	setter := &carManufacturerSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	return setter
}

type carBrandSetter = models.CarBrandSetter

type carBrandSetterBuilder struct{}

//nolint:whitespace // multiline signature style
func (b carBrandSetterBuilder) Build(
	msg carBrandRequest,
) *carBrandSetter {
	setter := &carBrandSetter{}

	if manufacturerID := msg.GetManufacturerId(); manufacturerID != 0 {
		setter.ManufacturerID = omit.From(int32(manufacturerID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	return setter
}

type carModelSetter = models.CarModelSetter

type carModelSetterBuilder struct{}

//nolint:whitespace // multiline signature style
func (b carModelSetterBuilder) Build(
	msg carModelRequest,
) *carModelSetter {
	setter := &carModelSetter{}

	if brandID := msg.GetBrandId(); brandID != 0 {
		setter.BrandID = omit.From(int32(brandID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateCarManufacturer(
	ctx context.Context,
	req *connect.Request[v1.CreateCarManufacturerRequest]) (
	*connect.Response[v1.CreateCarManufacturerResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateCarManufacturer")
	setter := (carManufacturerSetterBuilder{}).Build(req.Msg)

	var newCarManufacturer *models.CarManufacturer
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarManufacturer, err = s.repo.Cars().CarManufacturers().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create car manufacturer", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create car manufacturer")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car manufacturer created")
	return connect.NewResponse(&v1.CreateCarManufacturerResponse{
		CarManufacturer: s.conversion.CarManufacturerToCarManufacturer(newCarManufacturer),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateCarManufacturer(
	ctx context.Context,
	req *connect.Request[v1.UpdateCarManufacturerRequest]) (
	*connect.Response[v1.UpdateCarManufacturerResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateCarManufacturer")
	setter := (carManufacturerSetterBuilder{}).Build(req.Msg)

	var newCarManufacturer *models.CarManufacturer
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarManufacturer, err = s.repo.Cars().CarManufacturers().Update(
			ctx,
			int32(req.Msg.GetCarManufacturerId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update car manufacturer", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update car manufacturer")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car manufacturer updated")
	return connect.NewResponse(&v1.UpdateCarManufacturerResponse{
		CarManufacturer: s.conversion.CarManufacturerToCarManufacturer(newCarManufacturer),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteCarManufacturer(
	ctx context.Context,
	req *connect.Request[v1.DeleteCarManufacturerRequest]) (
	*connect.Response[v1.DeleteCarManufacturerResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteCarManufacturer")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Cars().CarManufacturers().DeleteByID(
			ctx,
			int32(req.Msg.GetCarManufacturerId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete car manufacturer", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete car manufacturer")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car manufacturer deleted")
	return connect.NewResponse(&v1.DeleteCarManufacturerResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateCarBrand(
	ctx context.Context,
	req *connect.Request[v1.CreateCarBrandRequest]) (
	*connect.Response[v1.CreateCarBrandResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateCarBrand")
	setter := (carBrandSetterBuilder{}).Build(req.Msg)

	var newCarBrand *models.CarBrand
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarBrand, err = s.repo.Cars().CarBrands().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create car brand", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create car brand")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car brand created")
	return connect.NewResponse(&v1.CreateCarBrandResponse{
		CarBrand: s.conversion.CarBrandToCarBrand(newCarBrand),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateCarBrand(
	ctx context.Context,
	req *connect.Request[v1.UpdateCarBrandRequest]) (
	*connect.Response[v1.UpdateCarBrandResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateCarBrand")
	setter := (carBrandSetterBuilder{}).Build(req.Msg)

	var newCarBrand *models.CarBrand
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarBrand, err = s.repo.Cars().CarBrands().Update(
			ctx,
			int32(req.Msg.GetCarBrandId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update car brand", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update car brand")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car brand updated")
	return connect.NewResponse(&v1.UpdateCarBrandResponse{
		CarBrand: s.conversion.CarBrandToCarBrand(newCarBrand),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteCarBrand(
	ctx context.Context,
	req *connect.Request[v1.DeleteCarBrandRequest]) (
	*connect.Response[v1.DeleteCarBrandResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteCarBrand")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Cars().CarBrands().DeleteByID(
			ctx,
			int32(req.Msg.GetCarBrandId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete car brand", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete car brand")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car brand deleted")
	return connect.NewResponse(&v1.DeleteCarBrandResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateCarModel(
	ctx context.Context,
	req *connect.Request[v1.CreateCarModelRequest]) (
	*connect.Response[v1.CreateCarModelResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateCarModel")
	setter := (carModelSetterBuilder{}).Build(req.Msg)

	var newCarModel *models.CarModel
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarModel, err = s.repo.Cars().CarModels().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create car model", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create car model")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model created")
	return connect.NewResponse(&v1.CreateCarModelResponse{
		CarModel: s.conversion.CarModelToCarModel(newCarModel),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateCarModel(
	ctx context.Context,
	req *connect.Request[v1.UpdateCarModelRequest]) (
	*connect.Response[v1.UpdateCarModelResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateCarModel")
	setter := (carModelSetterBuilder{}).Build(req.Msg)

	var newCarModel *models.CarModel
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarModel, err = s.repo.Cars().CarModels().Update(
			ctx,
			int32(req.Msg.GetCarModelId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update car model", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update car model")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model updated")
	return connect.NewResponse(&v1.UpdateCarModelResponse{
		CarModel: s.conversion.CarModelToCarModel(newCarModel),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteCarModel(
	ctx context.Context,
	req *connect.Request[v1.DeleteCarModelRequest]) (
	*connect.Response[v1.DeleteCarModelResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteCarModel")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Cars().CarModels().DeleteByID(
			ctx,
			int32(req.Msg.GetCarModelId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete car model", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete car model")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model deleted")
	return connect.NewResponse(&v1.DeleteCarModelResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) SetSimulationCarAliases(
	ctx context.Context,
	req *connect.Request[v1.SetSimulationCarAliasesRequest]) (
	*connect.Response[v1.SetSimulationCarAliasesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetSimulationCarAliases")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		carModelID := int32(req.Msg.GetCarModelId())
		simulationID := int32(req.Msg.GetSimulationId())

		existing, err := s.repo.Cars().SimulationCarAliases().LoadBySimulationID(
			ctx, simulationID)
		if err != nil {
			return err
		}

		for _, alias := range existing {
			if alias.CarModelID != carModelID {
				continue
			}
			if err := s.repo.Cars().SimulationCarAliases().DeleteByID(
				ctx, alias.ID); err != nil {
				return err
			}
		}

		user := s.execUser(ctx)
		for _, externalName := range req.Msg.GetExternalName() {
			_, err := s.repo.Cars().
				SimulationCarAliases().
				Create(ctx, &models.SimulationCarAliasSetter{
					CarModelID:   omit.From(carModelID),
					SimulationID: omit.From(simulationID),
					ExternalName: omit.From(externalName),
					CreatedBy:    omit.From(user),
					UpdatedBy:    omit.From(user),
				})
			if err != nil {
				return err
			}
		}

		return nil
	}); txErr != nil {
		l.Error("failed to set simulation car aliases", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error,
			"failed to set simulation car aliases")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation car aliases set")
	return connect.NewResponse(&v1.SetSimulationCarAliasesResponse{Updated: true}), nil
}
