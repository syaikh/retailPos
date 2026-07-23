-- Migration: 024_add_customers_store_id.sql
-- Description: Add store_id to customers table for store isolation
-- Created: 2026-07-10

ALTER TABLE customers ADD COLUMN IF NOT EXISTS store_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX IF NOT EXISTS idx_customers_store_id ON customers(store_id);
