-- Migration: 020_add_export_import_permissions.sql
-- Description: Add export/import permissions for master data (privileged users)
-- Created: 2026-06-28

INSERT INTO permissions (code, name, description) VALUES
  ('product:export', 'Export Produk', 'Bisa mengexport data produk'),
  ('product:import', 'Import Produk', 'Bisa mengimport data produk'),
  ('category:export', 'Export Kategori', 'Bisa mengexport data kategori'),
  ('category:import', 'Import Kategori', 'Bisa mengimport data kategori'),
  ('customer:export', 'Export Pelanggan', 'Bisa mengexport data pelanggan'),
  ('customer:import', 'Import Pelanggan', 'Bisa mengimport data pelanggan')
ON CONFLICT (code) DO NOTHING;

-- Assign to superadmin (id=1)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 1 AND p.code IN ('product:export','product:import','category:export','category:import','customer:export','customer:import')
ON CONFLICT DO NOTHING;

-- Assign to admin (id=2)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.id = 2 AND p.code IN ('product:export','product:import','category:export','category:import','customer:export','customer:import')
ON CONFLICT DO NOTHING;
