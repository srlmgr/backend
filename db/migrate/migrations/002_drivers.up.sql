BEGIN;

CREATE TABLE drivers (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    external_id text NOT NULL,
    name text NOT NULL,
    simulation_ids jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_active boolean NOT NULL DEFAULT true,
    joined_at timestamp with time zone NOT NULL DEFAULT now(),
    last_imported_from text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

COMMENT ON COLUMN drivers.frontend_id IS 'id used to reference in frontend';
COMMENT ON COLUMN drivers.simulation_ids IS 'map by simID to array of sim specific driver IDs';

ALTER TABLE drivers
    ADD CONSTRAINT drivers_external_id_unique UNIQUE (external_id);

CREATE INDEX idx_drivers_name ON drivers (name);
CREATE INDEX idx_drivers_is_active ON drivers (is_active);

COMMIT;
