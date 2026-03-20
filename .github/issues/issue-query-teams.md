# Feature: Implement Query Service for Teams

## Summary

Implement the query service handlers for `Team` in `services/query/teams.go`:

- `ListTeams`
- `GetTeam`

Also add missing `LoadAll` and `LoadBySeasonID` methods to `TeamsRepository` in `repository/teams/teams.go`.

Use `services/query/series.go` as the primary implementation reference.

## Why

The `QueryServiceHandler` interface requires these methods. Teams are season-scoped and must be listable for roster, standings, and event context views.

## Prerequisites

- Conversion function `TeamToTeam` from `issue-command-teams.md` must be available in `services/conversion/service.go`.

## Goals

- Add `LoadAll` and `LoadBySeasonID` to `TeamsRepository` and its implementation.
- Implement `ListTeams` and `GetTeam` in `services/query/teams.go`.
- `ListTeams` accepts optional season filter:
    - if `req.Msg.GetSeasonId() != 0`, call `LoadBySeasonID`
    - otherwise call `LoadAll`
- `GetTeam` resolves by `req.Msg.GetId()` via `LoadByID`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- Team-driver membership query methods.
- Team standings queries.
- Command (write) handlers - covered by `issue-command-teams.md`.

## Implementation Plan

1. **Extend `repository/teams/teams.go`**
    - Add to `TeamsRepository`:
        ```go
        LoadAll(ctx context.Context) ([]*models.Team, error)
        LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.Team, error)
        ```
    - Add implementations on `teamsRepository`:

        ```go
        func (r *teamsRepository) LoadAll(ctx context.Context) ([]*models.Team, error) {
            return models.Teams.Query().All(ctx, r.getExecutor(ctx))
        }

        func (r *teamsRepository) LoadBySeasonID(ctx context.Context, seasonID int32) ([]*models.Team, error) {
            return models.Teams.Query(
                sm.Where(models.Teams.Columns.SeasonID.EQ(psql.Arg(seasonID))),
            ).All(ctx, r.getExecutor(ctx))
        }
        ```

2. **Create `services/query/teams.go`**
    - Implement `ListTeams`:
        - choose repo method based on `season_id`.
        - convert with `s.conversion.TeamToTeam`.
        - return `ListTeamsResponse{Items: items}`.
    - Implement `GetTeam`:
        - call `s.repo.Teams().Teams().LoadByID(ctx, int32(req.Msg.GetId()))`.
        - return `GetTeamResponse{Team: s.conversion.TeamToTeam(item)}`.

3. **Create `services/query/teams_test.go`**

    Package `query`. Use `newDBBackedQueryService(t)`.

    Local helper in this test file:
    - `seedTeam(t, repo, seasonID, name)` creating team rows with `IsActive: true`, audit fields set to `testUserSeed`.

    Update `resetTestTables` in `services/query/test_setup_test.go`:
    - include `teams` in truncate list.

    Tests for `ListTeams`:
    - `TestListTeamsEmpty` - seeds nothing; verifies empty list.
    - `TestListTeamsReturnsAll` - seeds one season with two teams; verifies both returned.
    - `TestListTeamsBySeasonID` - seeds two seasons with one team each; filter by first season; verifies exactly one result with matching `GetSeasonId()`.

    Tests for `GetTeam`:
    - `TestGetTeamSuccess` - verifies `GetId()`, `GetSeasonId()`, `GetName()`, `GetIsActive()`.
    - `TestGetTeamNotFound` - expects `connect.CodeNotFound`.
