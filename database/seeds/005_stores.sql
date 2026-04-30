-- Seed: Default store
INSERT INTO stores (id, name, address, phone, is_active, created_at)
VALUES (1, 'Main Store', '123 Main Street', '081234567890', true, NOW())
ON CONFLICT (id) DO NOTHING;
