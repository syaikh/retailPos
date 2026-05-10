-- Seed: Products
INSERT INTO products (id, sku, name, barcode, category_id, brand_id, description, price, cost, stock, stock_min, stock_max, store_id, is_active, tax_class_id, unit_of_measure_id, weight_grams, is_active_for_sale, is_active_for_purchase, created_at, updated_at)
VALUES
(1, 'SKU-001', 'Indomie Goreng', '8999500972205', 1, 1, 'Mie goreng instan dengan bumbu spesial Indofood', 5000, 4000, 100, 10, 500, NULL, true, 1, 1, 85, true, true, NOW(), NOW()),
(2, 'SKU-002', 'Teh Botol Sosro', '8992721000155', 2, 2, 'Teh botol dalam kemasan special', 5000, 4500, 200, 20, 300, NULL, true, 1, 1, 300, true, true, NOW(), NOW()),
(3, 'SKU-003', 'Chitato BBQ', '8999909300268', 3, 3, 'Keripik kentang rasa BBQ', 10000, 8500, 75, 15, 200, NULL, true, 1, 1, 75, true, true, NOW(), NOW()),
(4, 'SKU-004', 'Kopiko 78° C', '8992721000611', 2, 4, 'Kopi instan dengan kafein tinggi', 8000, 6500, 150, 20, 400, NULL, true, 1, 1, 25, true, true, NOW(), NOW()),
(5, 'SKU-005', 'Beng-beng', '8999909300125', 3, 1, 'Coklat bar dengan wafer', 2500, 2000, 300, 30, 600, NULL, true, 1, 1, 45, true, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
