-- 007_sale_lookup.sql
-- Adds the cross-cashier transaction lookup capability.
--
-- Distinction (per design):
--   * sale.view  -> "My Transactions" — own sales only (existing behaviour).
--   * sale.lookup -> "Find Transaction" — search across all cashiers, but the
--     API returns a REDACTED summary (no items/cost, no customer PII, no
--     payment tender/reference). Required for cashiers to look up a
--     co-worker's transaction (returns/receipt reprints) without exposing
--     sensitive data or another cashier's full history.
--   * report.view -> full analytics (unchanged).

INSERT INTO permissions (code, name, description) VALUES
  ('sale.lookup', 'Cari Penjualan Lintas Kasir',
   'Cari transaksi kasir lain secara ringkas (tanpa detail item/biaya/pelanggan)')
ON CONFLICT (code) DO NOTHING;

-- Grant to cashier + manager by default (operational necessity).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('cashier', 'manager')
  AND p.code = 'sale.lookup'
ON CONFLICT (role_id, permission_id) DO NOTHING;
