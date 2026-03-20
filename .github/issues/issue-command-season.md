# Feature: Implement Command Service for Season

## Summary

Implement the command service handlers for `Season` in `services/command/season.go`:

- `CreateSeason`
- `UpdateSeason`
- `DeleteSeason`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires these methods. Season mutations are core to the application lifecycle — each racing season belongs to a series and must be creatable, updatable, and deletable through the API.

## Goals

- Implement `CreateSeason`, `UpdateSeason`, `DeleteSeason` in a new file `services/command/season.go`.
- Add a `seasonSetterBuilder` struct following the setter-builder pattern.
- Add conversion function to `services/conversion/service.go`:
    - `SeasonToSeason(model *models.Season) *commonv1.Season`
- Map all writable proto fields to `models.SeasonSetter`:
    - `SeriesId` → `setter.SeriesID`
    - `Name` → `setter.Name`
    - `PointSystemId` → `setter.PointSystemID`
    - `HasTeams` → `setter.HasTeams`
    - `SkipEvents` → `setter.SkipEvents`
    - `Status` → `setter.Status`
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Query (read) handlers – covered by `issue-query-season.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- `SeasonSetter` has additional fields not in the proto (`StartsAt`, `EndsAt`, `TeamPointsTopN`). Only map fields that are present in the proto request.
- The proto `Season` message has `GetStatus() string`. The DB model stores this as a plain string — no enum conversion needed.
- `UpdateSeasonRequest` also includes `GetSeasonId() uint32` as the target record identifier.

## Implementation Plan

1. **Create `services/command/season.go`**
    - Define `seasonRequest` interface with getter methods for all mappable proto fields:
        - `GetSeriesId() uint32`
        - `GetName() string`
        - `GetPointSystemId() uint32`
        - `GetHasTeams() bool`
        - `GetSkipEvents() int32`
        - `GetStatus() string`
    - Define `seasonSetterBuilder` with a `Build(msg seasonRequest) *models.SeasonSetter` method.
    - Implement `CreateSeason`:
        - Build setter from request.
        - Set `CreatedBy` and `UpdatedBy` inside transaction.
        - Call `s.repo.Seasons().Create(ctx, setter)`.
        - Return `CreateSeasonResponse` with converted model.
    - Implement `UpdateSeason`:
        - Build setter from request.
        - Set `UpdatedAt` and `UpdatedBy` inside transaction.
        - Call `s.repo.Seasons().Update(ctx, int32(req.Msg.GetSeasonId()), setter)`.
        - Return `UpdateSeasonResponse`.
    - Implement `DeleteSeason`:
        - Call `s.repo.Seasons().DeleteByID(ctx, int32(req.Msg.GetSeasonId()))`.
        - Return `DeleteSeasonResponse{Deleted: true}`.

2. **Add conversion function in `services/conversion/service.go`**
    - `SeasonToSeason` – maps `ID`, `SeriesID`, `Name`, `PointSystemID`, `HasTeams`, `SkipEvents`, `Status`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.SeasonErrors.ErrUniqueSeasonsSeriesIdNameUnique` → `connect.CodeAlreadyExists`

4. **Create `services/command/season_test.go`**

    Keep tests in package `command`.

    Add shared seed helpers to `test_setup_test.go`:
    - `seedPointSystem(t, repo, name)` (if not already added by `issue-command-pointsystem.md`) – required to satisfy the `point_system_id` FK when seeding seasons.
    - `seedSeason(t, repo, seriesID, pointSystemID, name)` – inserts a `Season` row with `CreatedBy: testUserSeed` and returns the model.
    - Add `"TRUNCATE TABLE seasons RESTART IDENTITY CASCADE"` to `resetTestTables` (ordering: seasons before series before racing_sims).

    Tests for `seasonSetterBuilder.Build`:
    - Success: maps all fields (`SeriesId`, `Name`, `PointSystemId`, `HasTeams`, `SkipEvents`, `Status`); zero-value fields are unset.

    Tests for `CreateSeason`:
    - `TestCreateSeasonSuccess` – verifies response fields, checks `CreatedBy`/`UpdatedBy` in DB, validates `SeriesID` and `PointSystemID` are stored.
    - `TestCreateSeasonFailureDuplicateNameSameSeries` – expects `connect.CodeAlreadyExists`.
    - `TestCreateSeasonSuccessDuplicateNameDifferentSeries` – same name under a different series should succeed.
    - `TestCreateSeasonFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateSeason`:
    - `TestUpdateSeasonSuccess` – verifies updated name/status and that `UpdatedBy`/`UpdatedAt` advance.
    - `TestUpdateSeasonFailureNotFound` – expects `connect.CodeNotFound`.
    - `TestUpdateSeasonFailureDuplicateNameSameSeries` – expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteSeason`:
    - `TestDeleteSeasonSuccess` – verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteSeasonFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.
