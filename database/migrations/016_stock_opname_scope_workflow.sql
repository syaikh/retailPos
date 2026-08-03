-- Migration: 016_stock_opname_scope_workflow.sql
-- Description: Extend stock opname to full & partial cycle count workflow.
--   - New workflow states: draft, open, counting, verification, needs_recount,
--     approved, posted, closed, cancelled  (replaces 012's pending_approval & approved).
--   - Adds session audit columns (opened/verified/posted/closed).
--   - Adds an extensible scopes table so new scope types (brand, supplier, manual,
--     and future location/rack) can be added without schema redesign.
--   - Replaces the global single-active-session guard with per-SKU overlap support.
--   - Adds verify / post / close / report permissions.
--   - Backfills: pending_approval -> verification; approved -> posted+closed
--     (v1 approval already posted the adjustment in the same transaction).

BEGIN;

-- ----------------------------------------------------------------
-- 1. stock_opnames: widen status + scope_type and add audit columns
-- ----------------------------------------------------------------
ALTER TABLE stock_opnames
    DROP CONSTRAINT IF EXISTS chk_stock_opname_status;

-- ----------------------------------------------------------------
-- 2. stock_opnames: add audit / summary / display columns
-- ----------------------------------------------------------------
ALTER TABLE stock_opnames
    ADD COLUMN IF NOT EXISTS opened_by        INTEGER REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS opened_at        TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verified_by      INTEGER REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS verified_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS posted_by        INTEGER REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS posted_at        TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS closed_by        INTEGER REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS closed_at        TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS scope_name       VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS title            VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS notes            TEXT,
    ADD COLUMN IF NOT EXISTS total_difference NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_adjustment NUMERIC(18,4) NOT NULL DEFAULT 0;

-- Backfill legacy rows (the audit columns referenced below now exist).
-- v1 'approved' meant the adjustment was already posted atomically -> migrate to posted|closed.
UPDATE stock_opnames
SET status = 'posted',
    posted_by = approved_by,
    posted_at = COALESCE(approved_at, NOW()),
    closed_by = approved_by,
    closed_at = COALESCE(approved_at, NOW())
WHERE status = 'approved';

-- v1 'pending_approval' is the verification review step.
UPDATE stock_opnames
SET status = 'verification'
WHERE status = 'pending_approval';

-- Apply the final status CHECK AFTER the legacy backfill above, so existing
-- 'pending_approval'/'approved' rows are remapped before the constraint is
-- validated (the new list omits 'pending_approval').
ALTER TABLE stock_opnames
    ADD CONSTRAINT chk_stock_opname_status
    CHECK (status IN ('draft','open','counting','verification','needs_recount','approved','posted','closed','cancelled'));

-- Drop old-denormalized-end-and-add index needs depend on existing columns only;
-- scope_name is kept for front-end back-compat and mirrored into the scopes table.

-- Widen scope_type CHECK to support brand/supplier/manual (and future location).
ALTER TABLE stock_opnames
    DROP CONSTRAINT IF EXISTS chk_stock_opname_scope_type;
ALTER TABLE stock_opnames
    ADD CONSTRAINT chk_stock_opname_scope_type
    CHECK (scope_type IN ('store','warehouse','category','brand','supplier','product','manual'));

-- ----------------------------------------------------------------
-- 3. Replace the global single-active-session guard with per-SKU overlap
--    enforcement. Overlap is checked and serialised at session creation in the
--    service layer (see repository/service); no global unique index is kept.
-- ----------------------------------------------------------------
DROP INDEX IF EXISTS uq_stock_opname_active;

-- ----------------------------------------------------------------
-- 4. stock_opname_items: reason per line + stock bucket scoping
-- ----------------------------------------------------------------
ALTER TABLE stock_opname_items
    ADD COLUMN IF NOT EXISTS reason       TEXT,
    ADD COLUMN IF NOT EXISTS warehouse_id INTEGER REFERENCES warehouses(id),
    ADD COLUMN IF NOT EXISTS store_id     INTEGER REFERENCES stores(id);

-- ----------------------------------------------------------------
-- 5. Extensible session scopes
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_opname_session_scopes (
    id             BIGSERIAL PRIMARY KEY,
    stock_opname_id INT NOT NULL REFERENCES stock_opnames(id) ON DELETE CASCADE,
    scope_type     VARCHAR(30) NOT NULL,
    scope_id       BIGINT,
    scope_name     VARCHAR(255) DEFAULT '',
    scope_data     JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_so_scope_type CHECK (scope_type IN
        ('store','warehouse','category','brand','supplier','product','manual'))
);

-- Backfill scopes from the legacy denormalized primary scope.
INSERT INTO stock_opname_session_scopes (stock_opname_id, scope_type, scope_id, scope_name)
SELECT so.id, so.scope_type, so.scope_id, so.scope_name
FROM stock_opnames so
ON CONFLICT DO NOTHING;

-- ----------------------------------------------------------------
-- 6. Recount request persistence (comments previously only in audit trail)
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_opname_recount_requests (
    id              BIGSERIAL PRIMARY KEY,
    stock_opname_id INT NOT NULL REFERENCES stock_opnames(id) ON DELETE CASCADE,
    requested_by    INT NOT NULL REFERENCES users(id),
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----------------------------------------------------------------
-- 7. Indexes
-- ----------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_stock_opname_status_created  ON stock_opnames(status, created_at);
CREATE INDEX IF NOT EXISTS idx_so_items_opname_status       ON stock_opname_items(stock_opname_id, status);
CREATE INDEX IF NOT EXISTS idx_so_scopes_opname             ON stock_opname_session_scopes(stock_opname_id);
CREATE INDEX IF NOT EXISTS idx_so_scopes_type_id            ON stock_opname_session_scopes(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_so_recounts_opname           ON stock_opname_recount_requests(stock_opname_id);

-- ----------------------------------------------------------------
-- 8. Permissions
-- ----------------------------------------------------------------
INSERT INTO permissions (code, name, description) VALUES
  ('stock_opname.verify', 'Verifikasi Stock Opname', 'Bisa memverifikasi hasil stock opname'),
  ('stock_opname.post',   'Posting Stock Opname',    'Bisa memposting penyesuaian stok'),
  ('stock_opname.close',  'Tutup Stock Opname',      'Bisa menutup sesi stock opname'),
  ('stock_opname.report', 'Laporan Stock Opname',    'Bisa melihat laporan stock opname')
ON CONFLICT (code) DO NOTHING;

-- Remove legacy permission codes replaced by the new workflow:
--   'stock_opname.approve'  -> verify + post (separation of duties)
--   'stock_opname.reject'   -> handled under stock_opname.verify
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('stock_opname.approve','stock_opname.reject'));

DELETE FROM permissions
WHERE code IN ('stock_opname.approve','stock_opname.reject');

-- superadmin / admin: all stock opname permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('superadmin','admin') AND p.code LIKE 'stock_opname.%'
ON CONFLICT DO NOTHING;

-- manager: add verify / post / close / report to existing grants
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code IN (
  'stock_opname.verify','stock_opname.post','stock_opname.close','stock_opname.report')
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('016_stock_opname_scope_workflow.sql');

COMMIT;