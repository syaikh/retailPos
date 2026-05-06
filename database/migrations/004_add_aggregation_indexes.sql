-- Migration: 004_add_aggregation_indexes.sql
-- Description: Add indexes and generated columns for sales aggregation performance
-- Created: 2026-05-06

-- Generated columns for faster week/month bucketing
ALTER TABLE sales
  ADD COLUMN week_bucket DATE GENERATED ALWAYS AS
    (DATE_TRUNC('week', created_at)::date) STORED,
  ADD COLUMN month_bucket DATE GENERATED ALWAYS AS
    (DATE_TRUNC('month', created_at)::date) STORED;

-- Covering index for sales aggregation queries (store + created_at + total_amount)
CREATE INDEX idx_sales_aggregation
ON sales (store_id, created_at DESC, total_amount)
INCLUDE (id, invoice_number, cashier_id, status)
WHERE deleted_at IS NULL;

-- Partial index for active sales only (since soft-delete pattern is used)
CREATE INDEX idx_sales_active_aggregations
ON sales (created_at DESC)
WHERE deleted_at IS NULL AND status = 'completed';

-- Indexes for the new generated columns
CREATE INDEX idx_sales_week_bucket ON sales (store_id, week_bucket) WHERE deleted_at IS NULL;
CREATE INDEX idx_sales_month_bucket ON sales (store_id, month_bucket) WHERE deleted_at IS NULL;

-- Add comments
COMMENT ON COLUMN sales.week_bucket IS 'Generated column for week-based aggregations (Monday start)';
COMMENT ON COLUMN sales.month_bucket IS 'Generated column for month-based aggregations';