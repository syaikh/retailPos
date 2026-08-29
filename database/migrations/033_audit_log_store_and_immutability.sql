-- Migration: 033_audit_log_store_and_immutability.sql
-- Description: Attribute every audit event to a store/branch and make
-- audit_logs append-only at the database level. Applied after 032_*.sql.
-- Deployment ordering: apply BEFORE deploying the binary that populates
-- store_id (audit events become store-scoped and the table rejects
-- UPDATE/DELETE).

BEGIN;

-- Store attribution for every audit event.
ALTER TABLE audit_logs
  ADD COLUMN IF NOT EXISTS store_id integer
  REFERENCES stores(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_store ON audit_logs (store_id);

-- Append-only enforcement: reject any UPDATE or DELETE on audit rows.
-- TRUNCATE (used by the test harness for cleanup) is a separate command
-- and is intentionally NOT blocked.
-- The optional maintenance bypass (session GUC) is added by migration 034.
CREATE OR REPLACE FUNCTION reject_audit_log_modification()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only: modifications are not permitted';
END;
$$;

DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;

CREATE TRIGGER trg_audit_logs_immutable
  BEFORE UPDATE OR DELETE ON audit_logs
  FOR EACH STATEMENT
  EXECUTE FUNCTION reject_audit_log_modification();

COMMIT;
