-- Seed: Permissions
INSERT INTO permissions (code, name, description) VALUES
-- Users & Roles
('user:read', 'Baca user', 'Lihat daftar user'),
('user:create', 'Tambah user', 'Tambah user baru'),
('user:update', 'Edit user', 'Edit data user'),
('user:delete', 'Hapus user', 'Hapus user (soft delete)'),
('users:read', 'Baca user', 'Lihat daftar user (alias)'),
('users:manage', 'Kelola user', 'CRUD user & role (alias)'),
('user:manage', 'Kelola user', 'CRUD user & role (alias)'),
('role:read', 'Baca role', 'Lihat daftar role'),
('role:create', 'Tambah role', 'Tambah role baru'),
('role:update', 'Edit role', 'Edit role & permissions'),
('role:delete', 'Hapus role', 'Hapus role'),
('users:roles:manage', 'Kelola role user', 'Kelola role & permission user (alias)'),

-- Products
('product:read', 'Baca produk', 'Lihat daftar produk'),
('product:create', 'Tambah produk', 'Tambah produk baru'),
('product:update', 'Edit produk', 'Edit data produk'),
('product:delete', 'Hapus produk', 'Hapus produk (soft delete)'),

-- Sales
('sale:create', 'Buat penjualan', 'Proses transaksi penjualan'),
('sale:read', 'Baca penjualan', 'Lihat riwayat penjualan'),

-- Inventory
('inventory:read', 'Baca inventory', 'Lihat daftar inventory'),
('inventory:adjust', 'Adjust inventory', 'Penyesuaian stok manual'),
('inventory:export', 'Export inventory', 'Export data inventory'),

-- Reports
('report:view', 'Lihat laporan', 'Akses dashboard & laporan'),
('reports:read', 'Lihat laporan', 'Akses dashboard & laporan (alias)'),

-- Dashboard
('dashboard:read', 'Lihat dashboard', 'Akses dashboard utama'),

-- POS
('pos:access', 'Akses POS', 'Akses halaman POS'),

-- System
('audit:read', 'Lihat audit log', 'Lihat log audit sistem')
ON CONFLICT (code) DO NOTHING;
