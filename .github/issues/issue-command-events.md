# Feature: Implement Command Service for Events

## Summary

Implement the command service handlers for `Event` in `services/command/events.go`:

- `CreateEvent`
- `UpdateEvent`
- `DeleteEvent`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires these methods. Events are the central scheduling entity — they link a season to a track layout and gate the entire import and results processing pipeline. They must be creatable and updatable through the API.

## Goals

- Implement `CreateEvent`, `UpdateEvent`, `DeleteEvent` in a new file `services/command/events.go`.
- Add an `eventSetterBuilder` struct following the setter-builder pattern.
- Add conversion function to `services/conversion/service.go`:
    - `EventToEvent(model *models.Event) *commonv1.Event`
- Map all writable proto fields to `models.EventSetter`:
    - `SeasonId` → `setter.SeasonID`
    - `TrackLayoutId` → `setter.TrackLayoutID`
    - `Name` → `setter.Name`
    - `EventDate` → `setter.EventDate` (convert from `*timestamppb.Timestamp` to `time.Time`)
    - `Status` → `setter.Status`
    - `ProcessingState` → `setter.ProcessingState`
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Import processing or result ingestion triggered by event status transitions.
- Query (read) handlers – covered by `issue-query-events.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- `EventSetter.EventDate` is `omit.Val[time.Time]`. The proto field is `*timestamppb.Timestamp`. Only set the DB field when `req.Msg.GetEventDate() != nil`; convert via `.AsTime()`.
- `EventSetter` has a `FinalizedAt` nullable field not present in the proto. Leave it un-set.
- `UpdateEventRequest.GetEventId()` provides the entity identifier.
- `Status` and `ProcessingState` are plain strings in both proto and DB model – no enum conversion needed.

## Implementation Plan

1. **Create `services/command/events.go`**
    - Define `eventRequest` interface:
        - `GetSeasonId() uint32`
        - `GetTrackLayoutId() uint32`
        - `GetName() string`
        - `GetEventDate() *timestamppb.Timestamp`
        - `GetStatus() string`
        - `GetProcessingState() string`
    - Define `eventSetterBuilder` with `Build(msg eventRequest) *models.EventSetter`.
    - Implement `CreateEvent`:
        - Build setter.
        - Set `CreatedBy` and `UpdatedBy` inside transaction.
        - Call `s.repo.Events().Create(ctx, setter)`.
        - Return `CreateEventResponse` with converted model.
    - Implement `UpdateEvent`:
        - Build setter.
        - Set `UpdatedAt` and `UpdatedBy` inside transaction.
        - Call `s.repo.Events().Update(ctx, int32(req.Msg.GetEventId()), setter)`.
        - Return `UpdateEventResponse`.
    - Implement `DeleteEvent`:
        - Call `s.repo.Events().DeleteByID(ctx, int32(req.Msg.GetEventId()))`.
        - Return `DeleteEventResponse{Deleted: true}`.

2. **Add conversion function in `services/conversion/service.go`**
    - `EventToEvent` – maps `ID`, `SeasonID`, `TrackLayoutID`, `Name`, `EventDate` (as `*timestamppb.Timestamp`), `Status`, `ProcessingState`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.EventErrors.ErrUniqueEventsSeasonIdNameUnique` → `connect.CodeAlreadyExists`

4. **Create `services/command/events_test.go`**

    Keep tests in package `command`.

    Add shared seed helpers to `test_setup_test.go` (reuse from other issues where applicable):
    - `seedEvent(t, repo, seasonID, trackLayoutID, name)` – inserts an `Event` row with `Status: "planned"`, `ProcessingState: "idle"`, `CreatedBy: testUserSeed` and returns the model.
    - Add truncations for `events`, `seasons`, `track_layouts`, `tracks`, `series`, `racing_sims`, `point_systems` in dependency order to `resetTestTables`. Consolidate into a single `TRUNCATE … CASCADE` on the root tables if most tests run the full hierarchy.

    Tests for `eventSetterBuilder.Build`:
    - Success: maps `SeasonId`, `TrackLayoutId`, `Name`, `Status`, `ProcessingState`.
    - `EventDate`: `nil` timestamp leaves `EventDate` unset; non-nil timestamp is converted to `time.Time` via `.AsTime()`.

    Tests for `CreateEvent`:
    - `TestCreateEventSuccess` – verifies response fields, checks `CreatedBy`/`UpdatedBy` in DB, checks `SeasonID` and `TrackLayoutID` stored correctly; seed a full hierarchy (simulation → series → point_system → season, and track → track_layout).
    - `TestCreateEventFailureDuplicateNameSameSeason` – expects `connect.CodeAlreadyExists`.
    - `TestCreateEventSuccessDuplicateNameDifferentSeason` – same name under different season should succeed.
    - `TestCreateEventFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateEvent`:
    - `TestUpdateEventSuccess` – verifies updated name/status and that `UpdatedBy`/`UpdatedAt` advance.
    - `TestUpdateEventFailureNotFound` – expects `connect.CodeNotFound`.
    - `TestUpdateEventFailureDuplicateNameSameSeason` – expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteEvent`:
    - `TestDeleteEventSuccess` – verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteEventFailureTransactionError` – uses `txManagerStub`; expects `connect.CodeInternal`.
