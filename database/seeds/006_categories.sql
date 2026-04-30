-- Seed: Categories
INSERT INTO categories (id, name, description, is_active, created_at)
VALUES
(1, 'Makanan', 'Produk makanan & minuman', true, NOW()),
(2, 'Minuman', 'Soft drink & juice', true, NOW()),
(3, 'Snack', 'Camilan & kerupuk', true, NOW()),
(4, 'Lainnya', 'Produk lainnya', true, NOW())
ON CONFLICT (id) DO NOTHING;
