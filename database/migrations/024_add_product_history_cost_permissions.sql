BEGIN;

-- ============================================================================
-- Sprint 1 Item 3 — New permissions: product.history.view & product.cost.view
-- ============================================================================
-- Source:      docs/audits/permission-additions-sprint1.md (approved 2026-08-04)
--
-- Ordering:    Apply BEFORE deploying the binary that enforces product.cost.view
--              (GET /products, GET /products/:id omit `cost` for non-holders).
--              Until this migration runs, nobody holds product.cost.view, so
--              cost is hidden for everyone — degraded, non-breaking.
--
-- Behavior delta (Sprint 1):
--   D1  product.history.view — controls the Audit Trail panel in the product
--       detail drawer, replacing the role-based superadmin/admin check.
--   D2  product.cost.view — controls sensitive cost data (cost, margin,
--       purchase price, markup, profit). Granted to the same set that could
--       previously see cost via pricing.view (SA/Admin/Manager) — no UX
--       regression after the split.
--
-- Idempotent: ON CONFLICT DO NOTHING — safe to re-run.
-- ============================================================================

INSERT INTO permissions (code, name, description)
VALUES
  ('product.history.view', 'Product History View', 'View product entity history (audit trail: created & updated timestamps)'),
  ('product.cost.view', 'Product Cost View', 'View sensitive cost data (cost, margin, purchase price, markup, profit)')
ON CONFLICT (code) DO NOTHING;

-- product.history.view -> superadmin, admin (matches the pre-Sprint-0 role-based check)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('superadmin', 'admin')
  AND p.code = 'product.history.view'
ON CONFLICT DO NOTHING;

-- product.cost.view -> superadmin, admin, manager (same set as pricing.view holders)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('superadmin', 'admin', 'manager')
  AND p.code = 'product.cost.view'
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('024_add_product_history_cost_permissions.sql');

COMMIT;
