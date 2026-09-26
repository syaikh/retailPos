-- Migration: 050_store_onboarding.sql
-- Description: Store onboarding prerequisites — forced first-login password
-- rotation and HQ-only store provisioning.
--
-- Part of the store onboarding wizard (docs/design/store-onboarding-wizard.md).
--
-- 1. users.must_change_password
--    The wizard creates staff accounts with a generated temp password. The flag
--    travels in the JWT (auth_service.AuthClaims) and is enforced by
--    middleware.NewModularAuthMiddleware, which aborts with 428
--    "PASSWORD_CHANGE_REQUIRED" on every protected route except the
--    change-password/logout allowlist. ChangePassword clears the column and
--    re-issues the access token. Default false keeps every existing account
--    unaffected.
--
-- 2. store.create revoked from manager
--    Store provisioning is an HQ action. A store-scoped manager cannot read the
--    new store's staff (list endpoints are scoped by JWT store_id, and
--    requireOwnStore 403s on any other store), so a manager provisioning a
--    second store could never verify readiness. Superadmin retains store.create.

BEGIN;

-- 1. Forced password rotation flag
ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password boolean NOT NULL DEFAULT false;

-- 2. HQ-only store provisioning
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
  AND permission_id = (SELECT id FROM permissions WHERE code = 'store.create');

-- Migration registration
INSERT INTO schema_migrations (filename) VALUES ('050_store_onboarding.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;

-- ROLLBACK:
-- DELETE FROM role_permissions
-- WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
--   AND permission_id = (SELECT id FROM permissions WHERE code = 'store.create');
-- ALTER TABLE users DROP COLUMN IF EXISTS must_change_password;
