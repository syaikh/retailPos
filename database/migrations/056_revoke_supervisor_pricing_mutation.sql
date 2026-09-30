-- Wave 2c: revoke pricing mutation from supervisor.
--
-- The supplier re-gating moved /suppliers onto supplier.* codes and dropped
-- pricing.create/update/delete from supervisor's grant list in 000_baseline.sql.
-- Editing that list is not enough, and this file exists because that is a
-- property of the baseline rather than an oversight:
--
--   000_baseline.sql only ever INSERTs into role_permissions. It has no DELETE
--   against that table, so a permission removed from a role's ARRAY stops being
--   added on a fresh install but is never removed from a database that already
--   granted it. Replaying the baseline on an upgraded database is therefore not
--   equivalent to a fresh one, and the removal silently did nothing.
--
-- This is the same reason 031_revoke_sale_lookup_manager.sql and
-- 049_revoke_store_view_finance_supervisor.sql exist as separate files. A grant
-- that must go away needs an explicit DELETE.
--
-- Nothing breaks: supervisor keeps pricing.view, so the pricing list and the
-- read-only pricing detail views still work, and it gains supplier.view, which
-- is what the supplier screen now requires.
--
-- After this runs, supervisor holds exactly the 56 codes in the 000_baseline.sql
-- array: the three revoked codes were the only difference between an upgraded
-- database and a fresh one. That equality is asserted by the array-length check
-- in TestBaselineRoleGrantsMatchPolicy, so the two cannot drift apart again
-- without a test failing.
--
-- The DELETE is naturally idempotent: a role that does not hold the code
-- matches no row, so re-running is a no-op.

BEGIN;

DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'supervisor')
  AND permission_id IN (
      SELECT id FROM permissions
      WHERE code IN ('pricing.create', 'pricing.update', 'pricing.delete')
  );

-- Migration registration
INSERT INTO schema_migrations (filename) VALUES ('056_revoke_supervisor_pricing_mutation.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;
