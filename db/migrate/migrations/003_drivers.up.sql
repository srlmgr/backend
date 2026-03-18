BEGIN;

CREATE TABLE drivers (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    external_id text NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    last_imported_from text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE driver_simulation_ids (
    id serial PRIMARY KEY,
    driver_id integer NOT NULL,
    simulation_id integer NOT NULL,
    simulation_driver_id text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE drivers
    ADD CONSTRAINT drivers_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE drivers
    ADD CONSTRAINT drivers_external_id_unique UNIQUE (external_id);


ALTER TABLE driver_simulation_ids
    ADD CONSTRAINT driver_simulation_ids_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE driver_simulation_ids
    ADD CONSTRAINT driver_simulation_ids_simulation_id_fk
    FOREIGN KEY (simulation_id) REFERENCES racing_sims (id);

ALTER TABLE driver_simulation_ids
    ADD CONSTRAINT driver_simulation_ids_driver_id_simulation_id_unique
    UNIQUE (driver_id, simulation_id);

ALTER TABLE driver_simulation_ids
    ADD CONSTRAINT driver_simulation_ids_simulation_id_driver_key_unique
    UNIQUE (simulation_id, simulation_driver_id);

CREATE INDEX idx_drivers_name ON drivers (name);
CREATE INDEX idx_drivers_is_active ON drivers (is_active);
CREATE INDEX idx_driver_simulation_ids_driver_id ON driver_simulation_ids (driver_id);
CREATE INDEX idx_driver_simulation_ids_simulation_id ON driver_simulation_ids (simulation_id);

COMMIT;
