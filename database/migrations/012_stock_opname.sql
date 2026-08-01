BEGIN;

-- Stock Opname session numbering
CREATE SEQUENCE IF NOT EXISTS so_seq START 1;

-- Main Stock Opname session table
CREATE TABLE stock_opnames (
    id SERIAL PRIMARY KEY,
    session_number VARCHAR(30) NOT NULL UNIQUE,
    scope_type VARCHAR(20) NOT NULL,
    scope_id BIGINT NOT NULL,
    warehouse_id BIGINT,
    blind_count BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_by INT NOT NULL,
    approved_by INT,
    approved_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_opname_scope_type CHECK (scope_type IN ('store', 'warehouse', 'category', 'product')),
    CONSTRAINT chk_stock_opname_status CHECK (status IN ('draft', 'counting', 'pending_approval', 'needs_recount', 'approved', 'cancelled'))
);

-- Snapshot of every product in the session scope
CREATE TABLE stock_opname_items (
    id SERIAL PRIMARY KEY,
    stock_opname_id INT NOT NULL REFERENCES stock_opnames(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id),
    opening_qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    expected_qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    physical_qty NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    difference_qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    adjustment_qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    sku VARCHAR(100) DEFAULT '',
    barcode VARCHAR(100) DEFAULT '',
    uom_name VARCHAR(50) DEFAULT 'pcs',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_opname_item_status CHECK (status IN ('pending', 'counted'))
);

-- Append-only count history (supports multi-counter + recount)
CREATE TABLE stock_opname_counts (
    id SERIAL PRIMARY KEY,
    stock_opname_item_id INT NOT NULL REFERENCES stock_opname_items(id) ON DELETE CASCADE,
    count_sequence INT NOT NULL CHECK (count_sequence >= 1),
    physical_qty NUMERIC(18,4) NOT NULL CHECK (physical_qty >= 0),
    counted_by INT NOT NULL REFERENCES users(id),
    counted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remarks TEXT
);

-- Counter / supervisor assignment history
CREATE TABLE stock_opname_assignments (
    id SERIAL PRIMARY KEY,
    stock_opname_id INT NOT NULL REFERENCES stock_opnames(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_opname_assignment_role CHECK (role IN ('counter', 'supervisor'))
);

-- Indexes
CREATE INDEX idx_stock_opname_status ON stock_opnames(status);
CREATE INDEX idx_stock_opname_scope ON stock_opnames(scope_type, scope_id);
CREATE INDEX idx_stock_opname_created ON stock_opnames(created_at DESC);
CREATE INDEX idx_stock_opname_items_opname ON stock_opname_items(stock_opname_id);
CREATE INDEX idx_stock_opname_items_product ON stock_opname_items(product_id);
CREATE INDEX idx_stock_opname_items_status ON stock_opname_items(status);
CREATE INDEX idx_stock_opname_counts_item ON stock_opname_counts(stock_opname_item_id);
CREATE INDEX idx_stock_opname_counts_counted ON stock_opname_counts(counted_at);
CREATE INDEX idx_stock_opname_assignments_opname ON stock_opname_assignments(stock_opname_id);
CREATE INDEX idx_stock_opname_assignments_user ON stock_opname_assignments(user_id);

-- BR-006/BR-033: a user can hold at most one assignment per session+role
CREATE UNIQUE INDEX uq_stock_opname_assignment ON stock_opname_assignments (stock_opname_id, user_id, role);

-- BR-001: only one active session. Scope filtering (store/warehouse/category/product)
-- is not implemented in v1; every session snapshots the full general stock, so allowing
-- multiple active sessions (even with different scope_type/scope_id) risks double-adjusting
-- the same product_stock rows at approval. Enforce a single active session globally.
CREATE UNIQUE INDEX uq_stock_opname_active ON stock_opnames ((1))
WHERE status IN ('draft', 'counting', 'pending_approval', 'needs_recount');

-- Stock Opname permissions
INSERT INTO permissions (code, name, description) VALUES
  ('stock_opname.view', 'Lihat Stock Opname', 'Bisa melihat daftar dan detail stock opname'),
  ('stock_opname.create', 'Buat Stock Opname', 'Bisa membuat sesi stock opname baru'),
  ('stock_opname.assign', 'Atur Petugas Stock Opname', 'Bisa menugaskan counter dan supervisor'),
  ('stock_opname.count', 'Hitung Stok Fisik', 'Bisa melakukan penghitungan stok fisik'),
  ('stock_opname.submit', 'Submit Stock Opname', 'Bisa mengirim hasil penghitungan'),
  ('stock_opname.approve', 'Setujui Stock Opname', 'Bisa menyetujui hasil stock opname'),
  ('stock_opname.reject', 'Tolak Stock Opname', 'Bisa menolak hasil stock opname'),
  ('stock_opname.recount', 'Minta Hitung Ulang', 'Bisa meminta penghitungan ulang'),
  ('stock_opname.cancel', 'Batalkan Stock Opname', 'Bisa membatalkan sesi stock opname'),
  ('stock_opname.export', 'Ekspor Stock Opname', 'Bisa mengekspor laporan stock opname')
ON CONFLICT (code) DO NOTHING;

-- superadmin: all stock opname permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin' AND p.code LIKE 'stock_opname.%'
ON CONFLICT DO NOTHING;

-- admin: all stock opname permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.code LIKE 'stock_opname.%'
ON CONFLICT DO NOTHING;

-- manager: create / assign / approve / reject / recount / cancel / view / export (no count/submit)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.code IN (
  'stock_opname.view', 'stock_opname.create', 'stock_opname.assign',
  'stock_opname.approve', 'stock_opname.reject', 'stock_opname.recount',
  'stock_opname.cancel', 'stock_opname.export')
ON CONFLICT DO NOTHING;

-- counter actors (cashier/staff): view / count / submit
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('cashier', 'staff') AND p.code IN (
  'stock_opname.view', 'stock_opname.count', 'stock_opname.submit')
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('012_stock_opname.sql');

COMMIT;
