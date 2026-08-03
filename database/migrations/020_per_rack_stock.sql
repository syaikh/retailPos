-- Migration: 020_per_rack_stock.sql
-- Description: Phase 2 of Storage Locations - per-rack stock tracking.
-- product_stock gains a location_id dimension so each rack/shelf can carry its
-- own quantity (a sub-account of the global row). stock_opnames gains a
-- location_id so sessions can count a single rack.

-- Invariant: rack stock rows ALWAYS mirror the rack's warehouse_id/store_id
-- (copied from storage_locations at write time). Global rows keep
-- warehouse_id/store_id/location_id all NULL. Existing queries that filter
-- "warehouse_id IS NULL AND store_id IS NULL" therefore keep selecting exactly
-- the global row.

-- ----------------------------------------------------------------
-- 1. product_stock: location dimension
-- ----------------------------------------------------------------
ALTER TABLE product_stock
    ADD COLUMN IF NOT EXISTS location_id INTEGER REFERENCES storage_locations(id) ON DELETE SET NULL;

ALTER TABLE product_stock DROP CONSTRAINT IF EXISTS uq_product_stock;
ALTER TABLE product_stock ADD CONSTRAINT uq_product_stock
    UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id, location_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_location_id ON product_stock(location_id);

-- ----------------------------------------------------------------
-- 2. stock_opnames: location scope column (mirrors warehouse/store)
-- ----------------------------------------------------------------
ALTER TABLE stock_opnames
    ADD COLUMN IF NOT EXISTS location_id INTEGER REFERENCES storage_locations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_stock_opnames_location_id ON stock_opnames(location_id);

-- ----------------------------------------------------------------
-- 3. Widen scope_type CHECK constraints to allow 'location'
-- ----------------------------------------------------------------
ALTER TABLE stock_opnames
    DROP CONSTRAINT IF EXISTS chk_stock_opname_scope_type;
ALTER TABLE stock_opnames
    ADD CONSTRAINT chk_stock_opname_scope_type
    CHECK (scope_type IN ('store','warehouse','category','brand','supplier','product','manual','location'));

ALTER TABLE stock_opname_session_scopes
    DROP CONSTRAINT IF EXISTS chk_so_scope_type;
ALTER TABLE stock_opname_session_scopes
    ADD CONSTRAINT chk_so_scope_type
    CHECK (scope_type IN ('store','warehouse','category','brand','supplier','product','manual','location'));

INSERT INTO schema_migrations (filename) VALUES ('020_per_rack_stock.sql');
