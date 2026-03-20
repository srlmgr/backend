# Feature: Implement Query Service for Drivers

## Summary

Implement the query service handlers for `Driver` in `services/query/drivers.go`:

- `ListDrivers`
- `GetDriver`

Also add missing `LoadAll` to `DriversRepository` in `repository/drivers/drivers.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires these methods. Driver records are required for roster selection, booking displays, and downstream standings views.

## Prerequisites

- Conversion function `DriverToDriver` from `issue-command-drivers.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll` to `DriversRepository` and its concrete implementation.
- Implement `ListDrivers` and `GetDriver` in `services/query/drivers.go`.
- `ListDrivers` has no request filter in the current proto; call `LoadAll`.
- `GetDriver` resolves by `req.Msg.GetId()` via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Query filtering by simulation (`driver_simulation_ids`) - not exposed by the current `ListDriversRequest`.
- Driver simulation identity handlers.
- Command (write) handlers - covered by `issue-command-drivers.md`.

## Implementation Plan

1. **Extend `repository/drivers/drivers.go`**
    - Add to `DriversRepository` interface:
        ```go
        LoadAll(ctx context.Context) ([]*models.Driver, error)
        ```
    - Add implementation on `driversRepository`:
        ```go
        func (r *driversRepository) LoadAll(ctx context.Context) ([]*models.Driver, error) {
            return models.Drivers.Query().All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/drivers.go`**
    - Implement `ListDrivers`:
        - Call `s.repo.Drivers().Drivers().LoadAll(ctx)`.
        - Convert each row with `s.conversion.DriverToDriver`.
        - Return `ListDriversResponse{Items: items}`.
    - Implement `GetDriver`:
        - Call `s.repo.Drivers().Drivers().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetDriverResponse{Driver: s.conversion.DriverToDriver(item)}`.

3. **Create `services/query/drivers_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)`.

    Add local seed helper in this file:
    - `seedDriver(t, repo, externalID, name)` - inserts `models.Driver` with:
        - `ExternalID` set as decimal string
        - `IsActive: true`
        - `CreatedBy` / `UpdatedBy: testUserSeed`

    Update `resetTestTables` in `services/query/test_setup_test.go`:
    - include `drivers` in the truncate list.

    Tests for `ListDrivers`:
    - `TestListDriversEmpty` - seeds nothing; verifies `GetItems()` is empty.
    - `TestListDriversReturnsAll` - seeds two drivers; verifies both IDs are present.

    Tests for `GetDriver`:
    - `TestGetDriverSuccess` - verifies `GetId()`, `GetExternalId()`, `GetName()`, `GetIsActive()`.
    - `TestGetDriverNotFound` - non-existent ID; expects `connect.CodeNotFound`.
