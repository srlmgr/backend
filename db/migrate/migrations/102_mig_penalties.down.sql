-- DB migration for renaming penalty_points to steward_penalty, part of issue #238
BEGIN;

-- Your migration SQL statements go here

ALTER TABLE booking_entries
    DROP CONSTRAINT booking_entries_source_type_check;

UPDATE booking_entries
    SET source_type = 'penalty_points'
    WHERE source_type = 'steward_penalty';

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
