-- Seed: Role-Permission mapping
-- Superadmin: ALL permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions
ON CONFLICT DO NOTHING;

-- Admin: all permissions (full store management)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions
ON CONFLICT DO NOTHING;

-- Manager: read permissions + reports + inventory adjust
INSERT INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions
WHERE code IN (
  'product:read', 'sale:read', 'report:view', 'reports:read', 'dashboard:read',
  'inventory:read', 'inventory:adjust', 'audit:read'
)
ON CONFLICT DO NOTHING;

-- Cashier: sale create + product read + pos access
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions
WHERE code IN (
  'product:read', 'sale:create', 'sale:read', 'pos:access'
)
ON CONFLICT DO NOTHING;
