# Plan: Tampilkan Info Supplier di Products Page

## Problem

Products page (`/inventory/products`) tidak menampilkan info supplier di tabel.
Backend `GetAllProducts` query tidak SELECT data supplier, Go struct `Product` tidak punya
supplier fields, dan `ProductTable.svelte` tidak ada kolom supplier.

User ingin melihat supplier info saat browse products, terutama saat navigate dari
Suppliers page (dengan filter `?supplier_id=X`).

## Context

### Data Model

```
products ─< product_suppliers >─ suppliers
```

- Relationship: **many-to-many** via `product_suppliers`
- Satu produk bisa punya banyak supplier
- Constraint: **maks 1 preferred supplier** per produk (partial unique index `WHERE is_preferred = true`)
- Index: `idx_product_suppliers_product ON product_suppliers(product_id)` — sudah ada

### Performance Consideration

Produk bisa punya N supplier. Kalau ditampilkan semua di tabel, row bisa sangat panjang.
Strategi: tampilkan **preferred supplier saja** (atau supplier pertama jika tidak ada preferred).

Dengan partial unique index `idx_product_suppliers_one_preferred`, LEFT JOIN
`WHERE is_preferred = true` sangat efektif —最多 1 row per produk.

## Strategi

### Backend

**Tidak** modify `v_products_full` view (digunakan banyak query lain).

Tambahkan LEFT JOIN langsung di `GetAllProducts` query:

```sql
-- Tambahkan ke SELECT:
COALESCE(ps_preferred.supplier_name, '') as supplier_name,
COALESCE(ps_preferred.supplier_id, 0) as supplier_id

-- Tambahkan JOIN:
LEFT JOIN LATERAL (
    SELECT s.name as supplier_name, ps.supplier_id
    FROM product_suppliers ps
    JOIN suppliers s ON ps.supplier_id = s.id AND s.deleted_at IS NULL
    WHERE ps.product_id = v.id AND ps.is_preferred = true
    LIMIT 1
) ps_preferred ON true
```

**Mengapa LATERAL JOIN + LIMIT 1:**
- Lebih efektif daripada subquery biasa karena PostgreSQL bisa "push down" filter
- Partial unique index `WHERE is_preferred = true` memastikan max 1 row → LIMIT 1 basically free
- Tidak ada N+1 karena semua data diambil dalam 1 query

**Alternatif (simpler):** Aggregated subquery tanpa LATERAL:

```sql
LEFT JOIN (
    SELECT DISTINCT ON (ps.product_id)
        ps.product_id, s.name as supplier_name, s.id as supplier_id
    FROM product_suppliers ps
    JOIN suppliers s ON ps.supplier_id = s.id AND s.deleted_at IS NULL
    WHERE ps.product_id = v.id AND ps.is_preferred = true
) ps_preferred ON true
```

**Rekomendasi:** Gunakan LATERAL karena lebih bersih dan efisien untuk case ini.

### Frontend

1. **Product type** (`types/index.ts`): tambah `supplier_name?: string`, `supplier_id?: number`
2. **ProductTable** (`ProductTable.svelte`): tambah kolom "SUPPLIER" antara BRAND dan UOM
   - Tampilkan `product.supplier_name || '—'`
   - Kolom width: `w-36` (cukup untuk nama supplier pendek)
   - Tidak perlu sortable (supplier info次要)
3. **Skeleton**: tambah skeleton kolom supplier di loading state

### Scope

- Hanya tampilkan **nama supplier** (preferred)
- Filter `?supplier_id=X` sudah bekerja (dari pekerjaan sebelumnya)
- Tidak perlu show semua supplier per produk di tabel (detail-nya di SupplierDetailDrawer)

## Files to Modify

### Backend
- `internal/product/domain.go` — tambah `SupplierID *int`, `SupplierName *string` ke Product struct
- `internal/product/query.go` — tambah LEFT JOIN LATERAL di `GetAllProducts` COUNT + SELECT queries
- `internal/product/repository.go` — update `scanProductFromRow` untuk scan supplier fields

### Frontend
- `web/src/modules/product/types/index.ts` — tambah `supplier_name?`, `supplier_id?`
- `web/src/modules/product/components/ProductTable.svelte` — tambah kolom SUPPLIER

### Tests
- `internal/product/repository_mock_test.go` — update mock untuk supplier fields
- `internal/product/service_mock_test.go` — tidak perlu (service tidak berubah)
- `internal/product/handler_mock_test.go` — tidak perlu (handler tidak berubah)
- `internal/product/repository_test.go` — tambah test case untuk supplier field

## Verification

1. `go build ./...`
2. `go test -p 1 -count=1 ./internal/product/...`
3. `npx svelte-check --tsconfig ./tsconfig.json`
4. `npx vitest run src/modules/product/`
5. Manual: navigate dari Suppliers → Products, verify supplier name tampil di kolom
