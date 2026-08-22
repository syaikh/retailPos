-- 008_revoke_sale_lookup_manager.sql
-- Revokes the cross-cashier lookup grant from the manager role.
--
-- Decision (per design): Find Transaction is a cashier-only capability. Managers,
-- admins, and superadmins already see every cashier's sales in "My Transactions"
-- via report.view (ownership.CanAccessAll), so the Find Transaction tab is both
-- redundant and a weaker redacted subset for them. The frontend hides the tab bar
-- entirely for report.view holders; this migration removes the now-unused manager
-- grant so the permission is cashier-only.
--
-- Note: startup only validates that permission *codes* exist, so revoking a role
-- grant does not break server startup. Apply before deploying the binary that
-- hides the tab for managers (migration-ordering rule).

DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
  AND permission_id = (SELECT id FROM permissions WHERE code = 'sale.lookup');
