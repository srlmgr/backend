# Feature: Implement Command Service for Teams

## Summary

Implement the command service handlers for `Team` in `services/command/teams.go`:

- `CreateTeam`
- `UpdateTeam`
- `DeleteTeam`

Use `services/command/series.go` as the primary implementation reference.

## Why

The `CommandServiceHandler` interface requires these methods. Teams are season-scoped entities required for team standings and team-based booking workflows.

## Goals

- Implement `CreateTeam`, `UpdateTeam`, `DeleteTeam` in `services/command/teams.go`.
- Add a `teamSetterBuilder` struct following the setter-builder pattern.
- Add conversion function to `services/conversion/service.go`:
    - `TeamToTeam(model *models.Team) *commonv1.Team`
- Map writable proto fields to `models.TeamSetter`:
    - `SeasonId` -> `setter.SeasonID`
    - `Name` -> `setter.Name`
    - `IsActive` -> `setter.IsActive`
- Set `CreatedBy` / `UpdatedBy` from `s.execUser(ctx)`.
- Set `UpdatedAt` on update.
- Wrap all writes in `s.withTx`.
- Map errors to Connect RPC codes via `s.conversion.MapErrorToRPCCode`.

## Non-Goals

- `AddDriverToTeam` command flow.
- Query (read) handlers - covered by `issue-query-teams.md`.
- End-to-end gRPC server tests.

## Implementation Notes

- Team uniqueness is scoped by season (`teams_season_id_name_unique`).
- `UpdateTeamRequest.GetTeamId()` provides the entity identifier.
- `TeamSetter` has no nullable optional fields relevant to this request; map only request-exposed fields.

## Implementation Plan

1. **Create `services/command/teams.go`**
    - Define `teamRequest` interface:
        - `GetSeasonId() uint32`
        - `GetName() string`
        - `GetIsActive() bool`
    - Define `teamSetterBuilder` with `Build(msg teamRequest) *models.TeamSetter`.
    - Implement `CreateTeam`:
        - Build setter.
        - Set `CreatedBy` and `UpdatedBy` in transaction.
        - Call `s.repo.Teams().Teams().Create(ctx, setter)`.
        - Return `CreateTeamResponse` with converted model.
    - Implement `UpdateTeam`:
        - Build setter.
        - Set `UpdatedAt` and `UpdatedBy` in transaction.
        - Call `s.repo.Teams().Teams().Update(ctx, int32(req.Msg.GetTeamId()), setter)`.
        - Return `UpdateTeamResponse`.
    - Implement `DeleteTeam`:
        - Call `s.repo.Teams().Teams().DeleteByID(ctx, int32(req.Msg.GetTeamId()))`.
        - Return `DeleteTeamResponse{Deleted: true}`.

2. **Add conversion function in `services/conversion/service.go`**
    - `TeamToTeam` maps `ID`, `SeasonID`, `Name`, `IsActive`.

3. **Wire up error sentinels**
    - Add mappings in `MapErrorToRPCCode` for:
        - `dberrors.TeamErrors.ErrUniqueTeamsSeasonIdNameUnique` -> `connect.CodeAlreadyExists`

4. **Create `services/command/teams_test.go`**

    Keep tests in package `command`.

    Add shared seed helper to `test_setup_test.go`:
    - `seedTeam(t, repo, seasonID, name)` - inserts `Team` with `IsActive: true`, `CreatedBy: testUserSeed`, `UpdatedBy: testUserSeed`.

    Reuse season hierarchy seeds (`seedSimulation`, `seedSeries`, `seedPointSystem`, `seedSeason`) to satisfy FK constraints.

    Tests for `teamSetterBuilder.Build`:
    - Success: maps `SeasonId`, `Name`, `IsActive`.
    - Zero values: empty name and `SeasonId == 0` leave those fields unset.

    Tests for `CreateTeam`:
    - `TestCreateTeamSuccess` - verifies response fields and DB audit fields.
    - `TestCreateTeamFailureDuplicateNameSameSeason` - expects `connect.CodeAlreadyExists`.
    - `TestCreateTeamSuccessDuplicateNameDifferentSeason` - same name in different season succeeds.
    - `TestCreateTeamFailureTransactionError` - uses `txManagerStub`; expects `connect.CodeInternal`.

    Tests for `UpdateTeam`:
    - `TestUpdateTeamSuccess` - verifies updated values and `UpdatedBy` / `UpdatedAt` advance.
    - `TestUpdateTeamFailureNotFound` - expects `connect.CodeNotFound`.
    - `TestUpdateTeamFailureDuplicateNameSameSeason` - expects `connect.CodeAlreadyExists`; DB row unchanged.

    Tests for `DeleteTeam`:
    - `TestDeleteTeamSuccess` - verifies `Deleted: true`; `LoadByID` returns `repoerrors.ErrNotFound`.
    - `TestDeleteTeamFailureTransactionError` - uses `txManagerStub`; expects `connect.CodeInternal`.
