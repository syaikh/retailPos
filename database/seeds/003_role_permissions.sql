-- Seed: Role-Permission mapping
-- Clean permission structure with proper RBAC separation

-- Superadmin: ALL permissions (including system-level audit.view)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions
ON CONFLICT DO NOTHING;

-- Admin: all permissions EXCEPT audit.view, role.update, role.delete, user.delete
-- Admin can manage users (create/read/update) but cannot delete users or modify roles
-- Only superadmin can manage roles and delete users
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions
WHERE code NOT IN ('audit.view', 'role.update', 'role.delete', 'user.delete')
ON CONFLICT DO NOTHING;

-- Manager: read + adjust permissions + category management
INSERT INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions
WHERE code IN (
  'product.view', 'product.update',
  'sale.view',
  'report.view', 'dashboard.view',
  'inventory.adjust',
  'category.view'
)
ON CONFLICT DO NOTHING;

-- Cashier: sale + POS access
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions
WHERE code IN (
  'product.view',
  'sale.create', 'sale.view',
  'customer.view',
  'shift.create', 'shift.view'
)
ON CONFLICT DO NOTHING;

-- Staff: inventory + product read (Stock Opname counting)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 5, id FROM permissions
WHERE code IN (
  'product.view',
  'inventory.adjust'
)
ON CONFLICT DO NOTHING;
