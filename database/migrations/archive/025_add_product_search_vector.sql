-- Migration: 025_add_product_search_vector.sql
-- Description: Add full-text search support to products table
-- Created: 2026-07-11

-- Add tsvector column for full-text search
ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN index for fast full-text search
CREATE INDEX IF NOT EXISTS idx_products_search_vector ON products USING GIN(search_vector);

-- Populate search_vector from existing data
UPDATE products SET search_vector =
    setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(sku, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(barcode, '')), 'C');

-- Create trigger to keep search_vector up to date on INSERT/UPDATE
CREATE OR REPLACE FUNCTION products_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.sku, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.barcode, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_products_search_vector ON products;
CREATE TRIGGER trg_products_search_vector
    BEFORE INSERT OR UPDATE OF name, sku, barcode ON products
    FOR EACH ROW
    EXECUTE FUNCTION products_search_vector_update();

-- Recreate v_products_full view to include search_vector
DROP VIEW IF EXISTS v_products_full;
CREATE VIEW v_products_full AS
SELECT
    p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name,
    p.price, p.cost, COALESCE(ps.quantity, 0) as stock,
    p.status, p.store_id, p.brand_id, b.name as brand_name,
    p.unit_of_measure_id, u.name as unit_of_measure,
    p.weight_grams, p.description,
    p.tax_class_id, tc.rate_percent as tax_rate,
    p.search_vector,
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
LEFT JOIN tax_classes tc ON tc.id = p.tax_class_id
WHERE p.deleted_at IS NULL;
