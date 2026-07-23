-- Migration: 010_partial_unique_barcode.sql
-- Description: Replace UNIQUE constraint with partial unique index on barcode
--              so that soft-deleted products can share barcodes with active ones.
-- Created: 2026-06-05

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_barcode_key;
CREATE UNIQUE INDEX idx_products_unique_active_barcode
ON products (barcode)
WHERE deleted_at IS NULL;
