-- Seed: Brands
INSERT INTO brands (id, name, description, is_active, created_at, updated_at)
VALUES
(1, 'Indofood', 'Produk makanan dari PT Indofood Sukses Makmur', true, NOW(), NOW()),
(2, 'Sosro', 'Minuman teh dalam kemasan', true, NOW(), NOW()),
(3, 'Wings', 'Snack dan makanan ringan', true, NOW(), NOW()),
(4, 'Unilever', 'Produk konsumsi sehari-hari', true, NOW(), NOW()),
(5, 'Lokal', 'Brand lokal/produk umum', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;