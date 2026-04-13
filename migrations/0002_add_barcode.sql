-- Add barcode column to products table
ALTER TABLE products ADD COLUMN barcode TEXT;

-- Add unique index for barcode
CREATE UNIQUE INDEX idx_products_barcode ON products(barcode);

-- Add index for search optimization
CREATE INDEX idx_products_barcode_search ON products(barcode);
