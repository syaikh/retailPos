-- 029_drop_products_stock.sql
--
-- Removes the legacy products.stock column (D6A of the security remediation
-- plan). product_stock is the canonical stock source (v_products_full already
-- reads stock from product_stock via ps.quantity), and the products.stock
-- mirror was written by a now-deleted sync path, so no code reads it anymore.
ALTER TABLE products DROP COLUMN IF EXISTS stock;
