BEGIN;

CREATE TABLE racing_sims (
    id serial PRIMARY KEY,

    name text NOT NULL,
    supported_import_formats jsonb NOT NULL DEFAULT '[]',
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);


ALTER TABLE racing_sims
    ADD CONSTRAINT racing_sims_name_unique UNIQUE (name);


COMMIT;
