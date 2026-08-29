-- Replace the append-only trigger function with a GUC-aware version so that
-- authorized maintenance tooling (cmd/backfill-audit-description) can opt out
-- of the restriction via `SET app.allow_audit_mod = 'on'`.
--
-- Normal application traffic never sets this GUC, so audit_logs remains
-- append-only at runtime.
CREATE OR REPLACE FUNCTION reject_audit_log_modification()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF current_setting('app.allow_audit_mod', true) = 'on' THEN
    RETURN NULL;
  END IF;
  RAISE EXCEPTION 'audit_logs is append-only: modifications are not permitted';
END;
$$;

DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;

CREATE TRIGGER trg_audit_logs_immutable
  BEFORE UPDATE OR DELETE ON audit_logs
  FOR EACH STATEMENT
  EXECUTE FUNCTION reject_audit_log_modification();
