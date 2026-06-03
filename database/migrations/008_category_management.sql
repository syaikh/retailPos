-- Migration: 008_category_management.sql
-- Description: Add slug, updated_at to categories; change FK to RESTRICT; add category management perms
-- Created: 2026-06-03

-- Add slug and updated_at columns
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS slug VARCHAR(120) UNIQUE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Backfill slug for existing categories (with collision handling)
-- Note: If slugs already exist with duplicates, use the slug_collision_fix.sql script instead
UPDATE categories SET slug = LOWER(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(TRIM(name), ' ', '-'), '''', ''), '"', ''), '&', 'and'), '/', '-'))
WHERE slug IS NULL;

-- Drop existing FK constraint and re-add with RESTRICT
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_category_id_fkey;
ALTER TABLE products ADD CONSTRAINT products_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

-- Partial index for optimized JOIN/exists queries
CREATE INDEX IF NOT EXISTS idx_products_category_active ON products(category_id) WHERE deleted_at IS NULL;

-- New permissions for category management
INSERT INTO permissions (code, name, description)
VALUES
    ('category.update', 'Edit Kategori', 'Bisa mengubah kategori'),
    ('category.delete', 'Hapus Kategori', 'Bisa menghapus kategori')
ON CONFLICT (code) DO NOTHING;

-- Grant to superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('category.update', 'category.delete')
WHERE r.name = 'superadmin'
ON CONFLICT DO NOTHING;

-- Grant to admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('category.update', 'category.delete')
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;