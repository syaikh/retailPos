-- Migration: 006_consolidate_permissions.sql
-- Description: Consolidate permission naming. Remove .read duplicates (use .view instead).
-- The system had a dual naming convention: .view (original) and .read (introduced later).
-- All handler code and frontend route guards use .view. This migration drops the unused .read variants.
--
-- Also removes colon-notation (:read/:create) permissions that were already converted to dot-notation.
-- The normalizePermissionCode function (which converted : → .) has been removed from both
-- backend middleware and frontend utilities.

BEGIN;

-- Add missing .view permissions to staff role (5)
-- Staff had .read variants but not .view for these entities
INSERT INTO role_permissions (role_id, permission_id)
SELECT 5, id FROM permissions WHERE code IN ('category.view', 'dashboard.view', 'product.view')
AND id NOT IN (SELECT permission_id FROM role_permissions WHERE role_id = 5)
ON CONFLICT DO NOTHING;

-- Remove old .read role_permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code LIKE '%.read'
);

-- Remove old .read permission codes
DELETE FROM permissions WHERE code LIKE '%.read';

COMMIT;
