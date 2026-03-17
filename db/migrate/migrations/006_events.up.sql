BEGIN;

CREATE TABLE events (
    id serial PRIMARY KEY,
    season_id integer NOT NULL,
    name text NOT NULL,
    round_number integer NOT NULL,
    venue text,
    starts_at timestamp with time zone,
    ends_at timestamp with time zone,
    status text NOT NULL DEFAULT 'scheduled',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE events
    ADD CONSTRAINT events_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE events
    ADD CONSTRAINT events_season_id_round_number_unique
    UNIQUE (season_id, round_number);

ALTER TABLE events
    ADD CONSTRAINT events_status_check
    CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled', 'postponed'));

ALTER TABLE events
    ADD CONSTRAINT events_date_order_check
    CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at >= starts_at);

CREATE INDEX idx_events_season_id ON events (season_id);
CREATE INDEX idx_events_status ON events (status);
CREATE INDEX idx_events_starts_at ON events (starts_at);

COMMIT;
