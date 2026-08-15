-- Migration: 004_supplier_code_sequence.sql
-- Description: Adds supplier_seq for auto-generating supplier codes (SUP-%06d)
-- when the create payload omits code. Idempotent (IF NOT EXISTS). The sequence
-- is seeded past the largest existing SUP-<digits> code so auto-generated codes
-- never collide with manually-entered codes that matched the auto pattern.
-- Deployment ordering: apply BEFORE deploying the binary whose supplier Create
-- auto-generates codes, otherwise a supplier created with a blank code fails on
-- the missing supplier_seq relation.

CREATE SEQUENCE IF NOT EXISTS supplier_seq START 1 INCREMENT 1;

SELECT setval('supplier_seq',
  GREATEST(COALESCE((SELECT MAX(CAST(substring(code FROM 'SUP-([0-9]+)') AS bigint))
                     FROM suppliers WHERE code ~ '^SUP-[0-9]+$'), 0), 0));

INSERT INTO schema_migrations (filename) VALUES ('004_supplier_code_sequence.sql')
ON CONFLICT (filename) DO NOTHING;