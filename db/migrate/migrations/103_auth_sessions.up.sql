-- Persistent authentication sessions, issue #250.
BEGIN;

CREATE TABLE auth_sessions (
    id serial PRIMARY KEY,
    session_id_hash bytea NOT NULL,
    principal jsonb NOT NULL,
    access_token text NOT NULL,
    refresh_token text NOT NULL,
    token_expires_at timestamp with time zone,
    expires_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system'
);

ALTER TABLE auth_sessions
    ADD CONSTRAINT auth_sessions_session_id_hash_unique
    UNIQUE (session_id_hash);

ALTER TABLE auth_sessions
    ADD CONSTRAINT auth_sessions_session_id_hash_length
    CHECK (octet_length(session_id_hash) = 32);

ALTER TABLE auth_sessions
    ADD CONSTRAINT auth_sessions_principal_object
    CHECK (jsonb_typeof(principal) = 'object');

CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions (expires_at);

COMMIT;
