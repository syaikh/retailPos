-- Migration: 006_product_schema_update.sql
-- Description: Update product table structure - add status; remove stock_max, is_active
-- Created: 2026-05-12

-- Add new column
ALTER TABLE products ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

-- Migrate is_active to status (reverse logic: false -> inactive, true -> active)
UPDATE products SET status = CASE WHEN is_active THEN 'active' ELSE 'inactive' END;

-- Add constraint for status values (requires separate statement in PostgreSQL)
ALTER TABLE products ADD CONSTRAINT chk_product_status CHECK (status IN ('draft', 'active', 'inactive', 'discontinued', 'archived'));

-- Remove stock_max column
ALTER TABLE products DROP COLUMN IF EXISTS stock_max;

-- Remove is_active column
ALTER TABLE products DROP COLUMN IF EXISTS is_active;