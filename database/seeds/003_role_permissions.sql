-- Seed: Role-Permission mapping
-- Clean permission structure with proper RBAC separation

-- Superadmin: ALL permissions (including system-level audit:read)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions
ON CONFLICT DO NOTHING;

-- Admin: ALL permissions EXCEPT audit:read (operational, not system-level)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions
WHERE code != 'audit:read'
ON CONFLICT DO NOTHING;

-- Manager: read + adjust permissions + category management
INSERT INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions
WHERE code IN (
  'product:read', 'product:update',
  'sale:read', 'sale:void',
  'report:read', 'dashboard:read',
  'inventory:read', 'inventory:adjust',
  'category:read'
)
ON CONFLICT DO NOTHING;

-- Cashier: sale + POS access only (Dashboard + POS on frontend)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions
WHERE code IN (
  'product:read',
  'sale:create', 'sale:read',
  'pos:access'
)
ON CONFLICT DO NOTHING;

-- Staff: inventory + category read (Dashboard + Inventory on frontend)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 5, id FROM permissions
WHERE code IN (
  'product:read',
  'inventory:read', 'inventory:adjust',
  'category:read'
)
ON CONFLICT DO NOTHING;
