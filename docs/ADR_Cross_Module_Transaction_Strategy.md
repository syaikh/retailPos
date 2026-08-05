# ADR: Strategi Transaksi Lintas Modul

| Field         | Nilai |
|---------------|-------|
| Status        | **Proposed** |
| Tanggal       | 2026-08-05 |
| Deciders      | Tim Produk, Tim Engineering |
| Referensi     | `docs/ADR_Modular_Monolith_Module_Boundaries.md` (batas modul dan kontrak komunikasi) |
| Scope         | Strategi atomicity antar modul, Unit of Work, domain event + outbox, keputusan per use case |
| Di luar scope | Implementasi outbox/relay spesifik, pilihan broker, migrasi microservice |

---

## 1. Konteks (Context)

Penerapan ADR batas modul menghilangkan SQL lintas-modul. Operasi yang saat ini mengandalkan satu transaksi database (mis. checkout: potong stok + insert `sales` + `sale_payments`) kehilangan atomicity jika ditulis sebagai beberapa transaksi terpisah milik modul berbeda.

Tanpa pedoman yang jelas, setiap modul akan memilih pola secara tidak konsisten: sebagian memakai panggilan sinkron dalam satu transaksi, sebagian memakai event eventual. Dampaknya: API modul menjadi tidak terduga, dan pemisahan database di masa depan menjadi sulit.

---

## 2. Keputusan (Decision)

### 2.1 Aturan kunci

> **Jika dua modul harus atomik, mereka bukan dua bounded context.**
>
> Konsistensi kuat dipertahankan dengan menempatkan aggregate tersebut di satu bounded context (satu database, satu Unit of Work). Efek lintas bounded context harus selalu *eventual*.

Aturan ini adalah filter pertama untuk setiap interaksi antar modul:

1. Apakah interaksi wajib atomik dengan command? → aggregate harus berada di **satu bounded context**; gunakan **Unit of Work** (satu transaksi).
2. Apakah interaksi hanya efek turunan yang boleh eventual? → gunakan **Domain Event + outbox**.
3. Apakah kedua jawaban masih ambigu? → pertimbangkan menggabungkan/pisahkan context, bukan memilih pola transaksi.

### 2.2 Unit of Work (sinkron, satu transaksi)

Digunakan untuk proses **wajib atomik** dalam satu bounded context. Contoh: **Checkout** di *Transactional Core* — `Reserve Stock → Create Sale → Create Payment` dalam satu transaksi. Kasir tidak boleh melihat transaksi berhasil sementara stok gagal dipotong.

### 2.3 Domain Event + outbox (eventual consistency)

Digunakan untuk efek turunan lintas bounded context yang boleh terlambat beberapa detik: dashboard, laporan, audit, notifikasi, sinkronisasi cache, analitik.

**Mandat outbox:** event lintas context tidak dikirim langsung ke bus. Event ditulis ke tabel `outbox` **dalam transaksi yang sama** dengan perubahan data, lalu sebuah relay mempublikasikannya ke bus/broker. Ini menjamin event tidak hilang saat proses crash antara commit data dan publish. Dead-letter tetap digunakan untuk kegagalan listener.

```
Aggregate commit ──▶ tulis data + tulis outbox (1 transaksi)
                                   │
                                   ▼ relay
                             publish ke bus/broker
                                   │
                    listener (retry → dead-letter)
```

### 2.4 Keputusan per use case

| # | Use case | Strategi | Alasan |
|---|----------|----------|--------|
| 1 | Checkout: reserve stok → create sale → create payment | **Unit of Work**, satu transaksi (Transactional Core) | Wajib atomik; kegagalan stok harus membatalkan seluruh transaksi |
| 2 | Goods Receipt (penerimaan barang) → update stok | **Domain Event + outbox** (`GoodsReceived` → inventory `AdjustStock`) | Mempertahankan single-writer `product_stock` (Inventory); operasi low-frequency non-customer-facing; toleran keterlambatan detik; kode saat ini sudah berbasis event |
| 3 | `SaleCreated` → laporan, dashboard websocket | **Domain Event** | Efek turunan; boleh eventual |
| 4 | `ProductUpdated` → cache invalidation, websocket | **Domain Event** | Invalidation lintas context |
| 5 | Stock opname posting → `inventory_adjustments` + `product_stock` | **Unit of Work internal** (dalam modul Stock Opname/Inventory) | Aggregate dalam satu context; harus atomik |
| 6 | PO `confirmed` / `cancelled` → websocket | **Domain Event** | Notification; tidak ada konsistensi yang perlu dijaga |

Catatan: untuk kasus lain di luar tabel di atas, gunakan prosedur keputusan di 2.1. Kasus #2 saat ini bersifat *per-kasus*: disepakati sebagai eventual + outbox (Model B), namun dapat ditinjau ulang bila ada requirement bisnis yang membutuhkan stok langsung konsisten pada saat GR; keputusan tersebut harus melalui revisi ADR ini.

---

## 3. Konsekuensi (Consequences)

- **Positif:** aturan keputusan tunggal dan konsisten; API modul dapat diprediksi; integritas stok tetap terjamin pada jalur checkout; GR tidak mengotori boundary modul purchase.
- **Negatif:** jendela kecil (detik) antara GR commit dan stok ter-update; diperlukan infrastruktur `outbox` + relay + reconciliation; laporan/dashboard bergantung pada keandalan event.
- **Kebutuhan baru:** tabel `outbox`, mekanisme relay, dan metrik keterlambatan pemrosesan event untuk observability.

---

## 4. Langkah migrasi

1. Terapkan pola outbox pada event lintas context yang ada (mulai `GoodsReceived`/`PurchaseReceiptCompleted`).
2. Pastikan checkout tetap dalam satu Unit of Work dan tidak ikut dipindah ke pola event.
3. Dokumentasikan keputusan per use case pada modul terkait.
