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

- `product_stock`/`products.stock` ditulis langsung oleh **lima titik** pada empat modul: `internal/product/bulk.go`, `internal/product/repository.go`, `internal/inventory/repository.go`, `internal/sale/service.go`, `internal/stockopname/repository.go`.
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
