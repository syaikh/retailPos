# Upcoming Features — Rencana Fitur Baru

Dokumen ini menjelaskan fitur-fitur baru yang akan ditambahkan ke Retail POS System, beserta urutan prioritas dan komponen yang perlu dibangun.

---

## Daftar Isi

1. [Shift Management](#1-shift-management)
2. [Hold & Recall Transaction (Parked Sales)](#2-hold--recall-transaction-parked-sales)
3. [Split Payment (Multi Payment Method)](#3-split-payment-multi-payment-method)
4. [Purchase Order & Goods Receiving](#4-purchase-order--goods-receiving)
5. [Stock Opname (Physical Count)](#5-stock-opname-physical-count)
6. [Product Image Upload](#6-product-image-upload)

---

## 1. Shift Management

### Penjelasan

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

### Komponen yang Perlu Dibangun

**Backend:**
- Tabel `shifts` (id, user_id, store_id, opening_balance, closing_balance, status, opened_at, closed_at)
- Handler & service: open shift, close shift, get active shift, list shift history
- Laporan per-shift: total cash, total non-cash, selisih

**Frontend:**
- Halaman shift: tombol Open Shift, form Close Shift (input closing balance)
- Riwayat shift (tabel dengan filter tanggal)
- Indikator shift aktif di sidebar/topbar
- Notifikasi otomatis jika sudah melebihi durasi shift normal

---

## 2. Hold & Recall Transaction (Parked Sales)

### Penjelasan

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

### Komponen yang Perlu Dibangun

**Backend:**
- Flag `status = 'parked'` pada sale atau tabel `parked_sales` terpisah
- Endpoint: park sale, recall sale, list parked sales, delete parked sale

**Frontend:**
- Tombol Hold/Recall di halaman POS
- Daftar parked transactions (bisa lebih dari satu, tampilkan jumlah item & total)
- Badge counter pada tombol Recall jika ada parked transactions
- Konfirmasi saat delete parked transaction

---

## 3. Split Payment (Multi Payment Method)

### Penjelasan

Pelanggan membayar dengan **lebih dari satu metode pembayaran** dalam satu transaksi. Saat ini sistem hanya support 1 payment method per sale.

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

### Komponen yang Perlu Dibangun

**Backend:**
- Tabel `sale_payments` (sale_id, payment_method_id, amount, reference_number, created_at)
- Ubah response sale detail untuk menyertakan multiple payments
- Validasi: total semua pembayaran harus = total belanja (toleransi 0 untuk cash, tolerance untuk rounding)

**Frontend:**
- Tombol "Add Payment" di area pembayaran POS
- Daftar payment yang sudah ditambahkan (dengan tombol hapus per baris)
- Sisa bayar (remaining) ditampilkan secara real-time
- Input reference number opsional per payment (untuk card/e-wallet)
- Jika remaining = 0, tombol "Complete Sale" aktif

---

## 4. Purchase Order & Goods Receiving

### Penjelasan

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

### Komponen yang Perlu Dibangun

**Backend:**
- Tabel `purchase_orders` (po_number, supplier_id, store_id, status, total, expected_date, notes, created_by, confirmed_at, ...)
- Tabel `purchase_order_items` (po_id, product_id, qty_ordered, qty_received, unit_cost, subtotal)
- Tabel `goods_receipts` (gr_number, po_id, store_id, received_by, received_at, notes)
- Tabel `goods_receipt_items` (gr_id, product_id, qty_received, condition [good/damaged], notes)
- Auto-generate PO number (sequential)
- Endpoint: CRUD PO, confirm PO, goods receive, list PO history
- Update stok otomatis saat goods receiving (inventory movement type: `purchase_receipt`)

**Frontend:**
- Halaman PO list (filter status, supplier, tanggal)
- Form create PO (pilih supplier, tambah item produk, input qty & harga)
- Halaman goods receive (pilih PO, input jumlah diterima per item, catat kondisi)
- Status badge berwarna untuk setiap status PO
- Print PO (optional)

---

## 5. Stock Opname (Physical Count)

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

→ Setelah disetujui:
  - Stok sistem Produk A: 150 → 148 (adjust -2)
  - Stok sistem Produk C: 200 → 195 (adjust -5)
  - Inventory movement tercatat: type = "stock_opname"
```

### Status Stock Opname

| Status | Keterangan |
|--------|------------|
| Draft | Sesi opname baru, bisa diedit |
| Pending Approval | Sudah selesai dihitung, menunggu approval manager |
| Approved | Disetujui, stok sistem otomatis di-adjust |
| Rejected | Ditolak, perlu dihitung ulang |

### Komponen yang Perlu Dibangun

**Backend:**
- Tabel `stock_opnames` (id, store_id, status, notes, counted_by, confirmed_by, counted_at, confirmed_at)
- Tabel `stock_opname_items` (opname_id, product_id, system_qty, physical_qty, difference, notes)
- Endpoint: create opname session, save counts, submit for approval, approve/reject, list history
- Saat approve: auto-adjust stok + create inventory_movements (type: `stock_opname`)
- Laporan selisih stok per sesi opname

**Frontend:**
- Halaman: buat sesi opname → input stok fisik per produk (dengan stok sistem sebagai referensi)
- Filter/search produk saat input
- Ringkasan selisih (total item, total selisih, item perlu perhatian)
- Tombol submit untuk approval
- Approval view untuk manager (lihat selisih, approve/reject)
- Riwayat stock opname

---

## 6. Product Image Upload

### Penjelasan

Produk saat ini hanya punya data text (nama, SKU, harga, stok). Fitur ini menambahkan **gambar produk** — memudahkan identifikasi visual, baik di halaman produk maupun di layar POS.

### Contoh Penggunaan

```
Produk: "Indomie Goreng"
SKU: PRD-00145
Gambar: [foto produk]

→ Di halaman produk: thumbnail ditampilkan di tabel/list
→ Di POS: gambar produk muncul saat dipilih/scanned
```

### Komponen yang Perlu Dibangun

**Backend:**
- Kolom `image_url` atau `image_path` di tabel `products`
- Endpoint upload gambar (`POST /api/products/:id/image`)
- Endpoint hapus gambar (`DELETE /api/products/:id/image`)
- Validasi: format (jpg, png, webp), max size (2MB)
- Resize otomatis (thumbnail 200x200, medium 600x600)
- Simpan file di `uploads/products/` (filesystem)

**Frontend:**
- Komponen upload gambar (drag & drop atau file picker) di form product create/edit
- Preview gambar di product list (thumbnail)
- Preview gambar besar di product detail
- Gambar produk di halaman POS (saat pilih/scan produk)
- Placeholder image jika produk belum ada gambar

---

## Prioritas Implementasi

| Urutan | Fitur | Alasan |
|--------|-------|--------|
| 1 | Shift Management | Operational kasir harian, fondasi untuk laporan |
| 2 | Split Payment | Fleksibilitas bayar, sering diminta pelanggan |
| 3 | Hold & Recall | UX kasir, meningkatkan produktivitas |
| 4 | Stock Opname | Akurasi inventori, mencegah selisih stok |
| 5 | Product Image | Peningkatan UX visual, relatif sederhana |
| 6 | Purchase Order | Alur pembelian, butuh dependency lebih banyak |

---

## Catatan

- **Returns & Refund:** Tidak akan diimplementasikan. Kebijakan toko: barang yang sudah dibeli tidak dapat direfund atau dikembalikan.
- **Supplier Returns:** Baru relevan setelah Purchase Order terbangun.
- Semua fitur baru harus mengikuti pattern Clean Architecture yang sudah ada (domain → repository → service → handler).
