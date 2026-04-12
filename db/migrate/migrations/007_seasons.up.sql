BEGIN;

CREATE TABLE seasons (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    series_id integer NOT NULL,
    point_system_id integer NOT NULL,
    name text NOT NULL,
    starts_at timestamp with time zone,
    ends_at timestamp with time zone,
    skip_events integer NOT NULL DEFAULT 0,
    has_teams boolean NOT NULL DEFAULT false,
    team_points_top_n integer,
	is_team_based boolean NOT NULL DEFAULT false,
	is_multiclass boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'planned',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);
COMMENT ON COLUMN seasons.team_points_top_n IS 'Top N team members considered for points';
COMMENT ON COLUMN seasons.has_teams IS 'Indicates team standings are supported';
COMMENT ON COLUMN seasons.is_team_based IS 'Primary entity for processing is a team';
COMMENT ON COLUMN seasons.is_multiclass IS 'Indicates if multiple classes are supported';

ALTER TABLE seasons
    ADD CONSTRAINT seasons_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_series_id_fk
    FOREIGN KEY (series_id) REFERENCES series (id);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_point_system_id_fk
    FOREIGN KEY (point_system_id) REFERENCES point_systems (id);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_series_id_name_unique
    UNIQUE (series_id, name);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_status_check
    CHECK (status IN ('planned', 'active', 'completed', 'cancelled'));

ALTER TABLE seasons
    ADD CONSTRAINT seasons_skip_events_check
    CHECK (skip_events >= 0);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_team_points_top_n_check
    CHECK (team_points_top_n IS NULL OR team_points_top_n > 0);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_date_order_check
    CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at >= starts_at);

CREATE INDEX idx_seasons_series_id ON seasons (series_id);
CREATE INDEX idx_seasons_point_system_id ON seasons (point_system_id);
CREATE INDEX idx_seasons_status ON seasons (status);
CREATE INDEX idx_seasons_starts_at ON seasons (starts_at);


CREATE TABLE season_car_classes (
	-- ugly workaround. don't need a pk here, but bob runs into errors creating dberrors text without it
	id serial PRIMARY KEY,
    season_id integer NOT NULL,
    car_class_id integer NOT NULL
);
ALTER TABLE season_car_classes
    ADD CONSTRAINT season_car_classes_car_class_id_fk
    FOREIGN KEY (car_class_id) REFERENCES car_classes (id);

ALTER TABLE season_car_classes
    ADD CONSTRAINT season_car_classes_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE season_car_classes
    ADD CONSTRAINT season_car_classes_car_class_id_season_id_unique
    UNIQUE (car_class_id, season_id);

COMMIT;
