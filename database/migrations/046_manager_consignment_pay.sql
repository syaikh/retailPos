-- Migration 046: Grant consignment.pay to manager role
--
-- Bug fix: migration 001 granted manager (role 2) consignment.view,
-- consignment.create, consignment.update and consignment.settle, but omitted
-- consignment.pay. As a result, the settlement screen hides the "Pay" button
-- for managers even though they can create settlements — an illogical state
-- (a manager can settle a consignment but cannot record the supplier payout).
--
-- This migration grants consignment.pay to manager, matching the role's
-- existing consignment.* permissions. Superadmin retains all five permissions
-- (already granted in 001).

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'manager'
  AND p.code = 'consignment.pay'
ON CONFLICT DO NOTHING;

-- ROLLBACK:
-- DELETE FROM role_permissions
-- WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
--   AND permission_id = (SELECT id FROM permissions WHERE code = 'consignment.pay');