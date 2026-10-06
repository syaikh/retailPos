-- 061_add_store_cg_import_export_perms.sql
-- Add import/export permissions for stores and customer_groups and grant to roles

DO $$
DECLARE
  v_count int;
BEGIN
  -- Add permissions if not exist
  INSERT INTO permissions (code, description)
  SELECT 'store.import', 'Import stores'
  WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'store.import');

  INSERT INTO permissions (code, description)
  SELECT 'store.export', 'Export stores'
  WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'store.export');

  INSERT INTO permissions (code, description)
  SELECT 'customer_group.import', 'Import customer groups'
  WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'customer_group.import');

  INSERT INTO permissions (code, description)
  SELECT 'customer_group.export', 'Export customer groups'
  WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'customer_group.export');

  -- Grant store.import/export to superadmin, manager, supervisor, inventory_staff
  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code IN ('store.import','store.export')
  WHERE r.name IN ('superadmin','manager','supervisor','inventory_staff')
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );

  -- Grant customer_group import/export to superadmin and manager
  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code IN ('customer_group.import','customer_group.export')
  WHERE r.name IN ('superadmin','manager','supervisor')
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );
END $$;
