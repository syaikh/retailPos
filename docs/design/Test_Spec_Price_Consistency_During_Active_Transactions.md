# Test Specification: Price Consistency During Active Transactions

| Field       | Nilai |
|-------------|-------|
| Status      | Draft |
| Tanggal     | 2026-07-31 |
| Basis desain | `docs/adr/ADR_Price_Consistency_During_Active_Transactions.md`, `docs/design/TDD_Price_Consistency_During_Active_Transactions.md` |
| Sumber bisnis | `docs/adr/BDR_Price_Consistency_During_Active_Transactions.txt`, `docs/adr/Business_Process_Product_Price_Changes_During_Active_Transactions.txt` |

---

## 1. Strategi Pengujian

Tiga lapis pengujian:

1. **Unit test (Go, table-driven)** — murni logika tanpa DB:
   - `internal/pricing/resolver_test.go` — `ResolveSnapshot`, snapshot fields, determinisme.
   - `internal/sale/service_test.go` — business rules cart (add/qty/void/hold/resume/checkout) dengan repository mock.
   - `internal/sale/domain_test.go` — perhitungan turunan (subtotal/dpp/tax), validasi.
2. **Integration test (Go + PostgreSQL)** — `internal/sale/repository_test.go` dan alur end-to-end service:
   - Pakai `shared/testdb.go`, DB `retail_pos_test`, auto-migrasi.
   - Jalankan dengan `-p 1` (AGENTS.md): `TEST_DB_PORT=5433 ... JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 ./...`
3. **Playwright E2E** — `tests/e2e/price-consistency.spec.ts`:
   - UI flow POS vs perubahan harga/promo pada kondisi master data yang berubah di tengah transaksi.

Setiap **Business Rule (BR)** dan **Edge Case** BDR harus dipetakan minimal satu test (lihat §6).

---

## 2. Unit Test (Table-Driven)

### 2.1 `internal/pricing/resolver_test.go`

**RT-01 — `ResolveSnapshot` mengembalikan snapshot lengkap**

Tabel kasus (product, qty, customer_group, store, expected):

| Nama kasus | Setup | UnitPrice | OriginalPrice | Discount | PricingType | Cost | TaxRate | SnapshotAt |
|------------|-------|-----------|---------------|----------|-------------|------|---------|------------|
| tanpa rule | base price 3500 | 3500 | 3500 | 0 | default | 2500 | 11.0 | dalam rentang now |
| promo aktif | rule discount 10% | 3150 | 3500 | 350 | promotion | 2500 | 11.0 | dalam rentang now |
| qty < minimum | rule min_qty 5, qty 2 | base | base | 0 | default | ... | ... | ... |
| qty >= minimum | rule min_qty 5, qty 5 | sesuai rule | base | >0 | ... | ... | ... | ... |

Assertion:
- seluruh field snapshot sesuai;
- `SnapshotAt` dalam `[now-1s, now+1s]` (Jakarta);
- `Cost`/`TaxRate` diambil dari master data.

**RT-02 — `ResolveSnapshot` deterministic** (sama input → sama output; tidak bergantung urutan pemanggilan).

**RT-03 — `ResolveSnapshotsBatch`** — beberapa item berbeda scope; urutan hasil sesuai urutan input (`ResolveItem`).

**RT-04 — Error path** — produk tidak ditemukan → error; repo gagal → error propagasi.

### 2.2 `internal/sale/service_test.go` (mock repository)

Gunakan mock `*Repository` + mock `pricing.PriceResolver` (pola yang sudah ada di `handler_mock_test.go` / `repository_mock_test.go`).

**RT-05 — AddCartItem membuat snapshot sekali & mempersistensikan**

| Kasus | Behavior |
|-------|----------|
| add item baru (cart open) | resolver dipanggil **tepat 1×**; `cart_items` tersimpan dengan field snapshot; totals cart diperbarui |
| resolver error | tidak ada insert; error propagasi |
| cart tidak open | `ErrCartNotOpen`, resolver **tidak** dipanggil |
| cart tidak ada | `ErrCartNotFound` |

**RT-06 — AddCartItem kedua (item berbeda) tidak menyentuh item pertama**

- item A di-resolve; ubah master data (mock) → add item B → snapshot A **tetap** (BR-06). Verifikasi resolver dipanggil hanya untuk B.

**RT-07 — UpdateCartItemQuantity tidak mengubah snapshot**

| Kasus | `unit_price` | `original_price` | `discount` | `cost` | `tax_rate` | `snapshot_created_at` |
|-------|--------------|------------------|------------|--------|------------|------------------------|
| qty naik | tetap | tetap | tetap | tetap | tetap | tetap |
| qty turun | tetap | tetap | tetap | tetap | tetap | tetap |
| qty = 0/negatif | error `ErrCartItemQuantity`; tidak ada update | | | | | |

`subtotal`, `dpp_amount`, `tax_amount` mengikuti qty baru.

**RT-08 — Void item**

- `RemoveCartItem` menghapus baris; totals diperbarui.
- Item yang dihapus tidak lagi muncul di cart.
- (integrasi) scan ulang menghasilkan `snapshot_created_at` baru.

**RT-09 — Hold tidak re-resolve**

- `HoldCart` mengubah status `'open'`→`'held'`, set `expired_at`; resolver **tidak** dipanggil; item tidak berubah.

**RT-10 — Resume tidak re-resolve**

- `ResumeCart` untuk cart `'held'` belum lewat → `'open'`, `expired_at=NULL`; item verbatim; resolver tidak dipanggil.
- Resume idempoten untuk cart `'open'`.
- Resume cart `'expired'` → `ErrCartExpired`; status jadi `'expired'`.

**RT-11 — Expired cart (lazy)**

| Kasus | Setup | Expected |
|-------|-------|----------|
| held, expired_at masa lalu | `expired_at < now` | resume/checkout → `ErrCartExpired`, status `'expired'` |
| held, expired_at masa depan | `expired_at > now` | resume → open, checkout → sukses |
| open, expired_at null | — | selalu bisa |

**RT-12 — CheckoutCart tidak memanggil resolver**

- Mock resolver yang melempar error bila dipanggil → checkout tetap sukses (bukti tidak ada re-resolve).
- Sale dibuat dari snapshot verbatim: `sale_items` sama dengan `cart_items` (unit_price, original_price, discount, rule, cost, tax_rate, snapshot_created_at, product_name).
- `total_amount = subtotal - discount` dari snapshot.
- Cart status → `'checked_out'`.
- `sale.created` dipublish (verifikasi eventbus mock).

**RT-13 — Checkout double / konflik**

| Kasus | Expected |
|-------|----------|
| checkout dua kali | kedua → `ErrCartAlreadyCheckedOut` / 409 |
| checkout cart `'checked_out'` | `ErrCartAlreadyCheckedOut` |
| add item setelah checkout | `ErrCartNotOpen` |

**RT-14 — Payment validation pada checkout**

- `validatePayments` total ≠ total_amount → `ErrPaymentTotalMismatch` (reuse kasus yang ada di `service_test.go`).
- Metode nonaktif, duplikat, cash ganda, ref wajib → error sesuai.

**RT-15 — CreateCartSession**

- berhasil untuk cashier tanpa cart open;
- cashier sudah punya cart open → error (unique index).

**RT-16 — Backward-compat `POST /sales` tanpa cart**

- Payload konsisten (`subtotal == unit_price * qty`) → tersimpan verbatim, tanpa resolver.
- Payload tidak konsisten (`subtotal != unit_price * qty`) → error 400.
- `cart_session_id` disediakan → berperilaku seperti `CheckoutCart`.

### 2.3 `internal/sale/domain_test.go`

| Fungsi | Kasus |
|--------|-------|
| hitung `dpp_amount`/`tax_amount` dari `unit_price*qty` + `tax_rate` | tax_rate 0, 11, 11.5 (pembulatan), dll. |
| `RecalcCartTotals` | campuran item taxed & non-taxed |

---

## 3. Integration Test (Go + PostgreSQL)

File: `internal/sale/repository_test.go` (alur cart) dan `internal/sale/service_integration_test.go` (alur lengkap).

> Setup: `shared/testdb.go`, migrasi otomatis, TRUNCATE antar test. Wajib `-p 1`.

### 3.1 `IT-01 — harga berubah saat hold (BR-05, Edge Case #1)`

1. Seed produk P harga **3500**, cost 2500, tax 11%.
2. Buat cart → add item P qty 1 → snapshot `unit_price=3500`.
3. Hold cart.
4. Update master data: harga P → **3000** (melalui `product.Service`/repository update).
5. Resume cart.
6. **Assert**: item tetap `unit_price=3500` (snapshot lama, tidak re-resolve).
7. Checkout → `sale_items.unit_price=3500`, `snapshot_created_at` = waktu add.

### 3.2 `IT-02 — item baru setelah resume memakai harga terbaru (BR-06, Edge Case #2)`

1. Lanjut dari IT-01 (master data sudah 3000, cart sudah di-resume dengan item lama 3500).
2. Add item P qty 1 lagi.
3. **Assert**: cart berisi dua baris P — `3500` (snapshot lama) dan `3000` (snapshot baru), `snapshot_created_at` berbeda.
4. `subtotal = 3500 + 3000 = 6500`.

### 3.3 `IT-03 — perubahan quantity tidak mengubah harga (BR-07)`

1. Add item (snapshot 3500). Update qty 1→3.
2. **Assert**: `unit_price` tetap 3500; `subtotal` 10500; `snapshot_created_at` tetap.

### 3.4 `IT-04 — void lalu scan ulang membuat snapshot baru (BR-08)`

1. Add item saat harga 3500 → void.
2. Update harga → 3000.
3. Add item yang sama.
4. **Assert**: item baru `unit_price=3000`, `snapshot_created_at` baru. Item void tidak ada.

### 3.5 `IT-05 — promo aktif saat transaksi berjalan tidak memengaruhi item lama (BR-09, Edge Case #5)`

1. Add item saat **tidak** ada promo → snapshot `pricing_type=default`.
2. Aktifkan promo discount 10% untuk produk tsb.
3. Add item lagi → snapshot `pricing_type=promotion`, harga diskon.
4. **Assert**: item pertama tetap harga lama; item kedua harga promo.

### 3.6 `IT-06 — promo aktif sebelum scan digunakan (Edge Case #6)`

1. Aktifkan promo terlebih dahulu.
2. Add item → snapshot promo.
3. Matikan promo → add item lagi → snapshot tanpa promo.
4. **Assert**: sesuai urutan snapshot.

### 3.7 `IT-07 — admin mengubah harga berkali-kali (Edge Case #7)`

1. Add item P (harga 3500).
2. Ubah harga 3000, 4000, 3200.
3. Add item P lagi.
4. **Assert**: item pertama 3500; item kedua 3200 (harga terakhir saat scan).

### 3.8 `IT-08 — checkout stok (regresi)`

- Checkout cart mengurangi stok; total cart sesuai snapshot; shift totals diperbarui; `sale.created` terbit.

### 3.9 `IT-09 — direct sale legacy (UC-09)`

- `POST /sales` tanpa cart → harga klien disimpan verbatim (konsisten); stok berkurang; tidak ada resolver.

### 3.10 `IT-10 — expired cart`

- Set `expired_at` masa lalu (direct update DB) → resume/checkout menolak, status jadi `'expired'`.

---

## 4. Playwright E2E

File baru: `tests/e2e/price-consistency.spec.ts`.

Konfigurasi: `playwright.config.js` yang ada (baseURL `http://localhost:5173`, API `API_BASE` dari `fixtures.ts`, users `TEST_USERS`). Pola helper: `loginUI`, `getToken`, `authHeader`, `request` (lihat `hold-recall.spec.ts`, `pos-flow.spec.ts`).

> Asumsi: test environment memiliki seed produk dasar. Helper setup membuat produk + pricing rule on-the-fly via API agar test tidak bergantung data tertentu.

### 4.1 E2E-01 — harga produk tidak berubah di tengah transaksi (BR-01/02)

1. Login superadmin → buka `/pos`.
2. Add produk P (harga 3500) via UI.
3. Via API (`request` + token): update harga P → 3000.
4. **Assert di cart**: item tetap menampilkan 3500 (tidak re-resolve).
5. Checkout → assert receipt/detail sale `unit_price=3500`.

### 4.2 E2E-02 — hold tidak refresh harga; resume memakai snapshot lama (BR-05, Edge Case #1)

1. Add produk P (3500) → **F6** (hold).
2. Via API ubah harga P → 3000.
3. **F5** → Recall cart.
4. **Assert**: cart item tetap 3500; tidak ada request `resolve` saat recall (verifikasi via `page.on('request')` atau periksa nilai cart).
5. Checkout → sale `unit_price=3500`.

### 4.3 E2E-03 — item baru setelah resume memakai harga terbaru (BR-06, Edge Case #2)

1. Lanjut dari E2E-02 (cart telah di-resume, item lama 3500, master data 3000).
2. Add produk P lagi.
3. **Assert**: cart menampilkan dua baris — 3500 dan 3000.
4. Checkout → `sale_items` memuat dua harga berbeda.

### 4.4 E2E-04 — perubahan quantity tidak mengubah harga (BR-07)

1. Add produk P (3500).
2. Ubah qty 1→3.
3. **Assert**: harga satuan tetap 3500, subtotal 10500.

### 4.5 E2E-05 — void lalu scan ulang memakai harga terbaru (BR-08, Edge Case #4)

1. Add produk P (3500) → hapus item.
2. Via API ubah harga → 3000.
3. Add produk P lagi.
4. **Assert**: cart item sekarang 3000.

### 4.6 E2E-06 — promo aktif saat transaksi berjalan tidak memengaruhi item lama (BR-09, Edge Case #5)

1. Add produk P (tanpa promo).
2. Via API buat + aktifkan promo 10% untuk P.
3. Add produk P lagi.
4. **Assert**: item pertama harga normal; item kedua harga promo (3150 bila base 3500).

### 4.7 E2E-07 — promo aktif sebelum scan digunakan (Edge Case #6)

1. Via API aktifkan promo 10%.
2. Add produk P → harga promo.
3. Via API nonaktifkan promo → add P lagi → harga normal.
4. **Assert**: dua harga berbeda sesuai urutan.

### 4.8 E2E-08 — admin ubah harga berkali-kali saat transaksi berjalan (Edge Case #7)

1. Add produk P (3500).
2. Via API ubah harga 3000 → 4000 → 3200.
3. Add P lagi.
4. **Assert**: 3500 (lama) dan 3200 (baru).

### 4.9 E2E-09 — tidak ada request re-resolve setelah add (regresi arsitektur)

- Selama skenario E2E-01..E2E-08, pasang `page.on('request', req => ...)` dan assert tidak ada panggilan `POST /api/pricing/resolve` setelah item ditambahkan (kecuali void+scan ulang). Ini menjaga agar `resolve-once-per-insertion` tidak regresi.

### 4.10 E2E-10 — checkout flow tetap berfungsi (regresi POS)

- Add → F4 → F7 → Selesai → cart kosong; sale muncul di history (pola `hold-recall.spec.ts`).

---

## 5. Test Data & Fixtures

- **Go integration**: helper seed di `internal/shared/testdb.go` atau `internal/sale/testdata` — produk dengan `price`/`cost`/`tax_class_id`; pricing rule aktif; gunakan TRUNCATE antar kasus (`-p 1`).
- **E2E**: perbesar `fixtures.ts` dengan helper:
  - `createProduct(request, { price, cost })`,
  - `updateProductPrice(request, id, price)`,
  - `createPricingRule(request, { product_id, type, method, value })`,
  - `setActiveCartByUI` / `addItem` helpers.
- Semua update master data saat transaksi berjalan dilakukan **via API** (bukan UI) agar deterministik dan cepat.
- Waktu: gunakan zona Jakarta; untuk test ekspirasi, set `expired_at` langsung via DB (helper admin) — tidak menunggu 24 jam.

---

## 6. Traceability Matrix BR → Test

| Business Rule / Edge Case | Unit | Integration | E2E |
|---------------------------|------|-------------|-----|
| BR-01 Snapshot dibuat saat add | RT-05, RT-06 | IT-01..IT-07 | E2E-01, E2E-09 |
| BR-02 Immutable setelah snapshot | RT-06, RT-12 | IT-01..IT-03 | E2E-01, E2E-03 |
| BR-03 Perubahan master data → transaksi berikutnya | RT-06 | IT-01, IT-02, IT-07 | E2E-03, E2E-08 |
| BR-04 Transaksi baru pakai harga terbaru | RT-05 | IT-02, IT-05 | E2E-03, E2E-07 |
| BR-05 Hold tidak refresh harga | RT-09, RT-10 | IT-01 | E2E-02 |
| BR-06 Add item setelah resume → snapshot baru item baru | RT-06 | IT-02 | E2E-03 |
| BR-07 Qty update tidak ubah harga | RT-07 | IT-03 | E2E-04 |
| BR-08 Void lalu scan ulang → snapshot baru | RT-08 | IT-04 | E2E-05 |
| BR-09 Promo hanya untuk add berikutnya | RT-01 (promo) | IT-05, IT-06 | E2E-06, E2E-07 |
| Hold expiration (BP Rule 9) | RT-11 | IT-10 | (opsional E2E) |
| Edge: harga berubah saat hold | RT-10 | IT-01 | E2E-02 |
| Edge: item baru setelah resume | RT-06 | IT-02 | E2E-03 |
| Edge: qty berubah | RT-07 | IT-03 | E2E-04 |
| Edge: void lalu scan ulang | RT-08 | IT-04 | E2E-05 |
| Edge: promo aktif saat transaksi berjalan | — | IT-05 | E2E-06 |
| Edge: promo aktif sebelum scan | — | IT-06 | E2E-07 |
| Edge: admin ubah harga berkali-kali | — | IT-07 | E2E-08 |
| Checkout tanpa re-resolve (arsitektur) | RT-12 | IT-08 | E2E-09 |
| Backward-compat `POST /sales` | RT-16 | IT-09 | — |

---

## 7. Kriteria Selesai (Definition of Done)

- Seluruh unit test table-driven di §2 hijau.
- Seluruh integration test §3 hijau dengan `-p 1` (mengikuti AGENTS.md).
- Seluruh E2E §4 hijau di Chromium (`npx playwright test tests/e2e/price-consistency.spec.ts`).
- E2E-09 (no re-resolve) terbukti tidak ada panggilan `POST /api/pricing/resolve` setelah item ditambahkan.
- Regresi: suite `pos-flow.spec.ts`, `hold-recall.spec.ts`, `pricing-rules.spec.ts` tetap hijau.
- Build: `go build ./...` dan `npm run build` (frontend) lulus.
