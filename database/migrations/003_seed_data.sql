-- Migration: 003_seed_data.sql
-- Description: Insert basic master data (Roles, Permissions, Superadmin user)
-- Created: 2026-05-02 (Refactored)

-- Roles
INSERT INTO roles (name, description, is_system)
VALUES 
  ('superadmin', 'Super Administrator', true),
  ('admin', 'Administrator', true),
  ('manager', 'Manager / Kepala Toko', true),
  ('cashier', 'Kasir', true),
  ('staff', 'Staff Gudang', true)
ON CONFLICT (name) DO NOTHING;

-- Permissions
INSERT INTO permissions (code, name, description)
VALUES 
  ('dashboard.view', 'Lihat Dashboard', 'Bisa melihat dashboard utama'),
  ('user.view', 'Lihat Pengguna', 'Bisa melihat daftar pengguna'),
  ('user.create', 'Tambah Pengguna', 'Bisa menambah pengguna baru'),
  ('user.update', 'Edit Pengguna', 'Bisa mengubah data pengguna'),
  ('user.delete', 'Hapus Pengguna', 'Bisa menghapus pengguna'),
  ('role.view', 'Lihat Role', 'Bisa melihat daftar role'),
  ('role.create', 'Tambah Role', 'Bisa menambah role baru'),
  ('role.update', 'Edit Role', 'Bisa mengubah role'),
  ('role.delete', 'Hapus Role', 'Bisa menghapus role'),
  ('product.view', 'Lihat Produk', 'Bisa melihat daftar produk'),
  ('product.create', 'Tambah Produk', 'Bisa menambah produk baru'),
  ('product.update', 'Edit Produk', 'Bisa mengubah data produk'),
  ('product.delete', 'Hapus Produk', 'Bisa menghapus produk'),
  ('category.view', 'Lihat Kategori', 'Bisa melihat kategori'),
  ('category.create', 'Tambah Kategori', 'Bisa menambah kategori'),
  ('sale.view', 'Lihat Penjualan', 'Bisa melihat daftar penjualan'),
  ('sale.create', 'Buat Penjualan', 'Bisa membuat transaksi penjualan'),
  ('sale.print', 'Cetak Struk', 'Bisa mencetak struk penjualan'),
  ('report.view', 'Lihat Laporan', 'Bisa melihat laporan keuangan & stok'),
  ('audit.view', 'Lihat Log Audit', 'Bisa melihat riwayat log audit')
ON CONFLICT (code) DO NOTHING;

-- Role Permissions (Superadmin full access)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'superadmin'
ON CONFLICT DO NOTHING;

-- Admin permissions (tanpa role & user delete)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM roles r
JOIN permissions p ON p.code IN (
  'dashboard.view','user.view','user.create','user.update',
  'role.view','role.create','role.update',
  'product.view','product.create','product.update','product.delete',
  'category.view','category.create',
  'sale.view','sale.create','sale.print',
  'report.view'
)
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Manager permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM roles r
JOIN permissions p ON p.code IN (
  'dashboard.view','product.view','product.update',
  'category.view','sale.view','sale.create','sale.print',
  'report.view'
)
WHERE r.name = 'manager'
ON CONFLICT DO NOTHING;

-- Cashier permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM roles r
JOIN permissions p ON p.code IN (
  'dashboard.view','product.view',
  'sale.view','sale.create','sale.print'
)
WHERE r.name = 'cashier'
ON CONFLICT DO NOTHING;

-- Users (Password default: admin123)
-- Note: Menggunakan gen_salt dari pgcrypto untuk hash bcrypt
INSERT INTO users (username, email, password_hash, role_id, is_active)
VALUES 
  ('superadmin', 'superadmin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), (SELECT id FROM roles WHERE name='superadmin'), true),
  ('admin', 'admin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), (SELECT id FROM roles WHERE name='admin'), true),
  ('manager', 'manager@retailpos.local', crypt('admin123', gen_salt('bf', 14)), (SELECT id FROM roles WHERE name='manager'), true),
  ('cashier', 'cashier@retailpos.local', crypt('admin123', gen_salt('bf', 14)), (SELECT id FROM roles WHERE name='cashier'), true)
ON CONFLICT (username) DO NOTHING;
