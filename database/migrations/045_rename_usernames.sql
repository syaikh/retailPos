-- Migration: 045_rename_usernames.sql
-- Description: Rename default usernames to match their new role names (migration 044).
--   admin      → manager       (store boss, role_id=2)
--   manager    → supervisor    (shift lead, role_id=3)
--   staff      → inventory_staff (stock ops, role_id=5)

BEGIN;

-- Step 1: staff → inventory_staff (must come first — no FK dependency on other usernames)
UPDATE users SET username = 'inventory_staff' WHERE username = 'staff';

-- Step 2: manager → supervisor (must come before admin → manager to avoid conflict)
UPDATE users SET username = 'supervisor' WHERE username = 'manager';

-- Step 3: admin → manager (now safe since "manager" username is free)
UPDATE users SET username = 'manager' WHERE username = 'admin';

COMMIT;
