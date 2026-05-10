-- Seed: Warehouses
INSERT INTO warehouses (id, name, code, address, store_id, is_active, created_at)
VALUES
(1, 'Main Warehouse', 'MAIN', 'Gudang utama toko', NULL, true, NOW()),
(2, 'Secondary Storage', 'SEC', 'Gudang cadangan', NULL, true, NOW())
ON CONFLICT (id) DO NOTHING;