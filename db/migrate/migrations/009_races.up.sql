BEGIN;

CREATE TABLE races (
    id serial PRIMARY KEY,

    event_id integer NOT NULL,
    name text NOT NULL,
    session_type text NOT NULL,
    sequence_no integer NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE races
    ADD CONSTRAINT races_event_id_fk
    FOREIGN KEY (event_id) REFERENCES events (id);


ALTER TABLE races
    ADD CONSTRAINT races_event_id_sequence_no_unique
    UNIQUE (event_id, sequence_no);

ALTER TABLE races
    ADD CONSTRAINT races_event_id_name_unique
    UNIQUE (event_id, name);

CREATE INDEX idx_races_event_id ON races (event_id);
CREATE INDEX idx_races_session_type ON races (session_type);

COMMIT;
