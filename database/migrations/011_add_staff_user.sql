-- 011: Add default staff user (role: staff, reports to manager)
-- Password: admin123
INSERT INTO users (username, email, password_hash, role_id, reports_to, is_active)
SELECT 'staff', 'staff@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id,
       (SELECT id FROM users WHERE username = 'manager'), true
FROM roles r
WHERE r.name = 'staff'
  AND NOT EXISTS (SELECT 1 FROM users WHERE username = 'staff' OR email = 'staff@retailpos.local')
ON CONFLICT (username) DO NOTHING;
