BEGIN;

CREATE TABLE point_systems (
    id serial PRIMARY KEY,
    name text NOT NULL,
    description text,
    is_active boolean NOT NULL DEFAULT true,
	guest_points boolean NOT NULL DEFAULT false,
	race_distance_pct numeric(5,4) NOT NULL DEFAULT 0 CHECK (race_distance_pct >= 0 AND race_distance_pct <= 1),
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE point_rules (
    id serial PRIMARY KEY,
    point_system_id integer NOT NULL,
	race_no integer NOT NULL DEFAULT 0,
	point_policy text NOT NULL,
    metadata_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);


ALTER TABLE point_systems
    ADD CONSTRAINT point_systems_name_unique UNIQUE (name);


ALTER TABLE point_rules
    ADD CONSTRAINT point_rules_point_system_id_fk
    FOREIGN KEY (point_system_id) REFERENCES point_systems (id);


CREATE INDEX idx_point_systems_is_active ON point_systems (is_active);
CREATE INDEX idx_point_rules_point_system_id ON point_rules (point_system_id);


COMMIT;
