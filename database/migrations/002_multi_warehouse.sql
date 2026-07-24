-- Migration: 002_multi_warehouse.sql
-- Description: Enable multi-warehouse inventory by making the UNIQUE constraint on
-- product_stock composite (product_id, warehouse_id, store_id) instead of just product_id.
-- Uses NULLS NOT DISTINCT so that the default "global" stock row (where both warehouse_id
-- and store_id are NULL) is still limited to one per product.
-- Created: 2026-07-24

ALTER TABLE product_stock DROP CONSTRAINT IF EXISTS product_stock_product_id_key;

ALTER TABLE product_stock ADD CONSTRAINT uq_product_stock
    UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id);
