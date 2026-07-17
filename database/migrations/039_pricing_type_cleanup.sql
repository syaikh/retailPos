-- Migration 039: Pricing Type Cleanup
-- 1. Remove 'default' pricing type (products.price is the single source of truth)
-- 2. Rename 'price_list' to 'special_price' for clearer semantics

BEGIN;

-- Delete all existing 'default' type rules (redundant with products.price)
DELETE FROM pricing_rules WHERE pricing_type = 'default';

-- Rename 'price_list' → 'special_price'
UPDATE pricing_rules SET pricing_type = 'special_price' WHERE pricing_type = 'price_list';

-- Update CHECK constraint
ALTER TABLE pricing_rules
  DROP CONSTRAINT IF EXISTS pricing_rules_pricing_type_check;

ALTER TABLE pricing_rules
  ADD CONSTRAINT pricing_rules_pricing_type_check
  CHECK (pricing_type IN ('special_price', 'promotion'));

COMMIT;
