# TDD: Price Consistency During Active Transactions

**Technical Design Document** — Implementasi backend Go (Clean Architecture), frontend Svelte 5, migrasi PostgreSQL, API, dan event flow.

| Field      | Nilai |
|------------|-------|
| Status     | Draft (menunggu review) |
| Tanggal    | 2026-07-31 |
| Basis desain | `docs/adr/ADR_Price_Consistency_During_Active_Transactions.md` (Server-Authorized Snapshot) |
| Sumber bisnis | `docs/adr/BDR_Price_Consistency_During_Active_Transactions.txt`, `docs/adr/Business_Process_Product_Price_Changes_During_Active_Transactions.txt` |

---

## 1. Gambaran Umum

Desain ini mengubah cara sistem memperoleh harga item dari **resolve-every-mutation** menjadi **resolve-once-per-item-insertion**.

- Backend menjadi *source of truth*: snapshot harga dibuat, dihitung, dan dipersistensikan oleh server **saat item ditambahkan** ke transaksi.
- Checkout, hold, resume, dan perubahan quantity **tidak** memanggil pricing engine.
- Transaksi aktif disimpan server-side dalam `cart_sessions`/`cart_items` (immutable per item), menggantikan ketergantungan pada state keranjang murni di frontend.

Alur tingkat tinggi:

```
Kasir add item → POST /api/pos/cart/:id/items
                 ├─ baca master data saat ini
                 ├─ pricing engine resolve 1× (ResolveSnapshot)
                 ├─ persist snapshot ke cart_items (immutable)
                 └─ kembalikan snapshot ke frontend
Kasir checkout → POST /api/pos/cart/:id/checkout
                 ├─ lock cart (FOR UPDATE)
                 ├─ validasi payment terhadap total snapshot
                 ├─ cek & kurangi stok
                 ├─ salin cart_items → sale_items (verbatim)
                 └─ publish sale.created
```

---

## 2. Arsitektur & Layer (Clean Architecture)

```
├── internal/pricing        (domain pricing, resolver, repository)
│     ├── domain.go          → PriceResolver, ResolveContext, ResolvedPrice
│     ├── resolver.go        → Resolver (algoritma 8-langkah deterministik)
│     └── repository.go      → akses master data (base price, scope, active rules)
├── internal/sale            (domain transaksi & keranjang)
│     ├── domain.go          → Sale, SaleItem, CartSession, CartItem, PriceSnapshot
│     ├── service.go         → Service (cart ops + checkout), processSaleItems → validasi
│     ├── repository.go      → Repository (SQL)
│     └── handler.go         → HTTP handlers + routes (sale + pos cart)
├── internal/shared          (response, timezone Jakarta, event bus)
└── internal/wiring/wiring.go (dependency injection)
```

Prinsip:

- **Domain layer** tidak bergantung pada HTTP atau driver DB; hanya berisi entitas, value object, dan error.
- **Service layer** berisi business rules (snapshot, immutability, qty/void/hold/resume).
- **Repository layer** mengakses PostgreSQL.
- **Handler layer** memetakan HTTP ↔ service.

---

## 3. Migrasi Database: `database/migrations/010_sale_price_snapshot.sql`

> Nomor migrasi berikutnya setelah `009_add_do_sequence.sql`. Mengikuti konvensi migrasi forward-only yang ada.

### 3.1 Tabel `cart_sessions`

Keranjang aktif / transaksi yang sedang berlangsung (termasuk yang di-hold).

```sql
CREATE TABLE IF NOT EXISTS cart_sessions (
    id             SERIAL PRIMARY KEY,
    cashier_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id       INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    shift_id       INTEGER REFERENCES shifts(id),
    customer_id    INTEGER REFERENCES customers(id),
    status         VARCHAR(20) NOT NULL DEFAULT 'open'
                   CHECK (status IN ('open', 'held', 'checked_out', 'cancelled', 'expired')),
    subtotal       INTEGER NOT NULL DEFAULT 0,
    discount       INTEGER NOT NULL DEFAULT 0,
    tax            INTEGER NOT NULL DEFAULT 0,
    total_amount   INTEGER NOT NULL DEFAULT 0,
    expired_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cart_sessions_cashier_status ON cart_sessions(cashier_id, status);
CREATE INDEX idx_cart_sessions_shift ON cart_sessions(shift_id);
```

Catatan:

- `status='open'` → keranjang aktif; `'held'` → di-hold; `'checked_out'` → sudah menjadi sale; `'cancelled'` → dibatalkan; `'expired'` → melewati masa berlaku hold.
- `expired_at` diisi saat hold (kebijakan TTL, default 24 jam). Untuk transaksi non-hold, `expired_at` NULL.
- **Satu cashier hanya boleh memiliki satu `cart_sessions` dengan status `'open'`** pada satu waktu (partial unique index di bawah).

```sql
CREATE UNIQUE INDEX uq_cart_sessions_open_cashier
    ON cart_sessions(cashier_id) WHERE status = 'open';
```

> **Pertimbangan:** jika nanti butuh lebih dari satu keranjang terbuka per kasir, indeks ini dihapus dan penegakan "satu open cart" dipindah ke service.

### 3.2 Tabel `cart_items`

Snapshot pricing per item. Satu baris = satu item dalam keranjang. **Immutable** kecuali `quantity`, `subtotal`, `dpp_amount`, `tax_amount`.

```sql
CREATE TABLE IF NOT EXISTS cart_items (
    id                  SERIAL PRIMARY KEY,
    cart_session_id     INTEGER NOT NULL REFERENCES cart_sessions(id) ON DELETE CASCADE,
    product_id          INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name        VARCHAR(200) NOT NULL,
    quantity            INTEGER NOT NULL CHECK (quantity > 0),
    unit_price          INTEGER NOT NULL CHECK (unit_price >= 0),
    original_price      INTEGER NOT NULL DEFAULT 0,
    discount            INTEGER NOT NULL DEFAULT 0,
    pricing_rule_id     INTEGER REFERENCES pricing_rules(id) ON DELETE SET NULL,
    pricing_rule_name   VARCHAR(200),
    pricing_rule_type   VARCHAR(50),
    pricing_type        VARCHAR(50),
    cost                INTEGER NOT NULL DEFAULT 0,
    tax_class_id        INTEGER REFERENCES tax_classes(id) ON DELETE SET NULL,
    tax_rate            NUMERIC(5,2),
    snapshot_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    subtotal            INTEGER NOT NULL DEFAULT 0,
    dpp_amount          INTEGER NOT NULL DEFAULT 0,
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_cart_item_subtotal CHECK (subtotal = quantity * unit_price)
);

CREATE INDEX idx_cart_items_session ON cart_items(cart_session_id);
```

Catatan:

- `product_name` disimpan agar tidak bergantung pada `JOIN products` (immutability penuh).
- `unit_price`, `original_price`, `discount`, `pricing_rule_*`, `pricing_type`, `cost`, `tax_class_id`, `tax_rate`, `snapshot_created_at` adalah **snapshot** — tidak pernah berubah setelah dibuat.
- Kolom yang berubah saat qty update: `quantity`, `subtotal`, `dpp_amount`, `tax_amount`, `updated_at`.

### 3.3 Perubahan `sale_items`

Menambahkan kolom snapshot agar `sale_items` merekam jejak harga penuh (tanpa mengubah kolom yang sudah ada).

```sql
ALTER TABLE sale_items
    ADD COLUMN IF NOT EXISTS cost                INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_class_id        INTEGER REFERENCES tax_classes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS tax_rate            NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS snapshot_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS product_name        VARCHAR(200);
```

> Kolom `product_name` diisi saat pembuatan sale. Untuk data lama (backfill), jalankan:
> `UPDATE sale_items si SET product_name = p.name FROM products p WHERE si.product_id = p.id AND si.product_name IS NULL;`

### 3.4 Kebijakan Hold Expiration (opsional, fase-1 direkomendasikan)

- Konfigurasi: env `CART_HOLD_TTL_HOURS` (default `24`), dibaca via `internal/config`.
- Pada `hold`: `expired_at = NOW() + interval '1 hour' * $CART_HOLD_TTL_HOURS`.
- Pada `resume`/`checkout`: jika `status='held'` dan `expired_at < NOW()` → ubah status `'expired'`, kembalikan error 410 (Gone). Item dapat di-pindah ke cart baru bila diinginkan kasir.
- Background job penanda `expired` bersifat opsional (lazy check cukup untuk fase pertama).

---

## 4. Domain & Value Objects (Go)

### 4.1 `internal/pricing/domain.go` — tambahan

Value object snapshot hasil resolve, memperluas `ResolvedPrice` dengan info cost & tax:

```go
// PriceSnapshot adalah hasil resolve yang immutable untuk satu item.
type PriceSnapshot struct {
    ProductID     int
    ProductName   string
    UnitPrice     int
    OriginalPrice int
    Discount      int
    PricingType   PricingType
    PricingMethod PricingMethod
    Rule          *PricingRule
    Cost          int
    TaxClassID    *int
    TaxRate       *float64
    SnapshotAt    time.Time
}
```

Perluasan interface `PriceResolver`:

```go
type PriceResolver interface {
    Resolve(ctx context.Context, rc ResolveContext) (*ResolvedPrice, error)
    ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error)
    // Baru:
    ResolveSnapshot(ctx context.Context, rc ResolveContext) (*PriceSnapshot, error)
    ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error)
}
```

### 4.2 `internal/pricing/resolver.go` — implementasi

- Tambah dependency ke `ResolverRepo` untuk membaca cost & tax produk:

```go
type ResolverRepo interface {
    // ... interface yang sudah ada (GetBasePrice, GetProductScope, GetActiveRules, dst.)
    GetProductCostAndTax(ctx context.Context, productID int) (cost int, taxClassID *int, taxRate *float64, err error)
    GetProductCostAndTaxBatch(ctx context.Context, productIDs []int) (map[int]ProductCostTax, error)
}
```

- `ResolveSnapshot` memanggil alur yang sama dengan `Resolve` (base price → scope → active rules → `resolvePricing`) kemudian menambahkan `Cost`, `TaxClassID`, `TaxRate`, `SnapshotAt: time.Now().In(shared.JakartaLocation())`, dan `ProductName` (dari repo atau argumen).
- **Penting**: `SnapshotAt` menggunakan waktu Jakarta (konsisten dgn `time.Now().In(shared.JakartaLocation())` yang sudah dipakai di `resolver.go:44`).

### 4.3 `internal/sale/domain.go` — entitas baru

```go
type CartSession struct {
    ID          int     `json:"id"`
    CashierID   int     `json:"cashier_id"`
    StoreID     *int    `json:"store_id,omitempty"`
    ShiftID     *int    `json:"shift_id,omitempty"`
    CustomerID  *int    `json:"customer_id,omitempty"`
    Status      string  `json:"status"`
    Subtotal    int     `json:"subtotal"`
    Discount    int     `json:"discount"`
    Tax         int     `json:"tax"`
    TotalAmount int     `json:"total_amount"`
    ExpiredAt   *string `json:"expired_at,omitempty"`
    Items       []CartItem `json:"items,omitempty"`
    CreatedAt   string  `json:"created_at,omitempty"`
    UpdatedAt   string  `json:"updated_at,omitempty"`
}

type CartItem struct {
    ID                int      `json:"id"`
    CartSessionID     int      `json:"cart_session_id"`
    ProductID         int      `json:"product_id"`
    ProductName       string   `json:"product_name"`
    Quantity          int      `json:"quantity"`
    UnitPrice         int      `json:"unit_price"`
    OriginalPrice     int      `json:"original_price"`
    Discount          int      `json:"discount"`
    PricingRuleID     *int     `json:"pricing_rule_id,omitempty"`
    PricingRuleName   *string  `json:"pricing_rule_name,omitempty"`
    PricingRuleType   *string  `json:"pricing_rule_type,omitempty"`
    PricingType       *string  `json:"pricing_type,omitempty"`
    Cost              int      `json:"cost"`
    TaxClassID        *int     `json:"tax_class_id,omitempty"`
    TaxRate           *float64 `json:"tax_rate,omitempty"`
    SnapshotCreatedAt string   `json:"snapshot_created_at,omitempty"`
    Subtotal          int      `json:"subtotal"`
    DPPAmount         int      `json:"dpp_amount"`
    TaxAmount         int      `json:"tax_amount"`
}
```

Perluasan `SaleItem` (menambah field yang sudah ada di DB setelah migrasi):

```go
type SaleItem struct {
    // field yang sudah ada ...
    Cost              int      `json:"cost,omitempty"`
    TaxClassID        *int     `json:"tax_class_id,omitempty"`
    TaxRate           *float64 `json:"tax_rate,omitempty"`
    SnapshotCreatedAt string   `json:"snapshot_created_at,omitempty"`
}
```

### 4.4 Error domain baru

```go
var (
    ErrCartNotFound         = errors.New("cart session not found")
    ErrCartNotOpen          = errors.New("cart session is not open")
    ErrCartExpired          = errors.New("cart session has expired")
    ErrCartItemNotFound     = errors.New("cart item not found")
    ErrCartItemQuantity     = errors.New("quantity must be greater than zero")
    ErrCartAlreadyCheckedOut = errors.New("cart session already checked out")
)
```

---

## 5. Alur Service (Use Case)

### UC-01 Create Cart Session

`Service.CreateCartSession(ctx, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error)`

1. Validasi tidak ada cart `'open'` untuk cashier (partial unique index menegakkan ini; tangani error `pgx.PgError` code `23505` → `ErrCartNotOpen` atau pesan ramah).
2. Insert `cart_sessions` status `'open'`.
3. Kembalikan entitas.

### UC-02 Add Item → Snapshot

`Service.AddCartItem(ctx, cartID int, productID, quantity int, customerGroupID, storeID *int) (*CartItem, *CartSession, error)`

1. `SELECT status FROM cart_sessions WHERE id=$1 FOR UPDATE` → jika bukan `'open'` → `ErrCartNotOpen`.
2. `qty := priceSnapshot.Quantity` — resolve snapshot via `pricing.PriceResolver.ResolveSnapshot`:
   - `ResolveContext{ProductID, Quantity, CustomerGroupID, StoreID}`.
   - Resolver membaca master data **pada saat itu** dan mengembalikan `PriceSnapshot`.
3. Hitung turunan:
   - `Subtotal = UnitPrice * Quantity`
   - `DPPAmount` dan `TaxAmount` dari `TaxRate` (rumus yang sama dengan POS saat ini):
     - `lineTotal = UnitPrice * Quantity`
     - `dpp = round(lineTotal * 100 / (100 + rate))`
     - `TaxAmount = lineTotal - dpp`
4. Insert `cart_items` dengan seluruh field snapshot.
5. `RecalcCartTotals`: `subtotal = Σ subtotal`, `discount = 0` (belum mendukung diskon level cart), `tax = Σ tax_amount`, `total_amount = subtotal - discount`.
6. Commit. Kembalikan `CartItem` + `CartSession` yang diperbarui.

> **Catatan penting:** satu item baru → satu snapshot baru. Item lama di keranjang **tidak** di-resolve ulang (BR-06).

### UC-03 Update Quantity

`Service.UpdateCartItemQuantity(ctx, cartID, itemID int, quantity int) (*CartItem, *CartSession, error)`

1. Validasi `quantity > 0` (jika `<= 0` → gunakan void).
2. `SELECT ... FOR UPDATE` pada cart; pastikan status `'open'`.
3. `UPDATE cart_items SET quantity=$n, subtotal=$qty*unit_price, dpp_amount=..., tax_amount=..., updated_at=NOW() WHERE id=$itemID AND cart_session_id=$cartID`.
   - **`unit_price`, `original_price`, `discount`, rule, `cost`, `tax_rate` tidak disentuh** (BR-07).
4. `RecalcCartTotals`.
5. Kembalikan item + cart yang diperbarui.

### UC-04 Void Item

`Service.RemoveCartItem(ctx, cartID, itemID int) (*CartSession, error)`

1. Pastikan cart `'open'`.
2. `DELETE FROM cart_items WHERE id=$itemID AND cart_session_id=$cartID`.
3. `RecalcCartTotals`.
4. Snapshot hilang; scan ulang → snapshot baru (BR-08).

### UC-05 Hold

`Service.HoldCart(ctx, cartID int) (*CartSession, error)`

1. Pastikan cart `'open'` (bisa juga dari `'held'` untuk perpanjang hold).
2. `UPDATE cart_sessions SET status='held', expired_at = NOW() + interval '1 hour' * $ttl, updated_at=NOW() WHERE id=$1`.
3. **Tidak** ada resolve ulang; item tetap seperti apa adanya (BR-05).

### UC-06 Resume

`Service.ResumeCart(ctx, cartID int) (*CartSession, error)`

1. `SELECT ... FROM cart_sessions WHERE id=$1 FOR UPDATE`.
2. Jika `status='expired'` atau (`status='held'` dan `expired_at < NOW()`) → set `status='expired'`, kembalikan `ErrCartExpired`.
3. Jika status `'open'` → idempoten, langsung kembalikan.
4. Jika `'held'` dan belum lewat → `UPDATE status='open', expired_at=NULL`.
5. Muat `cart_items` dan kembalikan. **Tanpa** re-resolve harga (BR-05).

### UC-07 Checkout (dari cart)

`Service.CheckoutCart(ctx, cartID int, payments []CreatePaymentRequest) (*Sale, error)`

Menggantikan panggilan pricing engine saat checkout.

1. `SELECT ... FROM cart_sessions WHERE id=$1 AND status IN ('open','held') FOR UPDATE`.
   - Tidak ditemukan → `ErrCartNotFound` / `ErrCartAlreadyCheckedOut`.
   - `'held'` dan lewat `expired_at` → `ErrCartExpired`.
2. Muat `cart_items` (snapshot immutable).
3. Hitung `subtotal`, `tax`, `total_amount` dari snapshot (bukan dari master data).
4. `validatePayments(ctx, total_amount, payments)` — tidak berubah dari implementasi saat ini (`internal/sale/service.go:186`).
5. **Cek & kurangi stok** (logika yang sudah ada, `processSaleItems` bagian stock) — menggunakan `product_id` + `quantity` dari `cart_items`.
6. Insert `sales` (status `'completed'`), lalu salin `cart_items` → `sale_items` **verbatim** (termasuk `product_name`, `cost`, `tax_rate`, `snapshot_created_at`).
7. Insert `sale_payments`, `UpdateShiftTotals`.
8. `UPDATE cart_sessions SET status='checked_out'`.
9. Commit. Publish `sale.created`.

**Pricing engine TIDAK dipanggil sama sekali di alur ini.**

### UC-08 Expired Cart

- Lazy: pada `ResumeCart`/`CheckoutCart`, cart `'held'` yang lewat `expired_at` ditandai `'expired'` dan ditolak.
- Opsional: job berkala `UPDATE cart_sessions SET status='expired' WHERE status='held' AND expired_at < NOW()`.

### UC-09 Direct Sale (backward compat)

`POST /api/sales` tetap ada untuk kompatibilitas API/import. Perilaku berubah:

- Jika payload menyertakan `cart_session_id` → alur `UC-07` (sumber item dari cart).
- Jika tanpa cart (legacy) → server **tidak** me-resolve harga; server melakukan **validasi konsistensi internal** pada item yang dikirim: `subtotal == unit_price * quantity`, `unit_price >= 0`, dan field rule konsisten. Harga klien disimpan verbatim.

> Alasan: mempertahankan kontrak `POST /sales` (dipakai integrasi lain), tetapi tidak lagi membiarkan server mengubah harga setelah transaksi terbentuk.

---

## 6. Perubahan `processSaleItems`

`internal/sale/service.go:49` dipecah menjadi dua tanggung jawab agar tidak lagi me-resolve harga:

1. **`validateCheckoutItems(ctx, tx, items []SaleItem)`** — hanya memvalidasi konsistensi snapshot yang masuk:
   - `quantity > 0`;
   - `unit_price >= 0`;
   - `subtotal == unit_price * quantity`;
   - (opsional) field pricing rule konsisten dengan `unit_price`.
2. **`deductStock(ctx, tx, items)`** — blok cek & pengurangan stok yang sudah ada (tidak berubah).

Blok `if s.resolver != nil { ... }` dan `else if s.priceStore != nil { ... }` di dalam `processSaleItems` **dihapus** untuk alur checkout cart. Resolver hanya dipanggil dari `AddCartItem`.

---

## 7. Repository & SQL (`internal/sale/repository.go`)

Metode baru:

| Metode | SQL inti |
|--------|----------|
| `CreateCartSession` | `INSERT INTO cart_sessions (cashier_id, store_id, shift_id, customer_id) VALUES (...) RETURNING id, ...` |
| `GetCartSessionByID` | `SELECT ... FROM cart_sessions WHERE id=$1` |
| `GetOpenCartByCashier` | `SELECT ... WHERE cashier_id=$1 AND status='open'` |
| `ListHeldCarts` | `SELECT ... WHERE cashier_id=$1 AND status='held' ORDER BY created_at DESC` |
| `InsertCartItem` | `INSERT INTO cart_items (...) VALUES (...)` |
| `GetCartItems` | `SELECT ... FROM cart_items WHERE cart_session_id=$1 ORDER BY id` |
| `UpdateCartItemQuantity` | `UPDATE cart_items SET quantity=$n, subtotal=$qty*unit_price, dpp_amount=..., tax_amount=..., updated_at=NOW() WHERE id=$i AND cart_session_id=$c` |
| `DeleteCartItem` | `DELETE FROM cart_items WHERE id=$i AND cart_session_id=$c` |
| `UpdateCartStatus` | `UPDATE cart_sessions SET status=$s, expired_at=$e, updated_at=NOW() WHERE id=$i` |
| `UpdateCartTotals` | `UPDATE cart_sessions SET subtotal=$a, discount=$b, tax=$c, total_amount=$d, updated_at=NOW() WHERE id=$i` |
| `LockCartSession` | `SELECT id, status, expired_at FROM cart_sessions WHERE id=$1 AND status IN ('open','held') FOR UPDATE` |

Tambahan pada `CreateSale` (repository) — kolom snapshot di `sale_items`:

```go
rows[i] = []interface{}{
    sale.ID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal,
    item.DPPAmount, item.TaxAmount,
    item.PricingRuleID, item.PricingRuleName, item.PricingRuleType, item.PricingType,
    origPrice,                       // original_price
    item.Cost,                       // cost
    item.TaxClassID,                 // tax_class_id
    item.TaxRate,                    // tax_rate
    time.Now(),                      // snapshot_created_at
    item.Name,                       // product_name
}
```

Penyesuaian `GetSaleByID`, `GetAllSales`, dan parked-sale query untuk membaca kolom baru (cost, tax_rate, snapshot_created_at, product_name) di JSON agregat item.

---

## 8. API Contracts

### 8.1 Daftar endpoint baru

Prefix seluruh route di bawah router `/api` (konsisten dengan frontend `apiClient`). Semua butuh auth; permission mengikuti pola yang ada (`sale.*`).

| Metode | Endpoint | Deskripsi | Permission |
|--------|----------|-----------|------------|
| POST | `/api/pos/cart` | Buat cart session | `sale.create` |
| GET | `/api/pos/cart` | List cart `'held'` milik cashier (query `?status=held`) | `sale.park` |
| GET | `/api/pos/cart/:id` | Ambil cart + items (untuk resume/reload) | `sale.view` |
| POST | `/api/pos/cart/:id/items` | **Add item → resolve + persist snapshot** | `sale.create` |
| PATCH | `/api/pos/cart/:id/items/:itemId` | Ubah quantity (harga beku) | `sale.create` |
| DELETE | `/api/pos/cart/:id/items/:itemId` | Void item | `sale.create` |
| POST | `/api/pos/cart/:id/hold` | Hold cart | `sale.park` |
| POST | `/api/pos/cart/:id/resume` | Resume cart (tanpa re-resolve) | `sale.park` |
| POST | `/api/pos/cart/:id/checkout` | Checkout (pakai snapshot, tanpa re-resolve) | `sale.create` |

Deprecated (tetap ada untuk kompatibilitas, tidak dipakai frontend POS baru):
- `POST /api/sales` — dukung `cart_session_id` opsional; tanpa cart → validasi konsistensi internal (UC-09).
- `POST /api/sales/parked`, `GET /api/sales/parked`, `POST /api/sales/parked/:id/recall`, `DELETE /api/sales/parked/:id` — dapat dipetakan ke cart hold/resume pada fase berikutnya; pada fase 1 dipertahankan apa adanya agar tidak memutus integrasi lama.

### 8.2 Contoh request/response

**POST `/api/pos/cart`**

```jsonc
// Request
{ "store_id": 1, "shift_id": 42, "customer_id": 7 }
// Response 201
{
  "data": {
    "id": 100,
    "cashier_id": 2,
    "store_id": 1,
    "shift_id": 42,
    "customer_id": 7,
    "status": "open",
    "subtotal": 0,
    "discount": 0,
    "tax": 0,
    "total_amount": 0,
    "items": [],
    "created_at": "2026-07-31T08:00:00+07:00"
  }
}
```

**POST `/api/pos/cart/100/items`**

```jsonc
// Request
{ "product_id": 55, "quantity": 2, "customer_group_id": 3 }
// Response 201 — snapshot immutable dikembalikan
{
  "data": {
    "id": 501,
    "cart_session_id": 100,
    "product_id": 55,
    "product_name": "Indomie Goreng",
    "quantity": 2,
    "unit_price": 3500,
    "original_price": 3500,
    "discount": 0,
    "pricing_rule_id": null,
    "pricing_type": "default",
    "cost": 2500,
    "tax_class_id": 1,
    "tax_rate": 11.0,
    "snapshot_created_at": "2026-07-31T08:05:12+07:00",
    "subtotal": 7000,
    "dpp_amount": 6306,
    "tax_amount": 694
  },
  "cart": { "id": 100, "subtotal": 7000, "tax": 694, "total_amount": 7000, "status": "open" }
}
```

**PATCH `/api/pos/cart/100/items/501`**

```jsonc
// Request
{ "quantity": 3 }
// Response 200 — unit_price tetap 3500; hanya quantity/subtotal yang berubah
{
  "data": { "id": 501, "product_id": 55, "quantity": 3, "unit_price": 3500, "subtotal": 10500, "tax_amount": 1041, "snapshot_created_at": "2026-07-31T08:05:12+07:00" },
  "cart": { "id": 100, "subtotal": 10500, "tax": 1041, "total_amount": 10500 }
}
```

**DELETE `/api/pos/cart/100/items/501`** → `204 No Content`

**POST `/api/pos/cart/100/hold`** → `200` `{ "data": { "id": 100, "status": "held", "expired_at": "2026-08-01T08:05:12+07:00", ... } }`

**POST `/api/pos/cart/100/resume`** → `200` dengan cart + items verbatim; **tanpa** field harga baru.

**POST `/api/pos/cart/100/checkout`**

```jsonc
// Request
{
  "payments": [
    { "payment_method_code": "CASH", "amount": 10500 }
  ]
}
// Response 201 — sale yang dibuat dari snapshot (items menampilkan snapshot)
{
  "data": {
    "id": 900,
    "invoice_number": "INV-2026-000900",
    "subtotal": 10500,
    "discount": 0,
    "tax": 1041,
    "total_amount": 10500,
    "status": "completed",
    "items": [ { "product_id": 55, "quantity": 3, "unit_price": 3500, "snapshot_created_at": "2026-07-31T08:05:12+07:00", ... } ]
  }
}
```

### 8.3 Status & error codes

| Kasus | HTTP | Kode/body |
|-------|------|-----------|
| Payload tidak valid | 400 | `{ "error": ... }` |
| Cart sudah di-checkout / tidak open | 409 | `ErrCartAlreadyCheckedOut` |
| Cart hold sudah kedaluwarsa | 410 | `ErrCartExpired` |
| Cart tidak ditemukan | 404 | `ErrCartNotFound` |
| Stok tidak cukup | 409 | `"insufficient stock"` (konsisten saat ini) |
| Payment tidak cocok | 400 | `ErrPaymentTotalMismatch` |

---

## 9. Concurrency & Race Condition

- **Double checkout**: dicegah dengan `SELECT ... FOR UPDATE` pada `cart_sessions` di `CheckoutCart` (pola yang sama dengan `CreateSaleWithParkedSale` di `internal/sale/service.go:385`). Baris cart dikunci hingga commit; checkout kedua gagal dengan status yang sudah `'checked_out'`.
- **Add vs Checkout**: `AddCartItem` juga mengunci baris cart (`FOR UPDATE`) sehingga tidak terjadi add setelah checkout dimulai.
- **Dua cart open per kasir**: partial unique index `uq_cart_sessions_open_cashier` + penanganan error `23505`.
- **Snapshot konsistensi**: `ResolveSnapshot` membaca master data dan rules dalam satu koneksi saat add; karena PostgreSQL default `READ COMMITTED`, snapshot merepresentasikan kondisi pada saat statement dijalankan — sesuai kebutuhan "saat item ditambahkan".

---

## 10. Frontend Svelte 5

### 10.1 Service layer — `web/src/modules/pos/services/pos-service.ts`

Fungsi baru (pakai `apiClient` yang ada):

```ts
export async function createCart(payload?: { store_id?: number; shift_id?: number; customer_id?: number }): Promise<CartSession>
export async function getHeldCarts(): Promise<CartSession[]>
export async function getCart(id: number): Promise<CartSession>
export async function addCartItem(cartId: number, item: { product_id: number; quantity: number; customer_group_id?: number }): Promise<{ data: CartItem; cart: CartSession }>
export async function updateCartItemQuantity(cartId: number, itemId: number, quantity: number): Promise<{ data: CartItem; cart: CartSession }>
export async function removeCartItem(cartId: number, itemId: number): Promise<void>
export async function holdCart(cartId: number): Promise<CartSession>
export async function resumeCart(cartId: number): Promise<CartSession>
export async function checkoutCart(cartId: number, payments: PaymentAllocation[]): Promise<Sale>
```

### 10.2 Types — `web/src/modules/pos/types/index.ts`

```ts
export interface CartItem {
  id: number;
  cart_session_id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  unit_price: number;        // snapshot — readonly untuk UI
  original_price: number;
  discount: number;
  pricing_rule_id?: number;
  pricing_rule_name?: string;
  pricing_rule_type?: string;
  pricing_type?: string;
  cost: number;
  tax_class_id?: number;
  tax_rate?: number;
  snapshot_created_at?: string;
  subtotal: number;
  dpp_amount: number;
  tax_amount: number;
}

export interface CartSession {
  id: number;
  cashier_id: number;
  store_id?: number;
  shift_id?: number;
  customer_id?: number;
  status: 'open' | 'held' | 'checked_out' | 'cancelled' | 'expired';
  subtotal: number;
  discount: number;
  tax: number;
  total_amount: number;
  expired_at?: string;
  items?: CartItem[];
  created_at?: string;
  updated_at?: string;
}
```

### 10.3 `PosPage.svelte`

Perubahan utama:

1. **Hapus** `resolveCartPrices()` dan pemanggilannya (di `addToCart`, `updateQty`, `removeFromCart`, `recallSale`, dan pada pemilihan customer).
2. Tambah state: `let activeCartId = $state<number | null>(null);` dan `let cartItems: CartItem[] = $state([]);` (menggantikan `cart: CartItem[]` lokal).
3. `addToCart(product)`:
   - jika `activeCartId` null → `createCart()` lalu simpan id;
   - `addCartItem(activeCartId, { product_id, quantity: 1, customer_group_id })`;
   - perbarui `cartItems` dari response; **jangan** resolve ulang.
4. `updateQty(id, delta)` → hitung qty baru → `updateCartItemQuantity(activeCartId, itemId, qty)` → perbarui dari response.
5. `removeFromCart(id)` → `removeCartItem(activeCartId, id)` → hapus dari list lokal.
6. `holdSale()` → `holdCart(activeCartId)` → reset `activeCartId`/`cartItems`.
7. `recallSale(cartId)` → `resumeCart(cartId)` → set `cartItems = cart.items` verbatim; **tanpa** `resolveCartPrices()`.
8. `processCheckout(payments)` → `checkoutCart(activeCartId, payments)` → sukses: reset state.
9. Derivasi totals diubah membaca dari response cart server (`cart.subtotal`, `cart.tax`, `cart.total_amount`) daripada menghitung ulang dari array lokal. Rumus `dppDisplay` tetap.
10. Ganti pelanggan (`CustomerSelectModal`) → panggil `PATCH /api/pos/cart/:id` (set `customer_id`) jika diperlukan; **tidak** me-resolve item lama.

### 10.4 Komponen lain

- **`CartPanel.svelte`**: tombol `+`/`-` memanggil `onupdateqty` (yang kini async ke server); tombol hapus → `onremovefromcart`. Tambahan UI opsional: label kecil "harga dibekukan" pada item dengan `snapshot_created_at`.
- **`ParkedSalesModal.svelte`**: daftar dari `getHeldCarts()`; tombol recall → `resumeCart`.
- **`CheckoutModal.svelte`**: tidak berubah secara kontrak; tetap menerima `payments` lalu diteruskan ke `checkoutCart`.

---

## 11. Event Flow

`internal/eventbus` (pub/sub in-memory) dipakai untuk integrasi real-time.

| Event | Trigger | Payload | Konsumen |
|-------|---------|---------|----------|
| `sale.created` | `CheckoutCart` (UC-07) | `*Sale` (dengan items snapshot) | Dashboard, laporan, websocket POS |
| `cart.checked_out` *(baru, observability)* | `CheckoutCart` | `cartID`, `saleID`, `cashierID` | Opsional: audit/notifikasi |
| `cart.held` *(baru, opsional)* | `HoldCart` | `cartID`, `cashierID` | Opsional |

Tidak ada event `price.changed` yang memicu re-resolve pada transaksi aktif — perubahan master data bersifat pasif dan hanya memengaruhi add item berikutnya.

---

## 12. Wiring (`internal/wiring/wiring.go`)

Tambahan registrasi:

```go
d.SaleSvc.SetPriceResolver(d.PricingResolver)  // sudah ada, tetap
// (opsional) d.PricingResolver kini mengimplementasikan ResolveSnapshot/ResolveSnapshotsBatch
d.SaleSvc.SetSnapshotResolver(d.PricingResolver) // setter baru untuk interface snapshot
```

Handler sale:

```go
// Route grup /api
d.SaleH.RegisterPosRoutes(api, auth, perm)
```

---

## 13. Timezone

- `snapshot_created_at`, `created_at`, `updated_at` di DB: `TIMESTAMPTZ` (UTC).
- Keluaran API: diformat ke Asia/Jakarta (`shared.JakartaLocation()`), pola yang sama dengan `internal/sale/repository.go:43`.
- `SnapshotAt` pada `ResolveSnapshot`: `time.Now().In(shared.JakartaLocation())`.

---

## 14. Data Dictionary (ringkas)

| Kolom | Tipe | Makna |
|-------|------|-------|
| `cart_sessions.status` | varchar | Lifecycle transaksi aktif: open → held/checked_out/cancelled/expired |
| `cart_items.unit_price` | int | Harga satuan **snapshot** (tidak berubah) |
| `cart_items.original_price` | int | Harga dasar produk saat snapshot (sebelum rule) |
| `cart_items.discount` | int | Diskon per unit dari pricing rule pada snapshot |
| `cart_items.cost` | int | Biaya produk saat snapshot (untuk margin/audit) |
| `cart_items.tax_rate` | numeric | Tarif pajak saat snapshot (persen) |
| `cart_items.snapshot_created_at` | timestamptz | Waktu pembuatan snapshot (Jakarta saat ditampilkan) |
| `sale_items.product_name` | varchar | Nama produk snapshot (imutabel) |

---

## 15. Backward Compatibility & Migration Plan

1. **Deploy order** (mengikuti kebijakan migrasi di AGENTS.md):
   - Terapkan `010_sale_price_snapshot.sql` sebelum binary baru.
   - Backfill `sale_items.product_name` untuk data lama.
2. **Rollout frontend**: deploy service cart + UI baru secara bersamaan; `POST /api/sales` lama tetap berfungsi untuk integrasi pihak ketiga.
3. **Non-regresi**: query lama (`GetSaleByID`, `GetAllSales`, parked sales) tetap mengembalikan data dengan kolom baru; unit test yang bergantung pada `processSaleItems` re-resolve diperbarui (lihat Test Spec).
