BEGIN;

-- Least-privilege grant of read access to storage location metadata for the
-- roles that render rack stock (product detail drawer RackStockPanel) and
-- stock opname "location" scopes (scope picker). Rack stock is a sub-account
-- of global stock, which these roles can already view via product.view, and
-- the rack panel is read-only for them (set/transfer need inventory.adjust).
-- Create/update/delete stay admin-only.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('manager', 'staff', 'cashier')
  AND p.code = 'storage_location.view'
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('021_grant_storage_location_view.sql');

COMMIT;
