-- Add shift_id to sales table for shift-based querying
-- This enables fraud prevention by linking each sale to a specific shift

ALTER TABLE sales ADD COLUMN IF NOT EXISTS shift_id INTEGER REFERENCES shifts(id);
CREATE INDEX IF NOT EXISTS idx_sales_shift_id ON sales(shift_id);
