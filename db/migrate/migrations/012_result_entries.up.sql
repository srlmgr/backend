BEGIN;

CREATE TABLE result_entries (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    race_grid_id integer NOT NULL,
    driver_id integer,
    team_id integer,
    car_model_id integer,
    car_class_id integer,
    raw_car_name text,
    raw_driver_name text,
    raw_team_name text,
    car_number text,
    is_guest_driver boolean NOT NULL DEFAULT false,
    start_position integer,
    finish_position integer NOT NULL,
    laps_completed integer NOT NULL DEFAULT 0,
    quali_lap_time_ms integer,
    fastest_lap_time_ms integer,
    total_time_ms integer,
    incidents integer,
    state text NOT NULL DEFAULT 'normal',
    admin_notes text,
    locked_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_frontend_id_unique UNIQUE (frontend_id);



ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_race_grid_id_fk
    FOREIGN KEY (race_grid_id) REFERENCES race_grids (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_car_model_id_fk
    FOREIGN KEY (car_model_id) REFERENCES car_models (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_car_class_id_fk
    FOREIGN KEY (car_class_id) REFERENCES car_classes (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_team_id_fk
    FOREIGN KEY (team_id) REFERENCES teams (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_laps_completed_check
    CHECK (laps_completed >= 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_fastest_lap_time_ms_check
    CHECK (fastest_lap_time_ms IS NULL OR fastest_lap_time_ms > 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_quali_lap_time_ms_check
    CHECK (quali_lap_time_ms IS NULL OR quali_lap_time_ms > 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_total_time_ms_check
    CHECK (total_time_ms IS NULL OR total_time_ms > 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_incidents_check
    CHECK (incidents IS NULL OR incidents >= 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_state_check
    CHECK (state IN ('mapping_error','normal', 'dq'));


CREATE UNIQUE INDEX idx_result_entries_race_grid_id_driver_id_unique
    ON result_entries (race_grid_id, driver_id)
    WHERE driver_id IS NOT NULL;
CREATE UNIQUE INDEX idx_result_entries_race_grid_id_team_id_unique
    ON result_entries (race_grid_id, team_id)
    WHERE team_id IS NOT NULL;


CREATE INDEX idx_result_entries_race_grid_id ON result_entries (race_grid_id);
CREATE INDEX idx_result_entries_driver_id ON result_entries (driver_id);
CREATE INDEX idx_result_entries_car_model_id ON result_entries (car_model_id);
CREATE INDEX idx_result_entries_car_class_id ON result_entries (car_class_id);
CREATE INDEX idx_result_entries_team_id ON result_entries (team_id);
CREATE INDEX idx_result_entries_state ON result_entries (state);

COMMIT;
