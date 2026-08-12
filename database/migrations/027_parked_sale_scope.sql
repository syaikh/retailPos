BEGIN;

-- ============================================================================
-- P2-6 — Parked-sale scope (IDOR fix) — Manager re-grant + hold note column
-- ============================================================================
-- Source:      .opencode/plans/security-remediation-plan.md, P2-6 (D4).
--
-- Ordering:    Apply BEFORE deploying the binary that:
--                * requires the sales.hold_note column (parked-sale list/detail
--                  SELECT and the CreateSale INSERT reference it), otherwise
--                  GET /sales/parked fails with a missing column;
--                * lets Manager recall any cashier's parked sale and complete a
--                  recalled sale via POST /sales/parked/:id/complete, otherwise
--                  managers 403 on every parked route (they hold no sale.park).
--
-- D4 (refined in review):
--   1. Re-grant `sale.park` to Manager — reversing the sale.park part of
--      014_remove_orphaned_role_grants.sql least-privilege cleanup. Manager
--      still gets NO `sale.create` re-grant; recall-only + scoped completion is
--      enforced in the service (role-name map pattern) and by the new
--      POST /sales/parked/:id/complete route.
--   2. sales.hold_note — optional free-text "held for" note captured at park
--      time and surfaced in the parked-sales list/detail responses.
--
-- Idempotent: ON CONFLICT DO NOTHING / ADD COLUMN IF NOT EXISTS.
-- ============================================================================

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'manager'
  AND p.code = 'sale.park'
ON CONFLICT DO NOTHING;

ALTER TABLE sales ADD COLUMN IF NOT EXISTS hold_note TEXT;

INSERT INTO schema_migrations (filename) VALUES ('027_parked_sale_scope.sql');

COMMIT;
