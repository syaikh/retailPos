-- Migration: 007_drop_dead_product_columns.sql
-- Description: Drop stock_min (moved to env config), dead unused columns
-- (dimensions_cm, is_track_expiry, is_track_batch, is_active_for_sale, is_active_for_purchase)
-- Created: 2026-05-22

ALTER TABLE products DROP COLUMN IF EXISTS stock_min;
ALTER TABLE products DROP COLUMN IF EXISTS dimensions_cm;
ALTER TABLE products DROP COLUMN IF EXISTS is_track_expiry;
ALTER TABLE products DROP COLUMN IF EXISTS is_track_batch;
ALTER TABLE products DROP COLUMN IF EXISTS is_active_for_sale;
ALTER TABLE products DROP COLUMN IF EXISTS is_active_for_purchase;
