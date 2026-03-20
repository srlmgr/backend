# Feature: Implement Command Service for PointSystem

## Summary

Implement the command service handlers for `PointSystem` in `services/command/pointsystem.go`:

- `CreatePointSystem`
- `UpdatePointSystem`
- `DeletePointSystem`

Use `services/command/series.go` and `services/command/simulation.go` as implementation references.

## Why

The `CommandServiceHandler` interface (generated from the protobuf definition) requires these methods to be implemented. Until implemented, the service panics or returns unimplemented errors for all point system mutation requests.

## Goals

- Implement `CreatePointSystem`, `UpdatePointSystem`, `DeletePointSystem` in a new file `services/command/pointsystem.go`.
- Add a `pointSystemSetterBuilder` struct following the existing setter-builder pattern.
- Add conversion functions to `services/conversion/service.go`:
    - `PointSystemToPointSystem(model *models.PointSystem) *commonv1.PointSystem`
    - `PointRuleToPointRule(model *models.PointRule) *commonv1.PointRule`
- Map all writable proto fields to `models.PointSystemSetter`:
    - `Name` → `setter.Name`
    - `Description` → `setter.Description` (nullable)
- Set `CreatedBy` / `UpdatedBy` from the authenticated principal (`s.execUser(ctx)`).
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map repository and domain errors to correct Connect RPC codes via `s.conversion.MapErrorToRPCCode`.
- Add unique-constraint error mappings to `MapErrorToRPCCode` for `PointSystem` (e.g. name uniqueness) once the DB error sentinels are confirmed in `db/dberrors`.

## Non-Goals

- Persistence of nested `PointRule` entries – the proto `rules` field on `CreatePointSystemRequest` and `UpdatePointSystemRequest` is out of scope for this issue. Point-rule management will be addressed in a follow-up.
- Query (read) handlers – covered by `issue-query-pointsystem.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- The `PointRule` DB model stores rule data in a single `MetadataJSON` column (`types.JSON[json.RawMessage]`). The proto separates this into `kind`, `position`, `points`, and `metadata_json` fields. Conversion between the two representations must be handled carefully and is deferred to the follow-up point-rule issue.
- `PointSystemSetter` does **not** have an `IsActive` field in the proto request, but the DB model does. Only set it if it is provided in the request.

## Implementation Plan

1. **Create `services/command/pointsystem.go`**
    - Define `pointSystemRequest` interface with `GetName()` and `GetDescription()`.
    - Define `pointSystemSetterBuilder` and its `Build` method.
    - Implement `CreatePointSystem`:
        - Build setter.
        - Set `CreatedBy` and `UpdatedBy` inside transaction.
        - Call `s.repo.PointSystems().PointSystems().Create(ctx, setter)`.
        - Return `CreatePointSystemResponse` with converted model.
    - Implement `UpdatePointSystem`:
        - Build setter.
        - Set `UpdatedAt` and `UpdatedBy` inside transaction.
        - Call `s.repo.PointSystems().PointSystems().Update(ctx, id, setter)`.
        - Return `UpdatePointSystemResponse`.
    - Implement `DeletePointSystem`:
        - Call `s.repo.PointSystems().PointSystems().DeleteByID(ctx, id)`.
        - Return `DeletePointSystemResponse{Deleted: true}`.

2. **Add conversion functions in `services/conversion/service.go`**
    - `PointSystemToPointSystem` – maps `ID`, `Name`, `Description`, `IsActive`.
    - `PointRuleToPointRule` – maps the `MetadataJSON` blob to proto fields (stub returning empty `PointRule` message until the full rule conversion is designed).

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.PointSystemErrors.ErrUniquePointSystemsNameUnique` → `connect.CodeAlreadyExists`

4. **Create `services/command/pointsystem_test.go`**

    Keep tests in package `command` (same as `simulation_test.go`).

    Add shared seed helpers to `test_setup_test.go`:
    - `seedPointSystem(t, repo, name)` – inserts a `PointSystem` row with `CreatedBy: testUserSeed` and returns the model.
    - Add `"TRUNCATE TABLE point_systems RESTART IDENTITY CASCADE"` to `resetTestTables`.

    Tests for `pointSystemSetterBuilder.Build`:
    - Success: maps `Name` and `Description` correctly; zero-value fields are unset.

    Tests for `CreatePointSystem`:
    - `TestCreatePointSystemSuccess` – verifies response fields, checks `CreatedBy`/`UpdatedBy` in DB.
    - `TestCreatePointSystemFailureDuplicateName` – expects `connect.CodeAlreadyExists`.
    - `TestCreatePointSystemFailureTransactionError` – uses `txManagerStub` that returns an error; expects `connect.CodeInternal` and the original error to be wrapped.

    Tests for `UpdatePointSystem`:
    - `TestUpdatePointSystemSuccess` – verifies updated name/description and that `UpdatedBy`/`UpdatedAt` advance.
    - `TestUpdatePointSystemFailureNotFound` – update with non-existent ID; expects `connect.CodeNotFound`.
    - `TestUpdatePointSystemFailureDuplicateName` – update second point system to name of first; expects `connect.CodeAlreadyExists` and DB row unchanged.

    Tests for `DeletePointSystem`:
    - `TestDeletePointSystemSuccess` – verifies `Deleted: true` and that `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeletePointSystemFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.
