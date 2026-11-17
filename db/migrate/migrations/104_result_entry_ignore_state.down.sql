-- Revert ignored result entries, issue #261.
BEGIN;

ALTER TABLE result_entries
    DROP CONSTRAINT result_entries_state_check;

UPDATE result_entries
    SET state = 'normal'
    WHERE state = 'ignore';

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_state_check
    CHECK (state IN ('mapping_error', 'normal', 'dq'));

COMMIT;
