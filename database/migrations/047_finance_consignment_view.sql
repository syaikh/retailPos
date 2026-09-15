-- Migration: 047_finance_consignment_view.sql
-- Description: Grant consignment.view to the finance role.
--
-- The finance role (created in 044) holds consignment.pay so it can record
-- supplier settlements, but the whole consignment module — frontend route
-- `/consignment` and the settlement list/detail endpoints — is gated on
-- consignment.view, so finance could not reach the pay flow at all
-- ("supervisor settles → finance pays" separation of duties, per
-- docs/design/store-first-and-finance-role.md).
--
-- Granting ONLY consignment.view keeps the separation intact: finance still
-- lacks consignment.create/update/settle, so it can view settlements and pay
-- them, but cannot originate or settle them.
--
-- Idempotent: ON CONFLICT DO NOTHING; safe to re-run.

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'finance' AND p.code = 'consignment.view'
ON CONFLICT DO NOTHING;