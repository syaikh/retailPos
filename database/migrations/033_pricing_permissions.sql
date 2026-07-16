-- Migration 033: Pricing rule management permissions
-- Dedicated pricing permissions, separate from product permissions (INV-P12).

INSERT INTO permissions (code, name, description) VALUES
  ('pricing:read', 'Lihat Aturan Harga', 'Bisa melihat daftar aturan harga'),
  ('pricing:create', 'Tambah Aturan Harga', 'Bisa menambah aturan harga baru'),
  ('pricing:update', 'Edit Aturan Harga', 'Bisa mengubah aturan harga'),
  ('pricing:delete', 'Hapus Aturan Harga', 'Bisa menghapus aturan harga')
ON CONFLICT (code) DO NOTHING;

-- Grant pricing:read, pricing:create, pricing:update, pricing:delete to superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin'
  AND p.code IN ('pricing:read', 'pricing:create', 'pricing:update', 'pricing:delete')
ON CONFLICT DO NOTHING;

-- Grant pricing:read, pricing:create, pricing:update to admin and manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
  AND p.code IN ('pricing:read', 'pricing:create', 'pricing:update')
ON CONFLICT DO NOTHING;

-- Grant pricing:delete to admin only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.code = 'pricing:delete'
ON CONFLICT DO NOTHING;

-- Grant pricing:read to cashier (read-only for POS display)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier'
  AND p.code = 'pricing:read'
ON CONFLICT DO NOTHING;

-- ROLLBACK:
-- DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('pricing:read', 'pricing:create', 'pricing:update', 'pricing:delete'));
-- DELETE FROM permissions WHERE code IN ('pricing:read', 'pricing:create', 'pricing:update', 'pricing:delete');
