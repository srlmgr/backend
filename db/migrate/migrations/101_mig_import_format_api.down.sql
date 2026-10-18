-- DB migration to support new import format API
BEGIN;

-- Your migration SQL statements go here
ALTER TABLE import_batches
DROP CONSTRAINT import_batches_import_format_check;

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_import_format_check CHECK ((import_format = ANY (ARRAY['json'::text, 'csv'::text, 'xml'::text])));
COMMIT;
