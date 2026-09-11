-- Migration: 044_store_first_and_finance_role.sql
-- Description: Store-first enforcement + finance role + role restructuring.
--   1. Seeds a default store if none exists
--   2. Renames roles to match retail job titles (order matters!)
--   3. Creates 'finance' role with payment/reporting permissions only
--   4. Grants supervisor: sale.create, store.view
--   5. Updates inventory_staff permissions
--   6. Backfills store_id for existing users

BEGIN;

-- ============================================================
-- 1. Default store (ensures fresh deploys have at least one)
-- ============================================================
INSERT INTO stores (name, address, phone, is_active, created_at)
SELECT 'Default Store', 'Alamat toko default', '0000000000', true, NOW()
WHERE NOT EXISTS (SELECT 1 FROM stores);

-- ============================================================
-- 2. Rename roles to match retail job titles (ORDER MATTERS!)
--    Idempotent: only renames if old name still exists.
-- ============================================================
-- Step 1: manager → supervisor (must come first to avoid conflict)
UPDATE roles SET name = 'supervisor', description = 'Supervisor — pengawasan operasional harian'
WHERE name = 'manager' AND NOT EXISTS (SELECT 1 FROM roles WHERE name = 'supervisor');

-- Step 2: admin → manager (now safe since "manager" is free)
UPDATE roles SET name = 'manager', description = 'Manajer Toko — pengelolaan toko secara penuh'
WHERE name = 'admin' AND NOT EXISTS (SELECT 1 FROM roles WHERE name = 'manager');

-- Step 3: staff → inventory_staff
UPDATE roles SET name = 'inventory_staff', description = 'Staf Inventaris — pengelolaan stok dan opname'
WHERE name = 'staff' AND NOT EXISTS (SELECT 1 FROM roles WHERE name = 'inventory_staff');

-- ============================================================
-- 3. Create finance role
-- ============================================================
INSERT INTO roles (name, description)
VALUES ('finance', 'Keuangan — mencatat pembayaran supplier dan melihat laporan')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- 4. Assign finance permissions
-- ============================================================
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'finance'
  AND p.code IN (
      'consignment.pay',   -- record payments to suppliers
      'report.view',       -- view financial reports
      'audit.view',        -- view audit trail
      'sale.view',         -- view sales for reconciliation
      'store.view',        -- see which store they're paying for
      'dashboard.view'     -- see dashboard
  )
ON CONFLICT DO NOTHING;

-- ============================================================
-- 5. Supervisor: add sale.create + store.view
-- ============================================================
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'supervisor'
  AND p.code IN (
      'sale.create',   -- supervisors can ring up sales at POS
      'store.view'     -- supervisors can see store dropdown
  )
ON CONFLICT DO NOTHING;

-- ============================================================
-- 6. Inventory staff: replace permissions with inventory-related ones
-- ============================================================
-- Remove old permissions
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'inventory_staff');

-- Add inventory-related permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'inventory_staff'
  AND p.code IN (
      'inventory.adjust',
      'stock_opname.view', 'stock_opname.create', 'stock_opname.assign',
      'stock_opname.count', 'stock_opname.submit', 'stock_opname.recount',
      'stock_opname.cancel', 'stock_opname.export', 'stock_opname.verify',
      'stock_opname.post', 'stock_opname.close', 'stock_opname.report',
      'storage_location.view', 'storage_location.create',
      'storage_location.update', 'storage_location.delete'
  )
ON CONFLICT DO NOTHING;

-- ============================================================
-- 7. Backfill store_id for existing users
-- ============================================================
UPDATE users SET store_id = (
    SELECT id FROM stores ORDER BY id LIMIT 1
)
WHERE store_id IS NULL
  AND role_id IN (SELECT id FROM roles WHERE name IN ('supervisor', 'manager', 'cashier', 'finance', 'inventory_staff'));

-- ============================================================
-- Migration registration
-- ============================================================
INSERT INTO schema_migrations (filename) VALUES ('044_store_first_and_finance_role.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;
