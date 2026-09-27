-- Migration: 051_pagination_indexes.sql
-- Description: Composite indexes backing keyset (cursor) pagination for the
-- two deepest list endpoints — audit logs and sales history.
--
-- Cursor seek predicate is (created_at, id) < ($1, $2) with
-- ORDER BY created_at DESC, id DESC, so a composite (created_at, id) index
-- lets Postgres walk the index backwards instead of a full sort of every
-- matching row (deep OFFSET pages previously cost a linear scan + sort on
-- each request).
--
-- The partial store index serves the common store-scoped audit query
-- (WHERE store_id = $n) so it too becomes an index walk.
-- Migration must be applied before deploying the keyset-enabled binary.
--
-- CONCURRENTLY is required: every authenticated request writes an audit row,
-- so a plain CREATE INDEX would hold a write lock on audit_logs/sales for the
-- whole build. CONCURRENTLY cannot run inside a transaction — hence no
-- BEGIN/COMMIT. Each statement commits on its own; if the migration fails
-- partway, DROP INDEX any INVALID indexes it left behind, then rerun
-- (IF NOT EXISTS skips the completed ones).

-- 1. Audit logs (default sort + cursor seek)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_created_id
  ON audit_logs (created_at DESC, id DESC);

-- 2. Audit logs, store-scoped (partial; store_id is nullable)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_store_created_id
  ON audit_logs (store_id, created_at DESC, id DESC)
  WHERE store_id IS NOT NULL;

-- 3. Sales history / lookup (default sort + cursor seek)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sales_created_id
  ON sales (created_at DESC, id DESC);

-- Migration registration
INSERT INTO schema_migrations (filename) VALUES ('051_pagination_indexes.sql')
ON CONFLICT (filename) DO NOTHING;

-- ROLLBACK:
-- DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_created_id;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_store_created_id;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_sales_created_id;
