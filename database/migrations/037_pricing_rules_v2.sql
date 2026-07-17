-- Migration 036: Pricing rules v2 — new columns, backfill, drop price
-- This migration transforms pricing_rules from simple fixed-price to multi-method.

-- ============================================================
-- Step 1: Add new columns
-- ============================================================

ALTER TABLE pricing_rules
    ADD COLUMN IF NOT EXISTS pricing_method VARCHAR(20) NOT NULL DEFAULT 'fixed_price',
    ADD COLUMN IF NOT EXISTS pricing_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS category_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS maximum_quantity INTEGER,
    ADD COLUMN IF NOT EXISTS customer_group_id INTEGER REFERENCES customer_groups(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS recurrence_days TEXT[],
    ADD COLUMN IF NOT EXISTS time_from TIME,
    ADD COLUMN IF NOT EXISTS time_to TIME,
    ADD COLUMN IF NOT EXISTS allow_combine BOOLEAN NOT NULL DEFAULT false;

-- Make product_id nullable (was NOT NULL — now rules can target category/brand instead)
ALTER TABLE pricing_rules ALTER COLUMN product_id DROP NOT NULL;

-- ============================================================
-- Step 2: Backfill existing data
-- ============================================================

-- All existing rules use fixed_price method with their current price value
UPDATE pricing_rules SET pricing_method = 'fixed_price', pricing_value = price;

-- Simplify pricing types
UPDATE pricing_rules SET pricing_type = 'price_list' WHERE pricing_type IN ('wholesale', 'member');
UPDATE pricing_rules SET pricing_type = 'promotion' WHERE pricing_type IN ('discount', 'promotion');
UPDATE pricing_rules SET pricing_type = 'default' WHERE pricing_type = 'normal';

-- ============================================================
-- Step 3: Drop old price column
-- ============================================================

ALTER TABLE pricing_rules DROP COLUMN price;

-- ============================================================
-- Step 4: Add constraints
-- ============================================================

-- At least one target must be set
ALTER TABLE pricing_rules ADD CONSTRAINT chk_pricing_target CHECK (
    product_id IS NOT NULL OR category_id IS NOT NULL OR brand_id IS NOT NULL
);

-- ============================================================
-- Step 5: New indexes
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_pricing_rules_category ON pricing_rules(category_id) WHERE category_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_brand ON pricing_rules(brand_id) WHERE brand_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_store ON pricing_rules(store_id) WHERE store_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_group ON pricing_rules(customer_group_id) WHERE customer_group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_method ON pricing_rules(pricing_method);
