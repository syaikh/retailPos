-- Wave 2d: split pricing approval from editing and block self-approval.
--
-- pricing_rules had no author column, so "you may not approve your own rule"
-- could not be enforced: audit_logs records who created a rule but they are
-- deletable, and the last *updater* is the person who usually submits it for
-- approval. created_by is the reliable signal, and it is nullable so that rules
-- predating this migration keep working (a NULL author is not a self-approval).
--
-- The permission rows themselves (supplier.*, pricing.approve) are seeded in
-- 000_baseline.sql, which is amended in place rather than duplicated here.

-- ---------------------------------------------------------------------------
-- pricing_rules.created_by
-- ---------------------------------------------------------------------------

ALTER TABLE public.pricing_rules
    ADD COLUMN IF NOT EXISTS created_by integer;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass
          AND conname = 'pricing_rules_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules
            ADD CONSTRAINT pricing_rules_created_by_fkey
            FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;
END
$$;

-- The self-approval check reads created_by on every approve, so index it.
CREATE INDEX IF NOT EXISTS idx_pricing_rules_created_by
    ON public.pricing_rules (created_by);
