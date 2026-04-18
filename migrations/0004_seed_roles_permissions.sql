-- Migration 0004: Seed roles and permissions, assign to admin/cashier

BEGIN;

-- Seed roles
INSERT INTO roles (name, description, is_system) VALUES
('admin',  'Full system access - all permissions', TRUE),
('cashier','Point of Sale staff - limited access', TRUE)
ON CONFLICT (name) DO NOTHING;

-- Seed permissions (20 total)
INSERT INTO permissions (code, description, category) VALUES
-- Dashboard
('dashboard:read',          'View dashboard overview',            'dashboard'),
-- POS
('pos:access',              'Access POS interface',               'pos'),
('pos:sale:create',         'Create new sales transactions',      'pos'),
('pos:sale:refund',         'Process refunds',                    'pos'),
-- Inventory - Products
('inventory:read',          'View products list',                 'inventory'),
('inventory:product:create','Add new products',                   'inventory'),
('inventory:product:update','Edit products',                      'inventory'),
('inventory:product:delete','Delete products',                    'inventory'),
('inventory:product:view_cost','View product cost prices',        'inventory'),
-- Inventory - Groups
('inventory:group:read',    'View product groups',                'inventory'),
('inventory:group:create',  'Add product groups',                 'inventory'),
('inventory:group:update',  'Edit product groups',                'inventory'),
('inventory:group:delete',  'Delete product groups',              'inventory'),
-- Reports
('reports:read',            'View reports dashboard',             'reports'),
('reports:sales:view',      'View sales reports',                'reports'),
('reports:sales:export',    'Export sales data',                  'reports'),
('reports:inventory:view',  'View inventory reports',            'reports'),
('reports:audit:view',      'View audit logs',                    'reports'),
-- User Management
('users:read',              'View user list',                     'users'),
('users:create',            'Create new users',                   'users'),
('users:update',            'Edit user accounts',                 'users'),
('users:delete',            'Delete users',                       'users'),
('users:roles:manage',      'Assign roles to users',              'users'),
-- Settings
('settings:read',           'View system settings',               'settings'),
('settings:write',          'Modify system settings',             'settings')
ON CONFLICT (code) DO NOTHING;

-- Admin gets ALL permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Cashier gets ONLY POS permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.code IN (
  'pos:access',
  'pos:sale:create'
)
WHERE r.name = 'cashier'
ON CONFLICT DO NOTHING;

COMMIT;
