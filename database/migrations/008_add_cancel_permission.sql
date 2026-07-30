BEGIN;

-- Add cancel permission for purchase orders
INSERT INTO permissions (code, name, description) VALUES
  ('purchase_order.cancel', 'Batalkan Purchase Order', 'Bisa membatalkan purchase order (draft/confirmed)')
ON CONFLICT (code) DO NOTHING;

-- Grant cancel permission to superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin'
  AND p.code = 'purchase_order.cancel'
ON CONFLICT DO NOTHING;

-- Grant cancel permission to admin and manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
  AND p.code = 'purchase_order.cancel'
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('008_add_cancel_permission.sql');

COMMIT;
