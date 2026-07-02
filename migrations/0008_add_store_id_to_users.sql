-- +migrate Up
ALTER TABLE users ADD COLUMN store_id INTEGER;

-- Optional: Set a default store (e.g., 1) for existing users
-- UPDATE users SET store_id = 1 WHERE store_id IS NULL;

-- +migrate Down
ALTER TABLE users DROP COLUMN store_id;
