BEGIN;

CREATE TABLE tracks (
    id serial PRIMARY KEY,
    name text NOT NULL,
    country text,
    latitude numeric(9, 6),
    longitude numeric(9, 6),
    website_url text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE track_layouts (
    id serial PRIMARY KEY,
    track_id integer NOT NULL,
    name text NOT NULL,
    length_meters integer,
    layout_image_url text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE simulation_track_layout_aliases (
    id serial PRIMARY KEY,
    track_layout_id integer NOT NULL,
    simulation_id integer NOT NULL,
    external_name text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);


ALTER TABLE tracks
    ADD CONSTRAINT tracks_name_unique UNIQUE (name);

ALTER TABLE track_layouts
    ADD CONSTRAINT track_layouts_track_id_fk
    FOREIGN KEY (track_id) REFERENCES tracks (id);

ALTER TABLE track_layouts
    ADD CONSTRAINT track_layouts_track_id_name_unique
    UNIQUE (track_id, name);

ALTER TABLE track_layouts
    ADD CONSTRAINT track_layouts_length_meters_check
    CHECK (length_meters IS NULL OR length_meters > 0);


ALTER TABLE simulation_track_layout_aliases
    ADD CONSTRAINT simulation_track_layout_aliases_track_layout_id_fk
    FOREIGN KEY (track_layout_id) REFERENCES track_layouts (id);

ALTER TABLE simulation_track_layout_aliases
    ADD CONSTRAINT simulation_track_layout_aliases_simulation_id_fk
    FOREIGN KEY (simulation_id) REFERENCES racing_sims (id);

ALTER TABLE simulation_track_layout_aliases
    ADD CONSTRAINT simulation_track_layout_aliases_track_layout_id_simulation_id_unique
    UNIQUE (track_layout_id, simulation_id);

ALTER TABLE simulation_track_layout_aliases
    ADD CONSTRAINT simulation_track_layout_aliases_simulation_id_external_name_unique
    UNIQUE (simulation_id, external_name);

CREATE INDEX idx_tracks_is_active ON tracks (is_active);
CREATE INDEX idx_track_layouts_track_id ON track_layouts (track_id);
CREATE INDEX idx_track_layouts_is_active ON track_layouts (is_active);
CREATE INDEX idx_simulation_track_layout_aliases_track_layout_id ON simulation_track_layout_aliases (track_layout_id);
CREATE INDEX idx_simulation_track_layout_aliases_simulation_id ON simulation_track_layout_aliases (simulation_id);

COMMIT;
