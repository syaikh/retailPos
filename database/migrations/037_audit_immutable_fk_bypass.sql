-- Allow FK-cascade updates (e.g. ON DELETE SET NULL from stores) to pass
-- through the append-only trigger.  Only direct application writes are blocked.
CREATE OR REPLACE FUNCTION reject_audit_log_modification()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  -- FK cascades fire at depth >= 1; allow them through.
  IF pg_trigger_depth() > 1 THEN
    RETURN NULL;
  END IF;
  IF current_setting('app.allow_audit_mod', true) = 'on' THEN
    RETURN NULL;
  END IF;
  RAISE EXCEPTION 'audit_logs is append-only: modifications are not permitted';
END;
$$;
