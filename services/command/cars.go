//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
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
type CarModelRequest interface {
	GetManufacturerId() uint32
	GetName() string
}

type carModelVariantRequest interface {
	GetModelId() uint32
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

type carModelSetter = models.CarModelSetter

type carModelSetterBuilder struct{}

//nolint:whitespace // multiline signature style
func (b carModelSetterBuilder) Build(
	msg CarModelRequest,
) *carModelSetter {
	setter := &carModelSetter{}

	if manufacturerID := msg.GetManufacturerId(); manufacturerID != 0 {
		setter.ManufacturerID = omit.From(int32(manufacturerID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	return setter
}

type carModelVariantSetter = models.CarModelVariantSetter

type carModelVariantSetterBuilder struct{}

//nolint:whitespace // multiline signature style
func (b carModelVariantSetterBuilder) Build(
	msg carModelVariantRequest,
) *carModelVariantSetter {
	setter := &carModelVariantSetter{}

	if modelID := msg.GetModelId(); modelID != 0 {
		setter.CarModelID = omit.From(int32(modelID))
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
		l.Error("failed to create car model v2", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create car model v2")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model v2 created")
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
		l.Error("failed to update car model v2", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update car model v2")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model v2 updated")
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
		l.Error("failed to delete car model v2", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete car model v2")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model v2 deleted")
	return connect.NewResponse(&v1.DeleteCarModelResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateCarModelVariant(
	ctx context.Context,
	req *connect.Request[v1.CreateCarModelVariantRequest]) (
	*connect.Response[v1.CreateCarModelVariantResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateCarModelVariant")
	setter := (carModelVariantSetterBuilder{}).Build(req.Msg)

	var newCarModelVariant *models.CarModelVariant
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarModelVariant, err = s.repo.Cars().CarModelVariants().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create car model variant", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to create car model variant")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model variant created")
	return connect.NewResponse(&v1.CreateCarModelVariantResponse{
		CarModelVariant: s.conversion.CarModelVariantToCarModelVariant(newCarModelVariant),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateCarModelVariant(
	ctx context.Context,
	req *connect.Request[v1.UpdateCarModelVariantRequest]) (
	*connect.Response[v1.UpdateCarModelVariantResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateCarModelVariant")
	setter := (carModelVariantSetterBuilder{}).Build(req.Msg)

	var newCarModelVariant *models.CarModelVariant
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarModelVariant, err = s.repo.Cars().CarModelVariants().Update(
			ctx,
			int32(req.Msg.GetCarModelVariantId()),
			setter,
		)
		if err != nil {
			return err
		}
		setters := s.createCarAliasSetters(
			ctx,
			newCarModelVariant.ID,
			req.Msg.GetSimulationAliases())
		aliases, aliasErr := s.repo.Cars().SimulationCarAliases().
			ReplaceForVariantID(
				ctx,
				newCarModelVariant.ID,
				setters,
			)
		_ = aliases // currently not used
		return aliasErr
	}); txErr != nil {
		l.Error("failed to update car model variant", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to update car model variant")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model variant updated")
	return connect.NewResponse(&v1.UpdateCarModelVariantResponse{
		CarModelVariant: s.conversion.CarModelVariantToCarModelVariant(newCarModelVariant),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteCarModelVariant(
	ctx context.Context,
	req *connect.Request[v1.DeleteCarModelVariantRequest]) (
	*connect.Response[v1.DeleteCarModelVariantResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteCarModelVariant")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Cars().CarModelVariants().DeleteByID(
			ctx,
			int32(req.Msg.GetCarModelVariantId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete car model variant", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to delete car model variant")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model variant deleted")
	return connect.NewResponse(&v1.DeleteCarModelVariantResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) createCarAliasSetters(
	ctx context.Context,
	carModelVariantID int32,
	aliases []*commonv1.SimulationAliases,
) []*models.SimulationCarAliasSetter {
	setters := make([]*models.SimulationCarAliasSetter, 0)
	for _, item := range aliases {
		for _, externalName := range item.Identifiers {
			setters = append(setters, &models.SimulationCarAliasSetter{
				CarModelVariantID: omit.From(carModelVariantID),
				SimulationID:      omit.From(int32(item.SimulationId)),
				ExternalName:      omit.From(externalName),
				CreatedBy:         omit.From(s.execUser(ctx)),
				UpdatedBy:         omit.From(s.execUser(ctx)),
			})
		}
	}
	return setters
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
		carModelVariantID := int32(req.Msg.GetCarModelVariantId())
		simulationID := int32(req.Msg.GetSimulationId())

		existing, err := s.repo.Cars().SimulationCarAliases().LoadBySimulationID(
			ctx, simulationID)
		if err != nil {
			return err
		}

		for _, alias := range existing {
			if alias.CarModelVariantID != carModelVariantID {
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
					CarModelVariantID: omit.From(carModelVariantID),
					SimulationID:      omit.From(simulationID),
					ExternalName:      omit.From(externalName),
					CreatedBy:         omit.From(user),
					UpdatedBy:         omit.From(user),
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

type carClassRequest interface {
	GetName() string
}

type carClassSetter = models.CarClassSetter

type carClassSetterBuilder struct{}

//nolint:whitespace // multiline signature style
func (b carClassSetterBuilder) Build(
	msg carClassRequest,
) *carClassSetter {
	setter := &carClassSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateCarClass(
	ctx context.Context,
	req *connect.Request[v1.CreateCarClassRequest]) (
	*connect.Response[v1.CreateCarClassResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateCarClass")
	setter := (carClassSetterBuilder{}).Build(req.Msg)

	var newCarClass *models.CarClass
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarClass, err = s.repo.Cars().CarClasses().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create car class", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car class created")
	return connect.NewResponse(&v1.CreateCarClassResponse{
		CarClass: s.conversion.CarClassToCarClass(newCarClass),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateCarClass(
	ctx context.Context,
	req *connect.Request[v1.UpdateCarClassRequest]) (
	*connect.Response[v1.UpdateCarClassResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateCarClass")
	setter := (carClassSetterBuilder{}).Build(req.Msg)

	var newCarClass *models.CarClass
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newCarClass, err = s.repo.Cars().CarClasses().Update(
			ctx,
			int32(req.Msg.GetCarClassId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update car class", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car class updated")
	return connect.NewResponse(&v1.UpdateCarClassResponse{
		CarClass: s.conversion.CarClassToCarClass(newCarClass),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteCarClass(
	ctx context.Context,
	req *connect.Request[v1.DeleteCarClassRequest]) (
	*connect.Response[v1.DeleteCarClassResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteCarClass")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Cars().CarClasses().DeleteByID(
			ctx,
			int32(req.Msg.GetCarClassId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete car class", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car class deleted")
	return connect.NewResponse(&v1.DeleteCarClassResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) AssignCarModelVariantToCarClass(
	ctx context.Context,
	req *connect.Request[v1.AssignCarModelVariantToCarClassRequest]) (
	*connect.Response[v1.AssignCarModelVariantToCarClassResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("AssignCarModelVariantToCarClass")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		return s.repo.Cars().CarClasses().AssignCarModelVariant(
			ctx,
			int32(req.Msg.GetCarClassId()),
			int32(req.Msg.GetCarModelVariantId()),
		)
	}); txErr != nil {
		l.Error("failed to assign car model variant to car class", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to assign car model variant to car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(
		codes.Ok, "car model variant assigned to car class")
	return connect.NewResponse(&v1.AssignCarModelVariantToCarClassResponse{}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UnassignCarModelVariantFromCarClass(
	ctx context.Context,
	req *connect.Request[v1.UnassignCarModelVariantFromCarClassRequest]) (
	*connect.Response[v1.UnassignCarModelVariantFromCarClassResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UnassignCarModelVariantFromCarClass")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		return s.repo.Cars().CarClasses().UnassignCarModelVariant(
			ctx,
			int32(req.Msg.GetCarClassId()),
			int32(req.Msg.GetCarModelVariantId()),
		)
	}); txErr != nil {
		l.Error("failed to unassign car model variant from car class",
			log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to unassign car model variant from car class")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model unassigned from car class")
	return connect.NewResponse(&v1.UnassignCarModelVariantFromCarClassResponse{}), nil
}
