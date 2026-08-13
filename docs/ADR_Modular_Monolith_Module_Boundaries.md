# ADR: Modular Monolith — Batas Modul dan Kontrak Komunikasi

| Field         | Nilai |
|---------------|-------|
| Status        | **Accepted** |
| Tanggal       | 2026-08-05 |
| Deciders      | Tim Produk, Tim Engineering |
| Referensi     | `docs/ADR_Cross_Module_Transaction_Strategy.md` (strategi transaksi lintas modul) |
| Scope         | Batas antar modul, mekanisme komunikasi, kontrak publik per modul, kontrak event, kepemilikan tabel |
| Di luar scope | Pilihan transport jaringan (gRPC/HTTP/message broker), migrasi ke microservice, strategi transaksi lintas modul (lihat ADR terpisah) |

---

## 1. Konteks (Context)

Aplikasi saat ini adalah *layered monolith* dengan satu database PostgreSQL bersama. Komunikasi antar modul terjadi melalui empat mekanisme:

1. **SQL lintas-tabel langsung** (dominan) — modul membaca dan menulis tabel milik modul lain.
2. **Panggilan interface in-process** — disambung di composition root (`internal/wiring/wiring.go`).
3. **Event bus in-memory** — payload berupa tipe domain mentah (mis. `*sale.Sale`).
4. **Shared cache & library bersama** — satu instance `pkg/cache` dipakai lintas repository.

Audit kode menemukan kebocoran boundary yang signifikan:

- `product_stock` ditulis langsung oleh **beberapa titik** pada beberapa modul: `internal/product/bulk.go`, `internal/product/repository.go`, `internal/inventory/repository.go`, `internal/sale/service.go`, `internal/stockopname/repository.go`. (Kolom legacy `products.stock` sudah dihapus di migrasi `029_drop_products_stock.sql` — D6A dari security remediation plan; `v_products_full` membaca `stock` dari `product_stock.quantity`.)
- `internal/report` dan `internal/shift` membaca `sales`/`sale_items`/`sale_payments` langsung.
- `internal/inventory/purchase_receipt_listener.go` dan `pkg/websocket/listener.go` mengimpor package domain modul lain demi type-assert payload event (mis. `*sale.Sale`, `purchase.PurchaseReceiptPayload`).
- Tidak ada penegakan arsitektur: `.golangci.yaml` tidak memiliki `depguard`, dan tidak ada test batas modul.

Akibatnya, modul tidak dapat diubah internalnya (schema, index, query, cache) tanpa berisiko merusak modul lain, dan pemisahan modul ke layanan/database terpisah membutuhkan perombakan besar.

---

## 2. Keputusan (Decision)

Arsitektur target adalah **modular monolith**: modul berkomunikasi hanya melalui **kontrak** (service dan event), bukan melalui implementasi database.

### 2.1 Hanya dua mekanisme komunikasi

| Mekanisme | Kegunaan | Transport saat ini |
|---|---|---|
| **Sync — Application Service / Query** | Request/response, butuh hasil sekarang | Panggilan method interface in-process (composition root) |
| **Async — Domain Event** | Notifikasi efek samping, boleh eventual | In-process event bus; kelak broker (Kafka/NATS/RabbitMQ) |

Tidak ada mekanisme ketiga. Khususnya: **tidak ada akses langsung ke tabel, repository, atau ORM milik modul lain.**

### 2.2 Database bukan media komunikasi

Modul tidak boleh mengetahui struktur database modul lain. Pemilik tabel bebas mengubah schema, index, query, dan cache tanpa memengaruhi pemanggil. Pola yang benar:

```
Sale ──▶ InventoryService ──▶ Inventory Repository ──▶ Database
```

bukan:

```
Sale ──▶ Database (tabel milik Inventory)
```

### 2.3 Tiga kontrak publik per modul

Setiap modul hanya boleh mengekspos tiga area publik:

```
Inventory
  ├── Application Service : Reserve(), Adjust(), Release()   (command)
  ├── Query               : LookupStock()                     (read)
  └── Events              : StockAdjusted, StockReserved      (outbox)
```

Repository, entity, ORM, SQL, cache, dan helper adalah **private**. Modul lain dilarang mengimpor package internal modul tersebut.

### 2.4 Interface kecil per use case

Anti-*God Service*. Interface diekspos per use case, bukan satu service raksasa:

```go
type StockReserver interface { Reserve(ctx, items) error }
type StockAdjuster interface { Adjust(ctx, ...) error }
```

bukan:

```go
type InventoryService interface { /* 25 method */ }
```

### 2.5 Query dan Command dipisah

Interface read (Query) terpisah dari interface command (Application Service) sejak awal, agar CQRS mudah diterapkan nanti. Contoh: `ProductLookup` (query) berbeda dari `ProductService.Create` (command).

### 2.6 Event hanya untuk notification, bukan query

Dilarang pola request/response via event:

```
Sale ──Publish "NeedProductPrice"──▶ Product ──Publish "ProductPrice"──▶ Sale
```

Jika butuh jawaban sekarang, gunakan interface sinkron. Event hanya untuk kejadian domain nyata yang memiliki subscriber, misalnya `SaleCreated`, `SaleCancelled`, `PurchaseConfirmed`, `GoodsReceived`, `StockAdjusted`, `ProductUpdated`. Kejadian tanpa subscriber bisnis (mis. `PriceResolved`, `ProductLoaded`) tidak dijadikan event.

### 2.7 Kontrak event versionable

Event memakai DTO terpisah dari model domain, di package `internal/events`, dengan nama dan versi:

- Nama topic: `sale.created.v1`
- Tipe: `SaleCreatedV1 { ... }`

Penambahan field memerlukan versi baru; subscriber lama tidak rusak. Publisher dan subscriber tidak saling mengimpor package domain.

### 2.8 Kepemilikan tabel (target)

| Bounded Context | Tabel yang dimiliki |
|---|---|
| Transaksional (1 DB) | `sales`, `sale_items`, `sale_payments`, `product_stock`, `inventory_movements`, `shifts`, `payment_methods`, `cart_sessions`, `cart_items` |
| Katalog | `products`, `categories`, `brands`, `units_of_measure`, `tax_classes`, `pricing_rules` |
| Prokuremen | `purchase_orders`, `purchase_order_items`, `goods_receipts`, `goods_receipt_items` |
| Stock Opname | `stock_opnames`, `stock_opname_items`, `stock_opname_counts`, `stock_opname_assignments`, `stock_opname_recount_requests`, `stock_opname_session_scopes`, `inventory_adjustments`, `inventory_adjustment_items` |
| Referensi | `stores`, `warehouses`, `storage_locations`, `customers`, `customer_groups`, `suppliers` |
| Analitik (read model) | materialized views + tabel laporan (hanya baca) |
| Platform | `users`, `roles`, `permissions`, `role_permissions`, `refresh_tokens`, `audit_logs`, `import_jobs`, `import_snapshots`, `import_rows`, `import_errors`, `outbox`, `dead_letter_events` |

Baca lintas context diizinkan hanya pada read model / reporting (CQRS); command tidak boleh.

### 2.9 Penegakan (enforcement)

- `depguard` di `.golangci.yaml`: larang import package internal modul lain; izinkan hanya `internal/events`, `internal/shared`, `internal/config`, `internal/middleware`, `internal/permissions`, `internal/eventbus`, `pkg/*`, dan package milik sendiri.
- Test arsitektur berbasis AST (`internal/archtest`): memastikan tidak ada modul non-pemilik yang menyentuh tabel tertentu (scan `INSERT`/`UPDATE`/`DELETE`/`FROM`) dan tidak ada import lintas domain.

---

## 3. Konsekuensi (Consequences)

- **Positif:** komunikasi tidak lagi bergantung pada topologi database; modul dapat diarahkan ke DB terpisah dengan mengubah DSN pool + transport, tanpa menyentuh logika bisnis; boundary eksplisit dan ter-enforce.
- **Negatif:** panggilan lintas modul yang tadinya gratis via SQL kini memerlukan interface/event; read path yang membutuhkan data banyak modul (reporting) harus didesain ulang sebagai read model.
- **Kebutuhan baru:** package `internal/events` untuk DTO, tabel `outbox`, dan mekanisme relay.

---

## 4. Langkah migrasi

1. Tetapkan ownership tabel (Tabel di 2.8) dan pasang `depguard` + `internal/archtest` terlebih dahulu.
2. Pindahkan kontrak event domain ke `internal/events` (mulai dari modul pilot `purchase`).
3. Enkapsulasi read lintas modul di belakang Query interface.
4. Enkapsulasi write lintas modul di belakang Application Service; pindahkan ke pola transaksi sesuai ADR strategi transaksi.
5. Terapkan aturan yang sama ke seluruh modul secara bertahap.

---

## 5. Status implementasi (2026-08-08)

Penegakan di `internal/archtest` sudah berjalan dan hampir menyeluruh. Seluruh modul domain **kecuali `report`** terdaftar di `strictModuleTables` dan hanya boleh menyentuh tabel yang dimilikinya (modul yang tidak terdaftar ditegakkan lewat `moduleContext`/`tableContext`: baca lintas context boleh, tulis lintas context dilarang).

### 5.1 Pola port sisi-konsumen (consumer-side port)

Baca/tulis lintas modul dienkapsulasi lewat interface kecil yang dideklarasikan **di modul pemakai**, diimplementasikan **di modul pemilik tabel** (structural typing — tidak ada import package pemakai), dan disambung di composition root (`internal/wiring/wiring.go`):

1. `internal/<pemakai>/ports.go` — deklarasi interface `XxxProvider` + dokumen tanggung jawab port.
2. `internal/<pemilik>/<xxx>_provider.go` — struct implementasi milik pemilik tabel.
3. DTO bersama di `internal/shared` bila signature port memakai tipe lintas modul (mis. `shared.StockSetItem`, `shared.InventoryMovement`, `shared.LocationStockReconcile`, `shared.UserRoleRef`).
4. `SetXxxProvider(...)` pada repository/service pemakai; composition root memanggil setter saat wiring; repository **fail-fast** (error runtime) bila port belum di-wire saat dipakai.

### 5.2 Kepemilikan per modul (kondisi aktual `strictModuleTables`)

| Modul | Tabel yang dimiliki | Status |
|---|---|---|
| `category` | `categories` | strict |
| `purchase` | `purchase_orders`, `purchase_order_items`, `goods_receipts`, `goods_receipt_items` | strict |
| `shift` | `shifts` | strict |
| `sale` | `sales`, `sale_items`, `sale_payments`, `payment_methods`, `cart_sessions`, `cart_items` | strict |
| `supplier` | `suppliers` | strict |
| `inventory` | `product_stock`, `inventory_movements` | strict |
| `product` | `products`, `product_suppliers`, `tax_classes`, `v_products_full`, `categories` | strict |
| `pricing` | `pricing_rules` | strict |
| `customer` | `customers` | strict |
| `brand` | `brands` | strict |
| `uom` | `units_of_measure` | strict |
| `customergroup` | `customer_groups` | strict |
| `stockopname` | `stock_opnames`, `stock_opname_items`, `stock_opname_counts`, `stock_opname_assignments`, `stock_opname_recount_requests`, `stock_opname_session_scopes`, `inventory_adjustments`, `inventory_adjustment_items` | strict |
| `store` | `stores`, `warehouses` | strict |
| `storagelocation` | `storage_locations` | strict |
| `user` | `users`, `roles`, `permissions`, `role_permissions`, `refresh_tokens`, `audit_logs` | strict |
| `platform` | `import_jobs`, `import_snapshots`, `import_rows`, `import_errors`, `outbox`, `dead_letter_events` | strict |
| `report` | read model `mv_*` (hanya baca) | **lax** |

Catatan:
- `categories` dimiliki bersama oleh `category` (CRUD) dan `product` (LEFT JOIN pada query restore produk by barcode) — pengecualian yang disengaja.
- `audit_logs` ditulis oleh `internal/audit` (shared infrastructure, di luar `domainModules`), tapi kepemilikan tabel ditetapkan ke `user` (platform).
- `inventory_movements` dimiliki `inventory`; `stockopname` menulisnya lewat port `MovementWriter` di dalam Unit of Work posting, bukan CopyFrom langsung.
- `warehouses` dimiliki `store`; `storage_locations` dimiliki `storagelocation`; `payment_methods` dan `tax_classes` (tabel referensi dari seed) dimiliki masing-masing `sale` dan `product`.

### 5.3 Debt lintas context yang diakui (`crossContextDebt`)

`crossContextDebt` adalah mekanisme untuk menandai referensi lintas modul yang sengaja dipertahankan sementara menunggu port; entri tersebut tetap memicu pelanggaran pada modul non-pemilik. Archtest memeriksa **stale entry** — entri yang sudah tidak relevan (referensi sudah diport) harus dihapus, sehingga debt tidak diam-diam tertinggal tanpa refactor. Saat ini **tidak ada entri** (semua referensi lintas modul sudah melalui port).

### 5.4 Batasan yang tersisa

- **`report` tetap lax (read model):** analitik diizinkan `SELECT` ke tabel domain mana pun (CQRS read-model allowance), tapi tidak boleh menulis tabel domain. Ini disengaja dan konsisten dengan §2.8.
- **`audit` adalah shared infrastructure:** boleh diimpor/dibaca dari mana saja; tidak ditegakkan oleh `internal/archtest`.
- Posting stock opname bergantung pada tiga port inventory (`StockApplier`, `StockLocker`, `MovementWriter`) yang berjalan pada `tx` pemanggil agar atomis (lihat `ADR_Cross_Module_Transaction_Strategy`).

