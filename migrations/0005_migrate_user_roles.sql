-- Migration 0005: Migrate existing users.role string to users.role_id FK
-- IMPORTANT: Run AFTER 0003 and 0004 have been applied successfully

BEGIN;

-- Backfill: map users.role string to roles.id
UPDATE users u
SET role_id = r.id
FROM roles r
WHERE u.role = r.name;

-- Verify: query should return all users with matched role_name
-- SELECT u.id, u.username, u.role, u.role_id, r.name AS resolved_role FROM users u JOIN roles r ON u.role_id = r.id;

-- If any users have NULL role_id after this, it means their role string didn't match seeded roles.
-- Check with: SELECT * FROM users WHERE role_id IS NULL;

-- Optionally, add NOT NULL constraint (uncomment after verification):
-- ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;

-- Keep old role column for backward compatibility during transition.
-- Cleanup (drop column) will happen in Phase 7.1 after 1-2 weeks of stable RBAC.

COMMIT;
