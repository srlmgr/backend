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
CREATE TABLE race_grids (
    id serial PRIMARY KEY,
    race_id integer NOT NULL,
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

ALTER TABLE race_grids
    ADD CONSTRAINT race_grids_race_id_fk
    FOREIGN KEY (race_id) REFERENCES races (id);


ALTER TABLE race_grids
    ADD CONSTRAINT race_grids_race_id_sequence_no_unique
    UNIQUE (race_id, sequence_no);

ALTER TABLE race_grids
    ADD CONSTRAINT race_grids_race_id_name_unique
    UNIQUE (race_id, name);

CREATE INDEX idx_race_grids_race_id ON race_grids (race_id);
CREATE INDEX idx_race_grids_session_type ON race_grids (session_type);

COMMIT;
