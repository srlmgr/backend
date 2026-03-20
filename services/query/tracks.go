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

// ListTracks returns a list of all tracks.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListTracks(
	ctx context.Context,
	req *connect.Request[queryv1.ListTracksRequest],
) (*connect.Response[queryv1.ListTracksResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListTracks")

	trackItems, err := s.repo.Tracks().Tracks().LoadAll(ctx)
	if err != nil {
		l.Error("failed to load tracks", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load tracks")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "tracks loaded")

	items := make([]*commonv1.Track, 0, len(trackItems))
	for _, item := range trackItems {
		if converted := s.conversion.TrackToTrack(item); converted != nil {
			items = append(items, converted)
		}
	}

	return connect.NewResponse(&queryv1.ListTracksResponse{
		Items: items,
	}), nil
}

// GetTrack returns a track by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetTrack(
	ctx context.Context,
	req *connect.Request[queryv1.GetTrackRequest],
) (*connect.Response[queryv1.GetTrackResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetTrack", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Tracks().Tracks().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load track", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load track")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track loaded")

	return connect.NewResponse(&queryv1.GetTrackResponse{
		Track: s.conversion.TrackToTrack(item),
	}), nil
}

// ListTrackLayouts returns a list of track layouts, optionally filtered by track ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListTrackLayouts(
	ctx context.Context,
	req *connect.Request[queryv1.ListTrackLayoutsRequest],
) (*connect.Response[queryv1.ListTrackLayoutsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListTrackLayouts", log.Uint32("track_id", req.Msg.GetTrackId()))

	layoutsRepo := s.repo.Tracks().TrackLayouts()

	var (
		layoutItems []*models.TrackLayout
		err         error
	)

	if trackID := int32(req.Msg.GetTrackId()); trackID != 0 {
		layoutItems, err = layoutsRepo.LoadByTrackID(ctx, trackID)
	} else {
		layoutItems, err = layoutsRepo.LoadAll(ctx)
	}

	if err != nil {
		l.Error("failed to load track layouts", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load track layouts")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track layouts loaded")

	items := make([]*commonv1.TrackLayout, 0, len(layoutItems))
	for _, item := range layoutItems {
		if converted := s.conversion.TrackLayoutToTrackLayout(item); converted != nil {
			items = append(items, converted)
		}
	}

	return connect.NewResponse(&queryv1.ListTrackLayoutsResponse{
		Items: items,
	}), nil
}

// GetTrackLayout returns a track layout by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetTrackLayout(
	ctx context.Context,
	req *connect.Request[queryv1.GetTrackLayoutRequest],
) (*connect.Response[queryv1.GetTrackLayoutResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetTrackLayout", log.Uint32("id", req.Msg.GetId()))

	item, err := s.repo.Tracks().TrackLayouts().LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load track layout", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load track layout")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "track layout loaded")

	return connect.NewResponse(&queryv1.GetTrackLayoutResponse{
		TrackLayout: s.conversion.TrackLayoutToTrackLayout(item),
	}), nil
}
