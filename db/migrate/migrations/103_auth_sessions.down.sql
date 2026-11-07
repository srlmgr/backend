-- Revert persistent authentication sessions, issue #250.
BEGIN;

DROP TABLE auth_sessions;

COMMIT;
