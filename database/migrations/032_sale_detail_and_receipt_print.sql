-- Phase 2 (Option B) of the Find Transaction cross-cashier feature:
--   * sale.detail  — lets a cashier drill into a co-worker's COMPLETED transaction
--     and see its itemized lines (for receipt reprint). Redacted (no cost/margin,
--     no payment reference, no customer PII) via /sales/lookup/:id.
--   * receipt.print — lets a cashier reprint another cashier's receipt from the
--     Find Transaction drawer.
-- Both are granted to cashier (primary user), and to manager/admin/superadmin so
-- the permission set stays consistent across roles (manager is a superset of
-- cashier; admin/superadmin are superusers). Managers/admin/superadmin do not see
-- the Find Transaction tab in the UI (they reach all cashiers' sales via report.view),
-- but holding the codes keeps the hierarchy coherent and lets the endpoints be
-- exercised directly.

INSERT INTO permissions (code, name, description)
VALUES
  ('sale.detail', 'Cari Detail Penjualan Lintas Kasir', 'Melihat rincian item transaksi kasir lain (tanpa harga pokok/margin) untuk cetak ulang struk.'),
  ('receipt.print', 'Cetak Struk', 'Mencetak ulang struk penjualan, termasuk transaksi kasir lain.')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('cashier', 'manager', 'admin', 'superadmin')
  AND p.code IN ('sale.detail', 'receipt.print')
ON CONFLICT DO NOTHING;
