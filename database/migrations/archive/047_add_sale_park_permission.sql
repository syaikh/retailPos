-- Migration 047: Add sale.park permission for Hold & Recall feature
-- Allows cashiers to park (hold), recall, and cancel parked sales.

INSERT INTO permissions (code, name, description) VALUES
  ('sale.park', 'Parkir Penjualan', 'Bisa menyimpan sementara, mengambil kembali, dan membatalkan penjualan yang diparkir')
ON CONFLICT (code) DO NOTHING;

-- Grant sale.park to superadmin, admin, manager, and cashier
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('superadmin', 'admin', 'manager', 'cashier')
  AND p.code = 'sale.park'
ON CONFLICT DO NOTHING;

-- ROLLBACK:
-- DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'sale.park');
-- DELETE FROM permissions WHERE code = 'sale.park';
