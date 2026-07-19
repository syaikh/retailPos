# Plan: Fix Unimplemented Buttons in Suppliers Module

## Temuan

### Temuan 1 & 3: "View all products" button di SupplierDetailDrawer + `onviewproducts` prop

**Lokasi:**
- `SupplierDetailDrawer.svelte:162` — button "View all X products" (handler default no-op)
- `SuppliersTable.svelte:45` — prop `onviewproducts` dideklarasikan tapi tidak ada UI atau wiring

**Maksud/Tujuan:** Navigasi dari supplier detail drawer atau table row ke halaman Products yang sudah ter-filter berdasarkan supplier.

**Status:** Button ter-render dan bisa diklik, tapi handler default-nya no-op `() => {}`. Parent (`SuppliersPage.svelte:254-262`) tidak pass prop `onviewproducts`.

### Temuan 2: `onduplicate` prop di SuppliersTable

**Lokasi:** `SuppliersTable.svelte:44`

**Maksud/Tujuan:** Fitur duplikasi supplier — user bisa mengklik tombol "Duplikasi" di table action menu untuk membuat salinan supplier dengan nama "X (Copy)", lalu langsung membuka form modal untuk diedit sebelum disimpan.

**Status:** Fungsi `duplicateSupplier()` sudah lengkap di parent (`SuppliersPage.svelte:186-190`), prop sudah di-pass ke table (`line 221`), tapi button tidak pernah ditambahkan ke template. Di Customer Groups, fitur serupa sudah jalan (`CustomerGroupsTable.svelte:226`).

---

## Rencana Perbaikan

### Temuan 1 & 3: Navigasi ke ProductsPage dengan `?supplier_id=X`

| Layer | File | Perubahan |
|---|---|---|
| Backend handler | `internal/product/handler.go` | Parse `supplier_id` dari query param, pass ke service |
| Backend service | `internal/product/service.go` | Tambah param `supplierID *int` ke `GetAllProducts()` |
| Backend repository | `internal/product/query.go` | Tambah filter `AND EXISTS (SELECT 1 FROM product_suppliers WHERE product_id = v.id AND supplier_id = $N)` |
| Backend tests | `internal/product/query_test.go` | Update tests untuk filter supplier_id |
| Frontend ProductsPage | `web/src/modules/product/components/ProductsPage.svelte` | Baca `supplier_id` dari URL di `onMount`, kirim ke backend |
| Frontend ProductsToolbar | `web/src/modules/product/components/ProductFiltersToolbar.svelte` | Tampilkan chip filter "Supplier: X" |
| Frontend SuppliersPage | `web/src/modules/supplier/components/SuppliersPage.svelte` | Pass `onviewproducts` ke both table + drawer, navigate ke `/inventory/products?supplier_id=X` |
| Frontend SuppliersTable | `web/src/modules/supplier/components/SuppliersTable.svelte` | Hapus prop `onviewproducts` yang mati |
| Frontend SupplierDetailDrawer | `web/src/modules/supplier/components/SupplierDetailDrawer.svelte` | Button sudah ada, wire ke parent |

### Temuan 2: Tombol "Duplikasi" di SuppliersTable

| File | Perubahan |
|---|---|
| `SuppliersTable.svelte` | Import `Copy` icon, tambah button "Duplikasi" di action area (ikuti pola CustomerGroupsTable line 226) |

---

## File yang berubah

| File | Temuan |
|---|---|
| `internal/product/handler.go` | 1&3 |
| `internal/product/service.go` | 1&3 |
| `internal/product/query.go` | 1&3 |
| `internal/product/query_test.go` | 1&3 |
| `web/src/modules/product/components/ProductsPage.svelte` | 1&3 |
| `web/src/modules/product/components/ProductFiltersToolbar.svelte` | 1&3 |
| `web/src/modules/supplier/components/SuppliersPage.svelte` | 1&3 |
| `web/src/modules/supplier/components/SuppliersTable.svelte` | 1&3 + 2 |
| `web/src/modules/supplier/components/SupplierDetailDrawer.svelte` | 1&3 |
