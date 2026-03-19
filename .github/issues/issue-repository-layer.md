# Feature: Add Repository Layer by Entity Group

## Summary

Introduce a repository layer that provides typed access to persisted entities through interfaces.

The repository layer should be organized by entity groups derived from migration scripts (one module per migration group). Each entity module must expose a consistent contract:

- LoadByID
- DeleteByID
- Create
- Update

## Why

Service code currently needs a stable abstraction for persistence operations.

A repository layer improves separation of concerns, enables easier testing via mocks/fakes, and centralizes persistence behavior for each entity group.

Using migration-defined groups keeps module boundaries aligned with schema ownership and evolution.

## Goals

- Add a repository package with modules grouped by migration domain.
- Expose one interface per entity module.
- Enforce a common method surface across modules:
    - LoadByID
    - DeleteByID
    - Create
    - Update
- Keep business services dependent on repository interfaces, not direct DB table access.
- collect all entity repositories in an additional interface `Repository`
- provide a full implementation for these interfaces in testsupport/repository module. This implementation should be based on maps and should provide meaningful and consistent sample data.
- Enable straightforward mocking in unit tests.

## Non-Goals

- Replacing generated db models/query builders.
- Introducing generic "one-size-fits-all" repository base types.
- Redesigning table schema or migration history.

## Entity Group Modules (from migrations)

- racing_sims
- point_systems (including point_rules)
- drivers (including driver_simulation_ids)
- tracks (including track_layouts and simulation_track_layout_aliases)
- cars (including car_manufacturers, car_brands, car_models, simulation_car_aliases)
- series
- seasons
- events
- races
- teams (including team_drivers)
- import_batches
- result_entries
- booking_entries
- event_processing_audit
- standings (including season_driver_standings, season_team_standings, event_driver_standings, event_team_standings)

## Interface Contract Requirements

Each module should provide an interface following this method shape (exact signatures may vary by entity type and key type):

- LoadByID(ctx, id) -> entity, error
- DeleteByID(ctx, id) -> error
- Create(ctx, input) -> entity, error
- Update(ctx, id, input) -> entity, error

Contract expectations:

- Use context.Context in all methods.
- Return domain-meaningful errors (not raw driver errors) where appropriate.
- Keep method naming consistent across all entity modules.

## Implementation Plan

1. Add repository package structure

- Create top-level repository package and one submodule per entity group.
- Keep module naming aligned with migration group names.

2. Define module interfaces

- Add one interface per module with required methods:
    - LoadByID
    - DeleteByID
    - Create
    - Update
- Keep method names identical across modules.

3. Add concrete implementations

- Implement interfaces using existing DB access patterns in the project.
- Keep SQL/query concerns encapsulated within repository implementations.

4. Integrate with services

- Refactor service constructors to depend on interfaces.
- Replace direct persistence usage in service logic with repositories.

5. Add tests

- Unit tests for each repository module covering success and failure paths.
- Service tests that use mocked repository interfaces.

## Acceptance Criteria

- A repository module exists for each migration-defined entity group.
- Every module exposes an interface with LoadByID, DeleteByID, Create, Update.
- Service layer depends on repository interfaces rather than direct DB model operations.
- Errors are mapped to project-level domain/database error conventions.
- Unit tests validate repository method behavior for representative modules.
- Existing functionality continues to pass test suite.

## Suggested Task Breakdown

- [ ] Create repository package and module directories by entity group
- [ ] Define interface contract in each module
- [ ] Implement repositories for high-use modules first (series/seasons/events/races)
- [ ] Implement remaining modules
- [ ] Refactor service constructors and wiring to use repository interfaces
- [ ] Add unit tests for repository implementations
- [ ] Add/update docs in README-dev for repository conventions

## Open Questions

- Should interfaces be per-table or per-group aggregate when a group has multiple tables?
- Should Update support partial updates (patch semantics) or full replacement semantics?
- What is the canonical error mapping strategy for not-found vs validation vs conflict errors?
- Do we need transaction-aware repository variants for multi-entity write operations?
