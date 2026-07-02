-- +migrate Up
ALTER TABLE products ADD COLUMN store_id INTEGER;

CREATE INDEX idx_products_store_id ON products(store_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_products_store_id;
ALTER TABLE products DROP COLUMN store_id;