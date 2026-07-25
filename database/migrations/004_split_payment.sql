-- Migration: 004_split_payment.sql
-- Description: Add sale_payments table for split payment support
-- Created: 2026-07-25

CREATE TABLE IF NOT EXISTS sale_payments (
    id SERIAL PRIMARY KEY,
    sale_id INTEGER NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    payment_method_id INTEGER NOT NULL REFERENCES payment_methods(id) ON DELETE RESTRICT,
    payment_method_code VARCHAR(30) NOT NULL,
    amount INTEGER NOT NULL CHECK (amount > 0),
    reference_number VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_payments_sale ON sale_payments(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_payments_method ON sale_payments(payment_method_id);
