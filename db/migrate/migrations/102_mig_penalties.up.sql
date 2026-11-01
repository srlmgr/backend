-- DB migration for renaming penalty_points to steward_penalty_points, part of issue #238
BEGIN;

-- Your migration SQL statements go here

ALTER TABLE booking_entries
    DROP CONSTRAINT booking_entries_source_type_check;

UPDATE booking_entries
    SET source_type = 'steward_penalty'
    WHERE source_type = 'penalty_points';

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
        'manual_adjustment',
        'steward_penalty',
        'team_contribution'
    ));


COMMIT;
