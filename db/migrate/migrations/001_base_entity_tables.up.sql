BEGIN;

CREATE TABLE racing_sims (
    id serial PRIMARY KEY,
    name text NOT NULL,
    supported_import_formats text[] NOT NULL DEFAULT '{}',
    data_mapping jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE drivers (
    id serial PRIMARY KEY,
    external_id text NOT NULL,
    name text NOT NULL,
    simulation_ids integer[] NOT NULL DEFAULT '{}',
    aliases text[] NOT NULL DEFAULT '{}',
    is_active boolean NOT NULL DEFAULT true,
    joined_at timestamp with time zone NOT NULL DEFAULT now(),
    last_imported_from text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE point_systems (
    id serial PRIMARY KEY,
    name text NOT NULL,
    description text,
    position_points jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE racing_sims
    ADD CONSTRAINT racing_sims_name_unique UNIQUE (name);

ALTER TABLE drivers
    ADD CONSTRAINT drivers_external_id_unique UNIQUE (external_id);

ALTER TABLE point_systems
    ADD CONSTRAINT point_systems_name_unique UNIQUE (name);

CREATE INDEX idx_racing_sims_is_active ON racing_sims (is_active);
CREATE INDEX idx_drivers_name ON drivers (name);
CREATE INDEX idx_drivers_is_active ON drivers (is_active);
CREATE INDEX idx_point_systems_is_active ON point_systems (is_active);

COMMIT;
