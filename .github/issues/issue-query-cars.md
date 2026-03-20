# Feature: Implement Query Service for Cars

## Summary

Implement the query service handlers for `CarManufacturer`, `CarBrand`, and `CarModel` in `services/query/cars.go`:

- `ListCarManufacturers`
- `GetCarManufacturer`
- `ListCarBrands`
- `GetCarBrand`
- `ListCarModels`
- `GetCarModel`

Also add missing `LoadAll` to `CarManufacturersRepository`, `LoadAll` and `LoadByManufacturerID` to `CarBrandsRepository`, and `LoadAll` and `LoadByManufacturerID` to `CarModelsRepository` in `repository/cars/cars.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires all six methods. The car hierarchy (manufacturer → brand → model) is reference data required to identify vehicles in race results and imports.

## Prerequisites

- Conversion functions `CarManufacturerToCarManufacturer`, `CarBrandToCarBrand`, and `CarModelToCarModel` from `issue-command-cars.md` must be available in `services/conversion/service.go`.

## Goals

- Extend `CarManufacturersRepository` with `LoadAll`.
- Extend `CarBrandsRepository` with `LoadAll` and `LoadByManufacturerID`.
- Extend `CarModelsRepository` with `LoadAll` and `LoadByManufacturerID`.
- Implement all six query handlers in `services/query/cars.go`.
- `ListCarManufacturers` has no proto filter – always call `LoadAll`.
- `ListCarBrands` accepts an optional `manufacturer_id` filter (`req.Msg.GetManufacturerId()`):
    - If non-zero, call `LoadByManufacturerID` on `CarBrandsRepository`.
    - Otherwise call `LoadAll`.
- `ListCarModels` accepts an optional `manufacturer_id` filter (`req.Msg.GetManufacturerId()`):
    - If non-zero, call `LoadByManufacturerID` on `CarModelsRepository`.
    - Otherwise call `LoadAll`.
- `GetCarManufacturer`, `GetCarBrand`, `GetCarModel` all resolve via `LoadByID` using `req.Msg.GetId()`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `SimulationCarAlias` query handlers – not in the current query proto.
- Command (write) handlers – covered by `issue-command-cars.md`.

## Implementation Notes

- `ListCarModelsRequest.GetManufacturerId()` filters by the manufacturer, not the brand. The `CarModel` DB model has a `BrandID` column, not `ManufacturerID`. To filter car models by manufacturer, a JOIN through the `car_brands` table is needed. Options:
    - Add a `LoadByManufacturerID` implementation that queries via a sub-select or JOIN on `brand_id IN (SELECT id FROM car_brands WHERE manufacturer_id = ?)`.
    - Alternatively, load all brands for the manufacturer first, then query models by those brand IDs.
    - The chosen approach should be reflected in the repository implementation with an appropriate `//nolint:lll` comment if the query gets long.

## Implementation Plan

1. **Extend `repository/cars/cars.go`**
    - Add `LoadAll(ctx context.Context) ([]*models.CarManufacturer, error)` to `CarManufacturersRepository`.
    - Add `LoadAll(ctx context.Context) ([]*models.CarBrand, error)` to `CarBrandsRepository`.
    - Add `LoadByManufacturerID(ctx context.Context, manufacturerID int32) ([]*models.CarBrand, error)` to `CarBrandsRepository`.
    - Add `LoadAll(ctx context.Context) ([]*models.CarModel, error)` to `CarModelsRepository`.
    - Add `LoadByManufacturerID(ctx context.Context, manufacturerID int32) ([]*models.CarModel, error)` to `CarModelsRepository`.
    - Implement each, following the `LoadAll` / `LoadBySimulationID` pattern from `repository/series/series.go`.

2. **Create `services/query/cars.go`**
    - Implement `ListCarManufacturers` – call `LoadAll`, return `ListCarManufacturersResponse`.
    - Implement `GetCarManufacturer` – call `LoadByID`, return `GetCarManufacturerResponse`.
    - Implement `ListCarBrands` – call `LoadByManufacturerID` or `LoadAll`, return `ListCarBrandsResponse`.
    - Implement `GetCarBrand` – call `LoadByID`, return `GetCarBrandResponse`.
    - Implement `ListCarModels` – call `LoadByManufacturerID` or `LoadAll`, return `ListCarModelsResponse`.
    - Implement `GetCarModel` – call `LoadByID`, return `GetCarModelResponse`.

3. **Create `services/query/cars_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` from `test_setup_test.go`.

    Local seed helpers (defined in this file):

    ```go
    func seedCarManufacturer(t *testing.T, repo rootrepo.Repository, name string) *models.CarManufacturer {
        t.Helper()
        m, err := repo.Cars().CarManufacturers().Create(context.Background(), &models.CarManufacturerSetter{
            Name:      omit.From(name),
            CreatedBy: omit.From(testUserSeed),
            UpdatedBy: omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed car manufacturer %q: %v", name, err)
        }
        return m
    }

    func seedCarBrand(t *testing.T, repo rootrepo.Repository, manufacturerID int32, name string) *models.CarBrand {
        t.Helper()
        b, err := repo.Cars().CarBrands().Create(context.Background(), &models.CarBrandSetter{
            ManufacturerID: omit.From(manufacturerID),
            Name:           omit.From(name),
            CreatedBy:      omit.From(testUserSeed),
            UpdatedBy:      omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed car brand %q: %v", name, err)
        }
        return b
    }

    func seedCarModel(t *testing.T, repo rootrepo.Repository, brandID int32, name string) *models.CarModel {
        t.Helper()
        cm, err := repo.Cars().CarModels().Create(context.Background(), &models.CarModelSetter{
            BrandID:   omit.From(brandID),
            Name:      omit.From(name),
            CreatedBy: omit.From(testUserSeed),
            UpdatedBy: omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed car model %q: %v", name, err)
        }
        return cm
    }
    ```

    Tests for `ListCarManufacturers`:
    - `TestListCarManufacturersEmpty` — seeds nothing; verifies `GetItems()` is empty.
    - `TestListCarManufacturersReturnsAll` — seeds 2 manufacturers; verifies both appear in the response.

    Tests for `GetCarManufacturer`:
    - `TestGetCarManufacturerSuccess` — seeds one; verifies `GetId()` and `GetName()`.
    - `TestGetCarManufacturerNotFound` — non-existent ID; expects `connect.CodeNotFound`.

    Tests for `ListCarBrands`:
    - `TestListCarBrandsEmpty` — verifies `GetItems()` is empty.
    - `TestListCarBrandsReturnsAll` — seeds 1 manufacturer with 2 brands; verifies both returned.
    - `TestListCarBrandsByManufacturerID` — seeds 2 manufacturers each with 1 brand; filters by first manufacturer ID; verifies exactly 1 brand returned with matching `GetManufacturerId()`.

    Tests for `GetCarBrand`:
    - `TestGetCarBrandSuccess` — seeds manufacturer → brand; verifies `GetId()`, `GetManufacturerId()`, and `GetName()`.
    - `TestGetCarBrandNotFound` — expects `connect.CodeNotFound`.

    Tests for `ListCarModels`:
    - `TestListCarModelsEmpty` — verifies `GetItems()` is empty.
    - `TestListCarModelsReturnsAll` — seeds manufacturer → brand → 2 models; verifies both returned.
    - `TestListCarModelsByManufacturerID` — seeds 2 manufacturers, each with 1 brand and 1 model; filters by first manufacturer ID; verifies exactly 1 model returned.

    Tests for `GetCarModel`:
    - `TestGetCarModelSuccess` — seeds manufacturer → brand → model; verifies `GetId()`, `GetBrandId()`, and `GetName()`.
    - `TestGetCarModelNotFound` — expects `connect.CodeNotFound`.
