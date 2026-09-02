-- Migration 038: Grant audit.view to admin role
--
-- Bug fix: admin was granted audit.export (migration 036) but not audit.view.
-- The intent of 036 was that export is "more restrictive than view", meaning
-- anyone who can export must first be able to view. Without audit.view, admin
-- can export audit logs but cannot view them in the UI — an illogical state.
--
-- This migration grants audit.view to admin, bringing admin to 68 permissions.
-- Superadmin retains exclusive access to audit.view + audit.export (already granted).
-- Manager/cashier/staff remain ungranted for both.

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.code = 'audit.view'
ON CONFLICT DO NOTHING;

-- ROLLBACK:
-- DELETE FROM role_permissions
-- WHERE role_id = (SELECT id FROM roles WHERE name = 'admin')
--   AND permission_id = (SELECT id FROM permissions WHERE code = 'audit.view');
