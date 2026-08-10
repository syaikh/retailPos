-- Migration: 025_add_supplier_to_products_full_view.sql
-- Description: Add the preferred-supplier columns to the katalog-owned
-- v_products_full read model so product reads stop JOINing suppliers directly.
--
-- suppliers is referensi-owned; v_products_full is the katalog read model that
-- already enriches referensi-owned names (categories, brands, units_of_measure,
-- tax_classes). The preferred-supplier pick (product_suppliers -> suppliers)
-- was the only cross-context read still living in Go (productSelectCols). Moving
-- it into the view removes the `suppliers` table reference from internal/product
-- while keeping the enrichment live (plain view, not materialized).
--
-- CREATE OR REPLACE VIEW can only append columns; supplier_id/supplier_name are
-- appended after updated_at. internal/product selects by name (productSelectCols),
-- so column order is irrelevant to consumers.

BEGIN;

DROP VIEW IF EXISTS v_products_full;

CREATE VIEW v_products_full AS
SELECT
    p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name,
    p.price, COALESCE(p.cost, 0) as cost, COALESCE(ps.quantity, 0) as stock,
    p.status, p.store_id, p.brand_id, b.name as brand_name,
    p.unit_of_measure_id, u.name as unit_of_measure,
    p.weight_grams, p.description,
    p.tax_class_id, tc.rate_percent as tax_rate,
    p.search_vector,
    p.created_at, p.updated_at,
    ps_preferred.supplier_id, ps_preferred.supplier_name
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
LEFT JOIN LATERAL (
    SELECT s.id as supplier_id, s.name as supplier_name
    FROM product_suppliers ps
    JOIN suppliers s ON ps.supplier_id = s.id AND s.deleted_at IS NULL
    WHERE ps.product_id = p.id AND ps.is_preferred = true
    LIMIT 1
) ps_preferred ON true
WHERE p.deleted_at IS NULL;

INSERT INTO schema_migrations (filename) VALUES ('025_add_supplier_to_products_full_view.sql');

COMMIT;
