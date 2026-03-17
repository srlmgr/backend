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

ALTER TABLE racing_sims
    ADD CONSTRAINT racing_sims_name_unique UNIQUE (name);

CREATE INDEX idx_racing_sims_is_active ON racing_sims (is_active);

COMMIT;
