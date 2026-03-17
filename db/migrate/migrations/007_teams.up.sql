BEGIN;

CREATE TABLE teams (
    id serial PRIMARY KEY,
    season_id integer NOT NULL,
    name text NOT NULL,
    external_id text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE teams
    ADD CONSTRAINT teams_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE teams
    ADD CONSTRAINT teams_season_id_name_unique
    UNIQUE (season_id, name);

CREATE INDEX idx_teams_season_id ON teams (season_id);
CREATE INDEX idx_teams_is_active ON teams (is_active);

CREATE UNIQUE INDEX idx_teams_season_id_external_id_unique
    ON teams (season_id, external_id)
    WHERE external_id IS NOT NULL;

COMMIT;
