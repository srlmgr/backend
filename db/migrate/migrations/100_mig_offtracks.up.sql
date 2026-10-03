-- DB migration for offtracks, part of issue #217
BEGIN;

-- Your migration SQL statements go here
ALTER TABLE result_entries
ADD COLUMN offtracks INT NOT NULL DEFAULT 0;

ALTER TABLE booking_entries
    DROP CONSTRAINT booking_entries_source_type_check;

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_source_type_check
    CHECK (source_type IN (
        'finish_pos',
        'fastest_lap',
        'least_incidents',
        'incidents_exceeded',
        'offtracks_exceeded',
        'qualification_pos',
        'top_n_finishers',
        'custom',
        'manual_adjustment',
        'penalty_points',
        'team_contribution'
    ));

COMMIT;
