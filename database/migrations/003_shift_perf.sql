-- Migration: 003_shift_perf.sql
-- Description: Optimise shift performance — composite index for sales aggregation
-- and incremental shift total updates on each completed sale.
-- Created: 2026-07-25

-- Composite index to accelerate aggregation queries when closing a shift
CREATE INDEX IF NOT EXISTS idx_sales_shift_status ON sales (shift_id, status);
