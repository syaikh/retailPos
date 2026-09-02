-- Migration 039: Business-perspective permission audit fixes
--
-- Grants derived from full RBAC audit (2026-09-02). Addresses operational
-- inconsistencies where roles could create entities but not edit/delete them,
-- and missing read permissions that affect daily workflows.
--
-- Summary of changes:
--   Manager:  +product.create, +category.update, +category.delete,
--             +customer.delete, +customer.export, +customer.import,
--             +customer_group.create, +customer_group.update, +customer_group.delete,
--             +pricing.delete, +stock_opname.count, +stock_opname.submit
--   Cashier:  +category.view, +pricing.view, +customer_group.view, +dashboard.view
--   Staff:    +category.view

-- ============================================================
-- MANAGER
-- ============================================================

-- Product: can edit but not add — operational bottleneck
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code = 'product.create'
ON CONFLICT DO NOTHING;

-- Category: can create but not edit/delete — inconsistent
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code IN ('category.update', 'category.delete')
ON CONFLICT DO NOTHING;

-- Customer: can create/edit but not delete/export/import
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code IN ('customer.delete', 'customer.export', 'customer.import')
ON CONFLICT DO NOTHING;

-- Customer group: can view only — can't manage loyalty/pricing groups
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code IN ('customer_group.create', 'customer_group.update', 'customer_group.delete')
ON CONFLICT DO NOTHING;

-- Pricing: can create/update but not delete
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code = 'pricing.delete'
ON CONFLICT DO NOTHING;

-- Stock opname: can create/assign/verify but not count/submit — odd lifecycle gap
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code IN ('stock_opname.count', 'stock_opname.submit')
ON CONFLICT DO NOTHING;

-- ============================================================
-- CASHIER
-- ============================================================

-- Category: can't view — product filter UX broken
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'category.view'
ON CONFLICT DO NOTHING;

-- Pricing: can't view rules — can't see active promotions at POS
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'pricing.view'
ON CONFLICT DO NOTHING;

-- Customer group: can't view — can't see loyalty tier pricing
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'customer_group.view'
ON CONFLICT DO NOTHING;

-- Dashboard: can't view daily sales summary at shift start
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'dashboard.view'
ON CONFLICT DO NOTHING;

-- ============================================================
-- STAFF
-- ============================================================

-- Category: can't view — same UX issue as cashier
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'staff' AND p.code = 'category.view'
ON CONFLICT DO NOTHING;

-- ROLLBACK:
-- Manager: DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
--   AND permission_id IN (SELECT id FROM permissions WHERE code IN (
--   'product.create','category.update','category.delete',
--   'customer.delete','customer.export','customer.import',
--   'customer_group.create','customer_group.update','customer_group.delete',
--   'pricing.delete','stock_opname.count','stock_opname.submit'));
-- Cashier: DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'cashier')
--   AND permission_id IN (SELECT id FROM permissions WHERE code IN (
--   'category.view','pricing.view','customer_group.view','dashboard.view'));
-- Staff: DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'staff')
--   AND permission_id IN (SELECT id FROM permissions WHERE code IN ('category.view'));
