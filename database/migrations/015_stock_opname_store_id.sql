BEGIN;

-- Store-scoping for stock opname sessions. The store column identifies the
-- store the session applies to, enabling per-store filtering of real-time
-- so_* websocket events. NULL means the session operates on shared/general
-- stock and broadcasts globally (current behavior for category/product scopes).
ALTER TABLE stock_opnames
    ADD COLUMN store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL;

-- Backfill data lama:
-- 1. scope_type='store' → store = scope_id (hanya bila store masih ada,
--    scope_id tidak divalidasi dan bisa menunjuk ke store yang sudah dihapus)
UPDATE stock_opnames so
SET store_id = so.scope_id::int
WHERE so.scope_type = 'store'
  AND EXISTS (SELECT 1 FROM stores s WHERE s.id = so.scope_id::int);

-- 2. scope_type='warehouse' → store = warehouse.store_id (jika ada)
UPDATE stock_opnames so
SET store_id = w.store_id
FROM warehouses w
WHERE w.id = so.warehouse_id
  AND so.scope_type = 'warehouse'
  AND w.store_id IS NOT NULL;

-- 3. scope_type='category'/'product' → tetap NULL (general stock, broadcast global)

CREATE INDEX IF NOT EXISTS idx_stock_opnames_store_id ON stock_opnames(store_id);

INSERT INTO schema_migrations (filename) VALUES ('015_stock_opname_store_id.sql');

COMMIT;
