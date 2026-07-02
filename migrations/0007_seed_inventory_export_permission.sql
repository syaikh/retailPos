-- Migration 0007: Add inventory export permission

BEGIN;

INSERT INTO permissions (code, description, category) VALUES
('inventory:product:export', 'Export inventory data', 'inventory')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin' AND p.code = 'inventory:product:export'
ON CONFLICT DO NOTHING;

COMMIT;