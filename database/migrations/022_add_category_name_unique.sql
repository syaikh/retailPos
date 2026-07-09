-- Add unique constraint on category name for import/export upsert support
-- The BulkUpsertCategories function uses ON CONFLICT (name) which requires a unique constraint
ALTER TABLE categories ADD CONSTRAINT categories_name_key UNIQUE (name);

-- Add unique constraint on product_stock.product_id for import/export upsert support
-- The BulkInsertProducts function uses ON CONFLICT (product_id) which requires a unique constraint
ALTER TABLE product_stock ADD CONSTRAINT product_stock_product_id_key UNIQUE (product_id);
