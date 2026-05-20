-- Migration: 004_add_aggregation_indexes.sql
-- Description: Add indexes for sales aggregation performance
-- Created: 2026-05-06

-- Covering index for sales aggregation queries (store + created_at + total_amount)
CREATE INDEX idx_sales_aggregation
ON sales (store_id, created_at DESC, total_amount)
INCLUDE (id, invoice_number, cashier_id, status);

-- Partial index for active sales only
CREATE INDEX idx_sales_active_aggregations
ON sales (created_at DESC)
WHERE status = 'completed';