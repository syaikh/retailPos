-- Migration: 013_add_dashboard_read_to_cashier_staff.sql
-- Description: Add dashboard:read permission to cashier and staff roles
-- Required for dashboard stat cards to load (GET /api/dashboard/live)
-- Created: 2026-06-07

-- Add dashboard:read to cashier (role_id = 4)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions WHERE code = 'dashboard:read'
ON CONFLICT DO NOTHING;

-- Add dashboard:read to staff (role_id = 5)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 5, id FROM permissions WHERE code = 'dashboard:read'
ON CONFLICT DO NOTHING;
