BEGIN;

CREATE TABLE car_manufacturers (
    id serial PRIMARY KEY,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE car_models (
    id serial PRIMARY KEY,
    manufacturer_id integer NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE car_model_variants (
    id serial PRIMARY KEY,
    car_model_id integer NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE car_classes (
    id serial PRIMARY KEY,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

CREATE TABLE car_classes_to_car_models (
	-- ugly workaround. don't need a pk here, but bob runs into errors creating dberrors text without it
	id serial PRIMARY KEY,
    car_class_id integer NOT NULL,
    car_model_variant_id integer
);

CREATE TABLE simulation_car_aliases (
    id serial PRIMARY KEY,
    car_model_variant_id integer NOT NULL,
    simulation_id integer NOT NULL,
    external_name text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE car_manufacturers
    ADD CONSTRAINT car_manufacturers_name_unique UNIQUE (name);

ALTER TABLE car_models
    ADD CONSTRAINT car_models_manufacturer_id_fk
    FOREIGN KEY (manufacturer_id) REFERENCES car_manufacturers (id);

ALTER TABLE car_models
    ADD CONSTRAINT car_models_manufacturer_id_name_unique
    UNIQUE (manufacturer_id, name);

ALTER TABLE car_model_variants
    ADD CONSTRAINT car_model_variants_car_model_id_fk
    FOREIGN KEY (car_model_id) REFERENCES car_models (id);

ALTER TABLE car_model_variants
    ADD CONSTRAINT car_model_variants_car_model_id_name_unique
    UNIQUE (car_model_id, name);

ALTER TABLE car_classes
    ADD CONSTRAINT car_classes_name_unique UNIQUE (name);

ALTER TABLE car_classes_to_car_models
    ADD CONSTRAINT car_classes_to_car_models_car_class_id_fk
    FOREIGN KEY (car_class_id) REFERENCES car_classes (id);


ALTER TABLE car_classes_to_car_models
    ADD CONSTRAINT car_classes_to_car_models_car_model_variant_id_fk
    FOREIGN KEY (car_model_variant_id) REFERENCES car_model_variants (id);

ALTER TABLE car_classes_to_car_models
    ADD CONSTRAINT car_classes_to_car_models_car_class_id_car_model_variant_id_unique
    UNIQUE (car_class_id, car_model_variant_id);

ALTER TABLE simulation_car_aliases
    ADD CONSTRAINT simulation_car_aliases_car_model_variant_id_fk
    FOREIGN KEY (car_model_variant_id) REFERENCES car_model_variants (id);

ALTER TABLE simulation_car_aliases
    ADD CONSTRAINT simulation_car_aliases_simulation_id_fk
    FOREIGN KEY (simulation_id) REFERENCES racing_sims (id);


ALTER TABLE simulation_car_aliases
    ADD CONSTRAINT simulation_car_aliases_simulation_id_external_name_unique
    UNIQUE (simulation_id, external_name);

CREATE INDEX idx_car_manufacturers_is_active ON car_manufacturers (is_active);
CREATE INDEX idx_car_models_manufacturer_id ON car_models (manufacturer_id);
CREATE INDEX idx_car_models_is_active ON car_models (is_active);
CREATE INDEX idx_car_model_variants_car_model_id ON car_model_variants (car_model_id);
CREATE INDEX idx_car_model_variants_is_active ON car_model_variants (is_active);
CREATE INDEX idx_car_classes_is_active ON car_classes (is_active);
CREATE INDEX idx_car_classes_to_car_models_car_class_id ON car_classes_to_car_models (car_class_id);
CREATE INDEX idx_car_classes_to_car_models_car_model_variant_id ON car_classes_to_car_models (car_model_variant_id);

CREATE INDEX idx_simulation_car_aliases_car_model_variant_id ON simulation_car_aliases (car_model_variant_id);
CREATE INDEX idx_simulation_car_aliases_simulation_id ON simulation_car_aliases (simulation_id);

COMMIT;
