-- Migration 041: Pricing Rules Name Uniqueness
-- Rename duplicate names by appending target info + ID, then add UNIQUE constraint.

BEGIN;

-- Step 1: Rename ALL rules to include target scope for disambiguation
-- Format: "original_name - product_name (ID:123)" or "original_name - category/brand (ID:456)"
UPDATE pricing_rules pr
SET name = pr.name || ' - ' || COALESCE(
    (SELECT p.name || ' (ID:' || p.id || ')' FROM products p WHERE p.id = pr.product_id),
    (SELECT c.name || ' (ID:' || c.id || ')' FROM categories c WHERE c.id = pr.category_id),
    (SELECT b.name || ' (ID:' || b.id || ')' FROM brands b WHERE b.id = pr.brand_id),
    'Unknown'
);

-- Step 2: Add UNIQUE constraint
ALTER TABLE pricing_rules
    ADD CONSTRAINT pricing_rules_name_unique UNIQUE (name);

COMMIT;
