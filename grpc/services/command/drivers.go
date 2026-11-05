//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"github.com/aarondl/opt/omit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type driverRequest interface {
	GetExternalId() string
	GetName() string
	GetIsActive() bool
}

type driverSetter = models.DriverSetter

type driverSetterBuilder struct{}

func (b driverSetterBuilder) Build(msg driverRequest) *driverSetter {
	setter := &driverSetter{}

	if externalID := msg.GetExternalId(); externalID != "" {
		setter.ExternalID = omit.From(externalID)
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	setter.IsActive = omit.From(msg.GetIsActive())

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateDriver(
	ctx context.Context,
	req *v1.CreateDriverRequest,
) (*v1.CreateDriverResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateDriver")
	setter := (driverSetterBuilder{}).Build(req)

	var newDriver *models.Driver
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newDriver, err = s.repo.Drivers().Drivers().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create driver")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver created")
	return &v1.CreateDriverResponse{
		Driver: s.conversion.DriverToDriver(newDriver),
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateDriver(
	ctx context.Context,
	req *v1.UpdateDriverRequest,
) (*v1.UpdateDriverResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateDriver")
	setter := (driverSetterBuilder{}).Build(req)

	var newDriver *models.Driver
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newDriver, err = s.repo.Drivers().Drivers().Update(
			ctx,
			int32(req.GetDriverId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update driver")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver updated")
	return &v1.UpdateDriverResponse{
		Driver: s.conversion.DriverToDriver(newDriver),
	}, nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteDriver(
	ctx context.Context,
	req *v1.DeleteDriverRequest,
) (*v1.DeleteDriverResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteDriver")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Drivers().Drivers().DeleteByID(
			ctx,
			int32(req.GetDriverId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete driver", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete driver")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver deleted")
	return &v1.DeleteDriverResponse{
		Deleted: true,
	}, nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) SetSimulationDriverAliases(
	ctx context.Context,
	req *v1.SetSimulationDriverAliasesRequest,
) (*v1.SetSimulationDriverAliasesResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetSimulationDriverAliases")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		driverID := int32(req.GetDriverId())
		simulationID := int32(req.GetSimulationId())

		existing, err := s.repo.Drivers().SimulationDriverAliases().GetDriverAliases(
			ctx,
			driverID,
		)
		if err != nil {
			return err
		}

		for _, alias := range existing {
			if alias.SimulationID != simulationID {
				continue
			}
			if err := s.repo.Drivers().
				SimulationDriverAliases().
				DeleteByID(ctx, alias.ID); err != nil {
				return err
			}
		}

		user := s.execUser(ctx)
		for _, simulationDriverID := range req.GetSimulationDriverId() {
			_, err := s.repo.Drivers().
				SimulationDriverAliases().
				Create(ctx, &models.SimulationDriverAliasSetter{
					DriverID:           omit.From(driverID),
					SimulationID:       omit.From(simulationID),
					SimulationDriverID: omit.From(simulationDriverID),
					CreatedBy:          omit.From(user),
					UpdatedBy:          omit.From(user),
				})
			if err != nil {
				return err
			}
		}

		return nil
	}); txErr != nil {
		l.Error("failed to set simulation driver aliases", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to set simulation driver aliases",
		)
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation driver aliases set")
	return &v1.SetSimulationDriverAliasesResponse{Updated: true}, nil
}
