-- Migration: 043_product_ownership_type.sql
-- Description: Add ownership_type to products table for hybrid consignment model.
-- Values: 'store' (default), 'consignment'
-- Enables unified product search with clear ownership indication.

BEGIN;

-- Add ownership_type column with default value
ALTER TABLE products ADD COLUMN IF NOT EXISTS ownership_type VARCHAR(20) DEFAULT 'store';

-- Add check constraint for valid values
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_products_ownership_type'
    ) THEN
        ALTER TABLE products ADD CONSTRAINT chk_products_ownership_type
            CHECK (ownership_type IN ('store', 'consignment'));
    END IF;
END $$;

-- Create index for filtering
CREATE INDEX IF NOT EXISTS idx_products_ownership_type ON products(ownership_type);

-- Backfill existing consignment products based on consignment_stock table
UPDATE products p
SET ownership_type = 'consignment'
WHERE EXISTS (
    SELECT 1 FROM consignment_stock cs WHERE cs.product_id = p.id
);

-- Update v_products_full view to include ownership_type
DROP VIEW IF EXISTS v_products_full;
CREATE VIEW v_products_full AS
 SELECT p.id,
    p.sku,
    p.name,
    p.barcode,
    p.category_id,
    c.name AS category_name,
    p.price,
    COALESCE(p.cost, 0) AS cost,
    COALESCE(ps.quantity, 0) AS stock,
    p.status,
    p.store_id,
    p.brand_id,
    b.name AS brand_name,
    p.unit_of_measure_id,
    u.name AS unit_of_measure,
    p.weight_grams,
    p.description,
    p.tax_class_id,
    tc.rate_percent AS tax_rate,
    p.search_vector,
    p.created_at,
    p.updated_at,
    ps_preferred.supplier_id,
    ps_preferred.supplier_name,
    p.ownership_type
   FROM ((((((public.products p
     LEFT JOIN categories c ON ((p.category_id = c.id)))
     LEFT JOIN brands b ON ((p.brand_id = b.id)))
     LEFT JOIN units_of_measure u ON ((p.unit_of_measure_id = u.id)))
     LEFT JOIN LATERAL ( SELECT product_stock.quantity
           FROM product_stock
          WHERE (product_stock.product_id = p.id)
          ORDER BY ((product_stock.warehouse_id IS NULL) AND (product_stock.store_id IS NULL)) DESC
         LIMIT 1) ps ON (true))
     LEFT JOIN tax_classes tc ON ((tc.id = p.tax_class_id)))
     LEFT JOIN LATERAL ( SELECT s.id AS supplier_id,
            s.name AS supplier_name
           FROM (public.product_suppliers ps_1
             JOIN suppliers s ON (((ps_1.supplier_id = s.id) AND (s.deleted_at IS NULL))))
          WHERE ((ps_1.product_id = p.id) AND (ps_1.is_preferred = true))
         LIMIT 1) ps_preferred ON (true))
  WHERE (p.deleted_at IS NULL);

COMMIT;
