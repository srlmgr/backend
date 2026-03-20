# Feature: Implement Query Service for Tracks

## Summary

Implement the query service handlers for `Track` and `TrackLayout` in `services/query/tracks.go`:

- `ListTracks`
- `GetTrack`
- `ListTrackLayouts`
- `GetTrackLayout`

Also add missing `LoadAll` to `TracksRepository`, and `LoadAll` and `LoadByTrackID` to `TrackLayoutsRepository` in `repository/tracks/tracks.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires all four methods. Track and layout data is needed by clients when creating events and displaying circuit information.

## Prerequisites

- Conversion functions `TrackToTrack` and `TrackLayoutToTrackLayout` from `issue-command-tracks.md` must be available in `services/conversion/service.go`.

## Goals

- Extend `TracksRepository` with `LoadAll`.
- Extend `TrackLayoutsRepository` with `LoadAll` and `LoadByTrackID`.
- Implement all four query handlers in `services/query/tracks.go`.
- `ListTracks` has no proto filter – always call `LoadAll` on `TracksRepository`.
- `ListTrackLayouts` accepts an optional `track_id` filter (`req.Msg.GetTrackId()`):
    - If non-zero, call `LoadByTrackID`.
    - Otherwise, call `LoadAll`.
- `GetTrack` and `GetTrackLayout` resolve via `LoadByID` using `req.Msg.GetId()`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `SimulationTrackLayoutAlias` query handlers – not in the current query proto.
- Command (write) handlers – covered by `issue-command-tracks.md`.

## Implementation Plan

1. **Extend `repository/tracks/tracks.go`**
    - Add `LoadAll(ctx context.Context) ([]*models.Track, error)` to `TracksRepository` interface.
    - Add `LoadAll(ctx context.Context) ([]*models.TrackLayout, error)` to `TrackLayoutsRepository` interface.
    - Add `LoadByTrackID(ctx context.Context, trackID int32) ([]*models.TrackLayout, error)` to `TrackLayoutsRepository` interface.
    - Implement all three on their respective concrete repository types:

        ```go
        func (r *tracksRepository) LoadAll(ctx context.Context) ([]*models.Track, error) {
            return models.Tracks.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *trackLayoutsRepository) LoadAll(ctx context.Context) ([]*models.TrackLayout, error) {
            return models.TrackLayouts.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *trackLayoutsRepository) LoadByTrackID(ctx context.Context, trackID int32) ([]*models.TrackLayout, error) {
            return models.TrackLayouts.Query(
                sm.Where(models.TrackLayouts.Columns.TrackID.EQ(psql.Arg(trackID))),
            ).All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/tracks.go`**
    - Implement `ListTracks`:
        - Call `s.repo.Tracks().Tracks().LoadAll(ctx)`.
        - Return `ListTracksResponse{Items: ...}`.
    - Implement `GetTrack`:
        - Call `s.repo.Tracks().Tracks().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetTrackResponse{Track: ...}`.
    - Implement `ListTrackLayouts`:
        - If `trackID := int32(req.Msg.GetTrackId()); trackID != 0`, call `LoadByTrackID`.
        - Otherwise call `LoadAll`.
        - Return `ListTrackLayoutsResponse{Items: ...}`.
    - Implement `GetTrackLayout`:
        - Call `s.repo.Tracks().TrackLayouts().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetTrackLayoutResponse{TrackLayout: ...}`.

3. **Create `services/query/tracks_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` and the shared seed helpers from `test_setup_test.go` (`seedTrack`, `seedTrackLayout`).

    Tests for `ListTracks`:
    - `TestListTracksEmpty` — seeds nothing; verifies `GetItems()` is empty.
    - `TestListTracksReturnsAll` — seeds 2 tracks (`"Monza"`, `"Spa"`); verifies both appear in the response.

    Tests for `GetTrack`:
    - `TestGetTrackSuccess` — seeds one track; calls `GetTrack` with its ID; verifies `GetId()` and `GetName()`.
    - `TestGetTrackNotFound` — calls `GetTrack` with a non-existent ID; expects `connect.CodeNotFound`.

    Tests for `ListTrackLayouts`:
    - `TestListTrackLayoutsEmpty` — seeds nothing; verifies `GetItems()` is empty.
    - `TestListTrackLayoutsReturnsAll` — seeds 1 track with 2 layouts; verifies both layouts are returned.
    - `TestListTrackLayoutsByTrackID` — seeds 2 tracks each with 1 layout; calls `ListTrackLayouts` filtering by the first track ID; verifies exactly 1 layout is returned and its `GetTrackId()` matches.

    Tests for `GetTrackLayout`:
    - `TestGetTrackLayoutSuccess` — seeds track → layout; calls `GetTrackLayout` with the layout ID; verifies `GetId()`, `GetTrackId()`, and `GetName()`.
    - `TestGetTrackLayoutNotFound` — calls `GetTrackLayout` with a non-existent ID; expects `connect.CodeNotFound`.
