-- Retire the pricing rule "draft" status.
--
-- Create has always overwritten a caller-supplied status with 'pending', so no
-- code path can write 'draft' any more and the submit route built around it
-- (POST /pricing-rules/:id/submit) could only ever return 400. 'pending' is the
-- pre-approval state, which makes 'draft' a redundant second way to spell it.
--
-- The constraint is dropped and recreated rather than altered in place so the
-- migration is re-runnable: on a database that already has the tightened
-- constraint the original name no longer exists, and CREATE ... IF NOT EXISTS
-- semantics for constraints are not portable.
--
-- No rows carry 'draft' (there were none when this was written, and nothing can
-- create one), so the tightening is a no-op for existing data. The WHERE guard
-- makes that explicit rather than relying on it: if a legacy install somehow
-- does hold a draft rule, it is remapped to 'pending' -- its semantically
-- correct state -- instead of aborting the whole migration.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pricing_rules WHERE status = 'draft'
  ) THEN
    UPDATE pricing_rules SET status = 'pending' WHERE status = 'draft';
  END IF;
END
$$;

ALTER TABLE pricing_rules DROP CONSTRAINT IF EXISTS chk_pricing_status;

ALTER TABLE pricing_rules
  ADD CONSTRAINT chk_pricing_status
  CHECK (status::text = ANY (ARRAY['pending', 'approved', 'rejected']));
