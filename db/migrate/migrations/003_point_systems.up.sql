BEGIN;

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

ALTER TABLE point_systems
    ADD CONSTRAINT point_systems_name_unique UNIQUE (name);

CREATE INDEX idx_point_systems_is_active ON point_systems (is_active);

COMMIT;
