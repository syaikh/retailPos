BEGIN;

-- Enforce the documented least-privilege RBAC split for the admin role.
-- seeds/003_role_permissions.sql grants admin every permission EXCEPT
-- system-level ones reserved for superadmin: audit.view, role.update,
-- role.delete, user.delete. This migration removes those grants from any
-- database seeded before that rule was applied. The permission records
-- themselves stay — they are still used by superadmin.
DELETE FROM role_permissions
WHERE (role_id, permission_id) IN (
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'admin'
    AND p.code IN ('audit.view', 'role.update', 'role.delete', 'user.delete')
);

INSERT INTO schema_migrations (filename) VALUES ('022_admin_least_privilege.sql');

COMMIT;
