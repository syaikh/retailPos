# Upcoming Features — Rencana Fitur Baru

Dokumen ini menjelaskan fitur-fitur baru yang akan ditambahkan ke Retail POS System, beserta urutan prioritas dan komponen yang perlu dibangun.

---

## Daftar Isi

1. [Shift Management](#1-shift-management)
2. [Hold & Recall Transaction (Parked Sales)](#2-hold--recall-transaction-parked-sales)
3. [Split Payment (Multi Payment Method)](#3-split-payment-multi-payment-method)
4. [Purchase Order & Goods Receiving](#4-purchase-order--goods-receiving)
5. [Stock Opname (Physical Count)](#5-stock-opname-physical-count)
6. [Store Management](#6-store-management)
7. [Storage Locations](#7-storage-locations)
8. [Time-based Pricing Update](#8-time-based-pricing-update)
9. [Admin Change Freeze During Active Shifts](#9-admin-change-freeze-during-active-shifts)
10. [Price Consistency During Active Transactions](#10-price-consistency-during-active-transactions)
11. [Konsinyasi Supplier](#11-konsinyasi-supplier)

---

## 1. Shift Management ✅

**Status:** SUDAH DIIMPLEMENTASI — `internal/shift/`, `web/src/modules/shifts/`

Sistem shift untuk kasir. Setiap kali kasir mulai kerja, mereka **open shift** — mencatat jumlah uang awal di kas (opening balance). Sepanjang shift, semua transaksi tercatat. Saat selesai, kasir **close shift** — menghitung uang fisik (closing balance) dan sistem menghitung selisihnya (discrepancy).

### Contoh Workflow

```
Open Shift
  - Kasir: "Budi"
  - Opening balance: Rp 2.000.000 (uang awal)

Sepanjang shift:
  - Transaksi #1: Cash Rp 150.000
  - Transaksi #2: QRIS Rp 85.000
  - Transaksi #3: Cash Rp 220.000
  - ...dst

Close Shift
  - Closing balance: Rp 3.400.000 (uang dihitung manual)
  - Expected: Rp 3.255.000 (2.000.000 + 150.000 + 220.000 = cash dari transaksi)
  - Selisih: +Rp 145.000 (kelebihan)
```

---

## 2. Hold & Recall Transaction (Parked Sales) ✅

**Status:** SUDAH DIIMPLEMENTASI — via `status='parked'/'recalled'` di sales table, `ParkedSalesModal.svelte`, badge counter

Saat transaksi sedang berjalan di POS, kasir bisa **menahan (hold/park)** transaksi tersebut untuk dilanjutkan nanti. Berguna saat:
- Pelanggan lupa dompet/hp
- Pelanggan ingin ambil barang lain dulu
- Ada antrian panjang, kasir mau proses pelanggan lain dulu

### Contoh Workflow

```
Kasir sedang scan barang:
  - Item A: Rp 50.000
  - Item B: Rp 35.000
  - Total: Rp 85.000

→ Pelanggan: "Bentar ya, saya ambil dompet dulu"
→ Kasir: klik "Hold" → transaksi tersimpan, layar bersih

→ Pelanggan baru datang, proses transaksi baru normal

→ Pelanggan pertama kembali:
→ Kasir: klik "Recall" → pilih dari daftar parked transactions
→ Item A + Item B muncul kembali, lanjut bayar
```

---

## 3. Split Payment (Multi Payment Method) ✅

**Status:** SUDAH DIIMPLEMENTASI — `sale_payments` table, `CheckoutModal.svelte` (allocations grid), backend validation (`validatePayments`)

Sistem saat ini sudah mendukung pembayaran dengan **lebih dari satu metode pembayaran** dalam satu transaksi. Setiap alokasi pembayaran dicatat di tabel `sale_payments` dan ditampilkan di UI POS, receipt, dan transaction drawer.

### Contoh Penggunaan

```
Total belanja: Rp 475.000

Contoh 1 - Cash + QRIS:
  - Cash:      Rp 300.000
  - QRIS:      Rp 175.000

Contoh 2 - Debit + E-Wallet:
  - Debit:     Rp 400.000
  - E-Wallet:  Rp  75.000

Contoh 3 - Cash + Card + E-Wallet:
  - Cash:      Rp 200.000
  - Kartu:     Rp 200.000
  - E-Wallet:  Rp  75.000
```

### Komponen yang Telah Dibangun

**Backend:**
- Tabel `sale_payments` (`004_split_payment.sql`) — menyimpan `sale_id`, `payment_method_id`, `payment_method_code`, `amount`, `reference_number`, `created_at`
- Validasi `validatePayments` — cek total pembayaran = total belanja, max 10 metode, batas 1 Cash, referensi untuk Card/E-Wallet
- Domain errors: `ErrPaymentTotalMismatch`, `ErrDuplicatePaymentMethod`, `ErrMultipleCashPayments`, `ErrMaxPaymentsExceeded`, `ErrPaymentReferenceRequired`, `ErrPaymentMethodInactive`
- Service & Repository: `CreateSalePayments`, `UpdateShiftTotals`, response `Sale.Payments []SalePayment`

**Frontend:**
- `CheckoutModal.svelte` — "Alokasi Pembayaran" dengan grid metode pembayaran, daftar alokasi (add/remove), input jumlah, nomor referensi opsional, denom uang cash, tombol "Tepat [F7]"
- `PosPage.svelte` — wiring `payments` ke payload POST `/sales`
- `ReceiptPrintOverlay.svelte` — cetak semua alokasi pembayaran
- `TransactionDrawer.svelte` — tampilkan array `payments` dengan ref number
- `TransactionTable.svelte` — badge metode ganda untuk transaksi split

### Aturan Validasi

| Aturan | Keterangan |
|--------|------------|
| Total split = total belanja | Wajib (toleransi 0) |
| Max 10 metode per transaksi | Ditentukan `MaxPaymentsPerSale` |
| Hanya 1 Cash | Mencegah duplikasi cash |
| Duplikasi metode lain | Ditolak (case-insensitive) |
| Referensi untuk Card/E-Wallet | Wajib jika payment method `requires_reference = true` |

### Test Coverage

- Backend: `internal/sale/service_test.go`, `handler_mock_test.go`, `repository_test.go` — include success, total mismatch, duplikat metode, multiple cash, max exceeded, reference required, invalid method, inactive method, 3-way split
- Frontend: `CheckoutModal.svelte.test.ts` (25 tests), `PosPage.svelte.test.ts` (26 tests)
- E2E: `TestE2E_SplitPayment` — cash+card, three-way split

---

## 4. Purchase Order & Goods Receiving ✅

**Status:** SUDAH DIIMPLEMENTASI — `internal/purchase/`, `web/src/modules/purchase-orders/`, migrasi `007_purchase_orders.sql`, `008_add_cancel_permission.sql`, `009_add_do_sequence.sql`

Alur pembelian barang dari supplier. Saat stok menipis atau butuh restock, toko buat **Purchase Order (PO)** ke supplier. Saat barang datang, admin **goods receiving** — mencatat berapa yang diterima dan stok otomatis bertambah.

### Contoh Workflow

```
1. Buat PO:
   - Supplier: PT Maju Jaya
   - Item:
     - Produk A × 100 unit @ Rp 10.000 = Rp 1.000.000
     - Produk B × 50 unit  @ Rp 15.000 = Rp   750.000
   - Estimasi tanggal: 15 Juli 2026

2. Saat barang datang → Goods Receive:
   - Dari PO #000123
   - Produk A: diterima 95 dari 100 (5 rusak/cacat)
   - Produk B: diterima 50 dari 50 ✓
   - Stok otomatis naik: A +95, B +50

3. Status PO berubah: Draft → Confirmed → Partial Received → Received
```

### Status PO

| Status | Keterangan |
|--------|------------|
| Draft | PO baru dibuat, bisa diedit |
| Confirmed | PO sudah dikirim ke supplier, menunggu pengiriman |
| Partial Received | Sebagian barang sudah diterima |
| Fully Received | Semua barang sudah diterima |
| Cancelled | PO dibatalkan |

### Komponen yang Telah Dibangun

**Backend:**
- Tabel `purchase_orders` (po_number, supplier_id, store_id, status, financial fields, approval_status, payment_status, invoice_status, warehouse_id, currency, ...)
- Tabel `purchase_order_items` (po_id, product_id, qty_ordered, qty_received, unit_cost, subtotal, snapshot name/sku/barcode/uom)
- Tabel `goods_receipts` + `goods_receipt_items` (gr_number, po_id, store_id, received_by, received_at, notes, condition good/damaged)
- Auto-generate nomor PO/GR + nomor DO (`do_seq`)
- Endpoint: CRUD PO, confirm, cancel, list receipts, goods receive, update stok otomatis (inventory movement `purchase_receipt`)
- Update PO real-time via WebSocket (store-scoped)

**Frontend:**
- Halaman PO list (filter status, supplier, tanggal) + status badge berwarna
- Form create/edit PO multi-step (pilih supplier, tambah item produk, qty & harga, financial summary)
- Halaman goods receive (input jumlah diterima per item, catat kondisi, auto DO number)
- Notification bell untuk update status PO

---

## 5. Stock Opname (Physical Count) ✅

**Status:** SUDAH DIIMPLEMENTASI — `internal/stockopname/`, `web/src/modules/stock-opname/`, migrasi `012_stock_opname.sql`, `015_stock_opname_store_id.sql`, `016_stock_opname_scope_workflow.sql`, `017_stock_opname_adjustment_ledger.sql`

### Penjelasan

Pencocokan **stok fisik** (yang dihitung manual di gudang/toko) dengan **stok sistem** di database. Penting untuk akurasi inventori karena selisih pasti terjadi (barang rusak, hilang, kesalahan scan, dll).

### Contoh Workflow

```
Stock Opname per 1 Juli 2026:

| Produk     | Stok Sistem | Stok Fisik | Selisih | Keterangan     |
|------------|-------------|------------|---------|----------------|
| Produk A   | 150         | 148        | -2      | Rusak terjatuh  |
| Produk B   | 80          | 80         |  0      | OK              |
| Produk C   | 200         | 195        | -5      | Hilang          |

→ Setelah diverifikasi & di-post:
  - Sesi: Draft → Open → Counting → Verification → Approved → Posted → Closed
  - Saat posting, stok sistem Produk A: 150 → 148 (adjust -2)
  - Stok sistem Produk C: 200 → 195 (adjust -5)
  - Inventory movement tercatat: type = "stock_opname"
  - Dokumen penyesuaian IA- dibuat di ledger (inventory_adjustments)
```

### Status Stock Opname

| Status | Keterangan |
|--------|------------|
| Draft | Sesi baru dibuat, bisa diedit |
| Open | Sesi dibuka, menunggu counting dimulai |
| Counting | Petugas menghitung stok fisik (autosave per item) |
| Verification | Hasil counting disubmit, menunggu verifikasi supervisor |
| Needs Recount | Ditolak / diminta hitung ulang |
| Approved | Terverifikasi, selisih di-persist (stok belum berubah) |
| Posted | Penyesuaian di-posting ke stok + dokumen IA- dibuat |
| Closed | Sesi ditutup (akhir alur) |
| Cancelled | Sesi dibatalkan (dari Draft/Open/Counting/Needs Recount) |

### Komponen yang Telah Dibangun

**Backend:**
- Tabel: `stock_opnames` (store_id, status, notes), `stock_opname_items`, `stock_opname_counts`, `stock_opname_assignments`, `stock_opname_session_scopes` (scope store/warehouse/category/product), `stock_opname_recount_requests`, `inventory_adjustments` + `inventory_adjustment_items` (ledger, sequence `ia_seq`)
- Multi-scope session (store / warehouse / category / product), store-scoped WebSocket broadcast, assign counter/supervisor (role-validated), blind count + recount workflow
- Endpoint: create, open, cancel, assign/reassign, start, count (autosave), counts history, submit, verify, reject, recount, resume, post-adjustment (auto-adjust stok + dokumen IA-), close, summary, difference report, export (CSV/Excel/PDF), adjustments report
- Permissions: `stock_opname.*` (view/create/assign/count/submit/verify/post/close/report/cancel/export), dot-notation

**Frontend:**
- Halaman: daftar sesi (filter status/scope, pagination), detail sesi, form create (multi-scope + dropdown scope), counting UI per item dengan stok sistem sebagai referensi, approval view untuk supervisor, riwayat counting, laporan selisih, adjustments report (dokumen IA-)
- Real-time status update via WebSocket (store-scoped) + notification bell (permission-gated)

---

## 6. Store Management ✅

**Status:** SUDAH DIIMPLEMENTASI — `internal/store/`, `web/src/modules/stores/`

### Penjelasan

Sistem saat ini mendukung **lebih dari satu toko/outlet** (`stores` table, `store_id` di sales, shifts, PO, cart, dll). Fitur ini menyediakan **halaman manajemen toko** di frontend agar admin bisa mengelola daftar toko/outlet (tambah, edit, nonaktifkan, hapus) tanpa harus akses database langsung.

### Contoh Workflow

```
1. Admin buka halaman Store Management:
   - Daftar toko: Toko Pusat, Cabang Malioboro, Cabang Dipatiukur
   - Status: aktif/nonaktif

2. Admin tambah toko baru:
   - Nama: "Cabang Bandung"
   - Alamat: Jl. Merdeka No. 1
   - Telepon: 022-123456

3. Admin nonaktifkan toko yang tutup:
   - Toko "Cabang Lama" → is_active = false
   - Tidak muncul di dropdown POS/sales baru
```

### Komponen Backend (Telah Dibangun)

**Backend:**
- Tabel `stores` (`000_squash.sql`) — name, address, phone, is_active
- `internal/store/`: domain, repository, service, handler
- Endpoint: `GET/POST /api/stores`, `GET/PUT/DELETE /api/stores/:id`, `GET /api/stores/active`
- Permission: `store.view` / `store.create` / `store.update` / `store.delete`
- Audit log untuk create/update/delete store
- Import/export via framework reusable (schema `stores`)

### Komponen yang Telah Dibangun

**Frontend:**
- `web/src/modules/stores/` — `StoresPage.svelte` (daftar toko dengan tabel, status badge aktif/nonaktif), service + types, unit tests
- Route `/stores` + label di sidebar/topbar, permission `store.view`
- Integrasi dengan dropdown pemilihan toko yang sudah ada (POS, sales, PO)
---

## 7. Storage Locations ✅

**Status:** SUDAH DIIMPLEMENTASI — `internal/storagelocation/`, `web/src/modules/storage-location/`, migrasi `018_storage_locations.sql`. Per-rack stock tracking (fase 2) juga sudah selesai — `internal/inventory/location_*`, `internal/stockopname` scope `location`, migrasi `020_per_rack_stock.sql`.

### Penjelasan

Master data **lokasi penyimpanan** (rak/gudang) tempat produk disimpan. Setiap lokasi bisa dikaitkan ke warehouse atau store (scope). Setelah migrasi `020`, tracking stok per lokasi (sub-akun dari stok global) dan stock opname per lokasi sudah aktif.

### Contoh Workflow

```
1. Admin buka halaman Storage Locations:
   - Daftar lokasi: Rak A-01 (Warehouse Pusat), Rak B-02 (Toko Pusat)
   - Status: aktif/nonaktif, scope (warehouse / store)

2. Admin tambah lokasi baru:
   - Kode: "RAK-C-01"
   - Nama: "Rak C-01 – Snack"
   - Scope: warehouse / store
   - Catatan: dekat pintu masuk

3. Admin buka detail produk → panel "Stok Rak (Lokasi)":
   - Set: tentukan jumlah eksak di sebuah lokasi
   - Transfer: pindahkan jumlah antar lokasi

4. Stock opname dengan scope "Storage Location (Rack)":
   - Hitung produk yang berada di satu rak
   - Saat di-post, stok rak dikoreksi ke angka fisik, lalu stok global dihitung ulang dari angka fisik tersebut: (global lama − rak lama, minimal 0) + rak baru
```

### Komponen yang Telah Dibangun

**Backend:**
- Tabel `storage_locations` (code, name, warehouse_id, store_id, notes, is_active) dengan constraint scope (warehouse atau store), unique code
- `internal/storagelocation/`: domain, repository, service, handler (+ audit log untuk create/update/delete)
- Endpoint: `GET/POST /api/storage-locations`, `GET/PUT/DELETE /api/storage-locations/:id`, `PUT/DELETE /api/storage-locations/bulk`
- Permissions: `storage_location.view` / `storage_location.create` / `storage_location.update` / `storage_location.delete`
- Migrasi `020_per_rack_stock.sql`: `product_stock.location_id` FK + unique `(product_id, warehouse_id, store_id, location_id)`, `stock_opnames.location_id`, scope `location`
- Inventory rack-stock: `GET/POST /api/inventory/locations`, `POST /api/inventory/locations/transfer` (reuse permission `inventory.adjust`)
- Stock opname scope `location` (wajib scope tunggal), snapshot/lock/update per lokasi, rekonsiliasi global saat posting

**Frontend:**
- `web/src/modules/storage-location/` — halaman list (`StorageLocationsPage.svelte`), tabel, toolbar/filter, modal create/edit/delete, service + types, unit tests
- Route `/storage-locations` + label di sidebar/topbar, permission `storage_location.view`
- `web/src/modules/inventory/components/RackStockPanel.svelte` — panel stok rak di detail produk (set/transfer)
- Scope picker stock opname mendukung tipe `location` ("Storage Location (Rack)")

---

 
## 8. Time-based Pricing Update

> **Status: DITOLAK / DIGANTI** — pendekatan ini **ditolak di BDR** dan digantikan oleh [Price Consistency During Active Transactions](#10-price-consistency-during-active-transactions) (server-authorized snapshot). Harga boleh berubah kapan saja; transaksi yang sedang berlangsung tidak terpengaruh karena harga dibekukan per-item saat ditambahkan. Sisa gagasan *scheduled price changes* masih relevan sebagai future enhancement (lihat ADR §6).

### Penjelasan

Perubahan harga (pricing rule, diskon, promosi) hanya diperbolehkan dilakukan pada **window waktu tertentu** saat toko tutup atau shift kasir tidak aktif. Alasan: perubahan harga di tengah shift dapat menyebabkan **price mismatch** antara harga yang ditampilkan di layar POS dengan harga authoritative di backend, yang berujung pada discrepancy fisik uang di kas.

### Contoh Workflow

```
Jam 02:00 - 05:00 (toko tutup, semua shift aktif sudah ditutup):
  → Admin bisa mengubah pricing rule:
    - Update diskon member
    - Ganti harga khusus
    - Upload promosi baru
  → Sistem menyimpan perubahan dan aktifkan otomatis saat toko buka

Jam 08:00 - Shift dimulai:
  → Semua kasir mendapatkan harga terbaru
  → Tidak ada perubahan harga di tengah shift

Jam 14:00 - Admin coba ubah harga:
  → Sistem blokir: "Tidak dapat mengubah harga saat ada shift aktif"
  → Admin harus menunggu semua shift ditutup atau menunggu window waktu khusus
```

### Komponen yang Perlu Dibangun

**Backend:**
- Config: `PRICING_UPDATE_WINDOW_START` dan `PRICING_UPDATE_WINDOW_END` (contoh: `02:00`, `05:00`)
- Middleware/check di endpoint pricing rule create/update/delete:
  - Cek apakah semua shift sudah ditutup (`status = 'closed'` untuk semua kasir)
  - ATAU cek apakah sekarang berada dalam window waktu yang diizinkan
- Response error jelas: `"Pricing changes are only allowed between 02:00-05:00 or when all shifts are closed"`
- Audit log untuk setiap percobaan perubahan harga yang diblokir

**Frontend:**
- Badge/indikator di halaman Pricing Rules: *"Pricing updates are frozen during active shifts"*
- Disable tombol tambah/edit/hapus pricing rule saat ada active shift
- Tampilkan window waktu yang diizinkan di help text

### Aturan

| Kondisi | Bisa Ubah Harga? |
|---------|------------------|
| Semua shift closed + dalam window waktu | ✅ Bisa |
| Ada shift aktif | ❌ Tidak bisa |
| Semua shift closed + di luar window | ❌ Tidak bisa |

---
 
## 9. Admin Change Freeze During Active Shifts

> **Status: DITOLAK / DIGANTI** — pendekatan ini **ditolak di BDR** (menghambat operasional toko multi-kasir / 24 jam) dan digantikan oleh [Price Consistency During Active Transactions](#10-price-consistency-during-active-transactions). Perubahan master data (harga, cost, pricing rule, discount) tetap diizinkan kapan saja; transaksi yang berlangsung tidak terpengaruh.

### Penjelasan

Perubahan pada data sensitif (harga, diskon, produk, stok) **diblokir secara otomatis** ketika ada setidaknya satu shift kasir yang masih berstatus `open`. Tujuan: mencegah perubahan yang dapat memengaruhi perhitungan discrepancy atau integritas data selama transaksi sedang berlangsung.

### Contoh Workflow

```
Skenario 1 - Ada shift aktif:
  Kasir "Budi" sedang open shift (belum close).
  Admin coba ubah harga produk "Indomie" dari Rp 3.500 menjadi Rp 3.000.
  → Sistem tolak:
    "Cannot update product price while shift #123 is still open.
     Please close all active shifts first."

Skenario 2 - Semua shift closed:
  Semua kasir sudah close shift untuk hari ini.
  Admin ubah harga "Indomie" menjadi Rp 3.000.
  → Sistem izinkan, perubahan langsung生效.

Skenario 3 - Percobaan bypass via API langsung:
  Developer/admin mencoba POST langsung ke /api/products/:id dengan harga baru.
  → Middleware backend tetap cek active shifts dan tolak permintaan.
```

### Data yang Diblokir

| Entity | Aksi yang Diblokir |
|--------|-------------------|
| Products | Update price, cost, status |
| Pricing Rules | Create, update, delete, activate/deactivate |
| Discounts | Create, update, delete |
| Stock adjustments | Manual stock correction |

### Komponen yang Perlu Dibangun

**Backend:**
- Helper: `HasActiveShifts(ctx) (bool, error)` — query `SELECT EXISTS(SELECT 1 FROM shifts WHERE status = 'open')`
- Middleware baru: `RequireNoActiveShifts` — dipasang di route sensitif
- Terapkan pada endpoint:
  - `PUT /api/products/:id`
  - `POST /api/pricing-rules`
  - `PUT /api/pricing-rules/:id`
  - `DELETE /api/pricing-rules/:id`
  - `POST /api/stock-adjustments`
- Response error:
  ```json
  {
    "error": "operation_not_allowed_during_active_shift",
    "message": "Cannot modify pricing while 2 shift(s) are still open. Please close all shifts first.",
    "active_shift_count": 2
  }
  ```
- Audit log untuk setiap percobaan yang diblokir (security relevant)

**Frontend:**
- Global check saat inisialisasi POS: fetch active shift count
- Jika ada active shift:
  - Disable/hide form edit harga di halaman Products
  - Disable/hide tombol tambah/edit pricing rule
  - Tampilkan banner: *"Pricing changes are frozen while shifts are active"*
- Saat admin coba aksi yang diblokir, tampilkan toast dengan pesan yang jelas

---
 
## 10. Price Consistency During Active Transactions ✅

**Status:** SUDAH DIIMPLEMENTASI — migrasi `010_sale_price_snapshot.sql`, `internal/pricing` (`ResolveSnapshot`/`ResolveSnapshotsBatch`), `internal/sale` (cart service + `CheckoutCart`), `web/src/modules/pos/services/pos-service.ts`, E2E `tests/e2e/price-consistency.spec.ts`

Prinsip: **freeze the transaction, not the master data.** Saat sebuah item ditambahkan ke transaksi, seluruh informasi yang memengaruhi harga (harga, cost, tax, pricing rule) menjadi snapshot yang immutable dan tidak lagi mengikuti perubahan master data.

### Alur Kerja

```
Kasir add item → POST /api/pos/cart/:id/items
                 ├─ baca master data terbaru
                 ├─ pricing engine resolve 1× (ResolveSnapshot)
                 ├─ persist snapshot ke cart_items (immutable)
                 └─ kembalikan snapshot ke frontend

Kasir checkout → POST /api/pos/cart/:id/checkout
                 ├─ lock cart (FOR UPDATE)
                 ├─ validasi payment terhadap total snapshot
                 ├─ cek & kurangi stok
                 ├─ salin cart_items → sale_items (verbatim, tanpa re-resolve)
                 └─ publish sale.created
```

### Komponen yang Telah Dibangun

**Backend:**
- Tabel `cart_sessions` (status open/held/checked_out/cancelled/expired, `expired_at` untuk TTL hold) dan `cart_items` (snapshot per item: unit_price, original_price, discount, cost, tax_rate, pricing_rule, snapshot_created_at)
- Kolom snapshot pada `sale_items` — jejak harga lengkap di transaksi final
- `internal/pricing`: `ResolveSnapshot` (+ batch) — pricing engine dijalankan **sekali saat add item**
- `internal/sale`: service cart (create/get/add/qty/void/hold/resume/checkout/customer), `CheckoutCart` menyalin snapshot verbatim tanpa re-resolve
- Endpoint POS cart: `POST/GET /api/pos/cart`, `POST /api/pos/cart/items`, `PATCH/DELETE /api/pos/cart/items/:itemId`, `GET /api/pos/cart/:id`, `POST /api/pos/cart/:id/hold|resume|checkout`, `PATCH /api/pos/cart/:id/customer`
- `POST /api/sales` dengan `cart_session_id` — checkout via alur sales biasa dengan perilaku yang sama seperti `CheckoutCart`

**Frontend:**
- `PosPage.svelte` — mutasi keranjang via service server-side (`addCartItem`, `holdCart`, `resumeCart`, `checkoutCart`); **tidak** ada lagi `resolveCartPrices()` / `pricing/resolve`
- Hold/recall via `ParkedSalesModal.svelte` + endpoint baru
- `pos-service.ts` — semua operasi cart server-side

### Aturan yang Dipenuhi

| Business Rule | Implementasi |
|---------------|--------------|
| BR-01 Snapshot dibuat saat add | `POST /api/pos/cart/:id/items` → resolver + persist ke `cart_items` |
| BR-02 Transaksi immutable | Checkout menyalin snapshot verbatim; tidak ada re-resolve |
| BR-03 Perubahan master data | Hanya memengaruhi snapshot yang dibuat setelahnya |
| BR-05 Hold/Resume | Tidak re-resolve harga; snapshot lama tetap |
| BR-07 Perubahan quantity | Hanya ubah quantity + subtotal; `unit_price` beku |
| BR-08 Void item | Hapus snapshot; scan ulang → snapshot baru |
| BR-09 Promo terjadwal | Aktivasi hanya memengaruhi add item berikutnya |

### Test Coverage

- Unit: `internal/pricing/resolver_test.go` (RT-01), `internal/sale/service_test.go`, `internal/sale/domain_test.go`
- Integration: `internal/sale/handler_test.go`, `internal/sale/cart_service_test.go`, `repository_test.go`
- E2E: `tests/e2e/price-consistency.spec.ts` — 11 skenario perubahan harga/promo di tengah transaksi

---

## 11. Konsinyasi Supplier ✅

**Status:** SUDAH DIIMPLEMENTASI — `internal/consignment/`, `web/src/modules/consignment/`

Barang titipan supplier: barang milik supplier yang dijual di toko, dengan terms harga & hak toko yang disepakati, dan supplier dibayar setelah barang terjual dan di-settlement.

### Alur Utama

1. Tandai supplier sebagai **konsinyasi** (Supplier → Edit → centang).
2. Buat **arrangement** per supplier.
3. Set **terms** per produk: harga jual + hak toko (persentase atau nominal tetap per unit).
4. Catat **penerimaan** (dibawa − ditolak = diterima) → stok konsinyasi bertambah.
5. Barang dijual lewat POS biasa; kepemilikan konsinyasi otomatis ter-record per transaksi.
6. Barang rusak/kadaluarsa: **retur tertunda** (ditarik dari display), lalu **retur formal** (diserahkan kembali ke supplier, menutup pending return).
7. **Settlement** mencakup semua penjualan belum diselesaikan; **payout** (bisa parsial) memakai payment method yang ada.

### Prinsip Kunci

- Setiap item dimiliki oleh tepat satu supplier. Available + pending return = masih milik supplier.
- Terms hanya berlaku untuk stok belum terjual; tidak mengubah penjualan yang sudah terjadi.
- SKU bersifat store-owned ATAU konsinyasi (mutually exclusive) — kepemilikan = available + pending return.
- Settlement tidak parsial; item yang sudah di-settle tidak pernah di-settle ulang.
- Hak toko dihitung berbasis qty (`computeStoreShare`); total terhutang = total penjualan − hak toko.

### Komponen yang Telah Dibangun

- Backend: `internal/consignment/` (domain, repository, service, handler, routes di `cmd/server/main.go`), penyesuaian checkout (`internal/sale/`) agar mencatat kepemilikan konsinyasi, `is_consignment` pada supplier.
- DB: `001_consignment.sql`, `002_settlement_items_product_id.sql`, `003_settlement_updated_at.sql`.
- Frontend: `web/src/modules/consignment/` — halaman arrangement (daftar + detail 6 tab), terms, penerimaan, retur tertunda, retur, stok, settlement & payout; toggle konsinyasi di modul supplier.
- Permission: `consignment.view/create/update/settle/pay` (Superadmin/Admin semua; Manager tanpa `pay`).

### Test Coverage

- `internal/consignment/` — arrangement, receipt, pending return, return, settlement, payout, checkout-insufficient-stock, store scope, PRD §9 scenario matrix.
- `internal/supplier/` — passthrough `is_consignment`.
- Frontend: `web/src/modules/consignment/` + build.

---

## Prioritas Implementasi

| Urutan | Fitur | Alasan |
|--------|-------|--------|
| 1 | ~~Shift Management~~ | ✅ Selesai |
| 2 | ~~Hold & Recall~~ | ✅ Selesai |
| 3 | ~~Split Payment~~ | ✅ Selesai |
| 4 | ~~Price Consistency During Active Transactions~~ | ✅ Selesai (server-authorized snapshot) |
| 5 | ~~Purchase Order~~ | ✅ Selesai |
| 6 | ~~Store Management~~ | ✅ Selesai |
| 7 | ~~Stock Opname~~ | ✅ Selesai (workflow 9-state + adjustment ledger) |
| ~~8~~ | ~~Admin Change Freeze During Active Shifts~~ | ~~Ditolak di BDR~~ — digantikan snapshot |
| ~~9~~ | ~~Time-based Pricing Update~~ | ~~Ditolak di BDR~~ — hanya scheduled changes yang relevan |
| 10 | Storage Locations | ✅ Selesai (master data; per-rack stock = fase berikutnya) |
| 11 | Konsinyasi Supplier | ✅ Selesai |

---

## Catatan

- **Returns & Refund (Damaged Items):** Sedang direncanakan — Phase 1. Pelanggan dapat meretur barang rusak dalam batas waktu yang dikonfigurasi (default 1 hari) untuk pengembalian penuh ke metode pembayaran asli. Persetujuan manager diperlukan untuk retur bernilai tinggi. Lihat `docs/prd/PRD-Sales-Return.md` dan `docs/design/sales-return-plan.md`.
- **Supplier Returns:** Baru relevan setelah Purchase Order terbangun.
- **Product Image Upload:** Dibatalkan untuk saat ini (tidak diprioritaskan).
- **Admin Change Freeze & Time-based Pricing Update:** Ditolak di BDR — memblokir perubahan master data saat shift aktif menghambat operasional. Digantikan oleh *Price Consistency During Active Transactions* (snapshot dibuat saat add item; master data tetap mutable).
- **Storage Locations:** Master data sudah dibangun. Fase berikutnya: per-rack stock tracking dan stock opname per lokasi.
- **Konsinyasi — deferred (PRD §15):** format dokumen cetak penerimaan/retur, dashboard analitik konsinyasi, aturan expiry-by-shelf-life, refund/void di luar eligible customer return, settlement policy saat supplier lama absen (termasuk forced-return release, Opsi A), rekonsiliasi fisik, alur pembayaran di luar payout record.
- Semua fitur baru harus mengikuti pattern Clean Architecture yang sudah ada (domain → repository → service → handler).
