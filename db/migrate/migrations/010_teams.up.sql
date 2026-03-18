BEGIN;

CREATE TABLE teams (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    season_id integer NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE team_drivers (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    team_id integer NOT NULL,
    driver_id integer NOT NULL,
    joined_at timestamp with time zone NOT NULL DEFAULT now(),
    left_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE teams
    ADD CONSTRAINT teams_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE teams
    ADD CONSTRAINT teams_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE teams
    ADD CONSTRAINT teams_season_id_name_unique
    UNIQUE (season_id, name);

ALTER TABLE team_drivers
    ADD CONSTRAINT team_drivers_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE team_drivers
    ADD CONSTRAINT team_drivers_team_id_fk
    FOREIGN KEY (team_id) REFERENCES teams (id);

ALTER TABLE team_drivers
    ADD CONSTRAINT team_drivers_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE team_drivers
    ADD CONSTRAINT team_drivers_left_at_check
    CHECK (left_at IS NULL OR left_at >= joined_at);

ALTER TABLE team_drivers
    ADD CONSTRAINT team_drivers_team_id_driver_id_joined_at_unique
    UNIQUE (team_id, driver_id, joined_at);

CREATE INDEX idx_teams_season_id ON teams (season_id);
CREATE INDEX idx_teams_is_active ON teams (is_active);
CREATE INDEX idx_team_drivers_team_id ON team_drivers (team_id);
CREATE INDEX idx_team_drivers_driver_id ON team_drivers (driver_id);
CREATE INDEX idx_team_drivers_joined_at ON team_drivers (joined_at);

COMMIT;
