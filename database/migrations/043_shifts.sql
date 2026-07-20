-- Migration 043: Create shifts table
-- Supports cashier shift management with opening/closing balances

CREATE TABLE IF NOT EXISTS shifts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    opening_balance INTEGER NOT NULL DEFAULT 0,
    closing_balance INTEGER,
    cash_sales INTEGER NOT NULL DEFAULT 0,
    non_cash_sales INTEGER NOT NULL DEFAULT 0,
    total_sales INTEGER NOT NULL DEFAULT 0,
    transaction_count INTEGER NOT NULL DEFAULT 0,
    discrepancy INTEGER,
    notes TEXT,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shifts_user_id ON shifts(user_id);
CREATE INDEX idx_shifts_store_id ON shifts(store_id);
CREATE INDEX idx_shifts_status ON shifts(status);
CREATE INDEX idx_shifts_opened_at ON shifts(opened_at);

COMMENT ON TABLE shifts IS 'Cashier shift management with opening/closing balances';
