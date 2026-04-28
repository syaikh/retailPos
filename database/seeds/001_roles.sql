-- Seed: Roles default
INSERT INTO roles (id, name, description, is_system) VALUES
(1, 'superadmin', 'Full system access, semua permissions', true),
(2, 'admin', 'Admin toko - kelola produk, users, laporan', true),
(3, 'manager', 'Manager - lihat laporan, kelola inventory', true),
(4, 'cashier', 'Kasir - transaksi penjualan saja', true)
ON CONFLICT (id) DO NOTHING;
