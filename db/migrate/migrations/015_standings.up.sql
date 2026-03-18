BEGIN;

CREATE TABLE season_driver_standings (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    season_id integer NOT NULL,
    driver_id integer NOT NULL,
    position integer NOT NULL,
    total_points integer NOT NULL,
    dropped_event_ids integer[] NOT NULL DEFAULT '{}',
    last_rebuilt_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE season_team_standings (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    season_id integer NOT NULL,
    team_id integer NOT NULL,
    position integer NOT NULL,
    total_points integer NOT NULL,
    dropped_event_ids integer[] NOT NULL DEFAULT '{}',
    last_rebuilt_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE event_driver_standings (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    event_id integer NOT NULL,
    season_id integer NOT NULL,
    driver_id integer NOT NULL,
    position integer NOT NULL,
    total_points integer NOT NULL,
    dropped_event_ids integer[] NOT NULL DEFAULT '{}',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE event_team_standings (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    event_id integer NOT NULL,
    season_id integer NOT NULL,
    team_id integer NOT NULL,
    position integer NOT NULL,
    total_points integer NOT NULL,
    dropped_event_ids integer[] NOT NULL DEFAULT '{}',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE season_driver_standings
    ADD CONSTRAINT season_driver_standings_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE season_driver_standings
    ADD CONSTRAINT season_driver_standings_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE season_driver_standings
    ADD CONSTRAINT season_driver_standings_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE season_driver_standings
    ADD CONSTRAINT season_driver_standings_season_id_driver_id_unique
    UNIQUE (season_id, driver_id);

ALTER TABLE season_team_standings
    ADD CONSTRAINT season_team_standings_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE season_team_standings
    ADD CONSTRAINT season_team_standings_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE season_team_standings
    ADD CONSTRAINT season_team_standings_team_id_fk
    FOREIGN KEY (team_id) REFERENCES teams (id);

ALTER TABLE season_team_standings
    ADD CONSTRAINT season_team_standings_season_id_team_id_unique
    UNIQUE (season_id, team_id);

ALTER TABLE event_driver_standings
    ADD CONSTRAINT event_driver_standings_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE event_driver_standings
    ADD CONSTRAINT event_driver_standings_event_id_fk
    FOREIGN KEY (event_id) REFERENCES events (id);

ALTER TABLE event_driver_standings
    ADD CONSTRAINT event_driver_standings_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE event_driver_standings
    ADD CONSTRAINT event_driver_standings_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE event_driver_standings
    ADD CONSTRAINT event_driver_standings_event_id_driver_id_unique
    UNIQUE (event_id, driver_id);

ALTER TABLE event_team_standings
    ADD CONSTRAINT event_team_standings_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE event_team_standings
    ADD CONSTRAINT event_team_standings_event_id_fk
    FOREIGN KEY (event_id) REFERENCES events (id);

ALTER TABLE event_team_standings
    ADD CONSTRAINT event_team_standings_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE event_team_standings
    ADD CONSTRAINT event_team_standings_team_id_fk
    FOREIGN KEY (team_id) REFERENCES teams (id);


ALTER TABLE event_team_standings
    ADD CONSTRAINT event_team_standings_event_id_team_id_unique
    UNIQUE (event_id, team_id);

CREATE INDEX idx_season_driver_standings_season_id_position ON season_driver_standings (season_id, position);
CREATE INDEX idx_season_team_standings_season_id_position ON season_team_standings (season_id, position);
CREATE INDEX idx_event_driver_standings_event_id_position ON event_driver_standings (event_id, position);
CREATE INDEX idx_event_team_standings_event_id_position ON event_team_standings (event_id, position);

COMMIT;
