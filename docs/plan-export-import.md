# Export & Import Bulk Action untuk Master Data

## Ringkasan

Menambahkan fitur export (CSV/XLSX) dan import (CSV) untuk master data guna memudahkan privileged user (superadmin & admin) dalam pengelolaan data secara massal.

## Master Data yang Ditangani

| Modul | Backend Path | Frontend Page |
|---|---|---|
| Products | `internal/product/` | `ProductsPage.svelte` |
| Categories | `internal/category/` | `CategoriesPage.svelte` |
| Brands | `internal/product/` (via brand routes) | `BrandsPage.svelte` |
| Units of Measure | `internal/product/` (via UOM routes) | `UnitsOfMeasurePage.svelte` |
| Customers | `internal/customer/` | `CustomersPage.svelte` |

---

## 1. Permission Baru

**File**: `database/migrations/019_add_export_import_permissions.sql`

Tambah permission baru:
- `product:export`, `product:import`
- `category:export`, `category:import`
- `customer:export`, `customer:import`

Assign ke role **superadmin** (id=1) dan **admin** (id=2). Brands/UOM ikut permission `product:export` / `product:import` karena satu handler.

---

## 2. Backend Shared Utility

**File baru**: `internal/importutil/parser.go`

- `ParseCSV(io.Reader) (headers []string, rows [][]string, err error)` — parse CSV dengan header row
- Struct `ImportResult { Inserted, Updated int, Errors []string }` — tipe return standar

---

## 3. Backend — Export

Setiap modul mendapat endpoint export mengikuti pola `audit/handler.go` (excelize untuk XLSX, encoding/csv untuk CSV).

### Products
| Layer | Detail |
|---|---|
| Route | `GET /api/products/export?format=csv\|xlsx` — `perm("product:export")` |
| Handler | `ExportProducts(c)` — fetch via `GetAllProductsForExport`, tulis CSV/XLSX |
| Service | `GetAllProductsForExport(ctx) ([]Product, error)` — tanpa pagination, include category/brand name |
| Columns | SKU, Name, Barcode, Category, Brand, Price, Cost, Stock, Status, TaxClass, UOM, Weight (grams), Description |

### Categories
| Route | `GET /api/categories/export?format=csv\|xlsx` — `perm("category:export")` |
|---|---|
| Columns | Name, Slug, Description, IsActive |

### Brands
| Route | `GET /api/brands/export?format=csv\|xlsx` — `perm("product:export")` |
|---|---|
| Columns | Name, Description, IsActive |

### Units of Measure
| Route | `GET /api/units-of-measure/export?format=csv\|xlsx` — `perm("product:export")` |
|---|---|
| Columns | Code, Name, Description, IsActive |

### Customers
| Route | `GET /api/customers/export?format=csv\|xlsx` — `perm("customer:export")` |
|---|---|
| Columns | Name, Phone, Email, Address, Note, IsActive |

---

## 4. Backend — Import

Semua import via **CSV only**, multipart/form-data field `file`. Operasi dalam **transaksi database**.

Response: `{ inserted: int, updated: int, errors: []string }`

### Products
| Layer | Detail |
|---|---|
| Route | `POST /api/products/import` — `perm("product:import")` |
| Handler | `ImportProducts(c)` — baca CSV → parse → resolve ref → validasi → bulk upsert |
| Service | `ImportProducts(ctx, records) (ImportResult, error)` — auto-resolve category/brand/UOM by name + auto-create |
| Repository | `BulkUpsertProducts(ctx, rows) (int, int, error)` — batch INSERT ON CONFLICT (sku) DO UPDATE |
| Validasi | name & SKU wajib, price > 0, stock >= 0, status valid |

### Categories
| Route | `POST /api/categories/import` — `perm("category:import")` |
|---|---|
| Validasi | name wajib |
| Repository | INSERT ON CONFLICT (name) DO NOTHING |

### Brands
| Route | `POST /api/brands/import` — `perm("product:import")` |
|---|---|
| Validasi | name wajib |
| Repository | INSERT ON CONFLICT (name) DO NOTHING |

### Units of Measure
| Route | `POST /api/units-of-measure/import` — `perm("product:import")` |
|---|---|
| Validasi | code & name wajib |
| Repository | INSERT ON CONFLICT (code) DO NOTHING |

### Customers
| Route | `POST /api/customers/import` — `perm("customer:import")` |
|---|---|
| Validasi | name, phone, email wajib |
| Repository | INSERT ON CONFLICT (email) DO UPDATE |

**Strategi referensi (product import)**: category by name, brand by name, UOM by code — auto-create jika belum ada, dalam satu transaksi.

---

## 5. Frontend — Service Layer

Tambah method di masing-masing service file:

| File | Tambahan Method |
|---|---|
| `web/src/modules/product/services/product-service.ts` | `exportProducts(format)`, `importProducts(file)` |
| `web/src/modules/settings/services/settings-service.ts` | `exportCategories(format)`, `importCategories(file)`, `exportBrands(format)`, `importBrands(file)`, `exportUnitsOfMeasure(format)`, `importUnitsOfMeasure(file)` |
| `web/src/modules/customers/services/customer-service.ts` | `exportCustomers(format)`, `importCustomers(file)` |

Export: download via blob response. Import: POST dengan FormData.

---

## 6. Frontend — Shared ImportModal Component

**File baru**: `web/src/shared/ui/ImportModal.svelte`

Props:
- `show: boolean`
- `title: string`
- `onImport: (file: File) => Promise<ImportResult>`
- `templateHeaders: string[]`

Features:
- Drag-and-drop / file picker (.csv)
- Preview tabel 10 baris pertama
- Tombol Import dengan loading state
- Display hasil: inserted count, updated count, error list

---

## 7. Frontend — Integrasi per Halaman

Di toolbar masing-masing halaman (ProductsPage, CategoriesPage, BrandsPage, UnitsOfMeasurePage, CustomersPage):
- Tombol **Export** → dropdown CSV / XLSX → trigger download
- Tombol **Import** → buka ImportModal
- Semua protected oleh permission `{entity}:export` / `{entity}:import`

---

## 8. Urutan Implementasi

| # | Step | Files |
|---|---|---|
| 1 | Migration permissions | `019_add_export_import_permissions.sql` |
| 2 | Shared CSV parser + ImportResult type | `internal/importutil/parser.go` |
| 3 | Category export + import | `internal/category/handler.go`, `service.go`, `repository.go` |
| 4 | Brand export + import | `internal/product/handler.go`, `service.go`, `repository.go` |
| 5 | UOM export + import | `internal/product/handler.go`, `service.go`, `repository.go` |
| 6 | Customer export + import | `internal/customer/handler.go`, `service.go`, `repository.go` |
| 7 | Product export + import | `internal/product/handler.go`, `service.go`, `repository.go` |
| 8 | Frontend shared ImportModal | `web/src/shared/ui/ImportModal.svelte` |
| 9 | Frontend services + page integration | Semua service & page component |
| 10 | Build & test | `go build ./...`, `go test ./...`, `npm run build` |
