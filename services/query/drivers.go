package query

import (
	"context"
	"sort"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

// ListDrivers returns a list of drivers.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListDrivers(
	ctx context.Context,
	req *connect.Request[queryv1.ListDriversRequest],
) (*connect.Response[queryv1.ListDriversResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListDrivers")
	var driverItems []*models.Driver
	var err error
	switch req.Msg.Filter.(type) {
	case *queryv1.ListDriversRequest_SeasonId:
		l.Debug("ListDrivers by seasonID",
			log.Uint32("seasonID", req.Msg.GetSeasonId()))
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	case *queryv1.ListDriversRequest_TeamId:
		l.Debug("ListDrivers by teamID",
			log.Uint32("teamID", req.Msg.GetTeamId()))
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	case *queryv1.ListDriversRequest_MultipleDrivers:
		ids := lo.Map(req.Msg.GetMultipleDrivers().GetDriverIds(),
			func(id uint32, _ int) int32 {
				return int32(id)
			})
		driverItems, err = s.repo.Drivers().Drivers().LoadByIDs(ctx, ids)
	default:
		driverItems, err = s.repo.Drivers().Drivers().LoadAll(ctx)
	}
	if err != nil {
		l.Error("failed to load drivers", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load drivers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	items := make([]*commonv1.Driver, 0, len(driverItems))
	for _, item := range driverItems {
		if converted := s.conversion.DriverToDriver(item); converted != nil {
			items = append(items, converted)
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "drivers loaded")
	return connect.NewResponse(&queryv1.ListDriversResponse{Items: items}), nil
}

// GetDriver returns a driver by ID.
//
//nolint:whitespace // editor/linter issue
func (s *service) GetDriver(
	ctx context.Context,
	req *connect.Request[queryv1.GetDriverRequest],
) (*connect.Response[queryv1.GetDriverResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetDriver", log.Uint32("id", req.Msg.GetId()))

	driverRepo := s.repo.Drivers().Drivers()
	aliasRepo := s.repo.Drivers().SimulationDriverAliases()
	var (
		item *models.Driver
		err  error
	)
	item, err = driverRepo.LoadByID(ctx, int32(req.Msg.GetId()))
	if err != nil {
		l.Error("failed to load driver", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load driver")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "driver loaded")
	response := &queryv1.GetDriverResponse{
		Driver: s.conversion.DriverToDriver(item),
	}
	if req.Msg.GetIncludeAliases() {
		aliases, err := aliasRepo.GetDriverAliases(ctx, int32(req.Msg.GetId()))
		if err != nil {
			l.Error("failed to load driver aliases", log.ErrorField(err))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load driver aliases")
			return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
		}
		response.Aliases = simulationDriverAliasesToProto(aliases)
	}
	return connect.NewResponse(response), nil
}

//nolint:whitespace // editor/linter issue
func simulationDriverAliasesToProto(
	items []*models.SimulationDriverAlias,
) []*commonv1.DriverAlias {
	if len(items) == 0 {
		return nil
	}

	grouped := lo.GroupBy(items, func(item *models.SimulationDriverAlias) int32 {
		return item.SimulationID
	})

	simulationIDs := make([]int32, 0, len(grouped))
	for simulationID := range grouped {
		simulationIDs = append(simulationIDs, simulationID)
	}
	sort.Slice(simulationIDs, func(i, j int) bool {
		return simulationIDs[i] < simulationIDs[j]
	})

	aliases := make([]*commonv1.DriverAlias, 0, len(grouped))
	for _, simulationID := range simulationIDs {
		group := grouped[simulationID]
		groupAliases := make([]string, 0, len(group))
		for _, item := range group {
			groupAliases = append(groupAliases, item.SimulationDriverID)
		}
		sort.Strings(groupAliases)
		aliases = append(aliases, &commonv1.DriverAlias{
			SimulationId:       uint32(simulationID),
			SimulationDriverId: groupAliases,
		})
	}

	return aliases
}
