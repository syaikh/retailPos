-- Migration: 017_customers_backfill_fields.sql
-- Description: Make phone/email NOT NULL and backfill dummy data for existing empty records
-- Created: 2026-06-12

-- Step 1: Backfill existing customers with NULL/phone or email
-- Use unique-per-id values to avoid UNIQUE constraint violations
UPDATE customers
SET
    phone = CONCAT('0812', LPAD(id::text, 8, '0'))
WHERE is_walk_in = false
  AND (phone IS NULL OR phone = '');

UPDATE customers
SET
    email = CONCAT('customer', id, '@retail-pos.local')
WHERE is_walk_in = false
  AND (email IS NULL OR email = '');

-- Step 2: Make phone NOT NULL
ALTER TABLE customers
    ALTER COLUMN phone SET NOT NULL;

-- Step 3: Make email NOT NULL
ALTER TABLE customers
    ALTER COLUMN email SET NOT NULL;
