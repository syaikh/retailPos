# Brand & Unit of Measure Management Plan

## Ringkasan

Membangun halaman management untuk Brand dan Unit of Measure (UOM) mengikuti pola
yang sudah ada pada `CategoriesPage`. Backend brand sudah siap, backend UOM masih
perlu ditambahkan CRUD.

---

## Layer 1 — Backend

### Brands ✅ (backend sudah siap)

Tidak perlu perubahan backend. Yang sudah ada:
- `POST /api/brands` → `CreateBrand`
- `PUT /api/brands/:id` → `UpdateBrand`
- `DELETE /api/brands/:id` → `DeleteBrand`
- `GET /api/brands` → `GetBrands` (public, tanpa pagination)

**Catatan**: `GET /api/brands` saat ini public tanpa pagination/search.
Untuk halaman management, ada dua opsi:
  1. **Gunakan endpoint yang sudah ada** — frontend handle search & pagination
     client-side. Sederhana, cocok karena data brand biasanya sedikit (< 50).
  2. **Buat endpoint baru** `/api/brands/manage` dengan pagination & search
     (seperti `/api/categories/manage`). Lebih scalable.

> **Rekomendasi**: Opsi 1 dulu. Upgrade ke opsi 2 jika diperlukan nanti.

### Units of Measure ❌ (perlu tambahan backend)

#### a. Repository (`internal/repository/postgres_repository.go`)

Tambah 3 method baru di `postgresRepository`:

```go
func (r *postgresRepository) CreateUnitOfMeasure(ctx context.Context, uom *domain.UnitOfMeasure) error
func (r *postgresRepository) UpdateUnitOfMeasure(ctx context.Context, uom *domain.UnitOfMeasure) error
func (r *postgresRepository) DeleteUnitOfMeasure(ctx context.Context, id int) error
```

Implementasi mengikuti pola `CreateBrand`/`UpdateBrand`/`DeleteBrand`.

#### b. Interface (`internal/repository/repository.go`)

Tambah 3 method di `ProductRepository` interface:

```go
CreateUnitOfMeasure(ctx context.Context, uom *domain.UnitOfMeasure) error
UpdateUnitOfMeasure(ctx context.Context, uom *domain.UnitOfMeasure) error
DeleteUnitOfMeasure(ctx context.Context, id int) error
```

#### c. Handler (`internal/delivery/http/handler/handler.go`)

Tambah 3 handler baru:

```go
func (h *Handler) CreateUnitOfMeasure(c *gin.Context)
func (h *Handler) UpdateUnitOfMeasure(c *gin.Context)
func (h *Handler) DeleteUnitOfMeasure(c *gin.Context)
```

Mengikuti pola `CreateBrand`/`UpdateBrand`/`DeleteBrand` yang sudah ada.

#### d. Routes (`cmd/server/main.go`)

Tambah di grup `protected`:

```go
protected.POST("/units-of-measure", middleware.RequirePermission("product:update"), h.CreateUnitOfMeasure)
protected.PUT("/units-of-measure/:id", middleware.RequirePermission("product:update"), h.UpdateUnitOfMeasure)
protected.DELETE("/units-of-measure/:id", middleware.RequirePermission("product:delete"), h.DeleteUnitOfMeasure)
```

---

## Layer 2 — Frontend: Settings Module

Letakkan brand & UOM management di module `settings` (sama seperti CategoriesPage).

### a. Types (`web/src/modules/settings/types/index.ts`)

Sesuaikan `MasterBrand` dan `MasterUnitOfMeasure` dengan response backend yang
sebenarnya, lalu tambah payload types:

```typescript
export interface MasterBrand {
  id: number;
  name: string;
  description?: string;
  is_active: boolean;
  product_count?: number;
  created_at?: string;
  updated_at?: string;
}

export interface MasterUnitOfMeasure {
  id: number;
  code: string;
  name: string;
  description?: string;
  is_active: boolean;
  created_at?: string;
}

export interface CreateBrandPayload { name: string; description?: string }
export interface UpdateBrandPayload { name?: string; description?: string; is_active?: boolean }
export interface CreateUnitOfMeasurePayload { code: string; name: string; description?: string }
export interface UpdateUnitOfMeasurePayload { code?: string; name?: string; description?: string; is_active?: boolean }
```

### b. Service Layer (`web/src/modules/settings/services/settings-service.ts`)

Tambah fungsi untuk brand & UOM, mengikuti pola yang sudah ada:

- `getBrands()`, `createBrand()`, `updateBrand()`, `deleteBrand()`
- `getUnitsOfMeasure()`, `createUnitOfMeasure()`, `updateUnitOfMeasure()`, `deleteUnitOfMeasure()`

### c. Halaman Pages (`web/src/modules/settings/components/`)

#### BrandsPage.svelte

- Kolom tabel: Name, Description, Product Count, Created, Actions
- Modal add/edit: Name (required), Description (optional), is_active toggle
  (edit only)
- Delete: proteksi jika masih dipakai produk

#### UnitsOfMeasurePage.svelte

- Kolom tabel: Code, Name, Description, Created, Actions
- Modal add/edit: Code (required, unique), Name (required), Description (optional)
- Delete: proteksi jika masih dipakai produk

Kedua halaman menggunakan:
- SearchBar, Button, Modal, Pagination, Skeleton dari `$shared/ui`
- `apiFetch` dari `$shared/api/http-client`
- `toast` dari `$shared/stores/toast`
- `formatDateInJakarta` dari `$shared/utils/jakartaTime`
- RBAC: create/edit/delete untuk `superadmin` dan `admin` saja

### d. Module Index (`web/src/modules/settings/index.ts`)

Export komponen dan fungsi baru.

---

## Layer 3 — Routing & Navigation

### a. Route Registration (`web/src/app/main.svelte`)

```typescript
// Tambah import
import BrandsPage from '$modules/settings/components/BrandsPage.svelte';
import UnitsOfMeasurePage from '$modules/settings/components/UnitsOfMeasurePage.svelte';

// di pageTitles
'/admin/brands': 'Brand Management',
'/admin/units-of-measure': 'Unit of Measure Management',

// di getComponent
case '/admin/brands': return BrandsPage;
case '/admin/units-of-measure': return UnitsOfMeasurePage;
```

### b. Sidebar (`web/src/app/layouts/Sidebar.svelte`)

Tambah sub-item di grup Master Data:

```typescript
const masterDataSubItems = [
  { label: 'Products',   href: '/inventory/products', icon: Package },
  { label: 'Categories', href: '/categories',          icon: Tag },
  { label: 'Brands',     href: '/admin/brands',        icon: Building2 },
  { label: 'Units',      href: '/admin/units-of-measure', icon: Ruler },
  { label: 'Customers',  href: '/customers',           icon: User },
];
```

Update `isMasterDataPath` untuk mencakup path baru.

---

## Urutan Implementasi

| Step | Apa | Estimasi |
|------|-----|----------|
| 1 | Backend: Repository + Interface untuk UOM CRUD | ~30 menit |
| 2 | Backend: Handler + Routes untuk UOM CRUD | ~20 menit |
| 3 | Frontend: Update types & service functions | ~15 menit |
| 4 | Frontend: BrandsPage.svelte | ~45 menit |
| 5 | Frontend: UnitsOfMeasurePage.svelte | ~45 menit |
| 6 | Frontend: Route + Sidebar integration | ~15 menit |
| 7 | Build & test | ~10 menit |
| **Total** | | **~3 jam** |

---

## Catatan

1. **Brands GET tanpa pagination** — Data brand biasanya sedikit (< 50), jadi
   client-side search cukup. Jika brand mencapai ratusan, migrasi ke endpoint
   manage dengan pagination.
2. **Hak akses UOM** — Saat ini menggunakan permission `product:update` dan
   `product:delete` seperti brand. Jika perlu permission khusus nanti, bisa
   ditambahkan `settings:manage`.
3. **Ikon sidebar** — `Building2` dan `Ruler` dari lucide-svelte perlu
   diimport. Opsi lain: `Tags` untuk brand, `Scale` untuk UOM.
4. **Audit log** — Brand & UOM handler saat ini tidak mencatat audit (berbeda
   dengan Category). Bisa ditambahkan jika diperlukan.
