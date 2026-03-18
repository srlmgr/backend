BEGIN;

CREATE TABLE racing_sims (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    name text NOT NULL,
    supported_import_formats text[] NOT NULL DEFAULT ARRAY['json', 'csv'],
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE racing_sims
    ADD CONSTRAINT racing_sims_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE racing_sims
    ADD CONSTRAINT racing_sims_name_unique UNIQUE (name);

ALTER TABLE racing_sims
    ADD CONSTRAINT racing_sims_supported_import_formats_check
    CHECK (supported_import_formats <@ ARRAY['json', 'csv']::text[]);

CREATE INDEX idx_racing_sims_is_active ON racing_sims (is_active);

COMMIT;
