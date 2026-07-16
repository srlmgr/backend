package command

import (
	"context"
	"fmt"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type seasonDriverInput struct {
	driverID          int32
	carModelVariantID int32
	carNumber         string
	isGuestDriver     bool
	joinedAt          *time.Time
	leftAt            *time.Time
}

//nolint:whitespace // editor/linter issue
func convertSetSeasonDriversInput(
	msg *v1.SetSeasonDriversRequest,
) []seasonDriverInput {
	inputs := make([]seasonDriverInput, 0, len(msg.GetDrivers()))
	for _, driver := range msg.GetDrivers() {
		input := seasonDriverInput{
			driverID:          int32(driver.GetDriverId()),
			carModelVariantID: int32(driver.GetCarModelVariantId()),
			carNumber:         driver.GetCarNumber(),
			isGuestDriver:     driver.GetIsGuestDriver(),
		}
		if driver.HasJoinedAt() {
			joinedAt := driver.GetJoinedAt().AsTime()
			input.joinedAt = &joinedAt
		}
		if driver.HasLeftAt() {
			leftAt := driver.GetLeftAt().AsTime()
			input.leftAt = &leftAt
		}

		inputs = append(inputs, input)
	}

	return inputs
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) SetSeasonDrivers(
	ctx context.Context,
	req *connect.Request[v1.SetSeasonDriversRequest]) (
	*connect.Response[v1.SetSeasonDriversResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetSeasonDrivers")

	seasonID := int32(req.Msg.GetSeasonId())
	inputs := convertSetSeasonDriversInput(req.Msg)

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		existing, err := s.repo.Drivers().SeasonDrivers().LoadBySeasonID(ctx, seasonID)
		if err != nil {
			return err
		}

		for _, item := range existing {
			if err := s.repo.Drivers().SeasonDrivers().DeleteByID(ctx, item.ID); err != nil {
				return err
			}
		}

		user := s.execUser(ctx)
		for i := range inputs {
			input := inputs[i]
			setter := &models.SeasonDriverSetter{
				DriverID:          omit.From(input.driverID),
				SeasonID:          omit.From(seasonID),
				CarModelVariantID: omit.From(input.carModelVariantID),
				CarNumber:         omit.From(input.carNumber),
				IsGuestStarter:    omit.From(input.isGuestDriver),
				CreatedBy:         omit.From(user),
				UpdatedBy:         omit.From(user),
			}
			if input.joinedAt != nil {
				setter.JoinedAt = omit.From(*input.joinedAt)
			}
			if input.leftAt != nil {
				setter.LeftAt = omitnull.From(*input.leftAt)
			}

			if _, err := s.repo.Drivers().SeasonDrivers().Create(ctx, setter); err != nil {
				return err
			}
		}

		return nil
	}); txErr != nil {
		l.Error("failed to set season drivers", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to set season drivers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season drivers set")
	return connect.NewResponse(&v1.SetSeasonDriversResponse{}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) AddSeasonDriver(
	ctx context.Context,
	req *connect.Request[v1.AddSeasonDriverRequest]) (
	*connect.Response[v1.AddSeasonDriverResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("AddSeasonDriver")

	carModelVariantID := int32(req.Msg.GetCarModelVariantId())
	if carModelVariantID <= 0 {
		err := fmt.Errorf(
			"invalid car_model_variant_id %d: must be greater than 0",
			carModelVariantID,
		)
		l.Error("invalid season driver payload", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "invalid season driver payload")
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	seasonID := int32(req.Msg.GetSeasonId())
	driverID := int32(req.Msg.GetDriverId())

	existing, err := s.repo.Drivers().SeasonDrivers().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season drivers", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season drivers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	for _, item := range existing {
		if item.DriverID == driverID && item.LeftAt.IsNull() {
			l.Debug("driver is already active in season, skipping")
			trace.SpanFromContext(ctx).
				SetStatus(codes.Ok, "driver is already active in season, skipping")
			return connect.NewResponse(&v1.AddSeasonDriverResponse{}), nil
		}
	}

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		setter := &models.SeasonDriverSetter{
			DriverID:          omit.From(driverID),
			SeasonID:          omit.From(seasonID),
			CarModelVariantID: omit.From(carModelVariantID),
			CarNumber:         omit.From(req.Msg.GetCarNumber()),
			IsGuestStarter:    omit.From(req.Msg.GetIsGuestDriver()),
			CreatedBy:         omit.From(s.execUser(ctx)),
			UpdatedBy:         omit.From(s.execUser(ctx)),
		}
		if req.Msg.HasJoinedAt() {
			setter.JoinedAt = omit.From(req.Msg.GetJoinedAt().AsTime())
		}

		_, err := s.repo.Drivers().SeasonDrivers().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to add season driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to add season driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season driver added")
	return connect.NewResponse(&v1.AddSeasonDriverResponse{}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) RemoveSeasonDriver(
	ctx context.Context,
	req *connect.Request[v1.RemoveSeasonDriverRequest]) (
	*connect.Response[v1.RemoveSeasonDriverResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("RemoveSeasonDriver")

	seasonID := int32(req.Msg.GetSeasonId())
	driverID := int32(req.Msg.GetDriverId())

	existing, err := s.repo.Drivers().SeasonDrivers().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season drivers", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season drivers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	var toClose *models.SeasonDriver
	for _, item := range existing {
		if item.DriverID == driverID && item.LeftAt.IsNull() {
			toClose = item
			break
		}
	}
	if toClose == nil {
		l.Debug("driver is not active in season, skipping")
		trace.SpanFromContext(ctx).SetStatus(
			codes.Ok, "driver is not active in season, skipping")
		return connect.NewResponse(&v1.RemoveSeasonDriverResponse{}), nil
	}

	leftAt := time.Now()
	if req.Msg.HasLeftAt() {
		leftAt = req.Msg.GetLeftAt().AsTime()
	}

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		_, err := s.repo.Drivers().SeasonDrivers().Update(
			ctx,
			toClose.ID,
			&models.SeasonDriverSetter{
				LeftAt:    omitnull.From(leftAt),
				UpdatedAt: omit.From(time.Now()),
				UpdatedBy: omit.From(s.execUser(ctx)),
			},
		)
		return err
	}); txErr != nil {
		l.Error("failed to remove season driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to remove season driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season driver removed")
	return connect.NewResponse(&v1.RemoveSeasonDriverResponse{}), nil
}

//nolint:whitespace,dupl // editor/linter issue
func (s *service) DeleteSeasonDriver(
	ctx context.Context,
	req *connect.Request[v1.DeleteSeasonDriverRequest]) (
	*connect.Response[v1.DeleteSeasonDriverResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteSeasonDriver")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		return s.repo.Drivers().SeasonDrivers().DeleteByID(
			ctx,
			int32(req.Msg.GetId()),
		)
	}); txErr != nil {
		l.Error("failed to delete season driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete season driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season driver deleted")
	return connect.NewResponse(&v1.DeleteSeasonDriverResponse{}), nil
}
