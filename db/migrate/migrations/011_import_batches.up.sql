BEGIN;

CREATE TABLE import_batches (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    event_id integer NOT NULL,
    race_id integer NOT NULL,
    import_format text NOT NULL,
    payload bytea NOT NULL,
    source_filename text,
    processing_state text NOT NULL DEFAULT 'raw_imported',
    metadata_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    processed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_event_id_fk
    FOREIGN KEY (event_id) REFERENCES events (id);

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_race_id_fk
    FOREIGN KEY (race_id) REFERENCES races (id);

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_import_format_check
    CHECK (import_format IN ('json', 'csv'));

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_processing_state_check
    CHECK (processing_state IN ('raw_imported', 'preprocessed', 'driver_entries_computed', 'team_entries_computed', 'finalized', 'failed'));

CREATE INDEX idx_import_batches_event_id ON import_batches (event_id);
CREATE INDEX idx_import_batches_race_id ON import_batches (race_id);
CREATE INDEX idx_import_batches_processing_state ON import_batches (processing_state);

COMMIT;
