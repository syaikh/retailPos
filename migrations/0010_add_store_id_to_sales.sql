-- +migrate Up
ALTER TABLE sales ADD COLUMN store_id INTEGER;

CREATE INDEX idx_sales_store_id ON sales(store_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_sales_store_id;
ALTER TABLE sales DROP COLUMN store_id;