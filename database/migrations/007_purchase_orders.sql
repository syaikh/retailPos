BEGIN;

-- Sequences for PO and GR numbering
CREATE SEQUENCE IF NOT EXISTS po_seq START 1;
CREATE SEQUENCE IF NOT EXISTS gr_seq START 1;

-- Main PO table
CREATE TABLE purchase_orders (
    id SERIAL PRIMARY KEY,
    po_number VARCHAR(30) NOT NULL UNIQUE,
    supplier_id INT NOT NULL REFERENCES suppliers(id),
    store_id INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    expected_date DATE,
    payment_term VARCHAR(50),
    delivery_address TEXT,
    supplier_reference_number VARCHAR(100),
    notes TEXT,
    confirmed_at TIMESTAMPTZ,
    confirmed_by INT,
    cancelled_at TIMESTAMPTZ,
    cancelled_by INT,
    created_by INT NOT NULL,
    updated_by INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Financial columns for PO header
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS subtotal INT NOT NULL DEFAULT 0;
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS discount_amount INT NOT NULL DEFAULT 0;
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS tax_amount INT NOT NULL DEFAULT 0;
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS grand_total INT NOT NULL DEFAULT 0;

-- Extension point columns (nullable, no business logic yet)
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS approval_status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS payment_status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS invoice_status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS warehouse_id INT;
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS currency_code VARCHAR(3) DEFAULT 'IDR';
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS exchange_rate INT DEFAULT 1;
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS approved_by INT;
ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;

-- PO line items
CREATE TABLE purchase_order_items (
    id SERIAL PRIMARY KEY,
    purchase_order_id INT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id),
    qty_ordered INT NOT NULL CHECK (qty_ordered > 0),
    qty_received INT NOT NULL DEFAULT 0 CHECK (qty_received >= 0),
    unit_cost INT NOT NULL CHECK (unit_cost >= 0),
    discount_amount INT NOT NULL DEFAULT 0,
    subtotal INT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(purchase_order_id, product_id)
);

-- Snapshot columns for PO items
ALTER TABLE purchase_order_items ADD COLUMN IF NOT EXISTS product_name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE purchase_order_items ADD COLUMN IF NOT EXISTS sku VARCHAR(100) DEFAULT '';
ALTER TABLE purchase_order_items ADD COLUMN IF NOT EXISTS barcode VARCHAR(100) DEFAULT '';
ALTER TABLE purchase_order_items ADD COLUMN IF NOT EXISTS uom_id INT;
ALTER TABLE purchase_order_items ADD COLUMN IF NOT EXISTS uom_name VARCHAR(50) DEFAULT 'pcs';

-- Goods Receipt header
CREATE TABLE goods_receipts (
    id SERIAL PRIMARY KEY,
    gr_number VARCHAR(30) NOT NULL UNIQUE,
    purchase_order_id INT NOT NULL REFERENCES purchase_orders(id),
    store_id INT NOT NULL,
    received_by INT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivery_order_number VARCHAR(100),
    shipping_method VARCHAR(50),
    driver_name VARCHAR(100),
    vehicle_plate_number VARCHAR(20),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Goods Receipt line items
CREATE TABLE goods_receipt_items (
    id SERIAL PRIMARY KEY,
    goods_receipt_id INT NOT NULL REFERENCES goods_receipts(id) ON DELETE CASCADE,
    purchase_order_item_id INT NOT NULL REFERENCES purchase_order_items(id),
    product_id INT NOT NULL REFERENCES products(id),
    qty_good INT NOT NULL CHECK (qty_good >= 0),
    qty_damaged INT NOT NULL DEFAULT 0 CHECK (qty_damaged >= 0),
    unit_cost INT NOT NULL DEFAULT 0,
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    supplier_id INT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_purchase_orders_status ON purchase_orders(status);
CREATE INDEX idx_purchase_orders_supplier ON purchase_orders(supplier_id);
CREATE INDEX idx_purchase_orders_store ON purchase_orders(store_id);
CREATE INDEX idx_purchase_orders_status_store ON purchase_orders(status, store_id);
CREATE INDEX idx_purchase_orders_created_at ON purchase_orders(created_at DESC);
CREATE INDEX idx_purchase_order_items_po ON purchase_order_items(purchase_order_id);
CREATE INDEX idx_goods_receipts_po ON goods_receipts(purchase_order_id);
CREATE INDEX idx_goods_receipt_items_gr ON goods_receipt_items(goods_receipt_id);

-- Purchase order permissions
INSERT INTO permissions (code, name, description) VALUES
  ('purchase_order.view', 'Lihat Purchase Order', 'Bisa melihat daftar dan detail purchase order'),
  ('purchase_order.create', 'Buat Purchase Order', 'Bisa membuat purchase order baru'),
  ('purchase_order.update', 'Edit Purchase Order', 'Bisa mengubah purchase order'),
  ('purchase_order.delete', 'Hapus Purchase Order', 'Bisa menghapus purchase order'),
  ('purchase_order.confirm', 'Konfirmasi Purchase Order', 'Bisa mengkonfirmasi purchase order'),
  ('purchase_order.receive', 'Terima Barang', 'Bisa membuat goods receipt untuk purchase order')
ON CONFLICT (code) DO NOTHING;

-- Grant purchase order permissions to superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'superadmin'
  AND p.code IN ('purchase_order.view', 'purchase_order.create', 'purchase_order.update', 'purchase_order.delete', 'purchase_order.confirm', 'purchase_order.receive')
ON CONFLICT DO NOTHING;

-- Grant purchase order permissions to admin and manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
  AND p.code IN ('purchase_order.view', 'purchase_order.create', 'purchase_order.update', 'purchase_order.confirm', 'purchase_order.receive')
ON CONFLICT DO NOTHING;

-- Grant purchase order:read to cashier (view only)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier'
  AND p.code = 'purchase_order.view'
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations (filename) VALUES ('007_purchase_orders.sql');

COMMIT;
