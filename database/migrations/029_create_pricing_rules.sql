-- Migration 029: Create pricing_rules table
-- Stores pricing rules for the Pricing Engine (ADR-002, ADR-006).
-- Products retain only `products.price` as the default selling price.
-- Pricing rules are overrides for specific conditions (discount, wholesale, etc.).

CREATE TABLE pricing_rules (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    pricing_type VARCHAR(50) NOT NULL,
    name VARCHAR(200),
    price INTEGER NOT NULL CHECK (price >= 0),
    minimum_quantity INTEGER NOT NULL DEFAULT 1 CHECK (minimum_quantity >= 1),
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    effective_from TIMESTAMPTZ,
    effective_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes (ADR-002 §Database Readiness)
CREATE INDEX idx_pricing_rules_product_id ON pricing_rules(product_id);
CREATE INDEX idx_pricing_rules_active ON pricing_rules(product_id, is_active) WHERE is_active = true;
CREATE INDEX idx_pricing_rules_type ON pricing_rules(pricing_type);
CREATE INDEX idx_pricing_rules_effective ON pricing_rules(effective_from, effective_until) WHERE is_active = true;

-- ROLLBACK:
-- DROP INDEX IF EXISTS idx_pricing_rules_effective;
-- DROP INDEX IF EXISTS idx_pricing_rules_type;
-- DROP INDEX IF EXISTS idx_pricing_rules_active;
-- DROP INDEX IF EXISTS idx_pricing_rules_product_id;
-- DROP TABLE IF EXISTS pricing_rules;
