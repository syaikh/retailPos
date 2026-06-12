-- Seed: 012_payment_methods.sql
-- Idempotent seed for payment methods
-- Can be re-run safely via ON CONFLICT DO NOTHING

INSERT INTO payment_methods (code, name, is_active, requires_reference, sort_order)
VALUES
    ('CASH', 'Cash', true, false, 1),
    ('CARD', 'Card', true, true, 2),
    ('E_WALLET', 'E-Wallet', true, true, 3),
    ('TRANSFER', 'Transfer', true, true, 4),
    ('QRIS', 'QRIS', true, false, 5)
ON CONFLICT (code) DO NOTHING;
