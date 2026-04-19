-- Migration 0006: Enforce valid payment methods (cash, card only)
-- Strategy: Truncate sales for fresh data, then add CHECK constraint

BEGIN;

-- Truncate sales cleanly (minimal WAL, RESTART IDENTITY)
TRUNCATE TABLE sales RESTART IDENTITY CASCADE;

-- Add CHECK constraint to only allow 'cash' or 'card'
ALTER TABLE sales
ADD CONSTRAINT valid_payment_method
CHECK (payment_method IN ('cash', 'card'));

COMMIT;
