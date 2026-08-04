# PERMISSION MATRIX FINAL — Sprint 0 Target State

**Status:** DRAFT (menunggu approval)
**Tanggal:** 2026-08-04
**Sumber kebenaran:** Live database `retail_pos` (`permissions`, `role_permissions`) — hasil query 2026-08-04. Definisi seed: `database/migrations/000_squash.sql:644-693` + migrasi `007`, `008`, `012`, `016`, `018`, `021` + `database/seeds/013_customer_permissions.sql`.

---

## 1. LANDASAN

- Total permission live di DB: **72** (0 ungranted).
- Permission `pos.access` dan `inventory.export` **TIDAK ada** di DB. Keduanya hanya didefinisikan di `database/seeds/002_permissions.sql` (legacy, tidak diaplikasikan). → koreksi atas audit v1 (74 → 72).
- `sale.print`, `sale.void`, `inventory.view`, `supplier_cost.view`, `supplier_cost.update` dihapus oleh `013_remove_dead_permissions.sql`.
- Total grants live: **193** (superadmin=72, admin=67, manager=36, cashier=11, staff=7).

## 2. PERUBAHAN MATRIKS (Migration 023)

Migration `023_sprint0_finalize_permissions.sql` berisi **tepat 2 revoke**, dieksekusi **PALING AKHIR** (setelah Fase 1–5 frontend selesai):

| # | Perubahan | Role | Permission | Alasan |
|---|-----------|------|------------|--------|
| R1 | REVOKE | staff | `product.update` | Staff tidak pernah bisa edit produk di UI (frontend `canEdit` hanya superadmin/admin/manager). Grant ini memberi hak yang tidak pernah terpakai (dead grant). Least privilege. |
| R2 | REVOKE | staff | `inventory.adjust` | Staff tetap bisa melakukan penghitungan stok via `stock_opname.count/submit`. Tombol "Adjust Stock" manual dihitung sebagai aksi inventory aktual — disisakan untuk superadmin/admin/manager. Least privilege. |

**TIDAK ada grant baru. TIDAK ada perubahan untuk superadmin/admin/manager/cashier.**

## 3. MATRIKS TARGET (72 × 5 role)

Legenda: ✅ = grant, — = tidak. Kolom **Staff** menunjukkan target setelah 023. Tanda ◀ menandai perubahan.

### 3.1 Sistem & Akun

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 1 | `dashboard.view` | ✅ | ✅ | ✅ | — | — |
| 2 | `user.view` | ✅ | ✅ | — | — | — |
| 3 | `user.create` | ✅ | ✅ | — | — | — |
| 4 | `user.update` | ✅ | ✅ | — | — | — |
| 5 | `user.delete` | ✅ | — | — | — | — |
| 6 | `role.view` | ✅ | ✅ | — | — | — |
| 7 | `role.create` | ✅ | ✅ | — | — | — |
| 8 | `role.update` | ✅ | — | — | — | — |
| 9 | `role.delete` | ✅ | — | — | — | — |
| 10 | `audit.view` | ✅ | — | — | — | — |
| 11 | `report.view` | ✅ | ✅ | ✅ | — | — |

### 3.2 Produk & Kategori

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 12 | `product.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| 13 | `product.create` | ✅ | ✅ | — | — | — |
| 14 | `product.update` | ✅ | ✅ | ✅ | — | ~~✅~~ → **— ◀ R1** |
| 15 | `product.delete` | ✅ | ✅ | — | — | — |
| 16 | `product.export` | ✅ | ✅ | — | — | — |
| 17 | `product.import` | ✅ | ✅ | — | — | — |
| 18 | `category.view` | ✅ | ✅ | ✅ | — | — |
| 19 | `category.create` | ✅ | ✅ | ✅ | — | — |
| 20 | `category.update` | ✅ | ✅ | — | — | — |
| 21 | `category.delete` | ✅ | ✅ | — | — | — |
| 22 | `category.export` | ✅ | ✅ | — | — | — |
| 23 | `category.import` | ✅ | ✅ | — | — | — |

### 3.3 Penjualan & Shift

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 24 | `sale.view` | ✅ | ✅ | ✅ | ✅ | — |
| 25 | `sale.create` | ✅ | ✅ | — | ✅ | — |
| 26 | `sale.park` | ✅ | ✅ | — | ✅ | — |
| 27 | `shift.view` | ✅ | ✅ | ✅ | ✅ | — |
| 28 | `shift.create` | ✅ | ✅ | ✅ | ✅ | — |
| 29 | `shift.review` | ✅ | ✅ | ✅ | — | — |
| 30 | `shift.audit` | ✅ | ✅ | ✅ | — | — |

### 3.4 Pelanggan & Harga

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 31 | `customer.view` | ✅ | ✅ | ✅ | ✅ | — |
| 32 | `customer.create` | ✅ | ✅ | ✅ | — | — |
| 33 | `customer.update` | ✅ | ✅ | ✅ | — | — |
| 34 | `customer.delete` | ✅ | ✅ | — | — | — |
| 35 | `customer.export` | ✅ | ✅ | — | — | — |
| 36 | `customer.import` | ✅ | ✅ | — | — | — |
| 37 | `pricing.view` | ✅ | ✅ | ✅ | — | — |
| 38 | `pricing.create` | ✅ | ✅ | ✅ | — | — |
| 39 | `pricing.update` | ✅ | ✅ | ✅ | — | — |
| 40 | `pricing.delete` | ✅ | ✅ | — | — | — |

### 3.5 Inventory

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 41 | `inventory.adjust` | ✅ | ✅ | ✅ | — | ~~✅~~ → **— ◀ R2** |

### 3.6 Toko & Customer Group

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 42 | `store.view` | ✅ | ✅ | — | — | — |
| 43 | `store.create` | ✅ | ✅ | — | — | — |
| 44 | `store.update` | ✅ | ✅ | — | — | — |
| 45 | `store.delete` | ✅ | ✅ | — | — | — |
| 46 | `customer_group.view` | ✅ | ✅ | ✅ | — | — |
| 47 | `customer_group.create` | ✅ | ✅ | — | — | — |
| 48 | `customer_group.update` | ✅ | ✅ | — | — | — |
| 49 | `customer_group.delete` | ✅ | ✅ | — | — | — |

### 3.7 Purchase Order

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 50 | `purchase_order.view` | ✅ | ✅ | ✅ | — | — |
| 51 | `purchase_order.create` | ✅ | ✅ | ✅ | — | — |
| 52 | `purchase_order.update` | ✅ | ✅ | ✅ | — | — |
| 53 | `purchase_order.delete` | ✅ | — | — | — | — |
| 54 | `purchase_order.confirm` | ✅ | ✅ | ✅ | — | — |
| 55 | `purchase_order.receive` | ✅ | ✅ | ✅ | — | — |
| 56 | `purchase_order.cancel` | ✅ | ✅ | ✅ | — | — |

### 3.8 Stock Opname

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 57 | `stock_opname.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| 58 | `stock_opname.create` | ✅ | ✅ | ✅ | — | — |
| 59 | `stock_opname.assign` | ✅ | ✅ | ✅ | — | — |
| 60 | `stock_opname.count` | ✅ | ✅ | — | ✅ | ✅ |
| 61 | `stock_opname.submit` | ✅ | ✅ | — | ✅ | ✅ |
| 62 | `stock_opname.recount` | ✅ | ✅ | ✅ | — | — |
| 63 | `stock_opname.cancel` | ✅ | ✅ | ✅ | — | — |
| 64 | `stock_opname.export` | ✅ | ✅ | ✅ | — | — |
| 65 | `stock_opname.verify` | ✅ | ✅ | ✅ | — | — |
| 66 | `stock_opname.post` | ✅ | ✅ | ✅ | — | — |
| 67 | `stock_opname.close` | ✅ | ✅ | ✅ | — | — |
| 68 | `stock_opname.report` | ✅ | ✅ | ✅ | — | — |

### 3.9 Storage Location

| # | Permission | SA | Admin | Manager | Cashier | Staff (target) |
|---|-----------|----|-------|---------|---------|----------------|
| 69 | `storage_location.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| 70 | `storage_location.create` | ✅ | ✅ | — | — | — |
| 71 | `storage_location.update` | ✅ | ✅ | — | — | — |
| 72 | `storage_location.delete` | ✅ | ✅ | — | — | — |

## 4. RINGKASAN PER ROLE (Setelah 023)

| Role | Sebelum | Setelah | Delta |
|------|---------|---------|-------|
| superadmin | 72 | 72 | 0 |
| admin | 67 | 67 | 0 |
| manager | 36 | 36 | 0 |
| cashier | 11 | 11 | 0 |
| staff | 7 | **5** | −2 |

## 5. CHANGE REGISTER (Migration 023)

| ID | Tipe | Detail | File Migrasi |
|----|------|--------|--------------|
| R1 | REVOKE `staff.product.update` | `DELETE FROM role_permissions` di mana role=staff & permission=product.update | `023_sprint0_finalize_permissions.sql` |
| R2 | REVOKE `staff.inventory.adjust` | `DELETE FROM role_permissions` di mana role=staff & permission=inventory.adjust | `023_sprint0_finalize_permissions.sql` |

Catatan: revoke ditulis sebagai `DELETE FROM role_permissions ... USING`/`WHERE (role_id, permission_id) IN (SELECT ...)` agar idempotent dan aman untuk DB yang sudah pernah di-seed sebelum 022.

## 6. BEHAVIOR DELTA REGISTER

| ID | Delta | Detail | Klasifikasi |
|----|-------|--------|-------------|
| D1 | Staff kehilangan tombol "Adjust Stock" (modal adjust) di `ProductsPage`/`ProductDetailDrawer` | Setelah 023 staff tidak punya `inventory.adjust` → tombol disembunyikan. Aksi stock opname counting (`stock_opname.count/submit`) TIDAK terpengaruh. | **Disengaja** (least privilege) |
| D2 | Bug 403 "Add Product" untuk manager/staff diperbaiki | Saat ini tombol Add Product aktif untuk manager/staff (via `canManageInventory`), tapi backend `product.create` hanya superadmin/admin → klik = 403. Setelah refactor, tombol digate `rbac.can('product.create')` → tombol hanya muncul untuk yang berhak. | **Perbaikan bug** |
| D3 | Staff tetap tidak bisa edit produk — tanpa regresi | `canEdit` saat ini superadmin/admin/manager; setelah refactor `rbac.can('product.update')`, dan staff kehilangan `product.update` via 023. Tidak ada perubahan perilaku untuk staff. | **Konsisten (tanpa regresi)** |

Semua perilaku lain (navigasi, default route, label role, kolom tabel, workflow) **tidak berubah** — No Behavior Change Policy.

## 9. DEPLOYMENT & VERIFICATION (Migration 023)

Migration `023_sprint0_finalize_permissions.sql` adalah artefak deployment. Tidak diaplikasikan ke DB development saat review — hanya pada environment deployment (staging/production) sesuai proses deployment.

### 9.1 Verification Query (setelah apply)

```sql
SELECT r.name AS role, p.code AS permission
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE r.name = 'staff'
  AND p.code IN ('product.update', 'inventory.adjust')
ORDER BY p.code;
```

**Expected result: 0 rows** — grant `staff.product.update` dan `staff.inventory.adjust` tidak ada lagi.

Cek total grant per role agar konsisten dengan §4:

```sql
SELECT r.name, COUNT(*) AS grants
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
GROUP BY r.name
ORDER BY r.name;
```

**Expected: superadmin=72, admin=67, manager=36, cashier=11, staff=5.**

### 9.2 UI Smoke Test Checklist

Login sebagai setiap role di bawah ini dan verifikasi sesuai Behavior Delta Register (§6):

| # | Skenario | Role | Expected |
|---|----------|------|----------|
| 1 | Edit Product → tombol Edit di ProductsPage | Staff | **Hilang** (D3) |
| 2 | Adjust Stock → modal adjust / tombol "Adjust Stock" | Staff | **Hilang** (D1) |
| 3 | Stock Opname → halaman counting tetap terbuka | Staff | **Normal** — `stock_opname.count/submit` tidak terpengaruh |
| 4 | Edit Product → tombol Edit di ProductsPage | Manager | **Tetap ada** (D3) |
| 5 | Add Product → tombol Add di ProductsPage | Manager | **Tidak muncul** (D2 — sudah berubah di Fase 3; hanya SA/A) |
| 6 | POS → buka halaman POS & transaksi | Cashier | **Normal** — POS & shift tetap jalan |
| 7 | Sidebar → navigasi sesuai role | Semua | **Tidak berubah** (presentasi, permission-gated) |

### 9.3 Exit Criteria Sprint 0

- [ ] `scripts/lint-rbac.sh` clean
- [ ] vitest hijau (termasuk matriks 72×5 `useRBAC.test.ts`)
- [ ] `npm run build` hijau
- [ ] `go test -p 1 -count=1 ./...` hijau
- [ ] Migration `023` applied di environment target
- [ ] Verification query §9.1 → 0 rows + total grant per role sesuai §4
- [ ] UI smoke test §9.2 lulus

---

## 7. TRACEABILITY

- Setiap perubahan permission dapat ditelusuri ke file ini.
- Format catatan commit/PR selama Sprint 0:
  - `Permission: staff.product.update`
  - `Decision: REVOKE (R1)`
  - `Section: 3.2 / 5`
  - `Status: intended`
  - `Reason: dead grant, least privilege`

## 8. SPRINT 0 SCOPE FREEZE

> **Sprint 0 Scope Freeze:** Setelah `permission-matrix-final.md` disetujui, perubahan terhadap permission matrix tidak diperbolehkan kecuali ditemukan bug implementasi. Perubahan kebutuhan bisnis akan dijadwalkan pada sprint berikutnya.

Selama Sprint 0:
- Jangan mengubah permission matrix.
- Jangan menambah permission baru.
- Jangan menambah role baru.
- Jangan mengubah workflow.
- Kasus baru yang ditemukan → masukkan ke backlog Sprint 1, bukan diubah di tengah Sprint 0.

---

*Dokumen ini disetujui secara resmi oleh pemilik proyek pada 2026-08-04. Perubahan di luar 2 revoke di atas membutuhkan amendemen dokumen ini terlebih dahulu.*
