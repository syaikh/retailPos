-- Migration: 005_product_extensions.sql
-- Description: Add brand, tax, unit of measure, and warehouse support for retail product form
-- Created: 2026-05-10

-- SKUs
CREATE SEQUENCE IF NOT EXISTS sku_seq START 1 INCREMENT 1;

-- Brands table
CREATE TABLE IF NOT EXISTS brands (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tax classes table
CREATE TABLE IF NOT EXISTS tax_classes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    rate_percent DECIMAL(5,2) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Units of measure table
CREATE TABLE IF NOT EXISTS units_of_measure (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Warehouses table (for multi-location inventory)
CREATE TABLE IF NOT EXISTS warehouses (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) UNIQUE NOT NULL,
    address TEXT,
    store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Extend products table with new fields
ALTER TABLE products 
    ADD COLUMN IF NOT EXISTS brand_id INTEGER REFERENCES brands(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS tax_class_id INTEGER REFERENCES tax_classes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS weight_grams INTEGER,
    ADD COLUMN IF NOT EXISTS dimensions_cm JSONB, -- {length: float, width: float, height: float}
    ADD COLUMN IF NOT EXISTS unit_of_measure_id INTEGER REFERENCES units_of_measure(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS default_discount_percent DECIMAL(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_track_expiry BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_track_batch BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_active_for_sale BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_active_for_purchase BOOLEAN DEFAULT true;

-- Set default values for existing products
UPDATE products SET 
    is_active_for_sale = true,
    is_active_for_purchase = true,
    default_discount_percent = 0,
    is_track_expiry = false,
    is_track_batch = false
WHERE is_active_for_sale IS NULL;

-- Add indexes for new fields
CREATE INDEX IF NOT EXISTS idx_products_brand ON products(brand_id);
CREATE INDEX IF NOT EXISTS idx_products_tax_class ON products(tax_class_id);
CREATE INDEX IF NOT EXISTS idx_products_uom ON products(unit_of_measure_id);

-- Index for dimensions JSONB (if querying by dimensions)
CREATE INDEX IF NOT EXISTS idx_products_dimensions ON products USING GIN (dimensions_cm) WHERE dimensions_cm IS NOT NULL;