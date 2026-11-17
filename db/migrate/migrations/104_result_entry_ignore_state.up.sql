-- Support ignored result entries, issue #261.
BEGIN;

ALTER TABLE result_entries
    DROP CONSTRAINT result_entries_state_check;

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_state_check
    CHECK (state IN ('mapping_error', 'normal', 'dq', 'ignore'));

COMMIT;
