-- Seed: Default users
-- Password for all: 'admin123' (bcrypt hash)
-- Hash generated with cost=10: $2a$10$YOUR_HASH_HERE
-- Replace with actual hash in production

INSERT INTO users (id, username, email, password_hash, role_id, store_id, is_active, created_at)
VALUES
(1, 'superadmin', 'superadmin@retailpos.local', '$2a$10$cV3/py5RCcZT.B8Hh1HTeO.07eOjtQXYqm6G.KQ/YOTHfKJouRznq', 1, NULL, true, NOW()),
(2, 'admin', 'admin@retailpos.local', '$2a$10$cV3/py5RCcZT.B8Hh1HTeO.07eOjtQXYqm6G.KQ/YOTHfKJouRznq', 2, NULL, true, NOW()),
(3, 'manager', 'manager@retailpos.local', '$2a$10$cV3/py5RCcZT.B8Hh1HTeO.07eOjtQXYqm6G.KQ/YOTHfKJouRznq', 3, NULL, true, NOW()),
(4, 'cashier', 'cashier@retailpos.local', '$2a$10$cV3/py5RCcZT.B8Hh1HTeO.07eOjtQXYqm6G.KQ/YOTHfKJouRznq', 4, NULL, true, NOW())
ON CONFLICT (id) DO NOTHING;
