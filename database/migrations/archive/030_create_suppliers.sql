-- Migration 030: Create suppliers and product_suppliers tables
-- Supplier domain for the Pricing Engine epic (ADR-003).

CREATE TABLE suppliers (
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

CREATE TABLE product_suppliers (
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

-- Partial unique index: enforce ONE preferred supplier per product (ADR-003, INV-S6)
CREATE UNIQUE INDEX idx_product_suppliers_one_preferred
    ON product_suppliers(product_id)
    WHERE is_preferred = true;

-- Lookup indexes
CREATE INDEX idx_product_suppliers_product ON product_suppliers(product_id);
CREATE INDEX idx_product_suppliers_supplier ON product_suppliers(supplier_id);
CREATE INDEX idx_suppliers_code ON suppliers(code);

-- ROLLBACK:
-- DROP INDEX IF EXISTS idx_suppliers_code;
-- DROP INDEX IF EXISTS idx_product_suppliers_supplier;
-- DROP INDEX IF EXISTS idx_product_suppliers_product;
-- DROP INDEX IF EXISTS idx_product_suppliers_one_preferred;
-- DROP TABLE IF EXISTS product_suppliers;
-- DROP TABLE IF EXISTS suppliers;
