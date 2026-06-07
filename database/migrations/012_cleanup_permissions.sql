-- Migration: 012_cleanup_permissions.sql
-- Description: Remove duplicate/alias permissions and add missing ones for RBAC refactoring
-- Created: 2026-06-06

-- Delete dot-notation (legacy) permissions - they are duplicates of colon-notation
DELETE FROM role_permissions WHERE permission_id IN (
  SELECT id FROM permissions WHERE code LIKE '%.view' OR code LIKE '%.create' OR code LIKE '%.update' OR code LIKE '%.delete' OR code LIKE '%.print'
);

-- Delete alias permissions
DELETE FROM permissions WHERE code IN (
  'users:read',
  'users:manage',
  'user:manage',
  'users:roles:manage',
  'reports:read'
);

-- Add is_system column to roles table (if not exists)
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT FALSE;

-- Mark superadmin as system role (immutable)
UPDATE roles SET is_system = TRUE WHERE id = 1;

-- Add missing permissions needed for RBAC refactoring
INSERT INTO permissions (code, name, description) VALUES
  ('category:update', 'Edit kategori', 'Edit data kategori'),
  ('category:delete', 'Hapus kategori', 'Hapus kategori'),
  ('sale:void', 'Void penjualan', 'Void/refund transaksi penjualan')
ON CONFLICT (code) DO NOTHING;
