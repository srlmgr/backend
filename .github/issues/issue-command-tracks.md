# Feature: Implement Command Service for Tracks

## Summary

Implement the command service handlers for `Track` and `TrackLayout` in `services/command/tracks.go`:

- `CreateTrack`
- `UpdateTrack`
- `DeleteTrack`
- `CreateTrackLayout`
- `UpdateTrackLayout`
- `DeleteTrackLayout`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires these methods. Tracks and their layouts are foundational reference data — events are linked to track layouts, so these must be manageable through the API.

## Goals

- Implement all six handlers in a new file `services/command/tracks.go`.
- Add a `trackSetterBuilder` and `trackLayoutSetterBuilder` struct following the setter-builder pattern.
- Add conversion functions to `services/conversion/service.go`:
    - `TrackToTrack(model *models.Track) *commonv1.Track`
    - `TrackLayoutToTrackLayout(model *models.TrackLayout) *commonv1.TrackLayout`
- Map all writable proto fields to `models.TrackSetter`:
    - `Name` → `setter.Name`
    - `Country` → `setter.Country` (nullable)
    - `Latitude` → `setter.Latitude` (nullable `decimal.Decimal`)
    - `Longitude` → `setter.Longitude` (nullable `decimal.Decimal`)
    - `WebsiteUrl` → `setter.WebsiteURL` (nullable)
- Map all writable proto fields to `models.TrackLayoutSetter`:
    - `TrackId` → `setter.TrackID`
    - `Name` → `setter.Name`
    - `LengthMeters` → `setter.LengthMeters` (nullable)
    - `LayoutImageUrl` → `setter.LayoutImageURL` (nullable)
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `SimulationTrackLayoutAlias` CRUD – not exposed via the command proto in the current version.
- Query (read) handlers – covered by `issue-query-tracks.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- `TrackSetter.Latitude` and `TrackSetter.Longitude` are `omitnull.Val[decimal.Decimal]`. The proto represents them as `float64`. Convert using `decimal.NewFromFloat(v)` from the `shopspring/decimal` package (already a transitive dependency).
- `TrackSetter.Country`, `TrackSetter.WebsiteURL`, `TrackSetter.LengthMeters`, and `TrackSetter.LayoutImageURL` are nullable (`omitnull.Val`). Only set them when the proto field carries a non-zero value.
- `UpdateTrackRequest.GetTrackId()` and `UpdateTrackLayoutRequest.GetTrackLayoutId()` provide the entity identifier for update routing.

## Implementation Plan

1. **Create `services/command/tracks.go`**
    - Define `trackRequest` interface:
        - `GetName() string`
        - `GetCountry() string`
        - `GetLatitude() float64`
        - `GetLongitude() float64`
        - `GetWebsiteUrl() string`
    - Define `trackSetterBuilder` with `Build(msg trackRequest) *models.TrackSetter`.
    - Define `trackLayoutRequest` interface:
        - `GetTrackId() uint32`
        - `GetName() string`
        - `GetLengthMeters() int32`
        - `GetLayoutImageUrl() string`
    - Define `trackLayoutSetterBuilder` with `Build(msg trackLayoutRequest) *models.TrackLayoutSetter`.
    - Implement `CreateTrack` and `UpdateTrack`, `DeleteTrack` using `s.repo.Tracks().Tracks()`.
    - Implement `CreateTrackLayout`, `UpdateTrackLayout`, `DeleteTrackLayout` using `s.repo.Tracks().TrackLayouts()`.

2. **Add conversion functions in `services/conversion/service.go`**
    - `TrackToTrack` – maps `ID`, `Name`, `Country`, `Latitude`, `Longitude`, `WebsiteUrl`, `IsActive`.
    - `TrackLayoutToTrackLayout` – maps `ID`, `TrackID`, `Name`, `LengthMeters`, `LayoutImageUrl`, `IsActive`.
    - For nullable decimal fields, convert to `float64` using `.InexactFloat64()`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.TrackErrors.ErrUniqueTracksNameUnique` → `connect.CodeAlreadyExists`
        - `dberrors.TrackLayoutErrors.ErrUniqueTrackLayoutsTrackIdNameUnique` → `connect.CodeAlreadyExists`

4. **Create `services/command/tracks_test.go`**

    Keep tests in package `command`.

    Add shared seed helpers to `test_setup_test.go`:
    - `seedTrack(t, repo, name)` – inserts a `Track` row with `CreatedBy: testUserSeed` and returns the model.
    - `seedTrackLayout(t, repo, trackID, name)` – inserts a `TrackLayout` row with `CreatedBy: testUserSeed` and returns the model.
    - Add `"TRUNCATE TABLE track_layouts RESTART IDENTITY CASCADE"` and `"TRUNCATE TABLE tracks RESTART IDENTITY CASCADE"` to `resetTestTables` (layouts before tracks).

    Tests for `trackSetterBuilder.Build`:
    - Success: maps `Name`, and verifies optional nullable fields (`Country`, `WebsiteUrl`, `Latitude`, `Longitude`) are only set when non-zero.

    Tests for `trackLayoutSetterBuilder.Build`:
    - Success: maps `TrackId`, `Name`, and verifies nullable `LengthMeters` and `LayoutImageUrl` are only set when non-zero.

    Tests for `CreateTrack`:
    - `TestCreateTrackSuccess` – verifies response fields, checks `CreatedBy`/`UpdatedBy` in DB.
    - `TestCreateTrackFailureDuplicateName` – expects `connect.CodeAlreadyExists`.
    - `TestCreateTrackFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateTrack`:
    - `TestUpdateTrackSuccess` – verifies updated name and that `UpdatedBy`/`UpdatedAt` advance.
    - `TestUpdateTrackFailureNotFound` – expects `connect.CodeNotFound`.
    - `TestUpdateTrackFailureDuplicateName` – expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteTrack`:
    - `TestDeleteTrackSuccess` – verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteTrackFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `CreateTrackLayout`:
    - `TestCreateTrackLayoutSuccess` – verifies response fields and `TrackID` linkage.
    - `TestCreateTrackLayoutFailureDuplicateNameSameTrack` – expects `connect.CodeAlreadyExists`.

    Tests for `UpdateTrackLayout`:
    - `TestUpdateTrackLayoutSuccess` – verifies updated name and `UpdatedAt` advance.
    - `TestUpdateTrackLayoutFailureNotFound` – expects `connect.CodeNotFound`.

    Tests for `DeleteTrackLayout`:
    - `TestDeleteTrackLayoutSuccess` – verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
