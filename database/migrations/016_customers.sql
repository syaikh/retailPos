-- Migration: 016_customers.sql
-- Description: Add customers master data
-- Created: 2026-06-10

CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    phone VARCHAR(20) UNIQUE,
    email VARCHAR(100),
    address TEXT,
    tax_id VARCHAR(50),
    loyalty_points INTEGER NOT NULL DEFAULT 0,
    total_spent INTEGER NOT NULL DEFAULT 0,
    last_purchase_at TIMESTAMPTZ,
    note TEXT,
    is_active BOOLEAN DEFAULT true,
    is_walk_in BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_customers_phone ON customers(phone);
CREATE INDEX idx_customers_is_active ON customers(is_active);

INSERT INTO customers (id, name, phone, email, is_walk_in, is_active)
VALUES
    (1, 'Pelanggan Umum / Walk-in', NULL, NULL, true, true),
    (2, 'Ahmad Fauzi', '081234567890', 'ahmad.fauzi@email.com', false, true),
    (3, 'Siti Nurhaliza', '082345678901', 'siti.nur@email.com', false, true),
    (4, 'Budi Santoso', '083456789012', 'budi.santoso@email.com', false, true),
    (5, 'Dewi Lestari', '084567890123', 'dewi.lestari@email.com', false, true),
    (6, 'Eko Prasetyo', '085678901234', 'eko.prasetyo@email.com', false, true),
    (7, 'Rina Wijaya', '086789012345', 'rina.wijaya@email.com', false, true),
    (8, 'Hendra Gunawan', '087890123456', 'hendra.g@email.com', false, true),
    (9, 'Maya Anggraini', '088901234567', 'maya.anggraini@email.com', false, true),
    (10, 'Rizky Pratama', '089012345678', 'rizky.pratama@email.com', false, true)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE sales ADD COLUMN IF NOT EXISTS customer_id INTEGER DEFAULT 1 REFERENCES customers(id);
