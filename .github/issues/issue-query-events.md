# Feature: Implement Query Service for Events

## Summary

Implement the query service handlers for `Event` in `services/query/events.go`:

- `ListEvents`
- `GetEvent`

Also add missing `LoadAll` and `LoadBySeasonID` methods to the `events.Repository` interface in `repository/events/events.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires these methods. Events are the central scheduling entity linking seasons to track layouts — they must be listable per-season for any calendar or standings view to work.

## Prerequisites

- Conversion function `EventToEvent` from `issue-command-events.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll` and `LoadBySeasonID` to the `events.Repository` interface and its concrete implementation.
- Implement `ListEvents` and `GetEvent` in `services/query/events.go`.
- `ListEvents` accepts an optional `season_id` filter (`req.Msg.GetSeasonId()`):
    - If non-zero, call `LoadBySeasonID`.
    - Otherwise, call `LoadAll`.
- `GetEvent` uses `req.Msg.GetId()` resolved via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `GetEventResults` or `GetEventBookingEntries` – these are separate handlers with their own logic, handled elsewhere.
- Command (write) handlers – covered by `issue-command-events.md`.

## Implementation Plan

1. **Extend `repository/events/events.go`**
    - Add to the `Repository` interface:
        ```go
        LoadAll(ctx context.Context) ([]*models.Event, error)
        LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.Event, error)
        ```
    - Add implementations on `eventsRepository` following the `series` repository pattern:

        ```go
        func (r *eventsRepository) LoadAll(ctx context.Context) ([]*models.Event, error) {
            return models.Events.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *eventsRepository) LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.Event, error) {
            return models.Events.Query(
                sm.Where(models.Events.Columns.SeasonID.EQ(psql.Arg(seasonID))),
            ).All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/events.go`**
    - Implement `ListEvents`:
        - If `seasonID := int32(req.Msg.GetSeasonId()); seasonID != 0`, call `s.repo.Events().LoadBySeasonID(ctx, seasonID)`.
        - Otherwise call `s.repo.Events().LoadAll(ctx)`.
        - Convert each item with `s.conversion.EventToEvent` and return `ListEventsResponse{Items: items}`.
    - Implement `GetEvent`:
        - Call `s.repo.Events().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetEventResponse{Event: s.conversion.EventToEvent(item)}`.

3. **Create `services/query/events_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` and the shared seed helpers from `test_setup_test.go` (`seedSimulation`, `seedSeries`, `seedSeason`, `seedTrack`, `seedTrackLayout`).

    Local seed helper (defined in this file):

    ```go
    func seedEvent(t *testing.T, repo rootrepo.Repository, seasonID, trackLayoutID int32, name string) *models.Event {
        t.Helper()
        event, err := repo.Events().Create(context.Background(), &models.EventSetter{
            SeasonID:      omit.From(seasonID),
            TrackLayoutID: omit.From(trackLayoutID),
            Name:          omit.From(name),
            CreatedBy:     omit.From(testUserSeed),
            UpdatedBy:     omit.From(testUserSeed),
        })
        if err != nil {
            t.Fatalf("failed to seed event %q: %v", name, err)
        }
        return event
    }
    ```

    The full seeding hierarchy is: `sim → series → season` and `track → trackLayout`, then `seedEvent(season.ID, trackLayout.ID, name)`.

    Tests for `ListEvents`:
    - `TestListEventsEmpty` — seeds nothing; verifies `GetItems()` is empty.
    - `TestListEventsReturnsAll` — seeds sim → series → season, track → 2 track layouts, 2 events (one per layout, same season); verifies both events are returned.
    - `TestListEventsBySeasonID` — seeds sim → series → 2 seasons, track → layout, 1 event per season; filters by first season ID; verifies exactly 1 event returned with matching `GetSeasonId()`.

    Tests for `GetEvent`:
    - `TestGetEventSuccess` — seeds the full hierarchy → event; calls `GetEvent` with its ID; verifies `GetId()`, `GetSeasonId()`, and `GetName()`.
    - `TestGetEventNotFound` — calls `GetEvent` with a non-existent ID; expects `connect.CodeNotFound`.
