BEGIN;

-- ============================================================================
-- Sprint 0 RBAC Final — Frontend permission-based authorization
-- ============================================================================
-- Source:      docs/audits/permission-matrix-final.md (Sprint 0 Target State)
--              docs/audits/rbac-sprint0-audit.md (implementasi & keputusan)
--
-- Ordering:    PALING AKHIR — apply SETELAH frontend Fase 1-5 (permission-based
--              UI) di-deploy. Binary lama yang masih memakai UI role-based akan
--              kehilangan tombol jika migration ini diaplikasikan lebih dulu.
--
-- Expected changes (tepat 2 REVOKE, TIDAK ada grant baru):
--   REVOKE  staff.product.update    -- R1 — dead grant: staff tidak pernah bisa
--                                   --      edit produk di UI (canEdit hanya
--                                   --      superadmin/admin/manager).
--   REVOKE  staff.inventory.adjust  -- R2 — least privilege: staff tetap bisa
--                                   --      menghitung stok via
--                                   --      stock_opname.count/submit; tombol
--                                   --      "Adjust Stock" manual disisakan
--                                   --      untuk SA/A/M.
--
-- Behavior delta (tercatat di permission-matrix-final.md §6):
--   D1  Staff kehilangan tombol "Adjust Stock" manual di ProductsPage/
--       ProductDetailDrawer (aplikasi menyesuaikan via rbac.can).
--   D2  (sudah diterapkan di Fase 3) tombol Add Product hanya untuk pemegang
--       product.create — memperbaiki bug 403 untuk manager/staff.
--   D3  Tidak ada regresi untuk staff: canEdit tetap hanya SA/A/M.
--
-- Idempotent: DELETE ... WHERE (role_id, permission_id) IN (SELECT ...) — aman
-- untuk DB yang sudah pernah di-seed, dijalankan berulang, atau sebagian
-- grant sudah hilang.
-- ============================================================================

DELETE FROM role_permissions
WHERE (role_id, permission_id) IN (
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'staff'
    AND p.code IN ('product.update', 'inventory.adjust')
);

INSERT INTO schema_migrations (filename) VALUES ('023_sprint0_finalize_permissions.sql');

COMMIT;
