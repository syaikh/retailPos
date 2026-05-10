-- Seed: Tax Classes
INSERT INTO tax_classes (id, name, rate_percent, description, is_active, created_at)
VALUES
(1, 'PPN 11%', 11.00, 'Pajak Pertambahan Nilai standar 11%', true, NOW()),
(2, 'PPN 0%', 0.00, 'Tidak dikenakan PPN', true, NOW()),
(3, 'Non PPN', 0.00, 'Produk tidak kena PPN', true, NOW())
ON CONFLICT (id) DO NOTHING;