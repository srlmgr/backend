# Feature: Implement Import Service Workflow

## Summary

Implement all `ImportService` handlers in `services/importsvc/`:

- `UploadResultsFile`
- `GetPreprocessPreview`
- `ApplyResultEdits`
- `ComputeDriverBookingEntries`
- `ComputeTeamBookingEntries`
- `FinalizeEventProcessing`

Use `services/command/series.go` and `services/query/series.go` as structure references for logging, error mapping, transaction boundaries, and response construction.

## Why

The Import API contract is already generated and wired in authz, but `services/importsvc/service.go` only exposes a constructor with no concrete handler methods.

Without these handlers:

- result payloads cannot be uploaded,
- preprocessing previews cannot be retrieved,
- edited rows cannot be applied,
- booking entries cannot be computed,
- event processing cannot be finalized.

This blocks the end-to-end event processing pipeline.

## Prerequisites

- `issue-command-events.md` (event write path and conversion support)
- `issue-command-resultentries.md` (result entry write path and conversion support)
- `issue-query-events.md` (event query path used by test setup and pipeline validation)

## Goals

- Implement all six import handlers in `services/importsvc/`.
- Add service-level helpers analogous to command service:
  - `withTx(ctx, fn)`
  - `principal(ctx)` and `execUser(ctx)`
- Add `conversion *conversion.Service` to import service struct and initialize it in `New`.
- Use `s.conversion.MapErrorToRPCCode(err)` for Connect RPC error mapping.
- Keep all write operations inside `s.withTx`.
- Update event/import-batch processing states consistently across workflow steps.
- Create event processing audit rows for every state transition.

## Non-Goals

- Replacing generated Bob models.
- Refactoring authz policy mappings (already present in `authz/mapping.go`).
- Adding new protobuf RPCs or changing proto payload shapes.
- Implementing UI-facing admin flows unrelated to import pipeline.

## Processing State Contract

Use existing DB state constraints:

- Event: `draft` -> `raw_imported` -> `preprocessed` -> `driver_entries_computed` -> `team_entries_computed` -> `finalized`
- Import batch: `raw_imported` -> `preprocessed` -> `driver_entries_computed` -> `team_entries_computed` -> `finalized` (or `failed`)

Each transition should:

- update `events.processing_state` (and `events.finalized_at` on finalize),
- update `import_batches.processing_state` where applicable,
- insert `event_processing_audit` with:
  - `event_id`
  - optional `import_batch_id`
  - `from_state`
  - `to_state`
  - `action`
  - `payload_json` (small structured metadata)
  - `created_by` / `updated_by`

## Repository Gaps to Close

Extend repository interfaces/implementations with methods required by import handlers (keep naming aligned with current style):

1. `repository/importbatches/importbatches.go`
   - `LoadLatestByEventIDAndRaceID(ctx context.Context, eventID, raceID int32) (*models.ImportBatch, error)`
   - Optional if needed by tests/workflow:
     - `LoadByEventIDAndRaceID(ctx context.Context, eventID, raceID int32) ([]*models.ImportBatch, error)`

2. `repository/resultentries/resultentries.go`
   - `LoadByImportBatchID(ctx context.Context, importBatchID int32) ([]*models.ResultEntry, error)`
   - `LoadByRaceID(ctx context.Context, raceID int32) ([]*models.ResultEntry, error)`

3. `repository/bookingentries/bookingentries.go`
   - `DeleteByEventIDAndSourceType(ctx context.Context, eventID int32, sourceType string) error`
   - Optional read helpers for assertions:
     - `LoadByEventID(ctx context.Context, eventID int32) ([]*models.BookingEntry, error)`

4. `repository/eventprocessingaudit/eventprocessingaudit.go`
   - Optional read helper for tests:
     - `LoadByEventID(ctx context.Context, eventID int32) ([]*models.EventProcessingAudit, error)`

## Implementation Plan

1. **Extend `services/importsvc/service.go`**
   - Add `conversion *conversion.Service` field.
   - Initialize conversion in `New(...)`.
   - Add:
     - `withTx(...)`
     - `principal(...)`
     - `execUser(...)`
   - Reuse patterns from command service (`services/command/service.go`).

2. **Add `services/importsvc/upload.go` (`UploadResultsFile`)**
   - Validate `event_id`, `race_id`, and payload are present.
   - Validate race belongs to event (`races.LoadByID` + `race.EventID == req.EventId`).
   - Convert proto `ImportFormat` to persisted format string.
   - Inside transaction:
     - create `import_batches` row,
     - set batch state to `raw_imported`,
     - update event state to `raw_imported`,
     - create audit row with action `upload_results_file`.
   - Return `UploadResultsFileResponse{ImportBatchId, ProcessingState}`.

3. **Add `services/importsvc/preprocess.go` (`GetPreprocessPreview`)**
   - Resolve latest import batch for `(event_id, race_id)`.
   - Load imported result entries tied to that batch.
   - Build unresolved mappings list from rows missing canonical IDs (driver/car model), with mapping types (e.g., `driver`, `car_model`).
   - Return:
     - `rows` via `s.conversion.ResultEntryToResultEntry`,
     - `unresolved_mappings` via direct proto construction.
   - If needed by workflow consistency, transition state to `preprocessed` in transaction and write audit action `get_preprocess_preview`.

4. **Add `services/importsvc/edits.go` (`ApplyResultEdits`)**
   - Validate IDs and non-empty `edited_rows`.
   - For each row:
     - ensure row belongs to target race/import batch context,
     - map editable fields from proto `common.v1.ResultEntry` to `models.ResultEntrySetter`,
     - update via repository.
   - Return `ApplyResultEditsResponse{UpdatedRows}`.
   - Transition state to `preprocessed` and write audit action `apply_result_edits`.

5. **Add `services/importsvc/booking_driver.go` (`ComputeDriverBookingEntries`)**
   - Resolve latest import batch for event.
   - Clear previously computed driver booking entries for idempotency.
   - Compute and insert driver-target booking entries from current result rows.
   - Return `ComputeDriverBookingEntriesResponse{CreatedEntries}`.
   - Transition to `driver_entries_computed` and write audit action `compute_driver_booking_entries`.

6. **Add `services/importsvc/booking_team.go` (`ComputeTeamBookingEntries`)**
   - Same pattern as driver computation, but for team-target booking entries.
   - Return `ComputeTeamBookingEntriesResponse{CreatedEntries}`.
   - Transition to `team_entries_computed` and write audit action `compute_team_booking_entries`.

7. **Add `services/importsvc/finalize.go` (`FinalizeEventProcessing`)**
   - Validate event exists and processing can be finalized.
   - Inside transaction:
     - set event `processing_state=finalized`,
     - set `finalized_at=time.Now()`,
     - set latest import batch state to `finalized`,
     - write audit action `finalize_event_processing`.
   - Return `FinalizeEventProcessingResponse{ProcessingState}`.

## Error Handling Requirements

- Always wrap handler failures with Connect errors:
  - `connect.NewError(s.conversion.MapErrorToRPCCode(err), err)`
- Keep logs and trace span status behavior consistent with command/query handlers.
- Use `repoerrors.ErrNotFound` mapping for missing event/race/import-batch/result-entry.
- Map invalid request/state transitions to `connect.CodeInvalidArgument` or `connect.CodeFailedPrecondition` as appropriate.

## Testing Plan

Create `services/importsvc/importsvc_test.go` (or split into focused files by handler).
Use DB-backed tests and existing seed helpers (`simulation -> series -> season -> event`, `track -> layout`, `race`).

Required tests:

1. `UploadResultsFile`
   - success: creates import batch, updates event state to `raw_imported`, creates audit row.
   - failure: invalid event/race relation -> `connect.CodeInvalidArgument`.
   - failure: transaction error -> `connect.CodeInternal`.

2. `GetPreprocessPreview`
   - success: returns rows for latest batch and unresolved mappings.
   - not found: no batch for event/race -> `connect.CodeNotFound`.

3. `ApplyResultEdits`
   - success: updates rows and returns updated count.
   - failure: row not found / wrong race -> `connect.CodeNotFound` or `connect.CodeInvalidArgument`.

4. `ComputeDriverBookingEntries`
   - success: creates driver booking entries and advances state.
   - idempotency: second call replaces/recomputes computed entries deterministically.

5. `ComputeTeamBookingEntries`
   - success: creates team booking entries and advances state.

6. `FinalizeEventProcessing`
   - success: sets event finalized state + timestamp and updates latest batch state.
   - failure: invalid transition (e.g., event still `draft`) -> `connect.CodeFailedPrecondition`.

## Acceptance Criteria

- `services/importsvc` implements all `ImportServiceHandler` methods and compiles without relying on unimplemented stubs.
- Import workflow state transitions are persisted on event/import batch rows.
- Every transition writes an `event_processing_audit` record.
- Handlers return expected response payload fields per proto contract.
- Errors are surfaced via Connect codes through conversion mapping.
- Import service tests cover success and representative failure paths for all six methods.

## Follow-up

- Add dedicated parser adapters per import format (`json`, `csv`) for robust row extraction and unresolved mapping detection.
- Add end-to-end tests covering full import flow from upload to finalize.
