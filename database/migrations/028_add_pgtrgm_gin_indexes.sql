-- Migration 028: Add pg_trgm GIN indexes for text search + composite indexes for aggregation
-- Enables fast ILIKE/ LIKE/ similarity searches on product names, customer names,
-- and invoice numbers. Also adds composite indexes for dashboard/report aggregation queries.

-- Create pg_trgm extension if not exists
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Product name search (used in product listing and sale export search)
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING GIN (name gin_trgm_ops);

-- Customer name search (used in customer listing and sale export search)
CREATE INDEX IF NOT EXISTS idx_customers_name_trgm ON customers USING GIN (name gin_trgm_ops);

-- Invoice number search (used in sale export search)
CREATE INDEX IF NOT EXISTS idx_sales_invoice_number_trgm ON sales USING GIN (invoice_number gin_trgm_ops);

-- Composite index for sales aggregation (dashboard stats, period comparison, chart data)
-- Covers: status filter + created_at range + store_id filter + total_amount sum
CREATE INDEX IF NOT EXISTS idx_sales_status_created_store ON sales (status, created_at, store_id) INCLUDE (total_amount);

-- Composite index for sale_items aggregation (export item counts)
CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items (sale_id) INCLUDE (product_id, quantity, unit_price, subtotal);
