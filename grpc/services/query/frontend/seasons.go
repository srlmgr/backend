// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

// ListSeasonsOverview returns active seasons together with their referenced
// series and simulations.
//
//nolint:whitespace,gocyclo,funlen // editor/linter issue
func (s *service) ListSeasonsOverview(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonsOverviewRequest],
) (*connect.Response[queryv1.ListSeasonsOverviewResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListSeasonsOverview",
		log.Bool("include_inactive", req.Msg.GetIncludeInactive()))

	seasonsRepo := s.repo.Seasons()

	seasons, err := func() ([]*commonv1.Season, error) {
		if req.Msg.GetIncludeInactive() {
			items, loadErr := seasonsRepo.LoadAll(ctx)
			if loadErr != nil {
				return nil, loadErr
			}
			out := make([]*commonv1.Season, 0, len(items))
			for _, item := range items {
				if converted := s.conversion.SeasonToSeason(item); converted != nil {
					out = append(out, converted)
				}
			}
			return out, nil
		}

		items, loadErr := seasonsRepo.LoadActiveAt(ctx, time.Now())
		if loadErr != nil {
			return nil, loadErr
		}
		out := make([]*commonv1.Season, 0, len(items))
		for _, item := range items {
			if converted := s.conversion.SeasonToSeason(item); converted != nil {
				out = append(out, converted)
			}
		}
		return out, nil
	}()
	if err != nil {
		l.Error("failed to load seasons", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load seasons")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	// Collect unique series IDs from seasons.
	seriesIDSet := make(map[int32]struct{}, len(seasons))
	for _, season := range seasons {
		seriesIDSet[int32(season.GetSeriesId())] = struct{}{}
	}

	seriesRepo := s.repo.Series()
	seriesItems := make([]*commonv1.Series, 0, len(seriesIDSet))
	simulationIDSet := make(map[int32]struct{})
	seriesDBItems, seriesErr := seriesRepo.LoadByIDs(ctx, lo.Keys(seriesIDSet))
	if seriesErr != nil {
		l.Error(
			"failed to load series",
			log.ErrorField(seriesErr),
		)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load series")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(seriesErr), seriesErr)
	}

	for _, item := range seriesDBItems {
		if converted := s.conversion.SeriesToSeries(item); converted != nil {
			seriesItems = append(seriesItems, converted)
		}
		simulationIDSet[item.SimulationID] = struct{}{}
	}

	// Load simulations (racing sims) referenced by the series.
	racingSimsRepo := s.repo.RacingSims()
	simulations := make([]*commonv1.Simulation, 0, len(simulationIDSet))
	racingSimsDBItems, racingSimsErr := racingSimsRepo.LoadByIDs(
		ctx, lo.Keys(simulationIDSet))
	if racingSimsErr != nil {
		l.Error(
			"failed to load simulations",
			log.ErrorField(racingSimsErr),
		)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load simulations")
		return nil, connect.NewError(
			s.conversion.MapErrorToRPCCode(racingSimsErr), racingSimsErr)
	}
	for _, item := range racingSimsDBItems {
		if converted := s.conversion.RacingSimToSimulation(item); converted != nil {
			simulations = append(simulations, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "seasons overview loaded")
	return connect.NewResponse(&queryv1.ListSeasonsOverviewResponse{
		Seasons:     seasons,
		Series:      seriesItems,
		Simulations: simulations,
	}), nil
}
