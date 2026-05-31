// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	"context"
	"sort"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

// ListSeasonDrivers returns season driver data with related driver and car data.
//
//nolint:whitespace,funlen // editor/linter issue
func (s *service) ListSeasonDrivers(
	ctx context.Context,
	req *connect.Request[queryv1.ListSeasonDriversRequest],
) (*connect.Response[queryv1.ListSeasonDriversResponse], error) {
	l := s.logger.WithCtx(ctx)
	seasonID := int32(req.Msg.GetSeasonId())
	l.Debug("ListSeasonDrivers", log.Int32("season_id", seasonID))

	seasonDrivers, err := s.repo.Drivers().SeasonDrivers().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load season drivers", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load season drivers")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	sort.Slice(seasonDrivers, func(i, j int) bool {
		return seasonDrivers[i].ID < seasonDrivers[j].ID
	})

	driverIDSet := make(map[int32]struct{}, len(seasonDrivers))
	carModelIDSet := make(map[int32]struct{}, len(seasonDrivers))
	seasonDriverProto := make([]*commonv1.SeasonDriver, 0, len(seasonDrivers))
	for _, item := range seasonDrivers {
		driverIDSet[item.DriverID] = struct{}{}
		carModelIDSet[item.CarModelID] = struct{}{}
		if converted := s.conversion.SeasonDriverToSeasonDriver(item); converted != nil {
			seasonDriverProto = append(seasonDriverProto, converted)
		}
	}

	driverItems, loadDriversErr := s.loadDriversByIDs(ctx, mapKeysSorted(driverIDSet))
	if loadDriversErr != nil {
		l.Error("failed to load drivers", log.ErrorField(loadDriversErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load drivers")
		return nil, connect.NewError(
			s.conversion.MapErrorToRPCCode(loadDriversErr),
			loadDriversErr,
		)
	}

	carData, loadCarDataErr := s.loadCarDataForSeasonDrivers(
		ctx, mapKeysSorted(carModelIDSet))
	if loadCarDataErr != nil {
		l.Error("failed to load car data", log.ErrorField(loadCarDataErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load car data")
		return nil, connect.NewError(
			s.conversion.MapErrorToRPCCode(loadCarDataErr),
			loadCarDataErr,
		)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season drivers loaded")
	return connect.NewResponse(&queryv1.ListSeasonDriversResponse{
		Items: []*queryv1.SeasonDriverContainer{{
			SeasonDrivers: seasonDriverProto,
			Drivers:       toDriverProto(s.conversion, driverItems),
			CarData:       carData,
		}},
	}), nil
}

//nolint:whitespace // editor/linter issue
func (s *service) loadDriversByIDs(
	ctx context.Context,
	driverIDs []int32,
) ([]*models.Driver, error) {
	if len(driverIDs) == 0 {
		return nil, nil
	}

	drivers, err := s.repo.Drivers().Drivers().LoadByIDs(ctx, driverIDs)
	if err != nil {
		return nil, err
	}

	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].ID < drivers[j].ID
	})
	return drivers, nil
}

//nolint:whitespace,funlen // editor/linter issue
func (s *service) loadCarDataForSeasonDrivers(
	ctx context.Context,
	carModelIDs []int32,
) ([]*queryv1.CarModelContainer, error) {
	if len(carModelIDs) == 0 {
		return nil, nil
	}

	carModelsRepo := s.repo.Cars().CarModels()
	carBrandsRepo := s.repo.Cars().CarBrands()
	carManufacturersRepo := s.repo.Cars().CarManufacturers()

	brandCache := make(map[int32]*models.CarBrand)
	manufacturerCache := make(map[int32]*models.CarManufacturer)
	containers := make([]*queryv1.CarModelContainer, 0, len(carModelIDs))

	for _, carModelID := range carModelIDs {
		carModel, err := carModelsRepo.LoadByID(ctx, carModelID)
		if err != nil {
			return nil, err
		}

		brand, ok := brandCache[carModel.BrandID]
		if !ok {
			brand, err = carBrandsRepo.LoadByID(ctx, carModel.BrandID)
			if err != nil {
				return nil, err
			}
			brandCache[carModel.BrandID] = brand
		}

		manufacturer, ok := manufacturerCache[brand.ManufacturerID]
		if !ok {
			manufacturer, err = carManufacturersRepo.LoadByID(ctx, brand.ManufacturerID)
			if err != nil {
				return nil, err
			}
			manufacturerCache[brand.ManufacturerID] = manufacturer
		}

		containers = append(containers, &queryv1.CarModelContainer{
			CarModel:        s.conversion.CarModelToCarModel(carModel),
			CarBrand:        s.conversion.CarBrandToCarBrand(brand),
			CarManufacturer: s.conversion.CarManufacturerToCarManufacturer(manufacturer),
		})
	}

	return containers, nil
}

//nolint:whitespace // editor/linter issue
func toDriverProto(
	conversionService conversionDriverToDriver,
	items []*models.Driver,
) []*commonv1.Driver {
	out := make([]*commonv1.Driver, 0, len(items))
	for _, item := range items {
		if converted := conversionService.DriverToDriver(item); converted != nil {
			out = append(out, converted)
		}
	}
	return out
}

type conversionDriverToDriver interface {
	DriverToDriver(model *models.Driver) *commonv1.Driver
}

func mapKeysSorted(values map[int32]struct{}) []int32 {
	ids := make([]int32, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}
