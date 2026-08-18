-- Migration: 005_app_settings.sql
-- Description: Creates the app_settings key-value table for global application
-- configuration (store branding, receipt text, default language). Seeds defaults
-- and grants app_settings.view/update to superadmin/admin.
-- Deployment ordering: apply BEFORE deploying the binary that reads/writes
-- app_settings, otherwise the server panics or returns 500 on /api/settings.

BEGIN;

-- ============================================================
-- Key-value settings table
-- ============================================================
CREATE TABLE IF NOT EXISTS app_settings (
    key        VARCHAR(100) PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- Seed defaults (idempotent)
-- ============================================================
INSERT INTO app_settings (key, value) VALUES
    ('store_name',        'RetailPOS'),
    ('store_jargon',      'Management System'),
    ('default_language',  'id'),
    ('receipt_header',    ''),
    ('receipt_footer',    'Terima kasih atas kunjungan Anda!')
ON CONFLICT (key) DO NOTHING;

-- ============================================================
-- Permission codes
-- ============================================================
INSERT INTO permissions (code, name, description) VALUES
    ('app_settings.view',   'Lihat Pengaturan Aplikasi',   'Bisa melihat pengaturan global aplikasi'),
    ('app_settings.update', 'Ubah Pengaturan Aplikasi',    'Bisa mengubah pengaturan global aplikasi')
ON CONFLICT (code) DO NOTHING;

-- Superadmin: view + update
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin'
  AND p.code IN ('app_settings.view', 'app_settings.update')
ON CONFLICT DO NOTHING;

-- Admin: view only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.code = 'app_settings.view'
ON CONFLICT DO NOTHING;

-- ============================================================
-- Migration registration (idempotent)
-- ============================================================
INSERT INTO schema_migrations (filename) VALUES ('005_app_settings.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;
