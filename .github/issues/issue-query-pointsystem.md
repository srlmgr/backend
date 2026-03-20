# Feature: Implement Query Service for PointSystem

## Summary

Implement the query service handlers for `PointSystem` in `services/query/pointsystem.go`:

- `ListPointSystems`
- `GetPointSystem`

Also add missing `LoadAll` method to the `PointSystemsRepository` interface in `repository/pointsystems/pointsystems.go`.

Use `services/query/simulation.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires these methods. Without them, all clients querying for point system data receive an unimplemented error.

## Prerequisites

- Conversion functions `PointSystemToPointSystem` and `PointRuleToPointRule` from `issue-command-pointsystem.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll(ctx context.Context) ([]*models.PointSystem, error)` to `PointSystemsRepository` interface and its concrete implementation in `repository/pointsystems/pointsystems.go`.
- Implement `ListPointSystems` and `GetPointSystem` in `services/query/pointsystem.go`.
- `ListPointSystems` has no filter field in the proto – always call `LoadAll`.
- `GetPointSystem` uses `req.Msg.GetId()` as the entity identifier, resolved via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Filtering point systems by active status or other fields not present in the current proto.
- Command (write) handlers – covered by `issue-command-pointsystem.md`.

## Implementation Notes

- `ListPointSystemsRequest` has no filter fields – the implementation always calls `LoadAll`.
- The proto `PointSystem` embeds `rules []*PointRule`. For the initial implementation, return an empty rules list if loading nested rules requires an additional query not yet supported by the repository. A follow-up can enrich this once `LoadByPointSystemID` is added to `PointRulesRepository`.

## Implementation Plan

1. **Extend `repository/pointsystems/pointsystems.go`**
    - Add `LoadAll(ctx context.Context) ([]*models.PointSystem, error)` to `PointSystemsRepository` interface.
    - Add implementation on `pointSystemsRepository`:
        ```go
        func (r *pointSystemsRepository) LoadAll(ctx context.Context) ([]*models.PointSystem, error) {
            return models.PointSystems.Query().All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/pointsystem.go`**
    - Implement `ListPointSystems`:
        - Call `s.repo.PointSystems().PointSystems().LoadAll(ctx)`.
        - Convert each item with `s.conversion.PointSystemToPointSystem`.
        - Return `ListPointSystemsResponse{Items: items}`.
    - Implement `GetPointSystem`:
        - Call `s.repo.PointSystems().PointSystems().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetPointSystemResponse{PointSystem: s.conversion.PointSystemToPointSystem(item)}`.

3. **Create `services/query/test_setup_test.go`** _(shared, create once for all query service tests)_

    Follow the same structure as `services/command/test_setup_test.go`. Key elements:
    - `TestMain` — calls `testdb.InitTestDB()`, stores the pool in `testPool`, calls `m.Run()`, and closes the pool.
    - `newDBBackedQueryService(t)` — calls `resetTestTables`, registers `t.Cleanup(resetTestTables)`, builds `postgresrepo.New(testPool)` and `rootrepo.NewBobTransactionFromPool(testPool)`, returns a `*service` and the `rootrepo.Repository`.
    - `resetTestTables(t)` — truncates all relevant tables:
        ```sql
        TRUNCATE TABLE racing_sims, series, point_systems, seasons, tracks, track_layouts, events,
            car_manufacturers, car_brands, car_models RESTART IDENTITY CASCADE
        ```
    - `seedSimulation(t, repo, name)` — inserts a `RacingSim` via `repo.RacingSims().Create(...)` with `IsActive=true` and `SupportedImportFormats=["json"]`.
    - `seedSeries(t, repo, simID, name)` — inserts a `Series` via `repo.Series().Create(...)` with `SimulationID` and `Name`.
    - `seedSeason(t, repo, seriesID, name)` — inserts a `Season` via `repo.Seasons().Create(...)` with `SeriesID`, `Name`, and `Status="draft"`.
    - `seedTrack(t, repo, name)` — inserts a `Track` via `repo.Tracks().Tracks().Create(...)` with `Name`.
    - `seedTrackLayout(t, repo, trackID, name)` — inserts a `TrackLayout` via `repo.Tracks().TrackLayouts().Create(...)` with `TrackID` and `Name`.

    All seed helpers use `CreatedBy` and `UpdatedBy` set to a `testUserSeed` constant.

4. **Create `services/query/pointsystem_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` for all test cases.

    Local seed helper (defined in this file):

    ```go
    func seedPointSystem(t *testing.T, repo rootrepo.Repository, name string) *models.PointSystem {
        t.Helper()
        ps, err := repo.PointSystems().PointSystems().Create(context.Background(), &models.PointSystemSetter{
            Name:      omit.From(name),
            CreatedBy: omit.From(testUserSeed),
            UpdatedBy: omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed point system %q: %v", name, err)
        }
        return ps
    }
    ```

    Tests for `ListPointSystems`:
    - `TestListPointSystemsEmpty` — seeds nothing; verifies `GetItems()` is empty (not nil).
    - `TestListPointSystemsReturnsAll` — seeds 2 point systems (`"Alpha Points"`, `"Beta Points"`); verifies response contains both IDs.

    Tests for `GetPointSystem`:
    - `TestGetPointSystemSuccess` — seeds one; calls `GetPointSystem` with its ID; verifies `GetId()` and `GetName()` match.
    - `TestGetPointSystemNotFound` — calls `GetPointSystem` with a non-existent ID; expects `connect.CodeNotFound`.
