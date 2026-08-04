# LAPORAN AUDIT RBAC — Sprint 0 (v4)

**Tanggal:** 2026-08-04 (revisi v4)
**Status:** FINAL — menunggu approval dokumen Fase 0
**Tujuan:** Fondasi RBAC — single source of truth untuk authorization di frontend, permission-based, tanpa mengubah workflow/business behavior.

**Riwayat revisi:**
- v1: audit awal (74 permission, 50 role-based check)
- v2: API generik `can/canAny/canAll` — tanpa helper per-permission
- v3: Permission Registry frontend + lint annotation-based
- v4: **Koreksi jumlah permission 74 → 72** (`pos.access`, `inventory.export` tidak ada di DB); integrasi seluruh keputusan review (DoD-1..4, No Behavior Change Policy, Exit Criteria, Traceability, sidebar permission-gate)

---

## 1. DAFTAR ROLE-BASED CHECK YANG DITEMUKAN (Frontend)

**50 lokasi** ditemukan menggunakan role-name-based authorization. Dikelompokkan berdasarkan tindakan:

### 1A. AUTHORIZATION — Mengontrol aksi/akses (harus diganti ke permission)

| # | File | Baris | Pola | Mengontrol |
|---|------|-------|------|------------|
| 1 | `shared/composables/useRBAC.svelte.ts` | 4-5, 21-38 | `ADMIN_ROLES`, `MANAGER_ROLES`, semua getter | **AKAR MASALAH** — 9 page consumer inherit role-based auth |
| 2 | `product/components/ProductsPage.svelte` | 47-48 | `allowedInventoryRoles`, `allowedStockRoles` | Tombol stock adjust, rack panel |
| 3 | `product/components/ProductsPage.svelte` | 401-406 | `getUserRoleName() === 'superadmin'` etc. | `canCreate`, `canEdit`, `isSensitive`, `isFullAudit` |
| 4 | `product/components/ProductsPage.svelte` | 424 | `allowedInventoryRoles.includes(...)` | `canManageInventory` |
| 5 | `product/components/ProductsPage.svelte` | 572-573 | `isSuperAdmin \|\| isAdmin`, role check | Props ke ProductTable |
| 6 | `product/components/ProductsPage.svelte` | 668-674 | Multi role-derived props | Props ke ProductDetailDrawer |
| 7 | `product/components/ProductFormModal.svelte` | 257 | `isSuperAdmin \|\| isAdmin` | Status "Archived" option |
| 8 | `product/components/ProductDetailDrawer.svelte` | 238-263 | `isSensitive` | Harga beli & margin visibility |
| 9 | `product/components/ProductDetailDrawer.svelte` | 324-341 | `isFullAudit` | Audit trail visibility |
| 10 | `product/components/ProductDetailDrawer.svelte` | 346 | `canEdit` | Tombol edit/delete |
| 11 | `admin/components/UsersPage.svelte` | 41-45, 319 | `rbac.canCreate/canEdit/canDelete/canView/canEditSuperadmin/canAssignManager` | CRUD users |
| 12 | `admin/components/RolesPage.svelte` | 196-198 | `rbac.canView/isSuperAdmin/canDelete` | CRUD roles |
| 13 | `admin/components/AuditLogsPage.svelte` | 37 | `rbac.isSuperAdmin` | Akses audit log |
| 14 | `admin/components/UserTable.svelte` | 157 | `user.role_id === 1 && !canEditSuperadmin` | Tombol edit user |
| 15 | `admin/components/UserTable.svelte` | 168 | `user.role_id === 1` | Tombol delete user |
| 16 | `shifts/components/ShiftsPage.svelte` | 24 | `rbac.isCashier` | Filter shift ke own shifts |
| 17 | `shifts/components/ShiftsPage.svelte` | 194 | `rbac.isCashier` | Sembunyikan status filter |
| 18 | `shifts/components/ShiftsPage.svelte` | 500 | `rbac.isManager` | Prop ke ShiftDetailDrawer |
| 19 | `shifts/components/ShiftDetailDrawer.svelte` | 129, 135 | `isManager` | Tombol "Review & Approve" & "Surprise Audit" |
| 20 | `sales/components/TransactionsPage.svelte` | 16-18 | `rbac.isCashier` | Filter transaksi ke own |
| 21 | `pos/components/PosPage.svelte` | 627-631 | `userRole === 'cashier'` | Redirect ke /shifts jika buka shift |
| 22 | `settings/components/CategoriesPage.svelte` | 39 | `rbac.isAdmin` | Tombol hapus |
| 23 | `settings/components/BrandsPage.svelte` | 32 | `rbac.isAdmin` | Tombol hapus |
| 24 | `settings/components/UnitsOfMeasurePage.svelte` | 33 | `rbac.isAdmin` | Tombol hapus |
| 25 | `stores/components/StoresPage.svelte` | 37-39 | `rbac.canCreate/canEdit/isAdmin` | CRUD stores |
| 26 | `shared/utils/default-route.ts` | 3-5 | `role === 'cashier'`, `role === 'staff'` | Default route setelah login |
| 27 | `app/layouts/Sidebar.svelte` | 57-61 | `role_id === 1/2/3/4/5` | Role resolution (akar) |
| 28 | `app/layouts/Sidebar.svelte` | 63 | `role !== 'cashier'` | Logout button |
| 29 | `app/layouts/Sidebar.svelte` | 138-140 | `role === 'staff'/'cashier'/'manager'` | Nav item selection |
| 30 | `app/layouts/Sidebar.svelte` | 145-147 | `role === 'staff'/'cashier'/'manager'` | Master data sub-item selection |
| 31 | `app/layouts/Sidebar.svelte` | 152, 155 | `role === 'superadmin'`, `role === 'admin'` | Admin section visibility |

### 1B. DISPLAY — Hanya tampilan label/warna (boleh pertahankan)

| # | File | Baris | Pola | Keterangan |
|---|------|-------|------|------------|
| 32 | `admin/components/UserTable.svelte` | 34-38 | `roleName === 'superadmin'` | Badge color variant |
| 33 | `admin/components/UserTable.svelte` | 41-43 | `role_id` fallback | Role label display |
| 34 | `shifts/components/ShiftsPage.svelte` | 316, 319, 344, 351, 362 | `rbac.isCashier` | Kolom lebar & sembunyi kolom cashier |
| 35 | `stock-opname/.../StockOpnameDetailPage.svelte` | 139-143, 462 | `a.role === 'counter'` | Session assignment (bukan system role) |

---

## 2. DAFTAR ENDPOINT TANPA PERMISSION (Backend)

186 total endpoints. 162 authorized. 24 bermasalah:

### 2A. Tanpa permission check (authenticated only)

| # | Endpoint | File | Risiko |
|---|----------|------|--------|
| 1 | `GET /api/products` | `product/handler.go:64` | **MEDIUM** — Semua user bisa list produk |
| 2 | `GET /api/products/next-sku` | `product/handler.go:65` | LOW |
| 3 | `GET /api/products/:id` | `product/handler.go:67` | **MEDIUM** — Semua user bisa view detail |
| 4 | `GET /api/categories` | `category/handler.go:35` | **MEDIUM** — Simple listing tanpa perm |
| 5 | `GET /api/payment-methods/:code` | `sale/handler.go:65` | LOW |
| 6 | `GET /api/shifts/active` | `shift/handler.go:48` | LOW (self-scoped) |
| 7 | `GET /api/import-export/modules` | `importexport/handler.go:81` | LOW |
| 8 | `GET /api/import-export/template/:module` | `importexport/handler.go:82` | **MEDIUM** |
| 9 | `GET /api/import-export/progress/:jobId` | `importexport/handler.go:85` | **MEDIUM** |
| 10 | `POST /api/import-export/cancel/:jobId` | `importexport/handler.go:86` | **HIGH** — Bisa cancel job user lain |

### 2B. Permission namespace salah (supplier pakai pricing.*)

| # | Endpoint | File | Permission Digunakan | Seharusnya |
|---|----------|------|---------------------|------------|
| 11-23 | Semua `/api/suppliers/*` (13 routes) | `supplier/handler.go:45-62` | `pricing.view/create/update/delete` | `supplier.*` atau `pricing.*` (documented) |

### 2C. Middleware unused

| Middleware | File | Status |
|---|---|---|
| `RoleMiddleware()` | `middleware/auth.go:58-73` | **TIDAK DIPAKAI** di route manapun |
| `AdminOnly()` | `middleware/auth.go:119-135` | **TIDAK DIPAKAI** di route manapun |
| `RequireAnyPermission()` | `middleware/auth.go:96-117` | **TIDAK DIPAKAI** di route manapun |

---

## 3. DAFTAR PERMISSION YANG DIPAKAI FRONTEND

### 3A. Route-level (permissions.ts)

| Permission | Route |
|---|---|
| `dashboard.view` | `/` |
| `sale.create` | `/pos` |
| `product.view` | `/inventory`, `/inventory/products`, `/brands`, `/units-of-measure` |
| `report.view` | `/reports` |
| `sale.view` | `/transactions` |
| `customer.view` | `/customers` |
| `category.view` | `/categories` |
| `user.view` | `/admin`, `/admin/users` |
| `role.view` | `/admin/roles` |
| `audit.view` | `/admin/audit-logs` |
| `store.view` | `/stores` |
| `pricing.view` | `/pricing-rules`, `/suppliers` |
| `customer_group.view` | `/customer-groups` |
| `purchase_order.view` | `/purchase-orders` |
| `shift.view` | `/shifts` |
| `stock_opname.view` | `/stock-opnames` |
| `stock_opname.report` | `/stock-opnames/adjustments` |
| `storage_location.view` | `/storage-locations` |

### 3B. Component-level (permission code checks)

| Permission | Component | Baris |
|---|---|---|
| `stock_opname.create` | `StockOpnamesPage.svelte` | 25 |
| `stock_opname.export` | `StockOpnamesPage.svelte:26`, `StockOpnameDetailPage.svelte:25` |
| `stock_opname.report` | `StockOpnamesPage.svelte:27` |
| `stock_opname.verify` | `StockOpnameDetailPage.svelte` | 17 |
| `stock_opname.post` | `StockOpnameDetailPage.svelte` | 18 |
| `stock_opname.close` | `StockOpnameDetailPage.svelte` | 19 |
| `stock_opname.recount` | `StockOpnameDetailPage.svelte` | 20 |
| `stock_opname.cancel` | `StockOpnameDetailPage.svelte` | 21 |
| `stock_opname.count` | `StockOpnameDetailPage.svelte` | 22 |
| `stock_opname.submit` | `StockOpnameDetailPage.svelte` | 23 |
| `stock_opname.assign` | `StockOpnameDetailPage.svelte` | 24 |
| `pricing.create` | `PricingRulesPage.svelte:103`, `SuppliersPage.svelte:20,24` |
| `pricing.update` | `PricingRulesPage.svelte:104`, `SuppliersPage.svelte:21` |

---

## 4. KOREKSI PERMISSION COUNT (v4)

**Audit v1 menyebut 74 permission aktif, dengan `pos.access` dan `inventory.export` dikategorikan "unused".**

**Fakta (verified 2026-08-04):** Kedua permission tersebut **TIDAK ADA** di database. Mereka hanya didefinisikan di `database/seeds/002_permissions.sql` (legacy, tidak diaplikasikan ke DB). Live DB `retail_pos` berisi **72 permission**, semuanya di-grant ke minimal satu role (0 ungranted).

**Sumber kebenaran permission:** `database/migrations/000_squash.sql:644-693` (48 code) + migrasi `007` (6: purchase_order.view/create/update/delete/confirm/receive), `008` (1: purchase_order.cancel), `012` (8: stock_opname.view/create/assign/count/submit/recount/cancel/export), `016` (4: stock_opname.verify/post/close/report), `018` (4: storage_location.view/create/update/delete), `seeds/013_customer_permissions.sql` (4: customer.view/create/update/delete), `seeds/002_permissions.sql` (1: inventory.adjust).

> **Konsekuensi:** §7 daftar "74 active" di v1 salah. Lihat §7.1 di bawah untuk daftar 72 yang benar.

---

## 5. PERMISSION DIPAKAI BACKEND TAPI TIDAK PERNAH DICEK FRONTEND

| Permission | Backend Route | Dampak |
|---|---|---|
| `product.create` | `POST /products`, `POST /brands`, `POST /uom` | Frontend pakai role-name `canCreate = isAdmin` |
| `product.update` | `PUT /products/:id`, `POST /products/bulk/status`, `PUT /brands/:id`, `PUT /uom/:id` | Frontend pakai role-name `canEdit = ['superadmin','admin','manager']` |
| `product.delete` | `DELETE /products/:id`, `DELETE /brands/:id`, `DELETE /uom/:id` | Frontend pakai role-name `canDelete = isSuperAdmin` |
| `product.export` | Export handler | Tidak dicek frontend |
| `product.import` | Import handler | Frontend dicek via `pricing.create` (salah!) |
| `category.create` | `POST /categories` | Frontend sudah benar via `useRBAC.canCreate` tapi role-based |
| `category.update` | `PUT /categories/:id` | Tidak ada UI edit category |
| `category.delete` | `DELETE /categories/:id` | Frontend pakai `rbac.isAdmin` |
| `category.export` | Export handler | Tidak dicek frontend |
| `category.import` | Import handler | Tidak dicek frontend |
| `sale.park` | `POST/GET/DELETE /sales/parked/*` | Frontend: park/hold pakai `sale.create` |
| `customer.create` | `POST /customers` | Tidak ada UI create customer di frontend |
| `customer.update` | `PUT /customers/:id` | Tidak ada UI edit customer di frontend |
| `customer.delete` | `DELETE /customers/:id` | Tidak ada UI delete customer di frontend |
| `customer.export` | Export handler | Tidak dicek frontend |
| `customer.import` | Import handler | Tidak dicek frontend |
| `inventory.adjust` | `POST /inventory/adjust`, `POST /inventory/locations`, `POST /inventory/locations/transfer` | Frontend pakai role-name `allowedStockRoles` |
| `shift.review` | `POST /shifts/:id/review` | Frontend pakai `isManager` prop |
| `shift.audit` | `POST /shifts/:id/audit` | Frontend pakai `isManager` prop |
| `storage_location.create/update/delete` | CRUD endpoints | Tidak ada UI manage storage location |
| `store.create/update/delete` | CRUD endpoints | Frontend pakai `rbac.canCreate/canEdit/isAdmin` |
| `customer_group.create/update/delete` | CRUD endpoints | Frontend pakai `rbac.canCreate/canEdit/isAdmin` |
| `user.create/update/delete` | CRUD endpoints | Frontend pakai `rbac.canCreate/canEdit/canDelete` (role-based) |
| `role.create/update/delete` | CRUD endpoints | Frontend pakai `rbac.isSuperAdmin` |
| `audit.view` | `GET /audit-logs/*` | Frontend pakai `rbac.isSuperAdmin` |
| `purchase_order.*` | Semua CRUD routes | Tidak ada granular check di frontend |
| `stock_opname.*` (semua) | Semua routes | **SUDAH BENAR** — `StockOpnameDetailPage.svelte` pakai permission code |

---

## 6. MASALAH INFRASTRUKTUR

| # | Masalah | Lokasi | Dampak |
|---|---------|--------|--------|
| 1 | **`useRBAC()` murni role-name** | `shared/composables/useRBAC.svelte.ts` | 9 page component inherit role-based auth |
| 2 | **`product-utils.ts:hasPermission()` MISNAMED** | `product/lib/product-utils.ts:72-74` | Cek role name, bukan permission — membingungkan |
| 3 | **3 duplikat `getUserRoleName()`** | `useRBAC`, `product-utils.ts`, `ProductsPage.svelte` + 2 inline | Hardcoded `role_id` mapping di 5 tempat |
| 4 | **Sidebar pakai dual system** | `Sidebar.svelte` | Role-name selection PARALEL dengan permission check |
| 5 | **String literal permission tersebar** | `permissions.ts`, `routePermissions`, komponen | Tidak ada single source of truth di kode |

---

## 7. COMPLETE PERMISSION LIST (72 live)

> **Koreksi v4:** daftar 74 di v1 salah. Berikut daftar **72 permission** yang benar, diverifikasi langsung dari DB. (Tidak termasuk `pos.access`, `inventory.export`.)

| # | Code | Created In |
|---|------|------------|
| 1 | `dashboard.view` | squash/000 |
| 2 | `user.view` | squash/000 |
| 3 | `user.create` | squash/000 |
| 4 | `user.update` | squash/000 |
| 5 | `user.delete` | squash/000 |
| 6 | `role.view` | squash/000 |
| 7 | `role.create` | squash/000 |
| 8 | `role.update` | squash/000 |
| 9 | `role.delete` | squash/000 |
| 10 | `audit.view` | squash/000 |
| 11 | `report.view` | squash/000 |
| 12 | `product.view` | squash/000 |
| 13 | `product.create` | squash/000 |
| 14 | `product.update` | squash/000 |
| 15 | `product.delete` | squash/000 |
| 16 | `product.export` | squash/000 |
| 17 | `product.import` | squash/000 |
| 18 | `category.view` | squash/000 |
| 19 | `category.create` | squash/000 |
| 20 | `category.update` | squash/000 |
| 21 | `category.delete` | squash/000 |
| 22 | `category.export` | squash/000 |
| 23 | `category.import` | squash/000 |
| 24 | `sale.view` | squash/000 |
| 25 | `sale.create` | squash/000 |
| 26 | `sale.park` | squash/000 |
| 27 | `shift.view` | squash/000 |
| 28 | `shift.create` | squash/000 |
| 29 | `shift.review` | squash/000 |
| 30 | `shift.audit` | squash/000 |
| 31 | `customer.view` | seeds/013 |
| 32 | `customer.create` | seeds/013 |
| 33 | `customer.update` | seeds/013 |
| 34 | `customer.delete` | seeds/013 |
| 35 | `customer.export` | squash/000 |
| 36 | `customer.import` | squash/000 |
| 37 | `pricing.view` | squash/000 |
| 38 | `pricing.create` | squash/000 |
| 39 | `pricing.update` | squash/000 |
| 40 | `pricing.delete` | squash/000 |
| 41 | `inventory.adjust` | seeds/002 |
| 42 | `store.view` | squash/000 |
| 43 | `store.create` | squash/000 |
| 44 | `store.update` | squash/000 |
| 45 | `store.delete` | squash/000 |
| 46 | `customer_group.view` | squash/000 |
| 47 | `customer_group.create` | squash/000 |
| 48 | `customer_group.update` | squash/000 |
| 49 | `customer_group.delete` | squash/000 |
| 50 | `purchase_order.view` | migration 007 |
| 51 | `purchase_order.create` | migration 007 |
| 52 | `purchase_order.update` | migration 007 |
| 53 | `purchase_order.delete` | migration 007 |
| 54 | `purchase_order.confirm` | migration 007 |
| 55 | `purchase_order.receive` | migration 007 |
| 56 | `purchase_order.cancel` | migration 008 |
| 57 | `stock_opname.view` | migration 012 |
| 58 | `stock_opname.create` | migration 012 |
| 59 | `stock_opname.assign` | migration 012 |
| 60 | `stock_opname.count` | migration 012 |
| 61 | `stock_opname.submit` | migration 012 |
| 62 | `stock_opname.recount` | migration 012 |
| 63 | `stock_opname.cancel` | migration 012 |
| 64 | `stock_opname.export` | migration 012 |
| 65 | `stock_opname.verify` | migration 016 |
| 66 | `stock_opname.post` | migration 016 |
| 67 | `stock_opname.close` | migration 016 |
| 68 | `stock_opname.report` | migration 016 |
| 69 | `storage_location.view` | migration 018 |
| 70 | `storage_location.create` | migration 018 |
| 71 | `storage_location.update` | migration 018 |
| 72 | `storage_location.delete` | migration 018 |

---

## 8. KEPUTUSAN DESAIN (v2–v4 REVIEW)

Keputusan berikut dikonfirmasi dalam review dan WAJIB ditaati:

| # | Keputusan | Detail |
|---|-----------|--------|
| K1 | **API generik** | `rbac.can(perm)` / `rbac.canAny(perms[])` / `rbac.canAll(perms[])`. TIDAK ada helper per-permission (`canCreateProduct`, `canAdjustStock`, dst). |
| K2 | **Permission Registry** | Semua permission code = konstanta di `web/src/shared/constants/permissions.ts`. String literal permission DILARANG dalam kode komponen (DoD-4). |
| K3 | **Role tetap ada untuk presentation/ownership** | `userRole`, `roleDisplayName`, `isCashier` (utk ownership) boleh dipertahankan. DILARANG untuk authorization. Komentar `// @display-only`, `// @ownership-only`. |
| K4 | **Sidebar metadata-driven** | Satu array route meta + `rbac.canAny(item.permissions)`. Group "Master Data" visible jika user punya minimal satu permission manage: `product.create`, `category.create`, `customer.create`, `pricing.create`, `store.create`, `customer_group.create`. Cashier & staff → menu minimal. |
| K5 | **Default route tetap role-based** | `default-route.ts` TIDAK diubah (navigasi UX, bukan authz). |
| K6 | **`isSensitive` → `pricing.view`** | Cost/margin visibility. Zero delta vs UI saat ini [SA,A,M]. TODO Sprint 1: `product.cost.view`. |
| K7 | **Entity history panel = compatibility layer lokal** | `shouldShowProductHistory = rbac.isAdmin || rbac.isSuperAdmin` di `ProductDetailDrawer.svelte`, diberi komentar `// @compatibility-layer` + `// TODO: Remove after product.history.view exists`. Bukan exception permanen. |
| K8 | **Ownership shift = data-scope layer** | `ShiftsPage.svelte:24` (`rbac.isCashier → store.userIdFilter`) diberi `// @ownership-only`. **Backend `GET /shifts` saat ini TIDAK menscope ownership** — enforcement backend = Sprint 1. |
| K9 | **Lint annotation-based** | Bukan path-allowlist. Blokir pola: `role ===`, `switch(role)`, `role?.name`, `roles.includes(`, `role_id === N`, `getUserRoleName`, `allowedStockRoles`/`allowedInventoryRoles`, `ADMIN_ROLES`/`MANAGER_ROLES`, string literal dalam `rbac.can('...')`. Izinkan dengan annotation `// @display-only`, `// @ownership-only`, `// @compatibility-layer`. |
| K10 | **Tidak ada compatibility layer jadi API publik** | useRBAC hanya export API generik + data display/ownership. |

---

## 9. NO BEHAVIOR CHANGE POLICY

Selama Sprint 0, **tidak ada perubahan pada workflow, business logic, navigasi, default route, atau hasil query ke backend**. Satu-satunya perubahan perilaku yang diizinkan adalah yang tercatat di **Behavior Delta Register** (`permission-matrix-final.md` §6):

- **D1:** Staff kehilangan tombol "Adjust Stock" (setelah 023) — disengaja.
- **D2:** Bug 403 "Add Product" untuk manager/staff diperbaiki (tombol digate `product.create`).
- **D3:** Staff tetap tidak bisa edit produk — tanpa regresi.

Perubahan frontend = **hanya** cara menentukan visibility/akses, dari "role name" ke "permission yang mewakili role tersebut". Setiap perubahan harus bisa menunjukkan delta perilaku = 0 (kecuali D1/D2/D3).

---

## 10. DEFINITION OF DONE (DoD)

| # | Kriteria | Verifikasi |
|---|----------|------------|
| DoD-1 | Tidak ada authorization berbasis role di frontend | `grep` role-based patterns = 0 (kecuali `@display-only`/`@ownership-only`/`@compatibility-layer`) |
| DoD-2 | Semua keputusan akses via permission (`rbac.can/canAny/canAll`) | Code review + lint |
| DoD-3 | Build + test + lint hijau | `go build ./...`, `go test -p 1 -count=1 ./...`, vitest, lint-rbac |
| DoD-4 | Tidak ada string literal permission di kode komponen | Semua via konstanta `permissions.ts`; lint memblokir string literal dalam `rbac.can('...')` |

---

## 11. EXIT CRITERIA (Sprint 0)

1. Semua 50 role-based check di §1A telah di-refactor ke permission-based.
2. `useRBAC` hanya export `can`/`canAny`/`canAll` + data display/ownership.
3. Sidebar, route, dan tombol-tombol CRUD menggunakan permission (metadata-driven).
4. Script `scripts/lint-rbac.sh` berjalan tanpa error di CI.
5. Unit test matrix role×permission + UI visibility test lulus (vitest).
6. `permission-matrix-final.md` disetujui; migration 023 = tepat 2 revoke.
7. Setiap perubahan permission dapat ditelusuri ke `permission-matrix-final.md` (format commit/PR: `Permission:` / `Decision:` / `Section:` / `Status:` / `Reason:`).
8. Build + full test suite hijau; tidak ada regresi yang tidak tercatat di Behavior Delta Register.

---

## 12. PLAN IMPLEMENTASI — URUTAN FASE

| Fase | Deskripsi | Isi | Mulai Setelah |
|------|-----------|-----|---------------|
| **0** | Dokumen & approval | Finalisasi `rbac-sprint0-audit.md` (ini) + `permission-matrix-final.md`. TANPA migration. | — |
| **1** | Permission Registry frontend | `web/src/shared/constants/permissions.ts` (72 konstanta) + `web/src/shared/constants/roles.ts`. Update `permissions.ts`/`routePermissions` merujuk konstanta. | Approval Fase 0 |
| **2** | Rewrite `useRBAC` | API generik `can/canAny/canAll`; hapus getter lama; pertahankan display/ownership (`@display-only`/`@ownership-only`). | Fase 1 |
| **3** | Refactor modul | Products (terbesar), Sidebar, admin (Users/Roles/AuditLogs), settings (Categories/Brands/UoM), stores, shifts, sales, POS, default-route (TIDAK diubah). | Fase 2 |
| **4** | Cleanup dead code | Hapus `getUserRoleName` duplikat, `hasPermission` misnamed, `allowed*Roles`, `ADMIN_ROLES`/`MANAGER_ROLES`. | Fase 3 |
| **5** | Lint + test | `scripts/lint-rbac.sh` (annotation-based), unit test matrix + UI visibility. | Fase 4 |
| **6** | **Migration 023 (PALING AKHIR)** | `023_sprint0_finalize_permissions.sql` — tepat 2 REVOKE (`staff.product.update`, `staff.inventory.adjust`). Hindari backend matrix baru vs frontend lama → false 403. | Fase 5 |
| — | Final verification | DoD-1..4 + Exit Criteria + Behavior Delta Register. | Fase 6 |

**PENTING:** Migration 023 dieksekusi SETELAH Fase 1–5 selesai. Backend permission matrix (`internal/permissions`) tidak berubah di Sprint 0.

### Fase 1 — Permission Registry

```typescript
// web/src/shared/constants/permissions.ts (draft)
export const Permissions = {
  dashboard: { view: 'dashboard.view' },
  user: { view: 'user.view', create: 'user.create', update: 'user.update', delete: 'user.delete' },
  role: { view: 'role.view', create: 'role.create', update: 'role.update', delete: 'role.delete' },
  audit: { view: 'audit.view' },
  report: { view: 'report.view' },
  product: { view: 'product.view', create: 'product.create', update: 'product.update', delete: 'product.delete', export: 'product.export', import: 'product.import' },
  category: { view: 'category.view', create: 'category.create', update: 'category.update', delete: 'category.delete', export: 'category.export', import: 'category.import' },
  sale: { view: 'sale.view', create: 'sale.create', park: 'sale.park' },
  shift: { view: 'shift.view', create: 'shift.create', review: 'shift.review', audit: 'shift.audit' },
  customer: { view: 'customer.view', create: 'customer.create', update: 'customer.update', delete: 'customer.delete', export: 'customer.export', import: 'customer.import' },
  pricing: { view: 'pricing.view', create: 'pricing.create', update: 'pricing.update', delete: 'pricing.delete' },
  inventory: { adjust: 'inventory.adjust' },
  store: { view: 'store.view', create: 'store.create', update: 'store.update', delete: 'store.delete' },
  customerGroup: { view: 'customer_group.view', create: 'customer_group.create', update: 'customer_group.update', delete: 'customer_group.delete' },
  purchaseOrder: { view: 'purchase_order.view', create: 'purchase_order.create', update: 'purchase_order.update', delete: 'purchase_order.delete', confirm: 'purchase_order.confirm', receive: 'purchase_order.receive', cancel: 'purchase_order.cancel' },
  stockOpname: { view: 'stock_opname.view', create: 'stock_opname.create', assign: 'stock_opname.assign', count: 'stock_opname.count', submit: 'stock_opname.submit', recount: 'stock_opname.recount', cancel: 'stock_opname.cancel', export: 'stock_opname.export', verify: 'stock_opname.verify', post: 'stock_opname.post', close: 'stock_opname.close', report: 'stock_opname.report' },
  storageLocation: { view: 'storage_location.view', create: 'storage_location.create', update: 'storage_location.update', delete: 'storage_location.delete' },
} as const;

// web/src/shared/constants/roles.ts (draft)
export const Roles = {
  superadmin: 'superadmin',
  admin: 'admin',
  manager: 'manager',
  cashier: 'cashier',
  staff: 'staff',
} as const;
```

### Fase 2 — useRBAC rewrite

```typescript
// API akhir:
rbac.can(permission: string): boolean
rbac.canAny(permissions: string[]): boolean
rbac.canAll(permissions: string[]): boolean

// Display / ownership (dipertahankan):
rbac.userRole            // @display-only
rbac.roleDisplayName     // @display-only
rbac.isCashier           // @ownership-only (hanya utk data-scope, bukan authz)
```

### Fase 3 — Mapping refactor (ringkas)

| File | Role-Based Check | Ganti Dengan |
|---|---|---|
| `CategoriesPage.svelte:39` | `rbac.isAdmin` | `rbac.can(Permissions.category.delete)` |
| `BrandsPage.svelte:32` | `rbac.isAdmin` | `rbac.can(Permissions.product.delete)` |
| `UnitsOfMeasurePage.svelte:33` | `rbac.isAdmin` | `rbac.can(Permissions.product.delete)` |
| `StoresPage.svelte:37-39` | `canCreate/canEdit/isAdmin` | `rbac.canAny([store.create, store.update, store.delete])` + `rbac.can(store.view)` |
| `UsersPage.svelte:41-45,319` | role-based getters | `rbac.can(user.view)` / `canAny([user.create, user.update])` / `can(user.delete)` + `user.role_id === 1` guard (`@display-only`? no — business guard utk superadmin protection, pakai `canAll([user.update, role.superadmin?])` → TODO Sprint 1 `user.superadmin.manage`; di Sprint 0 pertahankan dengan komentar) |
| `RolesPage.svelte:196-198` | `canView/isSuperAdmin/canDelete` | `rbac.can(role.view)` / `can(role.update)` / `can(role.delete)` |
| `AuditLogsPage.svelte:37` | `rbac.isSuperAdmin` | `rbac.can(audit.view)` |
| `ShiftsPage.svelte:24` | `rbac.isCashier` | `rbac.isCashier` (`@ownership-only`) — filter own shifts |
| `ShiftsPage.svelte:194` | `rbac.isCashier` | Sembunyikan status filter — `@display-only` |
| `ShiftsPage.svelte:500` | `rbac.isManager` | `rbac.canAny([shift.review, shift.audit])` |
| `ShiftDetailDrawer.svelte:129,135` | `isManager` | `rbac.can(shift.review)` / `rbac.can(shift.audit)` |
| `TransactionsPage.svelte:16-18` | `rbac.isCashier` | `@ownership-only` filter by user.id |
| `PosPage.svelte:627-631` | `userRole === 'cashier'` | `@display-only` (navigasi UX, bukan authz) |
| `ProductsPage.svelte:47-48` | `allowedInventoryRoles/allowedStockRoles` | `rbac.can(inventory.adjust)` |
| `ProductsPage.svelte:401-406` | `getUserRoleName() === ...` | `rbac.can(product.create/update/delete)`, `rbac.can(pricing.view)` (isSensitive), `rbac.can(audit.view)` (isFullAudit) |
| `ProductsPage.svelte:424` | `allowedInventoryRoles.includes(...)` | `rbac.can(inventory.adjust)` |
| `ProductsPage.svelte:572-573,668-674` | role-derived props | permission-based props (`canEdit=rbac.can(product.update)`, `canDelete=rbac.can(product.delete)`, `canAdjustStock=rbac.can(inventory.adjust)`, `isSensitive=rbac.can(pricing.view)`) |
| `ProductFormModal.svelte:257` | `isSuperAdmin \|\| isAdmin` | `rbac.can(product.delete)` (Archive) |
| `ProductDetailDrawer.svelte:238-263` | `isSensitive` | `rbac.can(pricing.view)` |
| `ProductDetailDrawer.svelte:324-341` | `isFullAudit` | `shouldShowProductHistory` — `@compatibility-layer` (K7), lihat nota resolusi di bawah. BUKAN `audit.view`. |
| `ProductDetailDrawer.svelte:346` | `canEdit` | `rbac.can(product.update)` |
| `Sidebar.svelte:57-155` | dual system | metadata-driven: satu array + `rbac.canAny(item.permissions)`; Master Data gate = `rbac.canAny([product.create, category.create, customer.create, pricing.create, store.create, customer_group.create])` |
| `default-route.ts:3-5` | role-based | **TIDAK DIUBAH** (K5) |

> Catatan UsersPage `role_id === 1` (superadmin protection) dipertahankan sementara dengan komentar `// @compatibility-layer // TODO: Sprint 1 — user.superadmin.manage`. Bukan bagian dari 50 hit authz (sudah berstatus permission-gate + guard ekstra).

> **Resolusi konflik K7 vs §12 (row 426):** Row 426 (`isFullAudit` → `rbac.can(audit.view)`) BERTENTANGAN dengan K7. `audit.view` hanya superadmin (matrix row 10), sedangkan panel Audit Trail saat ini tampil untuk [SA, A]. Menerapkan `audit.view` = perilaku berubah untuk admin (kehilangan panel) tanpa tercatat di delta register → melanggar No Behavior Change Policy. **Yang diterapkan: K7** — panel di-rename `shouldShowProductHistory`, dihitung di `ProductDetailDrawer.svelte` via `rbac.userRole === Roles.superadmin || rbac.userRole === Roles.admin` dengan `// @compatibility-layer` + TODO `product.history.view`. Implementasi final: `ProductDetailDrawer.svelte:11-17`.

---

## 13. FASE 4 — HAPUS DEAD CODE

| Item | Lokasi |
|---|---|
| Hapus `getUserRoleName()` dari `product-utils.ts` | `product/lib/product-utils.ts:60-69` |
| Hapus `hasPermission()` (role-based, misnamed) dari `product-utils.ts` | `product/lib/product-utils.ts:72-74` |
| Hapus duplikat `getUserRoleName()` dari `ProductsPage.svelte` | `ProductsPage.svelte:387-397` |
| Hapus `ADMIN_ROLES`, `MANAGER_ROLES` dari `useRBAC` | `useRBAC.svelte.ts:4-5` |
| Hapus getter authorization lama (`canCreate/canEdit/canDelete/canView/canEditSuperadmin/canAssignManager`) | `useRBAC.svelte.ts:21-38` |
| Hapus `allowedInventoryRoles`/`allowedStockRoles` | `ProductsPage.svelte:47-48` |
| Hapus export `hasPermission` dari `product/index.ts` | `product/index.ts` |

---

## 14. FASE 5 — LINT & TEST

### Lint (annotation-based, `scripts/lint-rbac.sh`)

**Status: SELESAI** — script ada di `scripts/lint-rbac.sh`, berjalan tanpa error di CI.

Blocked patterns (error):
- `role ===`, `role !==`, `switch(role)`, `role?.name`, `roles.includes(`
- `role_id === N` / `role_id !== N` (di luar file admin UserTable yang sudah dikategorikan)
- `userRole ===` / `userRole !==` (`rbac.userRole` comparisons) — WAJIB annotation
- `rbac.isCashier` — WAJIB annotation (`@ownership-only` untuk data-scope)
- `getUserRoleName`, `allowedStockRoles`, `allowedInventoryRoles`, `ADMIN_ROLES`, `MANAGER_ROLES`
- `rbac.can('...')` / `canAny([...])` dengan string literal permission (wajib konstanta registry)

Dizinkan hanya dengan annotation pada baris sebelum: `// @display-only`, `// @ownership-only`, `// @compatibility-layer`.

File yang di-exempt dari lint (implementasi / sudah dikategorikan):
- `web/src/shared/composables/useRBAC.svelte.ts` (inti komposable — menangani objek `role`)
- `web/src/modules/admin/components/UserTable.svelte` (`role_id` mapping sudah dikategorikan di §12)
- File test (`__tests/**`, `*.test.ts`, `*.spec.ts`)

Annotation yang ditambahkan saat Fase 5:
- `default-route.ts:3` → `// @display-only` (navigasi awal UX, bukan authz)
- `PosPage.svelte:627` → `// @display-only` (flow guard navigasi cashier → /shifts)
- `StockOpnameDetailPage.svelte:139` → `// @ownership-only` (data-scope counter assignee)
- `StockOpnameDetailPage.svelte:460` → `// @display-only` (badge tugas assignment)
- `UsersPage.svelte:92` → `// @display-only` (default role pada form, bukan authz)

### Test

| Test | Verifikasi |
|---|---|
| Unit matrix role×permission (`web/src/shared/composables/__tests__/useRBAC.test.ts`) | **SELESAI** — 10 test, termasuk matriks 72×5 lengkap (setiap (role, permission) sesuai `permission-matrix-final.md` target setelah 023) |
| UI visibility test per role | Sidebar items, tombol CRUD, default route |
| Existing tests pass | **SELESAI** — vitest 1275/1275, `go test -p 1 -count=1 ./...` hijau (backend tak tersentuh), `npm run build` hijau, `scripts/lint-rbac.sh` clean |

---

## 15. TEKNIKAL DEBT YANG DIBAWA KE SPRINT 1

| # | Debt | Prioritas |
|---|------|-----------|
| 1 | Backend Permission Registry `internal/permissions/permissions.go` (~162 string literal) — **SELESAI** (item 1 Sprint 1) | HIGH |
| 2 | Ownership enforcement backend: `GET /shifts` (`ListShifts`) tidak menscope ownership | HIGH |
| 3 | 10 backend endpoints tanpa permission check (§2A) | MEDIUM |
| 4 | Supplier endpoints pakai `pricing.*` namespace (§2B) | MEDIUM |
| 5 | Cleanup middleware `RoleMiddleware`, `AdminOnly`, `RequireAnyPermission` | LOW |
| 6 | Cleanup `seeds/002_permissions.sql` (legacy, berisi `pos.access`/`inventory.export`) | LOW |
| 7 | Permission baru `product.history.view`, `product.cost.view` → hapus compatibility layer | MEDIUM |

---

## 16. FILE YANG DIUBAH (Rencana Sprint 0)

**Baru:**
- `web/src/shared/constants/permissions.ts`
- `web/src/shared/constants/roles.ts`
- `scripts/lint-rbac.sh`
- `web/src/shared/composables/__tests__/useRBAC.test.ts` (matriks 72×5)
- `database/migrations/023_sprint0_finalize_permissions.sql` (**SELESAI dibuat** — header traceability + tepat 2 REVOKE idempotent; **TIDAK di-apply ke DB dev**, artefak deployment. Verification query + UI smoke test + exit checklist di `permission-matrix-final.md` §9)

**Diubah:**
- `web/src/shared/composables/useRBAC.svelte.ts` (rewrite)
- `web/src/shared/utils/permissions.ts` (sinkron registry — **DIPUTUSKAN: DIHAPUS**; satu-satunya export `hasPermission(userPerms, requiredPerm)` tidak punya consumer tersisa setelah Fase 3 dan duplikat `rbac.can`)
- `web/src/app/config/permissions.ts` (merujuk konstanta)
- `web/src/app/layouts/Sidebar.svelte` (metadata-driven)
- `web/src/modules/product/components/ProductsPage.svelte`
- `web/src/modules/product/components/ProductTable.svelte`
- `web/src/modules/product/components/ProductFormModal.svelte`
- `web/src/modules/product/components/ProductDetailDrawer.svelte`
- `web/src/modules/product/components/ProductFiltersToolbar.svelte`
- `web/src/modules/product/lib/product-utils.ts`
- `web/src/modules/product/index.ts`
- `web/src/modules/admin/components/UsersPage.svelte`
- `web/src/modules/admin/components/RolesPage.svelte`
- `web/src/modules/admin/components/AuditLogsPage.svelte`
- `web/src/modules/admin/components/UserTable.svelte`
- `web/src/modules/settings/components/CategoriesPage.svelte`
- `web/src/modules/settings/components/BrandsPage.svelte`
- `web/src/modules/settings/components/UnitsOfMeasurePage.svelte`
- `web/src/modules/stores/components/StoresPage.svelte`
- `web/src/modules/shifts/components/ShiftsPage.svelte`
- `web/src/modules/shifts/components/ShiftDetailDrawer.svelte`
- `web/src/modules/sales/components/TransactionsPage.svelte`
- `web/src/modules/pos/components/PosPage.svelte` (annotation `@display-only` pada flow guard cashier)
- `web/src/modules/stock-opname/components/StockOpnameDetailPage.svelte` (annotation `@ownership-only` + `@display-only`)
- `web/src/shared/utils/default-route.ts` (logika TIDAK DIUBAH — K5; hanya ditambah annotation `@display-only`)

**Total: ~26 file (21 diubah + 5 baru)**
