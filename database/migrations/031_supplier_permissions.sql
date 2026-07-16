-- Migration 031: Supplier cost visibility permissions
-- Permission-based cost visibility for supplier data (ADR-003, INV-S3).

INSERT INTO permissions (code, name, description) VALUES
  ('supplier_cost:view', 'Lihat Harga Beli Supplier', 'Bisa melihat unit_cost supplier pada produk dan tautan supplier'),
  ('supplier_cost:update', 'Edit Harga Beli Supplier', 'Bisa mengubah unit_cost saat menautkan produk ke supplier')
ON CONFLICT (code) DO NOTHING;

-- Grant supplier_cost:view to admin and manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
  AND p.code = 'supplier_cost:view'
ON CONFLICT DO NOTHING;

-- Grant supplier_cost:update to admin only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.code = 'supplier_cost:update'
ON CONFLICT DO NOTHING;

-- ROLLBACK:
-- DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('supplier_cost:view', 'supplier_cost:update'));
-- DELETE FROM permissions WHERE code IN ('supplier_cost:view', 'supplier_cost:update');
