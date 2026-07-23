-- Migration 032: Add pricing snapshot columns to sale_items
-- Stores pricing context for historical accuracy (ADR-004, INV-P6).
-- Fields are immutable after sale commit.

ALTER TABLE sale_items
    ADD COLUMN pricing_rule_id INTEGER REFERENCES pricing_rules(id) ON DELETE SET NULL,
    ADD COLUMN pricing_rule_name VARCHAR(200),
    ADD COLUMN pricing_rule_type VARCHAR(50),
    ADD COLUMN pricing_type VARCHAR(50),
    ADD COLUMN original_price INTEGER NOT NULL DEFAULT 0;

-- Backfill existing rows with sensible defaults
UPDATE sale_items SET
    pricing_type = 'normal',
    original_price = unit_price
WHERE pricing_type IS NULL;

-- Index for reporting queries (revenue by pricing type)
CREATE INDEX idx_sale_items_pricing_type ON sale_items(pricing_type);

-- ROLLBACK:
-- DROP INDEX IF EXISTS idx_sale_items_pricing_type;
-- ALTER TABLE sale_items DROP COLUMN IF EXISTS pricing_rule_id;
-- ALTER TABLE sale_items DROP COLUMN IF EXISTS pricing_rule_name;
-- ALTER TABLE sale_items DROP COLUMN IF EXISTS pricing_rule_type;
-- ALTER TABLE sale_items DROP COLUMN IF EXISTS pricing_type;
-- ALTER TABLE sale_items DROP COLUMN IF EXISTS original_price;
