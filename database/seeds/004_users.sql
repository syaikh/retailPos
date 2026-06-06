-- Seed: Default users
-- Password for all: 'admin123' (bcrypt hash, cost=10)
-- Generated with: bcrypt.GenerateFromPassword([]byte("admin123"), 10)

INSERT INTO users (id, username, email, password_hash, role_id, store_id, is_active, created_at)
VALUES
(1, 'superadmin', 'superadmin@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 1, NULL, true, NOW()),
(2, 'admin', 'admin@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 2, NULL, true, NOW()),
(3, 'manager', 'manager@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 3, NULL, true, NOW()),
(4, 'cashier', 'cashier@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 4, NULL, true, NOW()),
(5, 'staff', 'staff@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 5, NULL, true, NOW())
ON CONFLICT (id) DO NOTHING;
