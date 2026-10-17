-- 065_consignment_force_return.sql
-- Add the consignment.force_return permission (PRD §15.9 forced-return release)
-- and grant it to superadmin and manager only. This is the store-sovereign escape
-- hatch for BR-05b: releasing a supplier's abandoned stock so the SKU is no
-- longer locked against other consignment suppliers.

DO $$
BEGIN
  INSERT INTO permissions (code, name, description)
  SELECT 'consignment.force_return', 'Paksa Pengembalian Stok Konsinyasi',
         'Bisa memaksa pengembalian seluruh stok konsinyasi yang tersisa'
  WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'consignment.force_return');

  -- Store-sovereign action: superadmin and the store boss (manager). The
  -- abandoned-stock release is a store-level decision (PRD §15.9), while
  -- finance-only roles (supervisor/finance/cashier/inventory_staff) do not get it.
  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code = 'consignment.force_return'
  WHERE r.name IN ('superadmin', 'manager')
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );
END $$;
