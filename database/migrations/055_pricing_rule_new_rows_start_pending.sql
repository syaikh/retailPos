-- Wave 2d follow-up: make the schema agree with the approval workflow.
--
-- pricing_rules.status defaulted to 'approved' and is_active defaulted to true.
-- Every Go creation path now pins both values (internal/pricing/service.go
-- Create for the API, BulkInsertPricingRules for imports), so the defaults were
-- unreachable from the application. They were not unreachable from the
-- database, though: any INSERT that omitted the two columns -- a future
-- service, a psql fix, a script -- produced a rule that was approved and
-- active on insert, which is the immediate-active creation the approval
-- workflow exists to prevent. The default was the loophole under the code.
--
-- Changing a default affects new rows only, so rules that are already approved
-- and active stay active. This is the "existing rules remain until replaced"
-- part of the decision, and it needs no data backfill.
--
-- ALTER COLUMN ... SET DEFAULT is already idempotent, so this re-runs cleanly on
-- every migration pass; no IF NOT EXISTS guard is available or needed.

ALTER TABLE public.pricing_rules
    ALTER COLUMN status SET DEFAULT 'pending';

ALTER TABLE public.pricing_rules
    ALTER COLUMN is_active SET DEFAULT false;
