-- Migration: 015_payment_methods.sql
-- Description: Add payment method master data
-- Created: 2026-06-10

CREATE TABLE IF NOT EXISTS payment_methods (
    id SERIAL PRIMARY KEY,
    code VARCHAR(30) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    requires_reference BOOLEAN DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payment_methods_code ON payment_methods(code);
CREATE INDEX idx_payment_methods_is_active ON payment_methods(is_active);

INSERT INTO payment_methods (code, name, is_active, requires_reference, sort_order) VALUES
('CASH', 'Cash', true, false, 1),
('CARD', 'Card', true, true, 2),
('E_WALLET', 'E-Wallet', true, true, 3),
('TRANSFER', 'Transfer', true, true, 4),
('QRIS', 'QRIS', true, false, 5)
ON CONFLICT (code) DO NOTHING;
