-- Add product_name column to sale_items for snapshotting
ALTER TABLE sale_items ADD COLUMN product_name TEXT;

-- Move data from products table to sale_items for existing records
UPDATE sale_items si 
SET product_name = p.name 
FROM products p 
WHERE si.product_name IS NULL AND si.product_id = p.id;
