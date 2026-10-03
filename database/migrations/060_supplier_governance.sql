-- 060_supplier_governance.sql
--
-- Supplier master-data governance (Phase 2). Adds row provenance and optimistic
-- concurrency to the global suppliers table, and narrows the global code
-- uniqueness to live rows so a code freed by a soft delete is reusable.
--
-- Permanently re-runnable per the migration replay contract: every statement is
-- guarded. 000_baseline.sql now performs the same constraint->partial-index
-- swap so a replay cannot re-add the global constraint; this migration repeats
-- it because internal/shared/testdb.go skips files already recorded in
-- schema_migrations, so an existing database applies only this file.

-- Provenance. ON DELETE SET NULL keeps the supplier row if the actor is later
-- removed; existing rows keep NULL. ADD COLUMN with a constant default is
-- metadata-only on PostgreSQL 11+ (dev is 18.3).
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS created_by integer REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS updated_by integer REFERENCES users(id) ON DELETE SET NULL;

-- Optimistic concurrency. Every update bumps it; the client sends the version
-- it read and a stale value is refused. NOT NULL DEFAULT 1 backfills existing
-- rows in place.
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS version integer NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_suppliers_updated_by ON suppliers(updated_by);

-- F4: the global UNIQUE (code) counts soft-deleted rows, so delete-then-recreate
-- with the same code fails. Replace it with a partial unique index scoped to
-- live rows. The old constraint guaranteed the live subset is already unique, so
-- building the partial index cannot fail on existing data.
ALTER TABLE suppliers DROP CONSTRAINT IF EXISTS suppliers_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS suppliers_code_active_key ON suppliers (code) WHERE deleted_at IS NULL;
