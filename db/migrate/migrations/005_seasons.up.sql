BEGIN;

CREATE TABLE seasons (
    id serial PRIMARY KEY,
    series_id integer NOT NULL,
    point_system_id integer NOT NULL,
    name text NOT NULL,
    short_name text NOT NULL,
    starts_at timestamp with time zone,
    ends_at timestamp with time zone,
    status text NOT NULL DEFAULT 'draft',
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

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
    ADD CONSTRAINT seasons_series_id_short_name_unique
    UNIQUE (series_id, short_name);

ALTER TABLE seasons
    ADD CONSTRAINT seasons_status_check
    CHECK (status IN ('draft', 'scheduled', 'active', 'completed', 'cancelled', 'archived'));

ALTER TABLE seasons
    ADD CONSTRAINT seasons_date_order_check
    CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at >= starts_at);

CREATE INDEX idx_seasons_series_id ON seasons (series_id);
CREATE INDEX idx_seasons_point_system_id ON seasons (point_system_id);
CREATE INDEX idx_seasons_status ON seasons (status);
CREATE INDEX idx_seasons_is_active ON seasons (is_active);
CREATE INDEX idx_seasons_starts_at ON seasons (starts_at);

COMMIT;
