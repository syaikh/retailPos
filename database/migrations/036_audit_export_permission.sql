-- P2 audit hardening (#12): a dedicated, independently-grantable permission for
-- exporting the full audit trail. Exporting every audit row is more sensitive
-- than merely viewing the list, so it gets its own code rather than riding on
-- audit.view. Granted to superusers (superadmin, admin); managers/cashiers are
-- not granted it by default, keeping export more restrictive than view.
INSERT INTO permissions (code, name, description)
VALUES (
	'audit.export',
	'Ekspor Log Audit',
	'Mengekspor seluruh log audit ke berkas (CSV/XLSX). Lebih sensitif daripada sekadar melihat daftar.'
)
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('superadmin', 'admin')
  AND p.code = 'audit.export'
ON CONFLICT DO NOTHING;
