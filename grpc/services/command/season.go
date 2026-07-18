//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"slices"
	"strings"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type seasonRequest interface {
	GetName() string
	GetPointSystemId() uint32
	GetHasTeams() bool
	GetIsTeamBased() bool
	GetIsMulticlass() bool
	GetSkipEvents() int32
	GetTeamPointsTopN() int32
	GetStatus() string
	GetStartsAt() *timestamppb.Timestamp
	GetEndsAt() *timestamppb.Timestamp
}

type seasonSetter = models.SeasonSetter

type seasonSetterBuilder struct{}

func (b seasonSetterBuilder) Build(msg seasonRequest) *seasonSetter {
	setter := &seasonSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if pointSystemID := msg.GetPointSystemId(); pointSystemID != 0 {
		setter.PointSystemID = omit.From(int32(pointSystemID))
	}

	setter.HasTeams = omit.From(msg.GetHasTeams())
	setter.IsTeamBased = omit.From(msg.GetIsTeamBased())
	setter.IsMulticlass = omit.From(msg.GetIsMulticlass())

	if skipEvents := msg.GetSkipEvents(); skipEvents != 0 {
		setter.SkipEvents = omit.From(skipEvents)
	}

	if teamPointsTopN := msg.GetTeamPointsTopN(); teamPointsTopN != 0 {
		setter.TeamPointsTopN = omitnull.From(teamPointsTopN)
	}

	if status := msg.GetStatus(); status != "" {
		setter.Status = omit.From(status)
	}

	if startsAt := msg.GetStartsAt(); startsAt != nil {
		setter.StartsAt = omitnull.From(startsAt.AsTime())
	}

	if endsAt := msg.GetEndsAt(); endsAt != nil {
		setter.EndsAt = omitnull.From(endsAt.AsTime())
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateSeason(
	ctx context.Context,
	req *connect.Request[v1.CreateSeasonRequest]) (
	*connect.Response[v1.CreateSeasonResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateSeason")
	setter := (seasonSetterBuilder{}).Build(req.Msg)

	var newSeason *models.Season
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.SeriesID = omit.From(int32(req.Msg.GetSeriesId()))
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSeason, err = s.repo.Seasons().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season created")
	return connect.NewResponse(&v1.CreateSeasonResponse{
		Season: s.conversion.SeasonToSeason(newSeason),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateSeason(
	ctx context.Context,
	req *connect.Request[v1.UpdateSeasonRequest]) (
	*connect.Response[v1.UpdateSeasonResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateSeason")
	setter := (seasonSetterBuilder{}).Build(req.Msg)

	var newSeason *models.Season
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newSeason, err = s.repo.Seasons().Update(
			ctx,
			int32(req.Msg.GetSeasonId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season updated")
	return connect.NewResponse(&v1.UpdateSeasonResponse{
		Season: s.conversion.SeasonToSeason(newSeason),
	}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) SetSeasonCarClasses(
	ctx context.Context,
	req *connect.Request[v1.SetSeasonCarClassesRequest]) (
	*connect.Response[v1.SetSeasonCarClassesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetSeasonCarClasses")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		toSetCarModelVariantIDs := make([]int32, 0)
		collectCarModelVariants := func() error {
			if err := s.repo.Seasons().SetCarModelVariants(
				ctx, int32(req.Msg.GetSeasonId()), []int32{}); err != nil {
				return err
			}

			for _, carClassID := range lo.Uniq(req.Msg.GetCarClassIds()) {
				carModelVariants, err := s.repo.Cars().CarModelVariants().LoadByCarClassID(
					ctx, int32(carClassID))
				if err != nil {
					return err
				}
				slices.SortStableFunc(carModelVariants, func(a, b *models.CarModelVariant) int {
					return strings.Compare(a.Name, b.Name)
				})
				for _, cmv := range carModelVariants {
					toSetCarModelVariantIDs = append(toSetCarModelVariantIDs, cmv.ID)
				}
			}
			return nil
		}
		switch req.Msg.UpdateMode {
		case v1.SeasonCarModelsUpdateMode_SEASON_CAR_MODELS_UPDATE_MODE_UNSPECIFIED:
			// do nothing
		case v1.SeasonCarModelsUpdateMode_SEASON_CAR_MODELS_UPDATE_MODE_REPLACE:
			// collect new carModels by provided carClasses
			if err := collectCarModelVariants(); err != nil {
				return err
			}
		case v1.SeasonCarModelsUpdateMode_SEASON_CAR_MODELS_UPDATE_MODE_INSERT_WHEN_EMPTY:
			existingCarModelVariants, err := s.repo.Cars().CarModelVariants().LoadBySeasonID(
				ctx, int32(req.Msg.GetSeasonId()))
			if err != nil {
				return err
			}
			if len(existingCarModelVariants) == 0 {
				if err := collectCarModelVariants(); err != nil {
					return err
				}
			}
		}
		if len(toSetCarModelVariantIDs) > 0 {
			if err := s.repo.Seasons().SetCarModelVariants(
				ctx,
				int32(req.Msg.GetSeasonId()),
				toSetCarModelVariantIDs,
			); err != nil {
				return err
			}
		}
		return s.repo.Seasons().SetCarClasses(
			ctx,
			int32(req.Msg.GetSeasonId()),
			lo.Map(lo.Uniq(req.Msg.GetCarClassIds()),
				func(id uint32, _ int) int32 { return int32(id) }),
		)
	}); txErr != nil {
		l.Error("failed to set car classes for season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to set car classes for season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car classes set for season")
	return connect.NewResponse(&v1.SetSeasonCarClassesResponse{}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) SetSeasonCarModelVariants(
	ctx context.Context,
	req *connect.Request[v1.SetSeasonCarModelVariantsRequest]) (
	*connect.Response[v1.SetSeasonCarModelVariantsResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetSeasonCarModelVariants")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		return s.repo.Seasons().SetCarModelVariants(
			ctx,
			int32(req.Msg.GetSeasonId()),
			lo.Map(lo.Uniq(req.Msg.GetCarModelVariantIds()),
				func(id uint32, _ int) int32 { return int32(id) }),
		)
	}); txErr != nil {
		l.Error("failed to set car model variants for season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to set car model variants for season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "car model variants set for season")
	return connect.NewResponse(&v1.SetSeasonCarModelVariantsResponse{}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteSeason(
	ctx context.Context,
	req *connect.Request[v1.DeleteSeasonRequest]) (
	*connect.Response[v1.DeleteSeasonResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteSeason")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Seasons().DeleteByID(
			ctx,
			int32(req.Msg.GetSeasonId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete season", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season deleted")
	return connect.NewResponse(&v1.DeleteSeasonResponse{
		Deleted: true,
	}), nil
}
