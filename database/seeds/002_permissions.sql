-- Seed: Permissions (clean, consolidated version)
-- Format: colon-notation only (user:read, user:create, etc.)
INSERT INTO permissions (code, name, description) VALUES
-- Users & Roles
('user:read', 'Baca user', 'Lihat daftar user'),
('user:create', 'Tambah user', 'Tambah user baru'),
('user:update', 'Edit user', 'Edit data user'),
('user:delete', 'Hapus user', 'Hapus user (soft delete)'),
('user:view', 'Lihat detail user', 'Lihat detail satu user'),
('role:read', 'Baca role', 'Lihat daftar role'),
('role:create', 'Tambah role', 'Tambah role baru'),
('role:update', 'Edit role', 'Edit role & permissions'),
('role:delete', 'Hapus role', 'Hapus role'),

-- Products
('product:read', 'Baca produk', 'Lihat daftar produk'),
('product:create', 'Tambah produk', 'Tambah produk baru'),
('product:update', 'Edit produk', 'Edit data produk'),
('product:delete', 'Hapus produk', 'Hapus produk (soft delete)'),

-- Categories
('category:read', 'Baca kategori', 'Lihat daftar kategori'),
('category:create', 'Tambah kategori', 'Tambah kategori baru'),
('category:update', 'Edit kategori', 'Edit data kategori'),
('category:delete', 'Hapus kategori', 'Hapus kategori'),

-- Sales
('sale:read', 'Baca penjualan', 'Lihat riwayat penjualan'),
('sale:create', 'Buat penjualan', 'Proses transaksi penjualan'),
('sale:void', 'Void penjualan', 'Void/refund transaksi penjualan'),

-- Inventory
('inventory:read', 'Baca inventory', 'Lihat daftar inventory'),
('inventory:adjust', 'Adjust inventory', 'Penyesuaian stok manual'),
('inventory:export', 'Export inventory', 'Export data inventory'),

-- Reports
('report:read', 'Lihat laporan', 'Akses dashboard & laporan'),

-- Dashboard
('dashboard:read', 'Lihat dashboard', 'Akses dashboard utama'),

-- POS
('pos:access', 'Akses POS', 'Akses halaman POS'),

-- System
('audit:read', 'Lihat audit log', 'Lihat log audit sistem')
ON CONFLICT (code) DO NOTHING;
