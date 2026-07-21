-- Seed: 013_customer_permissions.sql
-- Idempotent seed for customer management permissions

INSERT INTO permissions (code, name, description)
VALUES
    ('customer.view', 'View Customers', 'View customer list and details'),
    ('customer.create', 'Create Customer', 'Add new customers'),
    ('customer.update', 'Update Customer', 'Edit customer information'),
    ('customer.delete', 'Delete Customer', 'Deactivate/delete customers')
ON CONFLICT (code) DO NOTHING;

-- Superadmin & Admin: full customer management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE p.code IN ('customer.view', 'customer.create', 'customer.update', 'customer.delete')
  AND r.name IN ('admin', 'superadmin')
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- Manager: read + create + update (no delete)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE p.code IN ('customer.view', 'customer.create', 'customer.update')
  AND r.name = 'manager'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- Cashier: read only (POS customer lookup, no management)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE p.code IN ('customer.view')
  AND r.name = 'cashier'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
