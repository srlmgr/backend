# Feature: Implement Command Service for Cars

## Summary

Implement the command service handlers for `CarManufacturer`, `CarBrand`, and `CarModel` in `services/command/cars.go`:

- `CreateCarManufacturer`
- `UpdateCarManufacturer`
- `DeleteCarManufacturer`
- `CreateCarBrand`
- `UpdateCarBrand`
- `DeleteCarBrand`
- `CreateCarModel`
- `UpdateCarModel`
- `DeleteCarModel`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires all nine handlers. Car manufacturers, brands, and models form a three-level reference hierarchy used throughout the application (race entries, standings). They must be manageable through the API before event data can be imported.

## Goals

- Implement all nine handlers in a new file `services/command/cars.go`.
- Add three setter builder structs: `carManufacturerSetterBuilder`, `carBrandSetterBuilder`, `carModelSetterBuilder`.
- Add conversion functions to `services/conversion/service.go`:
    - `CarManufacturerToCarManufacturer(model *models.CarManufacturer) *commonv1.CarManufacturer`
    - `CarBrandToCarBrand(model *models.CarBrand) *commonv1.CarBrand`
    - `CarModelToCarModel(model *models.CarModel) *commonv1.CarModel`
- Map all writable proto fields:
    - `CarManufacturerSetter`: `Name`
    - `CarBrandSetter`: `ManufacturerId` → `setter.ManufacturerID`, `Name`
    - `CarModelSetter`: `BrandId` → `setter.BrandID`, `Name`
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `SimulationCarAlias` CRUD – not exposed via command proto in the current version.
- Query (read) handlers – covered by `issue-query-cars.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- `IsActive` is present in `CarManufacturerSetter`, `CarBrandSetter`, and `CarModelSetter` but the current proto requests do not expose it. Leave it un-set (defaults to column default) unless a proto field is added.
- The three entity types follow an identical structural pattern (`Name` + optional parent FK). Use the `//nolint:dupl` comment where needed.
- `UpdateCarManufacturerRequest.GetCarManufacturerId()`, `UpdateCarBrandRequest.GetCarBrandId()`, and `UpdateCarModelRequest.GetCarModelId()` provide the entity identifiers.

## Implementation Plan

1. **Create `services/command/cars.go`**
    - Define `carManufacturerRequest` interface (`GetName() string`).
    - Define `carBrandRequest` interface (`GetManufacturerId() uint32`, `GetName() string`).
    - Define `carModelRequest` interface (`GetBrandId() uint32`, `GetName() string`).
    - Implement setter builders for each.
    - Implement `CreateCarManufacturer`, `UpdateCarManufacturer`, `DeleteCarManufacturer`
      using `s.repo.Cars().CarManufacturers()`.
    - Implement `CreateCarBrand`, `UpdateCarBrand`, `DeleteCarBrand`
      using `s.repo.Cars().CarBrands()`.
    - Implement `CreateCarModel`, `UpdateCarModel`, `DeleteCarModel`
      using `s.repo.Cars().CarModels()`.

2. **Add conversion functions in `services/conversion/service.go`**
    - `CarManufacturerToCarManufacturer` – maps `ID`, `Name`.
    - `CarBrandToCarBrand` – maps `ID`, `ManufacturerID`, `Name`.
    - `CarModelToCarModel` – maps `ID`, `BrandID`, `Name`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.CarManufacturerErrors.ErrUniqueCarManufacturersNameUnique` → `connect.CodeAlreadyExists`
        - `dberrors.CarBrandErrors.ErrUniqueCarBrandsManufacturerIdNameUnique` → `connect.CodeAlreadyExists`
        - `dberrors.CarModelErrors.ErrUniqueCarModelsBrandIdNameUnique` → `connect.CodeAlreadyExists`

4. **Create `services/command/cars_test.go`**

    Keep tests in package `command`.

    Add shared seed helpers to `test_setup_test.go`:
    - `seedCarManufacturer(t, repo, name)` – inserts a `CarManufacturer` row and returns the model.
    - `seedCarBrand(t, repo, manufacturerID, name)` – inserts a `CarBrand` row and returns the model.
    - `seedCarModel(t, repo, brandID, name)` – inserts a `CarModel` row and returns the model.
    - Add truncations for `car_models`, `car_brands`, `car_manufacturers` (in dependency order) to `resetTestTables`.

    Tests for `carManufacturerSetterBuilder.Build`:
    - Success: `Name` is mapped; zero `Name` leaves field unset.

    Tests for `carBrandSetterBuilder.Build`:
    - Success: maps `ManufacturerId` and `Name`.

    Tests for `carModelSetterBuilder.Build`:
    - Success: maps `BrandId` and `Name`.

    Tests for `CreateCarManufacturer`:
    - `TestCreateCarManufacturerSuccess` – verifies response and `CreatedBy`/`UpdatedBy` in DB.
    - `TestCreateCarManufacturerFailureDuplicateName` – expects `connect.CodeAlreadyExists`.
    - `TestCreateCarManufacturerFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateCarManufacturer`:
    - `TestUpdateCarManufacturerSuccess` – verifies updated name and `UpdatedAt` advance.
    - `TestUpdateCarManufacturerFailureNotFound` – expects `connect.CodeNotFound`.
    - `TestUpdateCarManufacturerFailureDuplicateName` – expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteCarManufacturer`:
    - `TestDeleteCarManufacturerSuccess` – verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.

    Tests for `CreateCarBrand`, `UpdateCarBrand`, `DeleteCarBrand`:
    - Mirror the same success/failure/duplicate/not-found pattern as manufacturer tests above.

    Tests for `CreateCarModel`, `UpdateCarModel`, `DeleteCarModel`:
    - Mirror the same pattern; seed a manufacturer + brand first to satisfy FK constraints.
