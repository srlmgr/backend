BEGIN;

CREATE TABLE result_entries (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    import_batch_id integer NOT NULL,
    race_id integer NOT NULL,
    driver_id integer,
    driver_name text NOT NULL,
    car_model_id integer,
    car_name text,
    finishing_position integer NOT NULL,
    completed_laps integer NOT NULL DEFAULT 0,
    fastest_lap_time_ms integer,
    incidents integer,
    state text NOT NULL DEFAULT 'normal',
    source_row_number integer,
    raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
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
    ADD CONSTRAINT result_entries_import_batch_id_fk
    FOREIGN KEY (import_batch_id) REFERENCES import_batches (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_race_id_fk
    FOREIGN KEY (race_id) REFERENCES races (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_car_model_id_fk
    FOREIGN KEY (car_model_id) REFERENCES car_models (id);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_completed_laps_check
    CHECK (completed_laps >= 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_fastest_lap_time_ms_check
    CHECK (fastest_lap_time_ms IS NULL OR fastest_lap_time_ms > 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_incidents_check
    CHECK (incidents IS NULL OR incidents >= 0);

ALTER TABLE result_entries
    ADD CONSTRAINT result_entries_state_check
    CHECK (state IN ('normal', 'dq'));

CREATE UNIQUE INDEX idx_result_entries_race_id_driver_id_unique
    ON result_entries (race_id, driver_id)
    WHERE driver_id IS NOT NULL;

CREATE INDEX idx_result_entries_import_batch_id ON result_entries (import_batch_id);
CREATE INDEX idx_result_entries_race_id ON result_entries (race_id);
CREATE INDEX idx_result_entries_driver_id ON result_entries (driver_id);
CREATE INDEX idx_result_entries_car_model_id ON result_entries (car_model_id);
CREATE INDEX idx_result_entries_state ON result_entries (state);

COMMIT;
