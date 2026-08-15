# ADR: Konsistensi Harga Selama Transaksi Aktif (Server-Authorized Snapshot)

| Field         | Nilai |
|---------------|-------|
| Status        | **Accepted** |
| Tanggal       | 2026-07-31 |
| Deciders      | Tim Produk, Tim Engineering |
| Referensi     | `docs/adr/BDR_Price_Consistency_During_Active_Transactions.txt`, `docs/adr/Business_Process_Product_Price_Changes_During_Active_Transactions.txt` |
| Scope         | Product Price, Product Cost, Pricing Rules, Discount Rules, Hold Transaction |
| Di luar scope | Refund, Return, Stock Opname, Inventory Valuation, Purchase Order |

---

## 1. Konteks (Context)

Administrator perlu mengubah master data (harga produk, biaya produk, pricing rule, discount rule) kapan saja untuk mendukung kebutuhan operasional, termasuk saat shift sedang berjalan. Namun, apabila transaksi yang sedang berlangsung terus membaca master data, perubahan tersebut dapat menyebabkan:

- harga item berubah di tengah transaksi;
- total transaksi menjadi tidak konsisten;
- audit menjadi sulit;
- pengalaman pelanggan menjadi buruk.

Membekukan seluruh perubahan master data selama masih ada shift aktif ditolak (rejected pada BDR) karena menghambat operasional toko dengan banyak kasir atau operasional 24 jam.

**Prinsip bisnis yang disepakati (BDR):**

> Master Data bersifat *mutable*. Transaction Data bersifat *immutable*.
>
> Ketika suatu item ditambahkan ke transaksi, seluruh informasi yang memengaruhi harga menjadi bagian dari transaksi dan tidak lagi mengikuti perubahan pada master data.
>
> **Freeze the transaction, not the master data.**

Keputusan kunci BDR: **snapshot harga dibuat saat item ditambahkan ke transaksi** — bukan saat checkout, pembayaran, atau penutupan transaksi.

### 1.1 Kondisi implementasi saat ini (gap)

Audit terhadap kode saat ini menemukan perilaku yang **tidak sesuai** dengan BDR:

| # | Lokasi | Perilaku sekarang | Pelanggaran |
|---|--------|-------------------|-------------|
| 1 | `internal/sale/service.go:99` (`processSaleItems`) | Harga di-resolve ulang oleh pricing engine saat **checkout** melalui `resolver.ResolveBatch`, lalu menimpa `unit_price` klien. | BR-01 (snapshot dibuat di *add time*, bukan checkout) |
| 2 | `web/src/modules/pos/components/PosPage.svelte:247` (`resolveCartPrices`) | Frontend memanggil `POST /api/pricing/resolve` pada **setiap** mutasi keranjang (add, remove, ubah qty, recall, ganti pelanggan). | BR-02 (harga dapat berubah), BR-07 (qty tidak boleh mengubah harga) |
| 3 | `web/src/modules/pos/components/PosPage.svelte:418` | Setelah recall transaksi hold, `resolveCartPrices()` dipanggil sehingga harga item lama dapat berubah. | BR-05 (resume tidak boleh refresh harga) |
| 4 | `web/src/modules/pos/components/PosPage.svelte:214` | Saat add, keranjang menyimpan `price: product.price` (base price mentah), lalu re-resolve. | BR-01 (snapshot tidak diambil sekali di server) |
| 5 | `internal/sale/handler.go:452` (`ParkSale`) | Hold menyimpan item tanpa `unit_price` dari request (hanya `product_id`, `quantity`, `subtotal`); snapshot lengkap tidak tersimpan. | BR-01, BR-05 |
| 6 | `internal/sale/repository.go:32` | `sale_items` tidak menyimpan snapshot `cost`, `tax_rate`, dan `snapshot_created_at`. | BR-01 (snapshot pricing minimal termasuk cost/tax bila diperlukan) |

Pricing engine saat ini berperilaku *resolve every cart mutation*; target yang diinginkan BDR adalah *resolve once per item insertion*.

---

## 2. Keputusan (Decision)

**Server-authorized Snapshot** — backend sebagai *source of truth* untuk pembuatan dan penyimpanan snapshot harga.

### 2.1 Snapshot dibuat di backend saat Add Item

Saat POS menambahkan item ke transaksi:

```
POST /api/pos/cart/:id/items
```

Backend akan:

1. membaca master data terbaru (harga, cost, tax, pricing rule aktif);
2. menjalankan pricing engine satu kali pada saat itu;
3. menghitung seluruh informasi pricing yang berlaku (unit price, original price, discount, rule, cost, tax rate);
4. membuat **snapshot pricing** dan mempersistensikannya ke `cart_items`;
5. mengembalikan snapshot tersebut ke frontend.

Sejak snapshot terbentuk, ia **immutable**. Frontend hanya menampilkan hasil snapshot dan tidak pernah me-resolve ulang.

### 2.2 Checkout tidak me-resolve ulang harga

Saat checkout:

- backend **tidak** memanggil pricing engine;
- backend **tidak** membaca ulang harga produk;
- backend hanya membaca snapshot yang sudah tersimpan di `cart_items` dan menyalinnya verbatim ke `sale_items`.

Hal ini menjamin total transaksi identik dengan yang telah dilihat kasir selama proses penjualan.

### 2.3 Hold / Resume tidak me-resolve ulang

Saat transaksi di-hold maupun di-resume:

- backend mengembalikan snapshot yang telah tersimpan;
- frontend **tidak** memanggil `resolveCartPrices()`;
- tidak ada refresh harga.

### 2.4 Perubahan Quantity

Mengubah quantity **tidak** membuat snapshot baru. Yang berubah hanya `quantity` dan `subtotal`. Harga satuan tetap memakai snapshot lama (BR-07).

### 2.5 Void Item

Jika item dihapus, snapshot item ikut dihapus. Apabila produk ditambahkan kembali, backend membuat snapshot baru menggunakan master data saat itu (BR-08).

### 2.6 Add Item setelah Resume

Setelah resume, item baru mendapat snapshot baru (harga master data saat itu); item lama tetap memakai snapshot lama. Satu transaksi dapat berisi item dengan snapshot bertimestamp berbeda — ini sesuai keputusan bisnis BDR.

### 2.7 Perubahan Master Data

Administrator tetap dapat mengubah harga produk, pricing rule, discount rule, dan cost kapan saja. Perubahan hanya memengaruhi snapshot yang dibuat **setelah** perubahan tersebut (BR-03, BR-04, BR-09). Aktivasi promo hanya memengaruhi item yang ditambahkan setelah promo aktif.

---

## 3. Alternatif yang Dipertimbangkan

### Opsi 1 — Server-Authorized Snapshot *(dipilih)*

Pembuatan snapshot dilakukan dan dipersistensikan oleh backend. Frontend membekukan hasil snapshot dan tidak lagi menjadi penentu harga.

**Kelebihan:**

- Sesuai prinsip Clean Architecture: business rules ditegakkan oleh backend; frontend boleh salah, boleh dimodifikasi, bahkan boleh diganti tanpa mengubah aturan bisnis.
- Snapshot tersimpan di server → mendukung audit, hold/resume multi-device, dan konsistensi data.
- Perubahan moderat: alur POS (cart client-side) tetap dipertahankan; hanya mutasi keranjang dialihkan ke endpoint server dan checkout berhenti me-resolve.

**Kekurangan:**

- Membutuhkan tabel baru (`cart_sessions`, `cart_items`) dan perubahan di beberapa titik frontend/backend.

### Opsi 2 — Frontend-Freeze *(ditolak)*

Frontend berhenti memanggil `resolveCartPrices()` setelah item ditambahkan, dan backend berhenti me-resolve saat checkout, mempercayai snapshot yang dikirim klien.

**Alasan ditolak:**

- Menjadikan frontend sebagai penegak business rule ("frontend yang membekukan harga").
- Bertentangan dengan prinsip Clean Architecture bahwa business rule harus ditegakkan oleh backend.
- Integritas bergantung pada perilaku klien; klien yang di-tamper atau bug klien dapat menghasilkan harga tidak konsisten.
- Audit sulit dibuktikan karena snapshot tidak pernah tervalidasi/tersimpan oleh server.

### Opsi 3 — Full Server-Side Transaction Session *(ditolak)*

Seluruh lifecycle keranjang (create, add, void, qty, hold, resume, checkout) dikelola penuh oleh server; frontend hanya menampilkan state.

**Alasan ditolak:**

- Secara arsitektur paling ideal, tetapi dampaknya terlalu besar untuk tujuan BDR:
  - flow POS berubah total;
  - seluruh lifecycle cart menjadi server-side (state machine ketat);
  - pekerjaan implementasi dan pengujian meningkat drastis;
  - migrasi menjadi jauh lebih kompleks.
- Tujuan BDR (konsistensi harga + fleksibilitas operasional) dapat dicapai dengan Opsi 1 tanpa perombakan tersebut.

**Catatan:** Opsi 3 tetap merupakan evolusi yang valid untuk fase berikutnya bila diperlukan sinkronisasi real-time antar perangkat atau offline mode.

---

## 4. Konsekuensi (Consequences)

### 4.1 Konsekuensi positif

- Administrator tidak perlu menunggu shift selesai untuk mengubah master data.
- Tidak ada perubahan harga di tengah transaksi.
- Audit lebih mudah: setiap `sale_items` adalah jejak snapshot lengkap (termasuk `snapshot_created_at`, `cost`, `tax_rate`, info rule).
- Mendukung multi-kasir dan operasional 24 jam.
- Backend menjadi *source of truth* harga → sesuai Clean Architecture.

### 4.2 Trade-off

- **Multi-snapshot dalam satu transaksi**: satu transaksi dapat memuat item dengan snapshot harga berbeda apabila item ditambahkan pada waktu berbeda (misal item lama saat harga Rp3.500, item baru setelah harga turun Rp3.000). Ini konsekuensi yang disepakati BDR.
- **Perilaku ganti pelanggan**: mengganti pelanggan di tengah transaksi **tidak** mengubah harga item yang sudah ada (berlaku untuk item baru saja), karena harga dibekukan saat add. Perilaku ini berbeda dari implementasi lama yang me-re-resolve.
- **Perluasan tabel**: dibutuhkan tabel baru dan penambahan kolom pada `sale_items` (migrasi forward-only).

### 4.3 Mapping ke Business Rules BDR

| Business Rule | Bagaimana dipenuhi |
|---------------|--------------------|
| BR-01 Snapshot Creation | `POST /api/pos/cart/:id/items` → resolver + persist `cart_items` lengkap (product_id, product_name, unit_price, discount, pricing rule, tax, cost, snapshot_created_at) |
| BR-02 Immutable Transaction | Snapshot tersimpan di `cart_items`; checkout menyalin verbatim ke `sale_items`; tidak ada re-resolve |
| BR-03 Master Data Update | Resolver membaca master data saat add; perubahan hanya memengaruhi snapshot berikutnya |
| BR-04 New Transaction | Transaksi baru → cart session baru → snapshot sesuai master data saat item ditambahkan |
| BR-05 Hold Transaction | Hold menyimpan `cart_items` apa adanya; resume/checkout tidak re-resolve |
| BR-06 Add Item After Resume | `POST items` pada cart yang di-resume → snapshot baru hanya untuk item baru |
| BR-07 Quantity Update | `PATCH items/:id` hanya mengubah `quantity` + `subtotal`; `unit_price` beku |
| BR-08 Void Item | `DELETE items/:id` menghapus snapshot; scan ulang → snapshot baru |
| BR-09 Scheduled Promotion | Pembuatan/penjadwalan promo diizinkan kapan saja; aktivasi hanya memengaruhi add item berikutnya |

### 4.4 Hold Expiration (Business Process Rule 9)

Transaksi hold tidak disimpan tanpa batas. Kebijakan yang direkomendasikan:

- masa berlaku hold default **24 jam** (dikonfigurasi, misal env `CART_HOLD_TTL_HOURS`); dan/atau
- otomatis dibatalkan saat shift ditutup (opsional, dapat diterapkan pada fase berikutnya).

Implementasi: kolom `expired_at` pada `cart_sessions`; verifikasi ekspirasi dilakukan secara *lazy* saat resume/checkout (status `expired` bila lewat). Background job pembersihan bersifat opsional.

---

## 5. Dampak Implementasi (ringkas)

Perubahan besar diuraikan detail pada `docs/design/TDD_Price_Consistency_During_Active_Transactions.md`. Ringkasannya:

1. **Database**: migrasi `010_sale_price_snapshot.sql`
   - tabel baru `cart_sessions`, `cart_items`;
   - kolom snapshot tambahan pada `sale_items` (`cost`, `tax_class_id`, `tax_rate`, `snapshot_created_at`, `product_name`).
2. **Backend Go**:
   - `internal/pricing`: metode baru `ResolveSnapshot` (+ batch) yang mengembalikan snapshot + cost + tax.
   - `internal/sale`: entity `CartSession`/`CartItem`; service cart (add/qty/void/hold/resume/checkout); `processSaleItems` diubah menjadi validasi konsistensi internal (tanpa re-resolve).
   - `internal/wiring`: registrasi repo/service/handler baru.
3. **API**: endpoint `POST /api/pos/cart`, `GET /api/pos/cart`, `POST /api/pos/cart/:id/items`, `PATCH/DELETE /api/pos/cart/:id/items/:itemId`, `POST /api/pos/cart/:id/hold`, `POST /api/pos/cart/:id/resume`, `GET /api/pos/cart/:id`, `POST /api/pos/cart/:id/checkout`.
4. **Frontend Svelte 5**: `PosPage.svelte` berhenti memanggil `resolveCartPrices()`; mutasi keranjang via service baru; hold/recall via endpoint baru.
5. **Event flow**: `sale.created` tetap; tambahan observability `cart.checked_out`.
6. **Audit**: audit log untuk operasi cart (add, void, qty, hold, resume, checkout) untuk memenuhi konsekuensi "audit lebih mudah".

---

## 6. Status & Lanjutan

- Keputusan ini **Accepted** dan menjadi dasar implementasi.
- Future enhancement yang tetap relevan (dari BDR): Scheduled Price Changes, Price Versioning, Approval Workflow, Bulk Price Update, Customer-specific Pricing, Membership Pricing, Multi-branch Pricing, Dynamic Pricing.
- Jika kelak dibutuhkan sinkronisasi real-time / offline mode, arsitektur dapat berevolusi menuju Opsi 3 (full server-side session) tanpa mengubah prinsip snapshot.
