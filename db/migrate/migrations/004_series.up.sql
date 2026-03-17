BEGIN;

CREATE TABLE series (
    id serial PRIMARY KEY,
    name text NOT NULL,
    description text,
    simulation_id integer,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE series
    ADD CONSTRAINT series_name_unique UNIQUE (name);

ALTER TABLE series
    ADD CONSTRAINT series_simulation_id_fk
    FOREIGN KEY (simulation_id) REFERENCES racing_sims (id);

CREATE INDEX idx_series_simulation_id ON series (simulation_id);
CREATE INDEX idx_series_is_active ON series (is_active);

COMMIT;
