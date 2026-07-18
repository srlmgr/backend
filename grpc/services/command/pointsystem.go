package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type pointSystemRequest interface {
	GetName() string
	GetDescription() string
	GetEligibility() *commonv1.PointEligibility
}

type pointSystemSetter = models.PointSystemSetter

type pointSystemSetterBuilder struct{}

func (b pointSystemSetterBuilder) Build(msg pointSystemRequest) *pointSystemSetter {
	setter := &pointSystemSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if description := msg.GetDescription(); description != "" {
		setter.Description = omitnull.From(description)
	}

	if eligibility := msg.GetEligibility(); eligibility != nil {
		setter.GuestPoints = omit.From(eligibility.GetGuests())
		setter.RaceDistancePCT = omit.From(decimal.NewFromFloat(
			eligibility.GetMinRaceDistancePercent(),
		))
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) replacePointRules(
	ctx context.Context,
	pointSystemID int32,
	raceSettings []*commonv1.PointRaceSettings,
) error {
	if err := s.repo.PointSystems().PointRules().DeleteByPointSystemID(
		ctx,
		pointSystemID,
	); err != nil {
		return err
	}

	user := s.execUser(ctx)
	for raceNo, raceSetting := range raceSettings {
		if raceSetting == nil {
			continue
		}

		for _, policy := range raceSetting.GetPolicies() {
			if policy == nil {
				continue
			}

			metadata, err := s.conversion.MarshalPointRuleMetadata(raceSetting.GetName(), policy)
			if err != nil {
				return err
			}

			_, err = s.repo.PointSystems().PointRules().Create(ctx, &models.PointRuleSetter{
				PointSystemID: omit.From(pointSystemID),
				RaceNo:        omit.From(int32(raceNo)),
				PointPolicy:   omit.From(policy.GetName().String()),
				MetadataJSON:  omit.From(metadata),
				CreatedBy:     omit.From(user),
				UpdatedBy:     omit.From(user),
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//nolint:whitespace // editor/linter issue
func (s *service) CreatePointSystem(
	ctx context.Context,
	req *connect.Request[v1.CreatePointSystemRequest]) (
	*connect.Response[v1.CreatePointSystemResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreatePointSystem")
	setter := (pointSystemSetterBuilder{}).Build(req.Msg)

	var newPointSystem *models.PointSystem
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newPointSystem, err = s.repo.PointSystems().PointSystems().Create(ctx, setter)
		if err != nil {
			return err
		}
		return s.replacePointRules(ctx, newPointSystem.ID, req.Msg.GetRaceSettings())
	}); txErr != nil {
		l.Error("failed to create point system", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	loadedPointSystem, loadErr := s.repo.PointSystems().
		PointSystems().
		LoadByID(ctx, newPointSystem.ID)
	if loadErr != nil {
		l.Error("failed to reload point system", log.ErrorField(loadErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to reload point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(loadErr), loadErr)
	}
	newPointSystem = loadedPointSystem

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system created")
	return connect.NewResponse(&v1.CreatePointSystemResponse{
		PointSystem: s.conversion.PointSystemToPointSystem(newPointSystem),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdatePointSystem(
	ctx context.Context,
	req *connect.Request[v1.UpdatePointSystemRequest]) (
	*connect.Response[v1.UpdatePointSystemResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdatePointSystem")
	setter := (pointSystemSetterBuilder{}).Build(req.Msg)

	var newPointSystem *models.PointSystem
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newPointSystem, err = s.repo.PointSystems().PointSystems().Update(
			ctx,
			int32(req.Msg.GetPointSystemId()),
			setter,
		)
		if err != nil {
			return err
		}
		if req.Msg.RaceSettings != nil {
			return s.replacePointRules(ctx, newPointSystem.ID, req.Msg.RaceSettings)
		}
		return nil
	}); txErr != nil {
		l.Error("failed to update point system", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	loadedPointSystem, loadErr := s.repo.PointSystems().
		PointSystems().
		LoadByID(ctx, newPointSystem.ID)
	if loadErr != nil {
		l.Error("failed to reload point system", log.ErrorField(loadErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to reload point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(loadErr), loadErr)
	}
	newPointSystem = loadedPointSystem

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system updated")
	return connect.NewResponse(&v1.UpdatePointSystemResponse{
		PointSystem: s.conversion.PointSystemToPointSystem(newPointSystem),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeletePointSystem(
	ctx context.Context,
	req *connect.Request[v1.DeletePointSystemRequest]) (
	*connect.Response[v1.DeletePointSystemResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeletePointSystem")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		if delErr := s.repo.PointSystems().PointRules().DeleteByPointSystemID(
			ctx,
			int32(req.Msg.GetPointSystemId()),
		); delErr != nil {
			return delErr
		}

		err = s.repo.PointSystems().PointSystems().DeleteByID(
			ctx,
			int32(req.Msg.GetPointSystemId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete point system", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete point system")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "point system deleted")
	return connect.NewResponse(&v1.DeletePointSystemResponse{
		Deleted: true,
	}), nil
}
