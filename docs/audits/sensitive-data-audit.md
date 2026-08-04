# Sprint 2 — Sensitive Data Audit: Cost Exposure di Sale & Cart

- **Status:** Implemented (2026-08-04) — fix aktif, regression suite hijau
- **Backlog source:** `docs/audits/permission-additions-sprint1.md` §Follow-up
- **Pola acuan:** `internal/product/presenter.go` (presenter/sanitizer + `product.cost.view`)
- **Sync source:** `database` (permissions table) → `internal/permissions/permissions.go` → `web/src/shared/constants/permissions.ts`

## Resolution

Implementasi sesuai dokumen ini (commit pending per kebijakan repo — hanya
eksekusi atas permintaan eksplisit):

- `internal/sale/presenter.go` (baru) — `canViewCost(c)` (gate
  `product.cost.view` via `ownership.CanAccessAll`), `presentSale`,
  `presentCart`, `presentCarts`. Cost di-shadow ke `*int json:"cost,omitempty"`
  sehingga field **di-omit**, bukan `null`, bagi non-holder.
- `internal/sale/handler.go` — `GetSaleByID` → `presentSale(sale, canViewCost(c))`.
- `internal/sale/cart_handler.go` — semua handler cart (termasuk `ListHeldCarts`,
  `CheckoutCart`) → `presentCart`/`presentCarts`/`presentSale`.
- `internal/sale/security_regression_test.go` (baru) — data-driven, memakai
  harness `internal/secregtest`; cashier: `cost` ABSENT + field non-sensitif
  VISIBLE; manager (+`product.cost.view`): `cost` VISIBLE.
- Verification: `go test -p 1 -count=1 ./...` (backend, seluruh paket) hijau;
  `npx vitest run` (1275 test) hijau; `npm run build` sukses.

## Executive Summary

Setelah Item 3 Sprint 1 meng-enforce `product.cost.view` pada `GET /products` dan
`GET /products/:id`, dua jalur lain yang mengekspos **purchase cost** ke cashier
masih terbuka di module `sale`:

1. `GET /sales/:id` — gate `sale.view` (cashier memegang) mengembalikan
   `'cost', si.cost` di dalam jsonb items.
2. `GET /pos/cart*` — gate `sale.create` (cashier memegang) mengembalikan
   `CartItem.Cost json:"cost"` (tanpa `omitempty` sekalipun).

Sprint ini memperbaiki kedua exposure tersebut dengan pola yang **sama persis**
dengan product: presenter/sanitizer di layer handler, gate `product.cost.view`.
Cashier tetap bisa melihat harga jual, subtotal, diskon, dan pajak — tapi tidak
bisa melihat purchase cost, margin internal, markup, atau keuntungan toko.

Selain fix, sprint ini membangun **security regression harness** (data-driven,
mendukung state `visible`/`null`/`absent`) sehingga exposure jenis ini terverifikasi
secara otomatis di CI dan fix berikutnya cukup menambah test case.

## Threat Model

**Aset yang dilindungi:** purchase cost & derivatifnya (margin, markup, keuntungan).
Ini adalah informasi bisnis sensitif: jika diketahui cashier/staff, mereka dapat
memprediksi margin toko dan memanfaatkannya di luar jalur resmi.

**Aktor:**
- Cashier — memegang `sale.view`, `sale.create`, `sale.park`, `product.view`.
  Berhak melihat harga jual & rincian transaksi, **tidak berhak** melihat cost.
- Staff — memegang `product.view`, `stock_opname.*`. Tidak berhak melihat cost.
- Manager — memegang `product.cost.view`. Berhak melihat cost.

**Kontrol yang sudah ada:**
- `product.cost.view` — permission yang mengontrol akses data cost (Sprint 1).
- `ownership.CanAccessAll` — helper reusable untuk memeriksa permission di handler.
- Presenter/sanitizer product — pola referensi enforcement.

**Serangan yang dicegah:** field injection / respons yang over-inclusive —
satu endpoint berbagi data sensitif ke konsumen yang tidak berhak (mis. melalui
API langsung, bukan UI).

## Endpoint Inventory

| Endpoint | Gate | Holder gate | Return cost saat ini | Kelas data |
| -------- | ---- | ----------- | -------------------- | ---------- |
| `GET /products` / `GET /products/:id` | `product.cost.view` (presenter) | SA/A/M | di-omit utk non-holder | product cost |
| `GET /sales` (history) | `sale.view` | SA/A/M/Cashier | **tidak** (query tidak select `si.cost`) | sale summary |
| `GET /sales/:id` | `sale.view` | SA/A/M/Cashier | **YA** (`'cost', si.cost` di jsonb) | sale detail |
| `GET /sales/export` | `report.view` | SA/A | tidak (hanya row export, tanpa cost) | export |
| `GET /sales/parked*` | `sale.park` | SA/A/M/Cashier | tidak (query kolom eksplisit tanpa cost) | parked sale |
| `GET /pos/cart` / `/pos/cart/:id` / `/pos/cart/held` | `sale.create` | SA/A/M/Cashier | **YA** (`CartItem.Cost`) | cart item |
| `POST /pos/cart` (create/get) | `sale.create` | SA/A/M/Cashier | **YA** | cart item |
| `POST /pos/cart/items`, `PATCH`, `DELETE`, `PATCH customer`, `POST hold/resume` | `sale.create` | SA/A/M/Cashier | **YA** (return cart) | cart item |
| `POST /pos/cart/:id/checkout` | `sale.create` | SA/A/M/Cashier | **YA** (return sale detail) | sale detail |
| `GET /purchase-orders*` | `purchase_order.view` | SA/A/M | `unit_cost` | PO (holder punya `product.cost.view`) |
| `GET /suppliers*` | `pricing.view` | SA/A/M | `unit_cost`, preferred supplier | supplier pricing (holder punya `product.cost.view`) |
| `GET /products/options` | (bebas) | Semua | tidak | id/sku/name |
| `GET /products/search` (pricing) | `pricing.view` | SA/A/M | tidak | id/name/sku/price |

## Sensitive Field Inventory

| Field | Tipe | Lokasi | Dikontrol `product.cost.view` |
| ----- | ---- | ------ | ----------------------------- |
| `Product.Cost` | `int` | `internal/product` | ✅ (Sprint 1, presenter) |
| `SaleItem.Cost` | `int json:"cost,omitempty"` | `internal/sale/domain.go` | ❌ (sprint ini) |
| `CartItem.Cost` | `int json:"cost"` | `internal/sale/cart.go` | ❌ (sprint ini) |
| `PurchaseItem.UnitCost` | `int json:"unit_cost"` | `internal/purchase` | n/a — gate `purchase_order.view` sudah = holder cost |
| `ProductSupplier.UnitCost` | `int` | `internal/supplier` | n/a — gate `pricing.view` sudah = holder cost |

Margin/markup/gross profit/COGS/inventory valuation **belum ada** sebagai field di
response manapun (verified grep, `internal/report`, `internal/inventory`) — tidak
ada exposure karena tidak ada data-nya.

## Existing Protection

- **Product:** `presentProduct` + `productWithoutCost` (field `Cost *int` shadow,
  `omitempty`, di-omit bukan `null`) — `internal/product/presenter.go`.
- **Sale list & parked:** query tidak men-select kolom `cost`, sehingga `Cost`
  tetap 0 dan ter-omit oleh `omitempty` — proteksi tidak disengaja, bukan desain.
- **PO & supplier:** gate permission sudah memastikan hanya holder
  `purchase_order.view`/`pricing.view` (SA/A/M) yang dapat mengakses — konsisten
  dengan holder `product.cost.view`.

## Confirmed Exposure

| Exposure | Severity | Endpoint | Bukti |
| -------- | -------- | -------- | ----- |
| `Product.Cost` | **Fixed** (Sprint 1) | `GET /products`, `GET /products/:id` | presenter `productWithoutCost` |
| `SaleItem.Cost` | **High** | `GET /sales/:id` | `internal/sale/repository.go:144` — `'cost', si.cost` dalam jsonb items |
| `CartItem.Cost` | **High** | `GET /pos/cart*` & mutasi cart | `internal/sale/cart.go:72` — `Cost int json:"cost"` |
| `SaleItem.Cost` (via checkout) | **High** | `POST /pos/cart/:id/checkout` | return `detail` dari `GetSaleByID` (sama dengan exposure detail) |
| `PurchaseItem.UnitCost` | **Accepted** | `GET /purchase-orders*` | gate `purchase_order.view` = SA/A/M = holder `product.cost.view` |
| `Supplier.UnitCost`/preferred | **Accepted** | `GET /suppliers*` | gate `pricing.view` = SA/A/M = holder `product.cost.view` |
| `SaleItem.Cost` (list/parked) | **Accepted** | `GET /sales`, `GET /sales/parked*` | query tidak select `cost`; zero → omit |

**Alasan severity High:** `sale.view` dan `sale.create` dipegang cashier. Melalui
sale detail / cart, cashier dapat menghitung margin toko per item — informasi
bisnis sensitif yang seharusnya hanya untuk holder `product.cost.view`.

## Planned Fix

Menerapkan pola yang sama persis dengan product, di layer handler:

```
Permission (product.cost.view)
        │
        ▼
Presenter / Sanitizer (presentSale / presentCart)
        │
        ▼
JSON Response (cost di-omit utk non-holder)
```

- **`presentSale(s *Sale, canViewCost bool) any`** — non-holder menerima
  `saleWithoutCost` (embed `Sale` + shadow `Items []saleItemWithoutCost`),
  item tanpa `cost`.
- **`presentCart(cart *CartSession, canViewCost bool) any`** — non-holder menerima
  `cartSessionWithoutCost` (embed `CartSession` + shadow `Items []cartItemWithoutCost`).
- **`presentCarts(carts []CartSession, canViewCost bool) any`** — untuk list held carts.
- **`canViewCost(c *gin.Context) bool`** — helper yang sama dengan product
  (`ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ProductCostView)`),
  di-refactor ke `internal/sale/presenter.go` (local, bukan shared package, karena
  pattern-nya sudah eksis di tiap module).

### Handler yang diubah

| Handler | File | Perubahan |
| ------- | ---- | --------- |
| `GetSaleByID` | `internal/sale/handler.go` | `shared.JSONSuccess(c, presentSale(sale, canViewCost(c)))` |
| `CheckoutCart` | `internal/sale/cart_handler.go` | `presentSale(detail, ...)` / `presentSale(sale, ...)` |
| `CreateOrGetOpenCart`, `GetOpenCart`, `GetCart`, `AddCartItem`, `UpdateCartItemQuantity`, `RemoveCartItem`, `UpdateCartCustomer`, `HoldCart`, `ResumeCart` | `internal/sale/cart_handler.go` | `presentCart(cart, canViewCost(c))` |
| `ListHeldCarts` | `internal/sale/cart_handler.go` | `presentCarts(carts, canViewCost(c))` |

**Keputusan desain:** field `cost` di-**omit** (bukan `null`) untuk non-holder,
konsisten dengan product — konsumen tidak bisa membedakan missing vs zero.
Pemegang `product.cost.view` menerima payload **identik** dengan sebelumnya
(no behavior change).

## Regression Test Strategy

**Security Regression Harness** — `internal/secregtest` (baru):

- `Field{Path, State}` dengan state **`visible` / `null` / `absent`** (bukan
  boolean), karena desain sanitizer bisa berbeda (omitted vs null).
- `Check(t, body, fields...)` — parse JSON, navigasi path dot-separated
  (`data.items.0.cost`; token numerik = index array), assert state tiap field.
- Constructor `secregtest.Visible(path)`, `secregtest.Null(path)`,
  `secregtest.Absent(path)` agar tabel test ringkas dan data-driven.

**Test case (data-driven):**

```go
{
  endpoint: "GET /sales/:id",
  role:     cashier (perms: sale.view),
  fields:   [product_name VISIBLE, quantity VISIBLE, unit_price VISIBLE,
             cost ABSENT, margin ABSENT]
},
{
  endpoint: "GET /sales/:id",
  role:     manager (perms: sale.view + product.cost.view),
  fields:   [cost VISIBLE]
},
{
  endpoint: "GET /pos/cart",
  role:     cashier (perms: sale.create),
  fields:   [product_name VISIBLE, unit_price VISIBLE, cost ABSENT]
},
{
  endpoint: "GET /pos/cart",
  role:     manager (perms: sale.create + product.cost.view),
  fields:   [cost VISIBLE]
}
```

File test: `internal/sale/security_regression_test.go` — mengisi data nyata
(product dengan cost > 0, sale item dengan cost > 0, cart item dengan cost > 0)
agar memastikan bahwa yang di-omit bukan karena zero-value.

## Out of Scope (sprint ini)

- **purchase** (`unit_cost`) — sudah terproteksi oleh `purchase_order.view`.
- **supplier** (pricing) — sudah terproteksi oleh `pricing.view`.
- **inventory valuation / stock value** — field belum ada.
- **report** (COGS, profit) — field belum ada.
- **Pencarian exposure baru** selain dua yang sudah terkonfirmasi di module `sale`.
- Frontend: tidak ada perubahan UI yang diperlukan — UI cashier sudah tidak
  merender `cost` (field cost hanya dipakai untuk perhitungan internal).

## Future Audit

- **Phase 2 — Margin/Profit:** audit saat field margin/markup/gross profit mulai
  ditambahkan ke response manapun (kemungkinan `internal/report`).
- **Phase 3 — Supplier:** audit lanjutan bila grant `pricing.view` berubah
  (mis. cashier diberi akses supplier) — saat itu `unit_cost` supplier harus
  digate `product.cost.view`.
- **Phase 4 — Financial:** audit saat COGS / inventory valuation / stock value
  diperkenalkan; pakai harness yang sama.
- **Backlog:** `SaleItem.Cost` di `sale_items` tetap disimpan (untuk analitik
  profit masa depan) — hanya exposure-nya yang digate, bukan penyimpanannya.
