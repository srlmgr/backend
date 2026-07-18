// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

// ListSeasonEvents returns season events including resolved track layouts and tracks.
//
//nolint:whitespace,gocyclo,funlen // editor/linter issue
func (s *service) ListSeasonEvents(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonEventsRequest],
) (*connect.Response[queryv1.ListSeasonEventsResponse], error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.Msg.GetSeasonId())
	l.Debug("ListSeasonEvents", log.Int32("season_id", seasonID))

	events, err := s.repo.Events().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season events", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season events")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trackLayoutIDSet := make(map[int32]struct{}, len(events))
	for _, event := range events {
		trackLayoutIDSet[event.TrackLayoutID] = struct{}{}
	}

	trackLayoutIDs := make([]int32, 0, len(trackLayoutIDSet))
	for id := range trackLayoutIDSet {
		trackLayoutIDs = append(trackLayoutIDs, id)
	}

	trackLayoutByID := make(map[int32]*commonv1.TrackLayout)
	trackIDSet := make(map[int32]struct{}, len(trackLayoutIDSet))

	if len(trackLayoutIDs) > 0 {
		dbTrackLayouts, loadTrackLayoutsErr := s.repo.Tracks().
			TrackLayouts().
			LoadByIDs(ctx, trackLayoutIDs)
		if loadTrackLayoutsErr != nil {
			l.Error("failed to load track layouts", log.ErrorField(loadTrackLayoutsErr))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load track layouts")
			return nil, connect.NewError(
				s.conversion.MapErrorToRPCCode(loadTrackLayoutsErr),
				loadTrackLayoutsErr,
			)
		}

		for _, item := range dbTrackLayouts {
			trackIDSet[item.TrackID] = struct{}{}
			if converted := s.conversion.TrackLayoutToTrackLayout(item); converted != nil {
				trackLayoutByID[item.ID] = converted
			}
		}
	}

	trackIDs := make([]int32, 0, len(trackIDSet))
	for id := range trackIDSet {
		trackIDs = append(trackIDs, id)
	}

	trackByID := make(map[int32]*commonv1.Track)
	if len(trackIDs) > 0 {
		dbTracks, loadTracksErr := s.repo.Tracks().Tracks().LoadByIDs(ctx, trackIDs)
		if loadTracksErr != nil {
			l.Error("failed to load tracks", log.ErrorField(loadTracksErr))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load tracks")
			return nil, connect.NewError(
				s.conversion.MapErrorToRPCCode(loadTracksErr),
				loadTracksErr,
			)
		}

		for _, item := range dbTracks {
			if converted := s.conversion.TrackToTrack(item); converted != nil {
				trackByID[item.ID] = converted
			}
		}
	}

	season, seasonErr := s.repo.Seasons().LoadByID(ctx, seasonID)
	if seasonErr != nil {
		l.Error(
			"failed to load season",
			log.ErrorField(seasonErr),
			log.Int32("season_id", seasonID),
		)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(seasonErr), seasonErr)
	}

	seriesItems, seriesErr := s.repo.Series().LoadByIDs(ctx, []int32{season.SeriesID})
	if seriesErr != nil {
		l.Error(
			"failed to load series",
			log.ErrorField(seriesErr),
			log.Int32("series_id", season.SeriesID),
		)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load series")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(seriesErr), seriesErr)
	}

	var seriesProto *commonv1.Series
	if len(seriesItems) > 0 {
		seriesProto = s.conversion.SeriesToSeries(seriesItems[0])
	}

	eventContainers := make([]*queryv1.EventContainer, 0, len(events))
	for _, event := range events {
		layout := trackLayoutByID[event.TrackLayoutID]
		var track *commonv1.Track
		if layout != nil {
			track = trackByID[int32(layout.GetTrackId())]
		}

		eventContainers = append(eventContainers, &queryv1.EventContainer{
			Event:       s.conversion.EventToEvent(event),
			TrackLayout: layout,
			Track:       track,
		})
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season events loaded")
	return connect.NewResponse(&queryv1.ListSeasonEventsResponse{
		Events: eventContainers,
		Season: s.conversion.SeasonToSeason(season),
		Series: seriesProto,
	}), nil
}
