-- Migration: 017_stock_opname_adjustment_ledger.sql
-- Description: Add a source-of-truth ledger of inventory adjustments created by
--   stock opname posting. Each posted line becomes an adjustment row referencing
--   the operation document number. Split from 016 so a restart-safe sequence
--   (ia_seq) is created with the table that consumes it.

BEGIN;

-- ----------------------------------------------------------------
-- 1. Adjustment document sequence (safe against partial deploy: the binary
--    that first calls GetNextAdjustmentNumber must be deployed only after this
--    migration has been applied).
-- ----------------------------------------------------------------
CREATE SEQUENCE IF NOT EXISTS ia_seq START 1;

-- ----------------------------------------------------------------
-- 2. Ledger header: one row per stock adjustment document.
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS inventory_adjustments (
    id               BIGSERIAL PRIMARY KEY,
    adjustment_number VARCHAR(30) NOT NULL UNIQUE,
    session_id       INT NOT NULL REFERENCES stock_opnames(id) ON DELETE RESTRICT,
    status           VARCHAR(20) NOT NULL DEFAULT 'posted',
    notes            TEXT,
    created_by       INT REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----------------------------------------------------------------
-- 3. Adjustment lines: one row per adjusted SKU.
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS inventory_adjustment_items (
    id                 BIGSERIAL PRIMARY KEY,
    adjustment_id      INT NOT NULL REFERENCES inventory_adjustments(id) ON DELETE CASCADE,
    product_id         INT NOT NULL REFERENCES products(id),
    warehouse_id       INTEGER REFERENCES warehouses(id),
    store_id           INTEGER REFERENCES stores(id),
    expected_qty       NUMERIC(18,4) NOT NULL DEFAULT 0,
    physical_qty       NUMERIC(18,4) NOT NULL DEFAULT 0,
    difference_qty     NUMERIC(18,4) NOT NULL DEFAULT 0,
    adjustment_qty     NUMERIC(18,4) NOT NULL DEFAULT 0,
    unit_cost          NUMERIC(18,4) NOT NULL DEFAULT 0,
    line_total         NUMERIC(18,4) NOT NULL DEFAULT 0,
    reason             TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----------------------------------------------------------------
-- 4. Indexes
-- ----------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_ia_session       ON inventory_adjustments(session_id);
CREATE INDEX IF NOT EXISTS idx_ia_created       ON inventory_adjustments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ia_items_adj     ON inventory_adjustment_items(adjustment_id);
CREATE INDEX IF NOT EXISTS idx_ia_items_product ON inventory_adjustment_items(product_id);

INSERT INTO schema_migrations (filename) VALUES ('017_stock_opname_adjustment_ledger.sql');

COMMIT;