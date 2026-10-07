-- Grant product.view to the operational roles that read the catalog in the UI.
--
-- The HTTP read routes (GET /products, GET /products/:id) used to be
-- authenticated but permission-free, so finance and inventory_staff — which do
-- not hold product.view — could still browse the product catalog. Catalog reads
-- are a legitimate need for both (stock workflows for inventory_staff, margin
-- context for finance), so the resolution is to align the grant set with the
-- route surface instead of cutting the reads: the permission that the routes now
-- enforce is exactly the one these roles hold.
--
-- Cashier, supervisor, manager and superadmin already carry product.view; only
-- the two ops roles are added here. Re-runnable: the grant is guarded by a
-- NOT EXISTS check, same as the other grant migrations.
DO $$
BEGIN
  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code = 'product.view'
  WHERE r.name IN ('inventory_staff','finance')
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );
END $$;