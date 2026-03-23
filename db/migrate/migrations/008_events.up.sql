BEGIN;

CREATE TABLE events (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    season_id integer NOT NULL,
    track_layout_id integer NOT NULL,
    name text NOT NULL,
    event_date timestamp with time zone NOT NULL,
    status text NOT NULL DEFAULT 'scheduled',
    processing_state text NOT NULL DEFAULT 'draft',
    finalized_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE events
    ADD CONSTRAINT events_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE events
    ADD CONSTRAINT events_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE events
    ADD CONSTRAINT events_track_layout_id_fk
    FOREIGN KEY (track_layout_id) REFERENCES track_layouts (id);

ALTER TABLE events
    ADD CONSTRAINT events_season_id_name_unique
    UNIQUE (season_id, name);

ALTER TABLE events
    ADD CONSTRAINT events_status_check
    CHECK (status IN ('scheduled', 'completed', 'cancelled'));

ALTER TABLE events
    ADD CONSTRAINT events_processing_state_check
    CHECK (processing_state IN ('draft', 'raw_imported', 'mapping_error', 'pre_processed','driver_entries_computed', 'team_entries_computed', 'finalized'));

CREATE INDEX idx_events_season_id ON events (season_id);
CREATE INDEX idx_events_track_layout_id ON events (track_layout_id);
CREATE INDEX idx_events_event_date ON events (event_date);
CREATE INDEX idx_events_processing_state ON events (processing_state);

COMMIT;
