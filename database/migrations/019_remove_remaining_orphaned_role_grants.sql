BEGIN;

-- Remove remaining orphaned grants on the seeded manager role that leak
-- management capabilities with no matching UI or route access:
--   manager: store.create, store.update, store.delete
--            customer_group.create, customer_group.update, customer_group.delete
-- Manager has no Stores page (store.view was revoked in 014) and only views
-- customer groups (customer_group.view). The permission records themselves
-- stay — they are still used by superadmin/admin.
DELETE FROM role_permissions
WHERE (role_id, permission_id) IN (
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'manager'
    AND p.code IN ('store.create', 'store.update', 'store.delete',
                   'customer_group.create', 'customer_group.update', 'customer_group.delete')
);

INSERT INTO schema_migrations (filename) VALUES ('019_remove_remaining_orphaned_role_grants.sql');

COMMIT;
