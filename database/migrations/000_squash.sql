-- Migration: 000_squash.sql
-- Description: Squashed migration — idempotent final schema covering all 001–047
-- Generated: 2026-07-23
-- Usage: CREATE TABLE IF NOT EXISTS + INSERT ON CONFLICT DO NOTHING for safe re-runs

BEGIN;

-- ============================================================
-- Cleanup stale schema_migrations (from migrated 001–047 files)
-- ============================================================
DELETE FROM schema_migrations WHERE filename LIKE '00%.sql' AND filename != '000_squash.sql';

-- ============================================================
-- Extensions
-- ============================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================
-- Sequences
-- ============================================================
CREATE SEQUENCE IF NOT EXISTS sku_seq START 1 INCREMENT 1;
CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1;

-- Sync invoice_seq with existing data (safe no-op on fresh DB)
DO $$
DECLARE max_seq bigint;
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'sales') THEN
        SELECT COALESCE(MAX(CAST(REGEXP_REPLACE(invoice_number, '^INV-\d+-0*', '') AS bigint)), 0) + 1000 INTO max_seq
        FROM sales WHERE invoice_number ~ '^INV-\d+-\d+$';
        IF max_seq > 1 THEN PERFORM setval('invoice_seq', max_seq); END IF;
    END IF;
END $$;

-- ============================================================
-- Core tables
-- ============================================================

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role_id INTEGER NOT NULL,
    store_id INTEGER,
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stores (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    address TEXT,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    slug VARCHAR(120) UNIQUE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT categories_name_key UNIQUE (name)
);

-- Backfill slug for existing categories without one
UPDATE categories SET slug = LOWER(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(TRIM(name), ' ', '-'), '''', ''), '"', ''), '&', 'and'), '/', '-'))
WHERE slug IS NULL;

CREATE TABLE IF NOT EXISTS brands (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tax_classes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    rate_percent DECIMAL(5,2) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS units_of_measure (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS warehouses (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) UNIQUE NOT NULL,
    address TEXT,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    barcode VARCHAR(50),
    category_id INTEGER REFERENCES categories(id) ON DELETE RESTRICT,
    brand_id INTEGER REFERENCES brands(id) ON DELETE SET NULL,
    description TEXT,
    price INTEGER NOT NULL CHECK (price >= 0),
    cost INTEGER DEFAULT 0 CHECK (cost >= 0),
    tax_class_id INTEGER REFERENCES tax_classes(id) ON DELETE SET NULL,
    weight_grams INTEGER,
    unit_of_measure_id INTEGER REFERENCES units_of_measure(id) ON DELETE SET NULL,
    default_discount_percent DECIMAL(5,2) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    search_vector tsvector,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_product_status CHECK (status IN ('draft', 'active', 'inactive', 'discontinued', 'archived'))
);

-- Partial unique index: allow soft-deleted products to share barcodes
DROP INDEX IF EXISTS idx_products_unique_active_barcode;
CREATE UNIQUE INDEX idx_products_unique_active_barcode ON products(barcode) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_stock (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id INTEGER REFERENCES warehouses(id) ON DELETE SET NULL,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reorder_point INTEGER NOT NULL DEFAULT 0 CHECK (reorder_point >= 0),
    reorder_quantity INTEGER NOT NULL DEFAULT 0 CHECK (reorder_quantity >= 0),
    last_restocked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE product_stock ADD CONSTRAINT uq_product_stock UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id);

CREATE TABLE IF NOT EXISTS inventory_movements (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity_change INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL,
    reference_id INTEGER,
    reference_table VARCHAR(50),
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customer_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    color VARCHAR(7),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    phone VARCHAR(20) UNIQUE NOT NULL,
    email VARCHAR(100) NOT NULL,
    address TEXT,
    tax_id VARCHAR(50),
    loyalty_points INTEGER NOT NULL DEFAULT 0,
    total_spent INTEGER NOT NULL DEFAULT 0,
    last_purchase_at TIMESTAMPTZ,
    note TEXT,
    is_active BOOLEAN DEFAULT true,
    is_walk_in BOOLEAN DEFAULT false,
    store_id INTEGER NOT NULL DEFAULT 1,
    customer_group_id INTEGER REFERENCES customer_groups(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_methods (
    id SERIAL PRIMARY KEY,
    code VARCHAR(30) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    requires_reference BOOLEAN DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS shifts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    opening_balance INTEGER NOT NULL DEFAULT 0,
    closing_balance INTEGER,
    cash_sales INTEGER NOT NULL DEFAULT 0,
    non_cash_sales INTEGER NOT NULL DEFAULT 0,
    total_sales INTEGER NOT NULL DEFAULT 0,
    transaction_count INTEGER NOT NULL DEFAULT 0,
    discrepancy INTEGER,
    notes TEXT,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    needs_review BOOLEAN NOT NULL DEFAULT false,
    reviewed_by INTEGER REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_shift_status CHECK (status IN ('open', 'closed'))
);

CREATE TABLE IF NOT EXISTS pricing_rules (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    pricing_type VARCHAR(50) NOT NULL,
    name VARCHAR(200),
    minimum_quantity INTEGER NOT NULL DEFAULT 1 CHECK (minimum_quantity >= 1),
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    effective_from TIMESTAMPTZ,
    effective_until TIMESTAMPTZ,
    pricing_method VARCHAR(20) NOT NULL DEFAULT 'fixed_price',
    pricing_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    category_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
    brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
    maximum_quantity INTEGER,
    customer_group_id INTEGER REFERENCES customer_groups(id) ON DELETE SET NULL,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    recurrence_days TEXT[],
    time_from TIME,
    time_to TIME,
    allow_combine BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'approved',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_pricing_target CHECK (product_id IS NOT NULL OR category_id IS NOT NULL OR brand_id IS NOT NULL),
    CONSTRAINT chk_pricing_type CHECK (pricing_type IN ('special_price', 'promotion')),
    CONSTRAINT chk_pricing_status CHECK (status IN ('draft', 'pending', 'approved', 'rejected'))
);

-- Name uniqueness: must be added after table creation to avoid ordering issues
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'pricing_rules_name_unique') THEN
        ALTER TABLE pricing_rules ADD CONSTRAINT pricing_rules_name_unique UNIQUE (name);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS sales (
    id SERIAL PRIMARY KEY,
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    cashier_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    customer_id INTEGER DEFAULT 1 REFERENCES customers(id),
    shift_id INTEGER REFERENCES shifts(id),
    subtotal INTEGER NOT NULL DEFAULT 0,
    discount INTEGER DEFAULT 0,
    tax INTEGER DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    payment_method VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'completed',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sale_items (
    id SERIAL PRIMARY KEY,
    sale_id INTEGER NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
    subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
    dpp_amount INTEGER NOT NULL DEFAULT 0,
    tax_amount INTEGER NOT NULL DEFAULT 0,
    pricing_rule_id INTEGER REFERENCES pricing_rules(id) ON DELETE SET NULL,
    pricing_rule_name VARCHAR(200),
    pricing_rule_type VARCHAR(50),
    pricing_type VARCHAR(50),
    original_price INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    contact_name VARCHAR(200),
    phone VARCHAR(50),
    email VARCHAR(200),
    address TEXT,
    notes TEXT,
    is_active BOOLEAN DEFAULT true,
    store_id INTEGER REFERENCES stores(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS product_suppliers (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    supplier_id INTEGER NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    supplier_sku VARCHAR(50),
    unit_cost INTEGER DEFAULT 0 CHECK (unit_cost >= 0),
    lead_time_days INTEGER DEFAULT 0,
    is_preferred BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(product_id, supplier_id)
);

CREATE TABLE IF NOT EXISTS import_jobs (
    id BIGSERIAL PRIMARY KEY,
    module VARCHAR(50) NOT NULL,
    schema_version VARCHAR(20) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    total_rows INT NOT NULL DEFAULT 0,
    inserted INT NOT NULL DEFAULT 0,
    updated INT NOT NULL DEFAULT 0,
    skipped INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    progress_pct INT NOT NULL DEFAULT 0,
    error_report_path TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INT,
    user_id INT NOT NULL REFERENCES users(id),
    store_id INT REFERENCES stores(id),
    cancel_requested BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS import_snapshots (
    id BIGSERIAL PRIMARY KEY,
    import_job_id BIGINT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    rows_data JSONB NOT NULL,
    schema_snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS import_rows (
    id BIGSERIAL PRIMARY KEY,
    import_job_id BIGINT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number INT NOT NULL,
    status VARCHAR(20) NOT NULL,
    entity_id INT,
    old_values JSONB,
    new_values JSONB,
    changed_fields TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS import_errors (
    id BIGSERIAL PRIMARY KEY,
    import_job_id BIGINT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number INT NOT NULL,
    field VARCHAR(100),
    value TEXT,
    reason TEXT NOT NULL,
    suggestion TEXT,
    stage VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    role VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100),
    entity_id INTEGER,
    old_values JSONB,
    new_values JSONB,
    description TEXT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dead_letter_events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload JSONB,
    error TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes
-- ============================================================

-- Users
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role_id);
CREATE INDEX IF NOT EXISTS idx_users_store ON users(store_id);

-- Products
CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_store ON products(store_id);
CREATE INDEX IF NOT EXISTS idx_products_brand ON products(brand_id);
CREATE INDEX IF NOT EXISTS idx_products_tax_class ON products(tax_class_id);
CREATE INDEX IF NOT EXISTS idx_products_uom ON products(unit_of_measure_id);
CREATE INDEX IF NOT EXISTS idx_products_search_vector ON products USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_category_active ON products(category_id) WHERE deleted_at IS NULL;

-- Sales
CREATE INDEX IF NOT EXISTS idx_sales_cashier ON sales(cashier_id);
CREATE INDEX IF NOT EXISTS idx_sales_store ON sales(store_id);
CREATE INDEX IF NOT EXISTS idx_sales_created ON sales(created_at);
CREATE INDEX IF NOT EXISTS idx_sales_shift_id ON sales(shift_id);
CREATE INDEX IF NOT EXISTS idx_sales_invoice_number_trgm ON sales USING GIN (invoice_number gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_sales_status_created_store ON sales (status, created_at, store_id) INCLUDE (total_amount);

-- Sale Items
CREATE INDEX IF NOT EXISTS idx_sale_items_sale ON sale_items(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_product ON sale_items(product_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items (sale_id) INCLUDE (product_id, quantity, unit_price, subtotal);
CREATE INDEX IF NOT EXISTS idx_sale_items_pricing_type ON sale_items(pricing_type);

-- Inventory movements
CREATE INDEX IF NOT EXISTS idx_inventory_movements_product ON inventory_movements(product_id);

-- Audit logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_ip_created ON audit_logs(action, ip_address, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);

-- Categories
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

-- Product stock
CREATE INDEX IF NOT EXISTS idx_product_stock_product_id ON product_stock(product_id);
CREATE INDEX IF NOT EXISTS idx_product_stock_warehouse_id ON product_stock(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_product_stock_store_id ON product_stock(store_id);

-- Payment methods
CREATE INDEX IF NOT EXISTS idx_payment_methods_code ON payment_methods(code);
CREATE INDEX IF NOT EXISTS idx_payment_methods_is_active ON payment_methods(is_active);

-- Customers
CREATE INDEX IF NOT EXISTS idx_customers_phone ON customers(phone);
CREATE INDEX IF NOT EXISTS idx_customers_is_active ON customers(is_active);
CREATE INDEX IF NOT EXISTS idx_customers_store_id ON customers(store_id);
CREATE INDEX IF NOT EXISTS idx_customers_name_trgm ON customers USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_customers_customer_group ON customers(customer_group_id);

-- Customer groups
CREATE INDEX IF NOT EXISTS idx_customer_groups_name ON customer_groups(name);
CREATE INDEX IF NOT EXISTS idx_customer_groups_active ON customer_groups(is_active) WHERE is_active = true;

-- Pricing rules
CREATE INDEX IF NOT EXISTS idx_pricing_rules_product_id ON pricing_rules(product_id);
CREATE INDEX IF NOT EXISTS idx_pricing_rules_active ON pricing_rules(product_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_type ON pricing_rules(pricing_type);
CREATE INDEX IF NOT EXISTS idx_pricing_rules_effective ON pricing_rules(effective_from, effective_until) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_category ON pricing_rules(category_id) WHERE category_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_brand ON pricing_rules(brand_id) WHERE brand_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_store ON pricing_rules(store_id) WHERE store_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_group ON pricing_rules(customer_group_id) WHERE customer_group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pricing_rules_method ON pricing_rules(pricing_method);
CREATE INDEX IF NOT EXISTS idx_pricing_rules_status ON pricing_rules(status);

-- Suppliers
CREATE INDEX IF NOT EXISTS idx_suppliers_code ON suppliers(code);

-- Product suppliers
CREATE INDEX IF NOT EXISTS idx_product_suppliers_product ON product_suppliers(product_id);
CREATE INDEX IF NOT EXISTS idx_product_suppliers_supplier ON product_suppliers(supplier_id);
DROP INDEX IF EXISTS idx_product_suppliers_one_preferred;
CREATE UNIQUE INDEX idx_product_suppliers_one_preferred ON product_suppliers(product_id) WHERE is_preferred = true;

-- Shifts
CREATE INDEX IF NOT EXISTS idx_shifts_user_id ON shifts(user_id);
CREATE INDEX IF NOT EXISTS idx_shifts_store_id ON shifts(store_id);
CREATE INDEX IF NOT EXISTS idx_shifts_status ON shifts(status);
CREATE INDEX IF NOT EXISTS idx_shifts_opened_at ON shifts(opened_at);

-- Import
CREATE INDEX IF NOT EXISTS idx_import_jobs_module ON import_jobs(module);
CREATE INDEX IF NOT EXISTS idx_import_jobs_user ON import_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);
CREATE INDEX IF NOT EXISTS idx_import_rows_job ON import_rows(import_job_id);
CREATE INDEX IF NOT EXISTS idx_import_errors_job ON import_errors(import_job_id);

-- Sales aggregation
CREATE INDEX IF NOT EXISTS idx_sales_aggregation
    ON sales (store_id, created_at DESC, total_amount)
    INCLUDE (id, invoice_number, cashier_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_active_aggregations
    ON sales (created_at DESC)
    WHERE status = 'completed';

-- Dead letter events
CREATE INDEX IF NOT EXISTS idx_dead_letter_events_created_at ON dead_letter_events (created_at);
CREATE INDEX IF NOT EXISTS idx_dead_letter_events_event_type ON dead_letter_events (event_type);

-- ============================================================
-- Function: products_search_vector_update (trigger)
-- ============================================================
CREATE OR REPLACE FUNCTION products_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.sku, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.barcode, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_products_search_vector ON products;
CREATE TRIGGER trg_products_search_vector
    BEFORE INSERT OR UPDATE OF name, sku, barcode ON products
    FOR EACH ROW
    EXECUTE FUNCTION products_search_vector_update();

-- Backfill search_vector for existing rows
UPDATE products SET search_vector =
    setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(sku, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(barcode, '')), 'C')
WHERE search_vector IS NULL;

-- ============================================================
-- View: v_products_full
-- ============================================================
DROP VIEW IF EXISTS v_products_full;
CREATE VIEW v_products_full AS
SELECT
    p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name,
    p.price, p.cost, COALESCE(ps.quantity, 0) as stock,
    p.status, p.store_id, p.brand_id, b.name as brand_name,
    p.unit_of_measure_id, u.name as unit_of_measure,
    p.weight_grams, p.description,
    p.tax_class_id, tc.rate_percent as tax_rate,
    p.search_vector,
    p.created_at, p.updated_at
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN brands b ON p.brand_id = b.id
LEFT JOIN units_of_measure u ON p.unit_of_measure_id = u.id
LEFT JOIN LATERAL (
    SELECT quantity FROM product_stock
    WHERE product_id = p.id
    ORDER BY (warehouse_id IS NULL AND store_id IS NULL) DESC
    LIMIT 1
) ps ON true
LEFT JOIN tax_classes tc ON tc.id = p.tax_class_id
WHERE p.deleted_at IS NULL;

-- ============================================================
-- Seed data
-- ============================================================

-- Roles
INSERT INTO roles (name, description, is_system) VALUES
    ('superadmin', 'Super Administrator', true),
    ('admin', 'Administrator', true),
    ('manager', 'Manager / Kepala Toko', true),
    ('cashier', 'Kasir', true),
    ('staff', 'Staff Gudang', true)
ON CONFLICT (name) DO NOTHING;

-- Payment methods
INSERT INTO payment_methods (code, name, is_active, requires_reference, sort_order) VALUES
    ('CASH', 'Cash', true, false, 1),
    ('CARD', 'Card', true, true, 2),
    ('E_WALLET', 'E-Wallet', true, true, 3),
    ('TRANSFER', 'Transfer', true, true, 4),
    ('QRIS', 'QRIS', true, false, 5)
ON CONFLICT (code) DO NOTHING;

-- Customer groups (seed defaults)
INSERT INTO customer_groups (name, description, is_active, color) VALUES
    ('Walk-in', 'Pelanggan umum tanpa kartu member', true, '#636E72'),
    ('Member', 'Pelanggan terdaftar dengan kartu member', true, '#00B894'),
    ('VIP', 'Pelanggan prioritas dengan harga khusus', true, '#6C5CE7')
ON CONFLICT (name) DO NOTHING;

-- Walk-in customer
INSERT INTO customers (id, name, phone, email, is_walk_in, is_active, store_id, customer_group_id)
VALUES (1, 'Pelanggan Umum / Walk-in', '0000000000', 'walk-in@retail-pos.local', true, true, 1, NULL)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Permissions
-- ============================================================

INSERT INTO permissions (code, name, description) VALUES
    ('dashboard.view', 'Lihat Dashboard', 'Bisa melihat dashboard utama'),
    ('user.view', 'Lihat Pengguna', 'Bisa melihat daftar pengguna'),
    ('user.create', 'Tambah Pengguna', 'Bisa menambah pengguna baru'),
    ('user.update', 'Edit Pengguna', 'Bisa mengubah data pengguna'),
    ('user.delete', 'Hapus Pengguna', 'Bisa menghapus pengguna'),
    ('role.view', 'Lihat Role', 'Bisa melihat daftar role'),
    ('role.create', 'Tambah Role', 'Bisa menambah role baru'),
    ('role.update', 'Edit Role', 'Bisa mengubah role'),
    ('role.delete', 'Hapus Role', 'Bisa menghapus role'),
    ('product.view', 'Lihat Produk', 'Bisa melihat daftar produk'),
    ('product.create', 'Tambah Produk', 'Bisa menambah produk baru'),
    ('product.update', 'Edit Produk', 'Bisa mengubah data produk'),
    ('product.delete', 'Hapus Produk', 'Bisa menghapus produk'),
    ('category.view', 'Lihat Kategori', 'Bisa melihat kategori'),
    ('category.create', 'Tambah Kategori', 'Bisa menambah kategori'),
    ('category.update', 'Edit Kategori', 'Bisa mengubah kategori'),
    ('category.delete', 'Hapus Kategori', 'Bisa menghapus kategori'),
    ('sale.view', 'Lihat Penjualan', 'Bisa melihat daftar penjualan'),
    ('sale.create', 'Buat Penjualan', 'Bisa membuat transaksi penjualan'),
    ('sale.print', 'Cetak Struk', 'Bisa mencetak struk penjualan'),
    ('sale.void', 'Void penjualan', 'Void/refund transaksi penjualan'),
    ('sale.park', 'Parkir Penjualan', 'Bisa menyimpan sementara, mengambil kembali, dan membatalkan penjualan yang diparkir'),
    ('shift.view', 'Baca shift', 'Lihat daftar dan detail shift'),
    ('shift.create', 'Kelola shift', 'Buka dan tutup shift'),
    ('shift.review', 'Review shift', 'Review dan setujui selisih shift'),
    ('shift.audit', 'Audit shift', 'Audit fisik cash shift'),
    ('report.view', 'Lihat Laporan', 'Bisa melihat laporan keuangan & stok'),
    ('audit.view', 'Lihat Log Audit', 'Bisa melihat riwayat log audit'),
    ('product.export', 'Export Produk', 'Bisa mengexport data produk'),
    ('product.import', 'Import Produk', 'Bisa mengimport data produk'),
    ('category.export', 'Export Kategori', 'Bisa mengexport data kategori'),
    ('category.import', 'Import Kategori', 'Bisa mengimport data kategori'),
    ('customer.export', 'Export Pelanggan', 'Bisa mengexport data pelanggan'),
    ('customer.import', 'Import Pelanggan', 'Bisa mengimport data pelanggan'),
    ('pricing.view', 'Lihat Aturan Harga', 'Bisa melihat daftar aturan harga'),
    ('pricing.create', 'Tambah Aturan Harga', 'Bisa menambah aturan harga baru'),
    ('pricing.update', 'Edit Aturan Harga', 'Bisa mengubah aturan harga'),
    ('pricing.delete', 'Hapus Aturan Harga', 'Bisa menghapus aturan harga'),
    ('supplier_cost.view', 'Lihat Harga Beli Supplier', 'Bisa melihat unit_cost supplier pada produk dan tautan supplier'),
    ('supplier_cost.update', 'Edit Harga Beli Supplier', 'Bisa mengubah unit_cost saat menautkan produk ke supplier'),
    ('store.view', 'Lihat Data Toko', NULL),
    ('store.create', 'Buat Data Toko', NULL),
    ('store.update', 'Edit Data Toko', NULL),
    ('store.delete', 'Hapus Data Toko', NULL),
    ('customer_group.view', 'Lihat Data Customer Group', NULL),
    ('customer_group.create', 'Buat Data Customer Group', NULL),
    ('customer_group.update', 'Edit Data Customer Group', NULL),
    ('customer_group.delete', 'Hapus Data Customer Group', NULL)
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- Role-Permission assignments
-- ============================================================

-- Superadmin: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.name = 'superadmin'
ON CONFLICT DO NOTHING;

-- Admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'
    AND p.code IN (
        'dashboard.view','user.view','user.create','user.update',
        'role.view','role.create','role.update',
        'product.view','product.create','product.update','product.delete',
        'category.view','category.create','category.update','category.delete',
        'sale.view','sale.create','sale.print','sale.void','sale.park',
        'shift.view','shift.create','shift.review','shift.audit',
        'report.view',
        'product.export','product.import',
        'category.export','category.import',
        'customer.export','customer.import',
        'pricing.view','pricing.create','pricing.update','pricing.delete',
        'supplier_cost.view','supplier_cost.update',
        'store.view','store.create','store.update','store.delete',
        'customer_group.view','customer_group.create','customer_group.update','customer_group.delete'
    )
ON CONFLICT DO NOTHING;

-- Manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'manager'
    AND p.code IN (
        'dashboard.view',
        'product.view','product.update',
        'category.view','category.create',
        'sale.view','sale.create','sale.print','sale.park',
        'shift.view','shift.create','shift.review','shift.audit',
        'report.view',
        'pricing.view','pricing.create','pricing.update',
        'supplier_cost.view',
        'store.view',
        'customer_group.view'
    )
ON CONFLICT DO NOTHING;

-- Cashier
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'cashier'
    AND p.code IN (
        'dashboard.view','product.view',
        'sale.view','sale.create','sale.print','sale.park',
        'shift.view','shift.create',
        'pricing.view',
        'store.view',
        'customer_group.view'
    )
ON CONFLICT DO NOTHING;

-- Staff: dashboard, product view/update, shift.view
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'staff'
    AND p.code IN ('dashboard.view','product.view','product.update','shift.view')
ON CONFLICT DO NOTHING;

-- ============================================================
-- Default users (password: admin123)
-- ============================================================
INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'superadmin', 'superadmin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'superadmin'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'admin', 'admin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'admin'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'manager', 'manager@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'manager'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'cashier', 'cashier@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'cashier'
ON CONFLICT (username) DO NOTHING;

-- ============================================================
-- Comments
-- ============================================================
COMMENT ON TABLE users IS 'Sistem pengguna (kasir, admin, manager)';
COMMENT ON TABLE roles IS 'Role pengguna (admin, cashier, dll)';
COMMENT ON TABLE permissions IS 'Permission berbasis kode (product.create, dll)';
COMMENT ON TABLE products IS 'Master produk';
COMMENT ON TABLE sales IS 'Transaksi penjualan';
COMMENT ON TABLE audit_logs IS 'Log audit untuk keamanan';
COMMENT ON TABLE shifts IS 'Cashier shift management with opening/closing balances';

COMMIT;
