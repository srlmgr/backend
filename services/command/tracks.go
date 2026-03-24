//nolint:dupl // crud operations are very similar across entities
package command

import (
	"context"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

type trackRequest interface {
	GetName() string
	GetCountry() string
	GetLatitude() float64
	GetLongitude() float64
	GetWebsiteUrl() string
}

type trackSetter = models.TrackSetter

type trackSetterBuilder struct{}

func (b trackSetterBuilder) Build(msg trackRequest) *trackSetter {
	setter := &trackSetter{}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if country := msg.GetCountry(); country != "" {
		setter.Country = omitnull.From(country)
	}

	if lat := msg.GetLatitude(); lat != 0 {
		setter.Latitude = omitnull.From(decimal.NewFromFloat(lat))
	}

	if lon := msg.GetLongitude(); lon != 0 {
		setter.Longitude = omitnull.From(decimal.NewFromFloat(lon))
	}

	if websiteURL := msg.GetWebsiteUrl(); websiteURL != "" {
		setter.WebsiteURL = omitnull.From(websiteURL)
	}

	return setter
}

type trackLayoutRequest interface {
	GetTrackId() uint32
	GetName() string
	GetLengthMeters() int32
	GetLayoutImageUrl() string
}

type trackLayoutSetter = models.TrackLayoutSetter

type trackLayoutSetterBuilder struct{}

func (b trackLayoutSetterBuilder) Build(msg trackLayoutRequest) *trackLayoutSetter {
	setter := &trackLayoutSetter{}

	if trackID := msg.GetTrackId(); trackID != 0 {
		setter.TrackID = omit.From(int32(trackID))
	}

	if name := msg.GetName(); name != "" {
		setter.Name = omit.From(name)
	}

	if lengthMeters := msg.GetLengthMeters(); lengthMeters != 0 {
		setter.LengthMeters = omitnull.From(lengthMeters)
	}

	if layoutImageURL := msg.GetLayoutImageUrl(); layoutImageURL != "" {
		setter.LayoutImageURL = omitnull.From(layoutImageURL)
	}

	return setter
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateTrack(
	ctx context.Context,
	req *connect.Request[v1.CreateTrackRequest]) (
	*connect.Response[v1.CreateTrackResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateTrack")
	setter := (trackSetterBuilder{}).Build(req.Msg)

	var newTrack *models.Track
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTrack, err = s.repo.Tracks().Tracks().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create track", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create track")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track created")
	return connect.NewResponse(&v1.CreateTrackResponse{
		Track: s.conversion.TrackToTrack(newTrack),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateTrack(
	ctx context.Context,
	req *connect.Request[v1.UpdateTrackRequest]) (
	*connect.Response[v1.UpdateTrackResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateTrack")
	setter := (trackSetterBuilder{}).Build(req.Msg)

	var newTrack *models.Track
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTrack, err = s.repo.Tracks().Tracks().Update(
			ctx,
			int32(req.Msg.GetTrackId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update track", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update track")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track updated")
	return connect.NewResponse(&v1.UpdateTrackResponse{
		Track: s.conversion.TrackToTrack(newTrack),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteTrack(
	ctx context.Context,
	req *connect.Request[v1.DeleteTrackRequest]) (
	*connect.Response[v1.DeleteTrackResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteTrack")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Tracks().Tracks().DeleteByID(
			ctx,
			int32(req.Msg.GetTrackId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete track", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete track")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track deleted")
	return connect.NewResponse(&v1.DeleteTrackResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) CreateTrackLayout(
	ctx context.Context,
	req *connect.Request[v1.CreateTrackLayoutRequest]) (
	*connect.Response[v1.CreateTrackLayoutResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("CreateTrackLayout")
	setter := (trackLayoutSetterBuilder{}).Build(req.Msg)

	var newTrackLayout *models.TrackLayout
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.CreatedBy = omit.From(s.execUser(ctx))
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTrackLayout, err = s.repo.Tracks().TrackLayouts().Create(ctx, setter)
		return err
	}); txErr != nil {
		l.Error("failed to create track layout", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to create track layout")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track layout created")
	return connect.NewResponse(&v1.CreateTrackLayoutResponse{
		TrackLayout: s.conversion.TrackLayoutToTrackLayout(newTrackLayout),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) UpdateTrackLayout(
	ctx context.Context,
	req *connect.Request[v1.UpdateTrackLayoutRequest]) (
	*connect.Response[v1.UpdateTrackLayoutResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("UpdateTrackLayout")
	setter := (trackLayoutSetterBuilder{}).Build(req.Msg)

	var newTrackLayout *models.TrackLayout
	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		setter.UpdatedAt = omit.From(time.Now())
		setter.UpdatedBy = omit.From(s.execUser(ctx))
		newTrackLayout, err = s.repo.Tracks().TrackLayouts().Update(
			ctx,
			int32(req.Msg.GetTrackLayoutId()),
			setter,
		)
		return err
	}); txErr != nil {
		l.Error("failed to update track layout", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to update track layout")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track layout updated")
	return connect.NewResponse(&v1.UpdateTrackLayoutResponse{
		TrackLayout: s.conversion.TrackLayoutToTrackLayout(newTrackLayout),
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) DeleteTrackLayout(
	ctx context.Context,
	req *connect.Request[v1.DeleteTrackLayoutRequest]) (
	*connect.Response[v1.DeleteTrackLayoutResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("DeleteTrackLayout")

	if txErr := s.withTx(ctx, func(ctx context.Context) (err error) {
		err = s.repo.Tracks().TrackLayouts().DeleteByID(
			ctx,
			int32(req.Msg.GetTrackLayoutId()),
		)
		return err
	}); txErr != nil {
		l.Error("failed to delete track layout", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to delete track layout")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track layout deleted")
	return connect.NewResponse(&v1.DeleteTrackLayoutResponse{
		Deleted: true,
	}), nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) SetSimulationTrackLayoutAliases(
	ctx context.Context,
	req *connect.Request[v1.SetSimulationTrackLayoutAliasesRequest]) (
	*connect.Response[v1.SetSimulationTrackLayoutAliasesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("SetSimulationTrackLayoutAliases")

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		trackLayoutID := int32(req.Msg.GetTrackLayoutId())
		simulationID := int32(req.Msg.GetSimulationId())

		existing, err := s.repo.Tracks().
			SimulationTrackLayoutAliases().
			LoadBySimulationID(ctx, simulationID)
		if err != nil {
			return err
		}

		for _, alias := range existing {
			if alias.TrackLayoutID != trackLayoutID {
				continue
			}
			if err := s.repo.Tracks().
				SimulationTrackLayoutAliases().
				DeleteByID(ctx, alias.ID); err != nil {
				return err
			}
		}

		user := s.execUser(ctx)
		for _, externalName := range req.Msg.GetExternalName() {
			_, err := s.repo.Tracks().
				SimulationTrackLayoutAliases().
				Create(ctx, &models.SimulationTrackLayoutAliasSetter{
					TrackLayoutID: omit.From(trackLayoutID),
					SimulationID:  omit.From(simulationID),
					ExternalName:  omit.From(externalName),
					CreatedBy:     omit.From(user),
					UpdatedBy:     omit.From(user),
				})
			if err != nil {
				return err
			}
		}

		return nil
	}); txErr != nil {
		l.Error("failed to set simulation track layout aliases", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).
			SetStatus(codes.Error, "failed to set simulation track layout aliases")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulation track layout aliases set")
	return connect.NewResponse(&v1.SetSimulationTrackLayoutAliasesResponse{
		Updated: true,
	}), nil
}
