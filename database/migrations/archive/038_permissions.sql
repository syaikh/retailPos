-- Migration 038: Pricing + Customer Group permissions

-- ============================================================
-- Customer Group permissions
-- ============================================================

INSERT INTO permissions (code, name) VALUES
    ('customer_group:read', 'Lihat Data Customer Group'),
    ('customer_group:create', 'Buat Data Customer Group'),
    ('customer_group:update', 'Edit Data Customer Group'),
    ('customer_group:delete', 'Hapus Data Customer Group')
ON CONFLICT (code) DO NOTHING;

-- Admin+Manager: full CRUD on customer groups
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('superadmin', 'admin', 'manager')
  AND p.code IN ('customer_group:read', 'customer_group:create', 'customer_group:update', 'customer_group:delete')
ON CONFLICT DO NOTHING;

-- Cashier: read-only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'customer_group:read'
ON CONFLICT DO NOTHING;

-- ============================================================
-- Verify pricing permissions include superadmin (fix from 033 if missing)
-- ============================================================

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin'
  AND p.code IN ('pricing:read', 'pricing:create', 'pricing:update', 'pricing:delete')
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
