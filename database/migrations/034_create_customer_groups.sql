-- Migration 034: Create customer_groups table
-- Customer groups allow pricing rules to target specific customer segments.

CREATE TABLE IF NOT EXISTS customer_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customer_groups_name ON customer_groups(name);
CREATE INDEX IF NOT EXISTS idx_customer_groups_active ON customer_groups(is_active) WHERE is_active = true;

-- Seed default groups
INSERT INTO customer_groups (name, description, is_active) VALUES
    ('Walk-in', 'Pelanggan umum tanpa kartu member', true),
    ('Member', 'Pelanggan terdaftar dengan kartu member', true),
    ('VIP', 'Pelanggan prioritas dengan harga khusus', true)
ON CONFLICT (name) DO NOTHING;
