BEGIN;

CREATE TABLE event_processing_audit (
    id serial PRIMARY KEY,
    frontend_id uuid NOT NULL DEFAULT uuid_generate_v4(),
    event_id integer NOT NULL,
    import_batch_id integer,
    from_state text,
    to_state text NOT NULL,
    action text NOT NULL,
    payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE event_processing_audit
    ADD CONSTRAINT event_processing_audit_frontend_id_unique UNIQUE (frontend_id);

ALTER TABLE event_processing_audit
    ADD CONSTRAINT event_processing_audit_event_id_fk
    FOREIGN KEY (event_id) REFERENCES events (id);

ALTER TABLE event_processing_audit
    ADD CONSTRAINT event_processing_audit_import_batch_id_fk
    FOREIGN KEY (import_batch_id) REFERENCES import_batches (id);

CREATE INDEX idx_event_processing_audit_event_id ON event_processing_audit (event_id);
CREATE INDEX idx_event_processing_audit_import_batch_id ON event_processing_audit (import_batch_id);
CREATE INDEX idx_event_processing_audit_to_state ON event_processing_audit (to_state);

COMMIT;
