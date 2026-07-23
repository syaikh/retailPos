-- Migration 035: Add customer_group_id FK to customers + store permissions

-- Add customer_group_id FK to customers table
ALTER TABLE customers ADD COLUMN IF NOT EXISTS customer_group_id INTEGER REFERENCES customer_groups(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_customers_customer_group ON customers(customer_group_id);

-- Store permissions (stores table already exists from migration 001)
INSERT INTO permissions (code, name) VALUES
    ('store:read', 'Lihat Data Toko'),
    ('store:create', 'Buat Data Toko'),
    ('store:update', 'Edit Data Toko'),
    ('store:delete', 'Hapus Data Toko')
ON CONFLICT (code) DO NOTHING;

-- Admin+Manager: full CRUD on stores
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('superadmin', 'admin', 'manager')
  AND p.code IN ('store:read', 'store:create', 'store:update', 'store:delete')
ON CONFLICT DO NOTHING;

-- Cashier: read-only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'store:read'
ON CONFLICT DO NOTHING;
