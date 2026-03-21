# Feature: Implement Query Service for Result Entries

## Summary

Implement the query service handlers for `ResultEntry` in `services/query/resultentries.go`:

- `ListResultEntries`
- `GetResultEntry`

Also add missing `LoadAll`, `LoadByRaceID`, `LoadByImportBatchID`, and `LoadByState` methods to the `resultentries.Repository` interface in `repository/resultentries/resultentries.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires these methods. Result entries represent race results and must be queryable by race, import batch, and state for result processing pipelines, standings calculations, and administrative review.

## Prerequisites

- Conversion function `ResultEntryToResultEntry` from `issue-command-resultentries.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll`, `LoadByRaceID`, `LoadByImportBatchID`, and `LoadByState` to the `resultentries.Repository` interface and its concrete implementation.
- Implement `ListResultEntries` and `GetResultEntry` in `services/query/resultentries.go`.
- `ListResultEntries` supports optional filters via request fields:
    - `req.Msg.GetRaceId()` – if non-zero, call `LoadByRaceID`.
    - `req.Msg.GetImportBatchId()` – if non-zero, call `LoadByImportBatchID`.
    - `req.Msg.GetState()` – if non-empty, call `LoadByState`.
    - If no filters provided, call `LoadAll`.
    - Support filtering by multiple criteria (e.g., race_id AND state) by combining conditions.
- `GetResultEntry` uses `req.Msg.GetId()` resolved via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Standings calculations or points computation – handled by separate services.
- Query (read) handlers – covered by `issue-query-resultentries.md`.

## Implementation Plan

1. **Extend `repository/resultentries/resultentries.go`**
    - Add to the `Repository` interface:
        ```go
        LoadAll(ctx context.Context) ([]*models.ResultEntry, error)
        LoadByRaceID(ctx context.Context, raceID int32) ([]*models.ResultEntry, error)
        LoadByImportBatchID(ctx context.Context, importBatchID int32) ([]*models.ResultEntry, error)
        LoadByState(ctx context.Context, state string) ([]*models.ResultEntry, error)
        ```
    - Add implementations on `resultEntriesRepository` following the `series` repository pattern:

        ```go
        func (r *resultEntriesRepository) LoadAll(ctx context.Context) ([]*models.ResultEntry, error) {
            return models.ResultEntries.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *resultEntriesRepository) LoadByRaceID(ctx context.Context, raceID int32) ([]*models.ResultEntry, error) {
            return models.ResultEntries.Query(
                sm.Where(models.ResultEntries.Columns.RaceID.EQ(psql.Arg(raceID))),
            ).All(ctx, r.getExecutor(ctx))
        }

        func (r *resultEntriesRepository) LoadByImportBatchID(ctx context.Context, importBatchID int32) ([]*models.ResultEntry, error) {
            return models.ResultEntries.Query(
                sm.Where(models.ResultEntries.Columns.ImportBatchID.EQ(psql.Arg(importBatchID))),
            ).All(ctx, r.getExecutor(ctx))
        }

        func (r *resultEntriesRepository) LoadByState(ctx context.Context, state string) ([]*models.ResultEntry, error) {
            return models.ResultEntries.Query(
                sm.Where(models.ResultEntries.Columns.State.EQ(psql.Arg(state))),
            ).All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/resultentries.go`**
    - Implement `ListResultEntries`:
        - Build conditions from provided filters (race_id, import_batch_id, state).
        - If race_id is non-zero, filter by race.
        - If import_batch_id is non-zero, filter by import batch.
        - If state is non-empty, filter by state.
        - Call appropriate repository method(s) or combine filters manually via query builder.
        - Convert each item with `s.conversion.ResultEntryToResultEntry` and return `ListResultEntriesResponse{Items: items}`.
    - Implement `GetResultEntry`:
        - Call `s.repo.ResultEntries().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetResultEntryResponse{ResultEntry: s.conversion.ResultEntryToResultEntry(item)}`.

3. **Create `services/query/resultentries_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` and the shared seed helpers from `test_setup_test.go`.

    Local seed helper (defined in this file):

    ```go
    func seedResultEntry(t *testing.T, repo rootrepo.Repository, importBatchID, raceID int32, driverName string, finishingPosition int32) *models.ResultEntry {
        t.Helper()
        entry, err := repo.ResultEntries().Create(context.Background(), &models.ResultEntrySetter{
            ImportBatchID:    omit.From(importBatchID),
            RaceID:           omit.From(raceID),
            DriverName:       omit.From(driverName),
            FinishingPosition: omit.From(finishingPosition),
            State:            omit.From("normal"),
            CreatedBy:        omit.From(testUserSeed),
            UpdatedBy:        omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed result entry for %q: %v", driverName, err)
        }
        return entry
    }
    ```

    The full seeding hierarchy is: `sim → series → season → event → race` and `import_batch`, then `seedResultEntry(batch.ID, race.ID, driverName, position)`.

    Tests for `ListResultEntries`:
    - `TestListResultEntriesEmpty` — seeds nothing; verifies `GetItems()` is empty.
    - `TestListResultEntriesReturnsAll` — seeds sim → series → season → event → race, import_batch; creates 3 result entries; verifies all are returned.
    - `TestListResultEntriesByRaceID` — seeds sim → series → season → event → 2 races, import_batch; 1 entry per race; filters by first race; verifies exactly 1 entry with matching `GetRaceId()`.
    - `TestListResultEntriesByImportBatchID` — seeds 2 import batches with 2 entries each; filters by first batch; verifies exactly 2 entries returned.
    - `TestListResultEntriesByState` — seeds 3 entries with mixed states ('normal', 'dq'); filters by 'dq'; verifies only 'dq' entries returned.
    - `TestListResultEntriesByRaceAndState` — seeds mixed entries; filters by both race_id and state; verifies intersection of results.

    Tests for `GetResultEntry`:
    - `TestGetResultEntrySuccess` — seeds the full hierarchy → entry; calls `GetResultEntry` with its ID; verifies `GetId()`, `GetRaceId()`, and `GetDriverName()`.
    - `TestGetResultEntryNotFound` — calls `GetResultEntry` with a non-existent ID; expects `connect.CodeNotFound`.
