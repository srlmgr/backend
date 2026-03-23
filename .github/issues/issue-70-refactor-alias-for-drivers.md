# Issue 70: Refactor alias for drivers

## Summary

Align driver simulation alias naming and schema with the existing alias patterns used for tracks and cars.

## Required Changes

1. Database migration updates:

- Rename table `driver_simulation_ids` -> `simulation_driver_aliases` in:
    - `db/migrate/migrations/003_drivers.up.sql`
    - `db/migrate/migrations/003_drivers.down.sql`
- Remove `event_id` from `import_batches` schema in:
    - `db/migrate/migrations/011_import_batches.up.sql`
- Remove `import_batch_id` from `result_entries` schema in:
    - `db/migrate/migrations/012_result_entries.up.sql`

2. Regenerate Bob artifacts:

- Run `make bob` to regenerate generated model/dbinfo/dberrors/factory files.

3. Repository refactor:

- Rename repository interface/member naming from `DriverSimulationIDsRepository` to `SimulationDriverAliasesRepository`.
- Update repository aggregate method from `DriverSimulationIDs()` to `SimulationDriverAliases()`.
- Update all call sites in services and testsupport repositories/tests.

4. Protobuf dependency update:

- Update API generated deps to commit `abbe45bbd4ff4c318b37c693dce20677` using:
    - `make update-bufbuild id=abbe45bbd4ff4c318b37c693dce20677`
- Fix compile errors caused by API type/field/method renames.

5. Validation:

- Run targeted tests for changed packages.
- Run full test suite (`make test`) and ensure green.

## Implementation Notes

- Keep migration SQL style consistent with existing files.
- Prefer minimal behavioral changes outside the schema/naming alignment requested.
- Let generated Bob files reflect schema source-of-truth, avoid manual edits in generated files where possible.
- For protobuf adjustments, prefer adapting to renamed RPC structures rather than preserving deprecated names.
