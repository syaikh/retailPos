-- Seed: Permissions (clean, consolidated version)
-- Format: dot-notation only (user.view, user.create, etc.)
INSERT INTO permissions (code, name, description) VALUES
-- Users & Roles
('user.view', 'Baca user', 'Lihat daftar user'),
('user.create', 'Tambah user', 'Tambah user baru'),
('user.update', 'Edit user', 'Edit data user'),
('user.delete', 'Hapus user', 'Hapus user (soft delete)'),
('role.view', 'Baca role', 'Lihat daftar role'),
('role.create', 'Tambah role', 'Tambah role baru'),
('role.update', 'Edit role', 'Edit role & permissions'),
('role.delete', 'Hapus role', 'Hapus role'),

-- Products
('product.view', 'Baca produk', 'Lihat daftar produk'),
('product.create', 'Tambah produk', 'Tambah produk baru'),
('product.update', 'Edit produk', 'Edit data produk'),
('product.delete', 'Hapus produk', 'Hapus produk (soft delete)'),

-- Categories
('category.view', 'Baca kategori', 'Lihat daftar kategori'),
('category.create', 'Tambah kategori', 'Tambah kategori baru'),
('category.update', 'Edit kategori', 'Edit data kategori'),
('category.delete', 'Hapus kategori', 'Hapus kategori'),

-- Sales
('sale.view', 'Baca penjualan', 'Lihat riwayat penjualan'),
('sale.create', 'Buat penjualan', 'Proses transaksi penjualan'),

-- Inventory
('inventory.adjust', 'Adjust inventory', 'Penyesuaian stok manual'),
('inventory.export', 'Export inventory', 'Export data inventory'),

-- Reports
('report.view', 'Lihat laporan', 'Akses dashboard & laporan'),

-- Dashboard
('dashboard.view', 'Lihat dashboard', 'Akses dashboard utama'),

-- POS
('pos.access', 'Akses POS', 'Akses halaman POS'),

-- System
('audit.view', 'Lihat audit log', 'Lihat log audit sistem')
ON CONFLICT (code) DO NOTHING;
