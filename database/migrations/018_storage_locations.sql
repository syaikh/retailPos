-- Migration: 018_storage_locations.sql
-- Description: Master data for storage locations (racks/shelves) where products are kept.
-- Phase 1 of per-rack stock tracking: master data only. Rack-aware stock and
-- rack-aware stock opname come in later phases.

-- ----------------------------------------------------------------
-- 1. Table
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS storage_locations (
    id           SERIAL PRIMARY KEY,
    code         VARCHAR(50) NOT NULL,
    name         VARCHAR(100) NOT NULL,
    warehouse_id INTEGER REFERENCES warehouses(id) ON DELETE SET NULL,
    store_id     INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    notes        TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_storage_locations_code UNIQUE (code),
    CONSTRAINT chk_storage_locations_scope CHECK (
        warehouse_id IS NOT NULL OR store_id IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_storage_locations_warehouse_id ON storage_locations(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_storage_locations_store_id     ON storage_locations(store_id);
CREATE INDEX IF NOT EXISTS idx_storage_locations_active       ON storage_locations(is_active);

-- ----------------------------------------------------------------
-- 2. Permissions
-- ----------------------------------------------------------------
INSERT INTO permissions (code, name, description) VALUES
  ('storage_location.view',   'Lihat Lokasi Penyimpanan', 'Bisa melihat daftar dan detail lokasi penyimpanan'),
  ('storage_location.create', 'Buat Lokasi Penyimpanan',  'Bisa membuat lokasi penyimpanan baru'),
  ('storage_location.update', 'Ubah Lokasi Penyimpanan',  'Bisa mengubah lokasi penyimpanan'),
  ('storage_location.delete', 'Hapus Lokasi Penyimpanan', 'Bisa menghapus lokasi penyimpanan')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('superadmin','admin') AND p.code LIKE 'storage_location.%'
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('018_storage_locations.sql');
