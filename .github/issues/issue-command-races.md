# Feature: Implement Command Service for Races

## Summary

Implement the command service handlers for `Race` in `services/command/races.go`:

- `CreateRace`
- `UpdateRace`
- `DeleteRace`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires these methods. Races are the execution units under an event (practice/qualifying/race sessions) and are referenced by imports and results.

## Goals

- Implement `CreateRace`, `UpdateRace`, `DeleteRace` in a new file `services/command/races.go`.
- Add a `raceSetterBuilder` struct following the setter-builder pattern.
- Add conversion function to `services/conversion/service.go`:
    - `RaceToRace(model *models.Race) *commonv1.Race`
- Map all writable proto fields to `models.RaceSetter`:
    - `EventId` -> `setter.EventID`
    - `Name` -> `setter.Name`
    - `SessionType` (enum) -> `setter.SessionType` (persisted DB string)
    - `SequenceNo` -> `setter.SequenceNo`
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` to `time.Now()` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Query (read) handlers - covered by `issue-query-races.md`.
- Result-entry ingestion and processing workflows.
- End-to-end gRPC server tests.

## Implementation Notes

- `CreateRaceRequest` fields are:
    - `GetEventId() uint32`
    - `GetName() string`
    - `GetSessionType() backend.common.v1.RaceSessionType`
    - `GetSequenceNo() int32`
- `UpdateRaceRequest.GetRaceId()` provides the target entity identifier.
- `SequenceNo` is already `int32` in proto and DB (`models.RaceSetter.SequenceNo`), so no conversion is required.
- `SessionType` requires explicit enum <-> DB string mapping:
    - Reject unsupported inbound enum values with `connect.CodeInvalidArgument`.
    - Convert known enum values to canonical DB strings for writes.
    - Convert unknown stored DB strings to proto `UNSPECIFIED` in reads and log a warning.
- There are two uniqueness constraints on `races` that should map to `connect.CodeAlreadyExists`:
    - `dberrors.RaceErrors.ErrUniqueRacesEventIdNameUnique`
    - `dberrors.RaceErrors.ErrUniqueRacesEventIdSequenceNoUnique`

## Implementation Plan

1. **Create `services/command/races.go`**
    - Define `raceRequest` interface:
        - `GetEventId() uint32`
        - `GetName() string`
        - `GetSessionType() string`
        - `GetSequenceNo() int32`
    - Define `raceSetterBuilder` with `Build(msg raceRequest) *models.RaceSetter`.
    - Implement `CreateRace`:
        - Build setter.
        - Set `CreatedBy` and `UpdatedBy` inside transaction.
        - Call `s.repo.Races().Create(ctx, setter)`.
        - Return `CreateRaceResponse` with converted model.
    - Implement `UpdateRace`:
        - Build setter.
        - Set `UpdatedAt` and `UpdatedBy` inside transaction.
        - Call `s.repo.Races().Update(ctx, int32(req.Msg.GetRaceId()), setter)`.
        - Return `UpdateRaceResponse`.
    - Implement `DeleteRace`:
        - Call `s.repo.Races().DeleteByID(ctx, int32(req.Msg.GetRaceId()))`.
        - Return `DeleteRaceResponse{Deleted: true}`.

2. **Add conversion function in `services/conversion/service.go`**
    - `RaceToRace` maps `ID`, `EventID`, `Name`, `SessionType`, `SequenceNo`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.RaceErrors.ErrUniqueRacesEventIdNameUnique` -> `connect.CodeAlreadyExists`
        - `dberrors.RaceErrors.ErrUniqueRacesEventIdSequenceNoUnique` -> `connect.CodeAlreadyExists`

4. **Create `services/command/races_test.go`**

    Keep tests in package `command`.

    Use existing helpers from `test_setup_test.go` and `events_test.go`:
    - `seedSimulation`, `seedSeries`, `seedPointSystem`, `seedSeason`, `seedTrack`, `seedTrackLayout`, `seedEvent`.

    Add local seed helper in `races_test.go`:

    ```go
    func seedRace(t *testing.T, repo rootrepo.Repository, eventID int32, name, sessionType string, sequenceNo int32) *models.Race {
        t.Helper()
        race, err := repo.Races().Create(context.Background(), &models.RaceSetter{
            EventID:     omit.From(eventID),
            Name:        omit.From(name),
            SessionType: omit.From(sessionType),
            SequenceNo:  omit.From(sequenceNo),
            CreatedBy:   omit.From(testUserSeed),
            UpdatedBy:   omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed race %q: %v", name, err)
        }
        return race
    }
    ```

    Seeding hierarchy for race tests:
    - `sim -> series -> pointSystem -> season -> track -> trackLayout -> event -> race`.

    Tests for `raceSetterBuilder.Build`:
    - Success: maps `EventId`, `Name`, `SessionType`, `SequenceNo`.
    - Zero/empty values: `EventId == 0`, empty `Name`/`SessionType` leave fields unset; `SequenceNo == 0` follows existing setter-builder conventions.

    Tests for `CreateRace`:
    - `TestCreateRaceSuccess` - verifies response fields, checks `CreatedBy`/`UpdatedBy` in DB, and verifies `EventID` persisted correctly.
    - `TestCreateRaceFailureDuplicateNameSameEvent` - expects `connect.CodeAlreadyExists`.
    - `TestCreateRaceFailureDuplicateSequenceSameEvent` - expects `connect.CodeAlreadyExists`.
    - `TestCreateRaceSuccessDuplicateNameDifferentEvent` - same race name under different event should succeed.
    - `TestCreateRaceFailureTransactionError` - uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateRace`:
    - `TestUpdateRaceSuccess` - verifies updated name/session_type/sequence_no and that `UpdatedBy`/`UpdatedAt` advance.
    - `TestUpdateRaceFailureNotFound` - expects `connect.CodeNotFound`.
    - `TestUpdateRaceFailureDuplicateNameSameEvent` - expects `connect.CodeAlreadyExists`; DB row unchanged.
    - `TestUpdateRaceFailureDuplicateSequenceSameEvent` - expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteRace`:
    - `TestDeleteRaceSuccess` - verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteRaceFailureTransactionError` - uses `txManagerStub`; expects `connect.CodeInternal`.
