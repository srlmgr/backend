# Feature: Implement Query Service for Races

## Summary

Implement the query service handlers for `Race` in `services/query/races.go`:

- `ListRaces`
- `GetRace`

Also add missing `LoadAll` and `LoadByEventID` methods to the `races.Repository` interface in `repository/races/races.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

Races are the per-event session rows consumed by scheduling UIs, result browsers, and import tooling. They should be listable and directly retrievable by ID in the query service.

## Prerequisites

- Query API contract from `issue-api-query-races.md` must be completed first.
- Query API must expose race RPCs and request/response messages:
    - `ListRaces(ListRacesRequest) returns (ListRacesResponse)`
    - `GetRace(GetRaceRequest) returns (GetRaceResponse)`
- Regenerate protobuf/connect Go code after API updates.
- Conversion function `RaceToRace` from `issue-command-races.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll` and `LoadByEventID` to `races.Repository` and its concrete implementation.
- Implement `ListRaces` and `GetRace` in `services/query/races.go`.
- `ListRaces` accepts an optional `event_id` filter (`req.Msg.GetEventId()`):
    - If non-zero, call `LoadByEventID`.
    - Otherwise, call `LoadAll`.
- `GetRace` uses `req.Msg.GetId()` resolved via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.
- Use explicit persisted string <-> proto enum mapping for `Race.session_type`:
    - Unknown stored DB strings map to enum `UNSPECIFIED` and emit a warning log.

## Non-Goals

- Event results aggregation (`GetEventResults`) and booking-entry calculations.
- Command (write) handlers - covered by `issue-command-races.md`.

## Implementation Plan

1. **Extend `repository/races/races.go`**
    - Add to the `Repository` interface:
        ```go
        LoadAll(ctx context.Context) ([]*models.Race, error)
        LoadByEventID(ctx context.Context, eventID int32) ([]*models.Race, error)
        ```
    - Add implementations on `racesRepository` following the `series`/`events` repository pattern:

        ```go
        func (r *racesRepository) LoadAll(ctx context.Context) ([]*models.Race, error) {
            return models.Races.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *racesRepository) LoadByEventID(ctx context.Context, eventID int32) ([]*models.Race, error) {
            return models.Races.Query(
                sm.Where(models.Races.Columns.EventID.EQ(psql.Arg(eventID))),
            ).All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/races.go`**
    - Implement `ListRaces`:
        - If `eventID := int32(req.Msg.GetEventId()); eventID != 0`, call `s.repo.Races().LoadByEventID(ctx, eventID)`.
        - Otherwise call `s.repo.Races().LoadAll(ctx)`.
        - Convert each item with `s.conversion.RaceToRace` and return `ListRacesResponse{Items: items}`.
    - Implement `GetRace`:
        - Call `s.repo.Races().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetRaceResponse{Race: s.conversion.RaceToRace(item)}`.

3. **Create `services/query/races_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` and shared seed helpers from `test_setup_test.go` plus `seedEvent` from `events_test.go`.

    Add local seed helper in this file:

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

    Seeding hierarchy:
    - `sim -> series -> pointSystem -> season -> track -> trackLayout -> event -> race`.

    Tests for `ListRaces`:
    - `TestListRacesEmpty` - seeds nothing; verifies `GetItems()` is empty.
    - `TestListRacesReturnsAll` - seeds one event and two races; verifies both races are returned.
    - `TestListRacesByEventID` - seeds two events with one race each; filters by first event ID; verifies exactly one item with matching `GetEventId()`.

    Tests for `GetRace`:
    - `TestGetRaceSuccess` - seeds full hierarchy; verifies `GetId()`, `GetEventId()`, `GetName()`, `GetSessionType()`, and `GetSequenceNo()`.
    - `TestGetRaceNotFound` - calls `GetRace` with a non-existent ID; expects `connect.CodeNotFound`.

## Notes

- At the time of writing, generated query bindings in this repository do not yet include `ListRacesRequest`/`GetRaceRequest`. Complete the API prerequisite first, then implement this issue.
- `GetSessionType()` is enum-based in protobuf; tests should assert enum values (not raw strings).
