-- Seed: Default users
-- Password for all: 'admin123' (bcrypt hash, cost=10)
-- Generated with: bcrypt.GenerateFromPassword([]byte("admin123"), 10)

-- Reports_to hierarchy: superadmin (1) <- admin (2) <- manager (3) <- cashier (4), staff (5)
INSERT INTO users (id, username, email, password_hash, role_id, store_id, reports_to, is_active, created_at)
VALUES
(1, 'superadmin', 'superadmin@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 1, NULL, NULL, true, NOW()),
(2, 'admin', 'admin@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 2, NULL, 1, true, NOW()),
(3, 'manager', 'manager@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 3, NULL, 2, true, NOW()),
(4, 'cashier', 'cashier@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 4, NULL, 3, true, NOW()),
(5, 'staff', 'staff@retailpos.local', '$2a$10$WZqxlkKl12W8tEERWjnXFuowrPHwheO2deI9IzAQJzaapK2UKIh4a', 5, NULL, 3, true, NOW())
ON CONFLICT (id) DO NOTHING;
