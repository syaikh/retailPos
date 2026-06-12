-- Migration: 014_split_stock_to_product_stock.sql
-- Description: Separate stock from product master data
-- Created: 2026-06-10

CREATE TABLE IF NOT EXISTS product_stock (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id INTEGER REFERENCES warehouses(id) ON DELETE SET NULL,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reorder_point INTEGER NOT NULL DEFAULT 0 CHECK (reorder_point >= 0),
    reorder_quantity INTEGER NOT NULL DEFAULT 0 CHECK (reorder_quantity >= 0),
    last_restocked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_product_stock_product_id ON product_stock(product_id);
CREATE INDEX idx_product_stock_warehouse_id ON product_stock(warehouse_id);
CREATE INDEX idx_product_stock_store_id ON product_stock(store_id);

CREATE OR REPLACE VIEW v_products_full AS
SELECT
    p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name,
    p.price, p.cost, COALESCE(ps.quantity, 0) as stock,
    p.status, p.store_id, p.brand_id, b.name as brand_name,
    p.unit_of_measure_id, u.name as unit_of_measure,
    p.weight_grams, p.description,
    p.created_at, p.updated_at
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN brands b ON p.brand_id = b.id
LEFT JOIN units_of_measure u ON p.unit_of_measure_id = u.id
LEFT JOIN LATERAL (
    SELECT quantity FROM product_stock
    WHERE product_id = p.id
    ORDER BY (warehouse_id IS NULL AND store_id IS NULL) DESC
    LIMIT 1
) ps ON true
WHERE p.deleted_at IS NULL;

INSERT INTO product_stock (product_id, quantity, created_at, updated_at)
SELECT id, COALESCE(stock, 0), created_at, updated_at
FROM products
WHERE deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM product_stock WHERE product_id = products.id);
