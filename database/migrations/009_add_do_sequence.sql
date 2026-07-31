BEGIN;

-- Sequence for Delivery Order numbering
CREATE SEQUENCE IF NOT EXISTS do_seq START 1;

INSERT INTO schema_migrations (filename) VALUES ('009_add_do_sequence.sql');

COMMIT;
