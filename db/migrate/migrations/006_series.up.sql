BEGIN;

CREATE TABLE series (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    simulation_id integer NOT NULL,
    name text NOT NULL,
    description text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE series
    ADD CONSTRAINT series_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE series
    ADD CONSTRAINT series_simulation_id_fk
    FOREIGN KEY (simulation_id) REFERENCES racing_sims (id);

ALTER TABLE series
    ADD CONSTRAINT series_simulation_id_name_unique
    UNIQUE (simulation_id, name);

CREATE INDEX idx_series_simulation_id ON series (simulation_id);
CREATE INDEX idx_series_is_active ON series (is_active);

COMMIT;
