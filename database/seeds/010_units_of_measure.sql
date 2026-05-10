-- Seed: Units of Measure
INSERT INTO units_of_measure (id, code, name, description, is_active, created_at)
VALUES
(1, 'pcs', 'Pieces', 'Satuan individual/buah', true, NOW()),
(2, 'box', 'Box', 'Kemasan kotak', true, NOW()),
(3, 'dus', 'Dus', 'Kemasan karton/dus', true, NOW()),
(4, 'kg', 'Kilogram', 'Kilogram', true, NOW()),
(5, 'gram', 'Gram', 'Gram', true, NOW()),
(6, 'liter', 'Liter', 'Liter', true, NOW()),
(7, 'ml', 'Mililiter', 'Mililiter', true, NOW()),
(8, 'pack', 'Pack', 'Kemasan/pack', true, NOW())
ON CONFLICT (id) DO NOTHING;