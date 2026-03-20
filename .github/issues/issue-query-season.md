# Feature: Implement Query Service for Season

## Summary

Implement the query service handlers for `Season` in `services/query/season.go`:

- `ListSeasons`
- `GetSeason`

Also add missing `LoadAll` and `LoadBySeriesID` methods to the `seasons.Repository` interface in `repository/seasons/seasons.go`.

Use `services/query/series.go` as the primary implementation reference (it demonstrates the same `LoadAll` / `LoadByParentID` pattern).

## Why

The `QueryServiceHandler` interface requires these methods. Seasons are the primary grouping entity for events and standings — they must be listable and retrievable before any downstream rendering can work.

## Prerequisites

- Conversion function `SeasonToSeason` from `issue-command-season.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll` and `LoadBySeriesID` to the `seasons.Repository` interface and its concrete implementation.
- Implement `ListSeasons` and `GetSeason` in `services/query/season.go`.
- `ListSeasons` accepts an optional `series_id` filter (`req.Msg.GetSeriesId()`):
    - If non-zero, call `LoadBySeriesID`.
    - Otherwise, call `LoadAll`.
- `GetSeason` uses `req.Msg.GetId()` resolved via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Command (write) handlers – covered by `issue-command-season.md`.

## Implementation Plan

1. **Extend `repository/seasons/seasons.go`**
    - Add to the `Repository` interface:
        ```go
        LoadAll(ctx context.Context) ([]*models.Season, error)
        LoadBySeriesID(ctx context.Context, seriesID int32) ([]*models.Season, error)
        ```
    - Add implementations on `seasonsRepository` following the `series` repository pattern:

        ```go
        func (r *seasonsRepository) LoadAll(ctx context.Context) ([]*models.Season, error) {
            return models.Seasons.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *seasonsRepository) LoadBySeriesID(ctx context.Context, seriesID int32) ([]*models.Season, error) {
            return models.Seasons.Query(
                sm.Where(models.Seasons.Columns.SeriesID.EQ(psql.Arg(seriesID))),
            ).All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/season.go`**
    - Implement `ListSeasons`:
        - If `seriesID := int32(req.Msg.GetSeriesId()); seriesID != 0`, call `s.repo.Seasons().LoadBySeriesID(ctx, seriesID)`.
        - Otherwise call `s.repo.Seasons().LoadAll(ctx)`.
        - Convert with `s.conversion.SeasonToSeason` and return `ListSeasonsResponse{Items: items}`.
    - Implement `GetSeason`:
        - Call `s.repo.Seasons().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - Return `GetSeasonResponse{Season: s.conversion.SeasonToSeason(item)}`.

3. **Create `services/query/season_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)` and the shared seed helpers from `test_setup_test.go` (`seedSimulation`, `seedSeries`, `seedSeason`).

    Tests for `ListSeasons`:
    - `TestListSeasonsEmpty` — seeds nothing; verifies `GetItems()` is empty.
    - `TestListSeasonsReturnsAll` — seeds sim → series → 2 seasons; verifies both appear in the response with correct `GetSeriesId()`.
    - `TestListSeasonsBySeriesID` — seeds sim → 2 series, each with 1 season; calls `ListSeasons` filtering by the first series ID; verifies exactly 1 season is returned and its `GetSeriesId()` matches.

    Tests for `GetSeason`:
    - `TestGetSeasonSuccess` — seeds sim → series → season; calls `GetSeason` with the season ID; verifies `GetId()`, `GetSeriesId()`, and `GetName()`.
    - `TestGetSeasonNotFound` — calls `GetSeason` with a non-existent ID; expects `connect.CodeNotFound`.
