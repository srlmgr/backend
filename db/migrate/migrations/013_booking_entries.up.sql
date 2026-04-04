BEGIN;

CREATE TABLE booking_entries (
    id serial PRIMARY KEY,
    event_id integer NOT NULL,
	race_id integer not null,
	race_grid_id integer not null,

    target_type text NOT NULL,
    driver_id integer,
    team_id integer,
    source_type text NOT NULL,
    points integer NOT NULL,
    description text NOT NULL DEFAULT '',
    is_manual boolean NOT NULL DEFAULT false,
    locked_at timestamp with time zone,
    metadata_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);


ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_event_id_fk
    FOREIGN KEY (event_id) REFERENCES events (id);

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_race_id_fk
    FOREIGN KEY (race_id) REFERENCES races (id);

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_race_grid_id_fk
    FOREIGN KEY (race_grid_id) REFERENCES race_grids (id);

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_team_id_fk
    FOREIGN KEY (team_id) REFERENCES teams (id);

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_target_type_check
    CHECK (target_type IN ('driver', 'team'));

ALTER TABLE booking_entries
    ADD CONSTRAINT booking_entries_source_type_check
    CHECK (source_type IN ('finish_pos', 'fastest_lap', 'least_incidents','incidents_exceeded','qualification_pos','top_n_finishers','custom','manual_adjustment'));


CREATE INDEX idx_booking_entries_event_id ON booking_entries (event_id);
CREATE INDEX idx_booking_entries_race_id ON booking_entries (race_id);
CREATE INDEX idx_booking_entries_race_grid_id ON booking_entries (race_grid_id);
CREATE INDEX idx_booking_entries_target_type ON booking_entries (target_type);
CREATE INDEX idx_booking_entries_driver_id ON booking_entries (driver_id);
CREATE INDEX idx_booking_entries_team_id ON booking_entries (team_id);
CREATE INDEX idx_booking_entries_source_type ON booking_entries (source_type);

COMMIT;
