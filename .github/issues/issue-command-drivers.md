# Feature: Implement Command Service for Drivers

## Summary

Implement the command service handlers for `Driver` in `services/command/drivers.go`:

- `CreateDriver`
- `UpdateDriver`
- `DeleteDriver`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface includes driver write operations. Drivers are foundational entities used by booking entries, results, and standings.

## Goals

- Implement `CreateDriver`, `UpdateDriver`, `DeleteDriver` in `services/command/drivers.go`.
- Add a `driverSetterBuilder` following the setter-builder pattern.
- Add conversion function to `services/conversion/service.go`:
    - `DriverToDriver(model *models.Driver) *commonv1.Driver`
- Map writable proto fields from create/update requests to `models.DriverSetter`:
    - `ExternalId` (`uint32`) -> `setter.ExternalID` (`string`)
    - `Name` -> `setter.Name`
    - `IsActive` -> `setter.IsActive`
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)` on create.
- Set `UpdatedAt` and `UpdatedBy` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `UpsertDriverSimulationIdentity`.
- Query (read) handlers - covered by `issue-query-drivers.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- Proto requests expose `GetExternalId() uint32` while DB uses `drivers.external_id` as `string`.
    - In setter builder, convert via `strconv.FormatUint(uint64(msg.GetExternalId()), 10)` when non-zero.
- `DriverSetter` has additional writable fields (`LastImportedFrom`, `FrontendID`) that are not part of the current command requests; leave them unset.
- `UpdateDriverRequest.GetDriverId()` provides the target entity identifier.

## Implementation Plan

1. **Create `services/command/drivers.go`**
    - Define `driverRequest` interface:
        - `GetExternalId() uint32`
        - `GetName() string`
        - `GetIsActive() bool`
    - Define `driverSetterBuilder` with `Build(msg driverRequest) *models.DriverSetter`.
    - Implement `CreateDriver`:
        - Build setter.
        - Set `CreatedBy` and `UpdatedBy` inside transaction.
        - Call `s.repo.Drivers().Drivers().Create(ctx, setter)`.
        - Return `CreateDriverResponse` with converted model.
    - Implement `UpdateDriver`:
        - Build setter.
        - Set `UpdatedAt` and `UpdatedBy` inside transaction.
        - Call `s.repo.Drivers().Drivers().Update(ctx, int32(req.Msg.GetDriverId()), setter)`.
        - Return `UpdateDriverResponse`.
    - Implement `DeleteDriver`:
        - Call `s.repo.Drivers().Drivers().DeleteByID(ctx, int32(req.Msg.GetDriverId()))`.
        - Return `DeleteDriverResponse{Deleted: true}`.

2. **Add conversion function in `services/conversion/service.go`**
    - `DriverToDriver` maps `ID`, `ExternalID`, `Name`, `IsActive`.
    - Convert `model.ExternalID` (string) to `uint32` using `strconv.ParseUint(..., 10, 32)`.
    - If parse fails, return `ExternalId: 0` (and optionally leave a short TODO comment for follow-up validation policy).

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.DriverErrors.ErrUniqueDriversExternalIdUnique` -> `connect.CodeAlreadyExists`

4. **Create `services/command/drivers_test.go`**

    Keep tests in package `command`.

    Add shared seed helper to `test_setup_test.go`:
    - `seedDriver(t, repo, externalID, name)` - inserts a `Driver` row with `IsActive: true`, `CreatedBy: testUserSeed`, `UpdatedBy: testUserSeed` and returns the model.

    Tests for `driverSetterBuilder.Build`:
    - Success: maps `ExternalId`, `Name`, `IsActive`.
    - Zero values: `ExternalId == 0` and empty `Name` leave those fields unset.

    Tests for `CreateDriver`:
    - `TestCreateDriverSuccess` - verifies response fields, checks `CreatedBy` / `UpdatedBy` in DB, verifies external ID persisted as decimal string.
    - `TestCreateDriverFailureDuplicateExternalID` - expects `connect.CodeAlreadyExists`.
    - `TestCreateDriverFailureTransactionError` - uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateDriver`:
    - `TestUpdateDriverSuccess` - verifies updated name/is_active and `UpdatedBy` / `UpdatedAt` advance.
    - `TestUpdateDriverFailureNotFound` - expects `connect.CodeNotFound`.
    - `TestUpdateDriverFailureDuplicateExternalID` - expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteDriver`:
    - `TestDeleteDriverSuccess` - verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteDriverFailureTransactionError` - uses `txManagerStub`; expects `connect.CodeInternal`.
