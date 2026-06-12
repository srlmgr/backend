BEGIN;

CREATE TABLE season_drivers (
    id serial PRIMARY KEY,
    driver_id integer NOT NULL,
    season_id integer NOT NULL,
    car_model_id integer NOT NULL,
	car_number text not null,
	is_guest_starter boolean NOT NULL DEFAULT false,
    joined_at timestamp with time zone NOT NULL DEFAULT now(),
    left_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE season_drivers
    ADD CONSTRAINT season_drivers_driver_id_fk
    FOREIGN KEY (driver_id) REFERENCES drivers (id);

ALTER TABLE season_drivers
    ADD CONSTRAINT season_drivers_season_id_fk
    FOREIGN KEY (season_id) REFERENCES seasons (id);

ALTER TABLE season_drivers
    ADD CONSTRAINT season_drivers_car_model_id_fk
    FOREIGN KEY (car_model_id) REFERENCES car_models (id);

CREATE INDEX idx_season_drivers_driver_id ON season_drivers (driver_id);
CREATE INDEX idx_season_drivers_season_id ON season_drivers (season_id);
CREATE INDEX idx_season_drivers_car_model_id ON season_drivers (car_model_id);

COMMIT;
