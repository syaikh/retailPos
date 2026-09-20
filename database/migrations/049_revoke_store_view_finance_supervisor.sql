-- Migration: 049_revoke_store_view_finance_supervisor.sql
-- Description: Revoke store.view from finance and supervisor roles.
--
-- Both roles are already store-scoped via RequireStoreID middleware and JWT
-- claims. The store.view permission was granted with the intent "see which
-- store they're paying for" / "see store dropdown", but the system enforces
-- store isolation at the middleware level — these roles can only access their
-- assigned store's data regardless of store.view.
--
-- Removing this permission:
--   - Eliminates confusion (the comment implied they needed visibility to
--     know which store, when the constraint is already enforced)
--   - Follows least-privilege (no reason for finance/supervisor to browse
--     the store management page)
--   - Does not break any existing flow (settlements, sales, reports are all
--     scoped by JWT store_id, not by store.view)

BEGIN;

-- Revoke from finance
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'finance')
  AND permission_id = (SELECT id FROM permissions WHERE code = 'store.view');

-- Revoke from supervisor
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'supervisor')
  AND permission_id = (SELECT id FROM permissions WHERE code = 'store.view');

-- Migration registration
INSERT INTO schema_migrations (filename) VALUES ('049_revoke_store_view_finance_supervisor.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;
