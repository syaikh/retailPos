BEGIN;

-- Remove dead permissions: granted to roles but never enforced by any backend
-- endpoint and unused by the frontend.
--   sale.print, sale.void, inventory.view, supplier_cost.view, supplier_cost.update
DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT id FROM permissions
  WHERE code IN ('sale.print', 'sale.void', 'inventory.view', 'supplier_cost.view', 'supplier_cost.update')
);

DELETE FROM permissions
WHERE code IN ('sale.print', 'sale.void', 'inventory.view', 'supplier_cost.view', 'supplier_cost.update');

INSERT INTO schema_migrations (filename) VALUES ('013_remove_dead_permissions.sql');

COMMIT;
