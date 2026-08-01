-- Seed: Shift permissions
INSERT INTO permissions (code, name, description) VALUES
('shift.view', 'Baca shift', 'Lihat daftar dan detail shift'),
('shift.create', 'Kelola shift', 'Buka dan tutup shift'),
('shift.review', 'Review shift', 'Review dan setujui selisih shift'),
('shift.audit', 'Audit shift', 'Audit fisik cash shift')
ON CONFLICT (code) DO NOTHING;

-- Superadmin: add shift permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions
WHERE code IN ('shift.view', 'shift.create', 'shift.review', 'shift.audit')
ON CONFLICT DO NOTHING;

-- Admin: add shift permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions
WHERE code IN ('shift.view', 'shift.create', 'shift.review', 'shift.audit')
ON CONFLICT DO NOTHING;

-- Manager: add shift permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions
WHERE code IN ('shift.view', 'shift.create', 'shift.review', 'shift.audit')
ON CONFLICT DO NOTHING;

-- Cashier: add shift permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions
WHERE code IN ('shift.view', 'shift.create')
ON CONFLICT DO NOTHING;
