# Feature: Add Unit Tests for Command Simulation Service

## Summary

Add database-backed tests for all methods in [services/command/simulation.go](services/command/simulation.go):

- `racingSimSetterBuilder.Build`
- `(*service).CreateSimulation`
- `(*service).UpdateSimulation`
- `(*service).DeleteSimulation`

Tests must include both successful and failure scenarios, with service-method tests executing against a real Postgres-backed repository.

## Why

Simulation command handlers perform request-to-model mapping, transactional writes, and RPC error mapping. Missing tests here increase regression risk when updating command behavior, transaction handling, or conversion logic.

## Goals

- Cover successful execution for every method in the file.
- Cover representative failure paths for every method in the file.
- Validate behavior that is easy to regress:
    - import format conversion from proto enums
    - transaction handling and error mapping
    - write metadata set by service (`CreatedBy`, `UpdatedBy`, `UpdatedAt`)

## Non-Goals

- End-to-end gRPC server tests.
- Refactoring business logic in simulation handlers.

## Implementation Plan

1. Add a new test file

- Create [services/command/simulation_test.go](services/command/simulation_test.go)
- Keep tests in package `command` to test unexported builder logic directly.

2. Add test scaffolding

- Use the database test helpers in [testsupport/testdb/setup.go](testsupport/testdb/setup.go) and [testsupport/tcpostgres/setuptestdb.go](testsupport/tcpostgres/setuptestdb.go) to provision a migrated Postgres database for tests.
- Construct the real repository and transaction manager from the database pool.
- Add helper setup/cleanup so each test starts from a known database state.
- Keep a narrow transaction-manager wrapper only where needed to force transaction-manager failure paths that cannot be produced reliably from request input alone.

3. Add tests for `racingSimSetterBuilder.Build`

- Success:
    - maps `Name`
    - maps `IsActive=true`
    - maps supported formats (`JSON`, `CSV`)
- Failure:
    - invalid/unsupported import format enum returns error

4. Add tests for `CreateSimulation`

- Success:
    - returns created simulation response
    - persists mapped fields in Postgres
    - sets `CreatedBy`/`UpdatedBy` from principal context
- Failure:
    - invalid format in request returns `connect.CodeInvalidArgument`
    - forced transaction-manager failure returns mapped internal RPC code
    - create two simulations with the same name

5. Add tests for `UpdateSimulation`

- Success:
    - updates existing simulation row and returns updated payload
    - sets `UpdatedBy` and `UpdatedAt`
- Failure:
    - invalid format in request returns `connect.CodeInvalidArgument`
    - update of missing ID returns `connect.CodeNotFound`

6. Add tests for `DeleteSimulation`

- Success:
    - delete returns `Deleted=true`
    - entity row is no longer loadable from repository
- Failure:
    - forced transaction-manager failure returns mapped internal RPC code
    - update a simulation with a name of an already existing simulation

7. Validate

- Run focused tests for [services/command/simulation_test.go](services/command/simulation_test.go).
- Fix test/lint issues if any.

## Acceptance Criteria

- Every method in [services/command/simulation.go](services/command/simulation.go) has at least one success-path test.
- Every method in [services/command/simulation.go](services/command/simulation.go) has at least one failure-path test.
- Tests verify RPC error code mapping for invalid arguments, not-found, and transaction failures.
- Service-method tests use a real Postgres-backed repository rather than the in-memory test repository.
- Tests run successfully with `go test` for the command package.
