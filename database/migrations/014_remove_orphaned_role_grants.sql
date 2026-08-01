BEGIN;

-- Revoke orphaned permissions from seeded roles: functional permissions the
-- roles have no sidebar UI for (least-privilege cleanup). The permission
-- records themselves stay — they are still used by other roles.
--   Manager: store.view, sale.create, sale.park
--   Staff:   dashboard.view, shift.view, category.view
--   Cashier: dashboard.view, pricing.view, customer_group.view, store.view
DELETE FROM role_permissions
WHERE (role_id, permission_id) IN (
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE (r.name = 'manager' AND p.code IN ('store.view', 'sale.create', 'sale.park'))
     OR (r.name = 'staff'   AND p.code IN ('dashboard.view', 'shift.view', 'category.view'))
     OR (r.name = 'cashier' AND p.code IN ('dashboard.view', 'pricing.view', 'customer_group.view', 'store.view'))
);

INSERT INTO schema_migrations (filename) VALUES ('014_remove_orphaned_role_grants.sql');

COMMIT;
