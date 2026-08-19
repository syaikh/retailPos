-- Migration: 006_user_preferences.sql
-- Description: Add per-user language and theme preference columns to the users
-- table. Removes the dead `default_language` key from app_settings.
-- Deployment ordering: apply BEFORE deploying the binary that reads/writes
-- language/theme on users, otherwise login responses omit those fields.

BEGIN;

-- ============================================================
-- Per-user preference columns
-- ============================================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS language VARCHAR(5) DEFAULT 'id';
ALTER TABLE users ADD COLUMN IF NOT EXISTS theme VARCHAR(10) DEFAULT 'light';

-- ============================================================
-- Remove dead default_language from app_settings
-- ============================================================
DELETE FROM app_settings WHERE key = 'default_language';

-- ============================================================
-- Migration registration (idempotent)
-- ============================================================
INSERT INTO schema_migrations (filename) VALUES ('006_user_preferences.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;
