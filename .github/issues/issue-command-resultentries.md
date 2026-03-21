# Feature: Implement Command Service for Result Entries

## Summary

Implement the command service handlers for `ResultEntry` in `services/command/resultentries.go`:

- `CreateResultEntry`
- `UpdateResultEntry`
- `DeleteResultEntry`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires these methods. Result entries represent completed race results imported from external sources. They must be creatable during import batch processing and updatable for administrative corrections (e.g., changing state to 'dq', adding admin notes).

## Goals

- Implement `CreateResultEntry`, `UpdateResultEntry`, `DeleteResultEntry` in a new file `services/command/resultentries.go`.
- Add a `resultEntrySetterBuilder` struct following the setter-builder pattern.
- Add conversion function to `services/conversion/service.go`:
    - `ResultEntryToResultEntry(model *models.ResultEntry) *commonv1.ResultEntry`
- Map all writable proto fields to `models.ResultEntrySetter`:
    - `RaceId` → `setter.RaceID`
    - `ImportBatchId` → `setter.ImportBatchID`
    - `DriverId` → `setter.DriverID` (nullable)
    - `DriverName` → `setter.DriverName`
    - `CarModelId` → `setter.CarModelID` (nullable)
    - `CarName` → `setter.CarName` (nullable)
    - `FinishingPosition` → `setter.FinishingPosition`
    - `CompletedLaps` → `setter.CompletedLaps`
    - `FastestLapTimeMs` → `setter.FastestLapTimeMs` (nullable)
    - `Incidents` → `setter.Incidents` (nullable)
    - `State` → `setter.State`
    - `AdminNotes` → `setter.AdminNotes` (nullable)
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Raw result payload processing or validation – these are handled by import services.
- Query (read) handlers – covered by `issue-query-resultentries.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- `ResultEntrySetter` has nullable fields (`DriverID`, `CarModelID`, `CarName`, `FastestLapTimeMs`, `Incidents`, `AdminNotes`). Only set them when provided in the proto request (non-zero/non-empty).
- `ResultEntrySetter` has `RawPayload` and `SourceRowNumber` fields not present in the proto. Leave them un-set.
- `State` is a plain string ('normal' or 'dq') – no enum conversion needed.
- `UpdateResultEntryRequest.GetResultEntryId()` provides the entity identifier.

## Implementation Plan

1. **Create `services/command/resultentries.go`**
    - Define `resultEntryRequest` interface:
        - `GetRaceId() uint32`
        - `GetImportBatchId() uint32`
        - `GetDriverId() uint32`
        - `GetDriverName() string`
        - `GetCarModelId() uint32`
        - `GetCarName() string`
        - `GetFinishingPosition() int32`
        - `GetCompletedLaps() int32`
        - `GetFastestLapTimeMs() int32`
        - `GetIncidents() int32`
        - `GetState() string`
        - `GetAdminNotes() string`
    - Define `resultEntrySetterBuilder` with `Build(msg resultEntryRequest) *models.ResultEntrySetter`.
    - Implement `CreateResultEntry`:
        - Build setter.
        - Set `CreatedBy` and `UpdatedBy` inside transaction.
        - Call `s.repo.ResultEntries().Create(ctx, setter)`.
        - Return `CreateResultEntryResponse` with converted model.
    - Implement `UpdateResultEntry`:
        - Build setter.
        - Set `UpdatedAt` and `UpdatedBy` inside transaction.
        - Call `s.repo.ResultEntries().Update(ctx, int32(req.Msg.GetResultEntryId()), setter)`.
        - Return `UpdateResultEntryResponse`.
    - Implement `DeleteResultEntry`:
        - Call `s.repo.ResultEntries().DeleteByID(ctx, int32(req.Msg.GetResultEntryId()))`.
        - Return `DeleteResultEntryResponse{Deleted: true}`.

2. **Add conversion function in `services/conversion/service.go`**
    - `ResultEntryToResultEntry` – maps `ID`, `RaceID`, `ImportBatchID`, `DriverID`, `DriverName`, `CarModelID`, `CarName`, `FinishingPosition`, `CompletedLaps`, `FastestLapTimeMs`, `Incidents`, `State`, `AdminNotes`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.ResultEntryErrors.ErrUniqueResultEntriesRaceIdDriverIdUnique` → `connect.CodeAlreadyExists`

4. **Create `services/command/resultentries_test.go`**

    Keep tests in package `command`.

    Add shared seed helpers to `test_setup_test.go` (reuse from other issues where applicable):
    - `seedResultEntry(t, repo, importBatchID, raceID, driverName, finishingPosition)` – inserts a `ResultEntry` row with `State: "normal"`, `CreatedBy: testUserSeed` and returns the model.
    - Add truncations for `result_entries`, `import_batches`, `races`, `events`, `drivers`, `car_models` in dependency order to `resetTestTables`.

    Tests for `resultEntrySetterBuilder.Build`:
    - Success: maps `RaceId`, `ImportBatchId`, `DriverId`, `DriverName`, `CarModelId`, `CarName`, `FinishingPosition`, `CompletedLaps`, `FastestLapTimeMs`, `Incidents`, `State`, `AdminNotes`.
    - Optional fields: `DriverId`, `CarModelId`, `CarName`, `FastestLapTimeMs`, `Incidents`, `AdminNotes` are unset when zero/empty in request.

    Tests for `CreateResultEntry`:
    - `TestCreateResultEntrySuccess` – verifies response fields, checks `CreatedBy`/`UpdatedBy` in DB; seed import_batch → race → driver hierarchy.
    - `TestCreateResultEntryFailureDuplicateRaceDriver` – expects `connect.CodeAlreadyExists` when unique constraint violated.
    - `TestCreateResultEntrySuccessDifferentDriver` – same race with different driver should succeed.
    - `TestCreateResultEntryFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateResultEntry`:
    - `TestUpdateResultEntrySuccess` – verifies updated state/admin_notes and that `UpdatedBy`/`UpdatedAt` advance.
    - `TestUpdateResultEntryFailureNotFound` – expects `connect.CodeNotFound`.
    - `TestUpdateResultEntryToDisqualified` – update state to 'dq'; verify state change.

    Tests for `DeleteResultEntry`:
    - `TestDeleteResultEntrySuccess` – verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteResultEntryFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.
