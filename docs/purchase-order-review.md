# Purchase Order & Goods Receiving — Critical Design Review

**Document Version:** 1.0  
**Review Date:** 2026-07-27  
**Reviewer:** Senior Software Architect  
**Scope:** `docs/purchase-order-implementation-plan.md` and `.kilo/plans/1785159133766-purchase-order-goods-receiving.md`  
**Status:** REQUIRES REVISION BEFORE IMPLEMENTATION

---

## Executive Summary

Implementasi modul Purchase Order & Goods Receiving secara umum mengikuti pola Clean Architecture yang sudah ada di project. Namun, terdapat **satu masalah kritis** yang berpotensi menyebabkan inkonsistensi data produksi: **ketidak-atomanan transaksi antara Goods Receipt dan Inventory Adjustment**. Saat ini desain mengandalkan dua transaksi terpisah (GR transaction + AdjustStock transaction) yang tidak saling melingkari, sehingga partial failure dapat menghasilkan state yang tidak konsisten.

Selain itu, terdapat beberapa **high-priority issues** pada database design, status machine, dan historical data integrity yang perlu diperbaiki sebelum implementasi.

**Rekomendasi utama:** Lakukan revisi desain terlebih dahulu, khususnya pada transaction flow dan inventory integration, sebelum menulis kode apa pun.

---

## Critical Findings

### CRIT-01: Non-Atomic Transaction Between Goods Receipt and Inventory Adjustment

**Masalah:**
Plan saat ini mendeskripsikan flow `CreateGoodsReceipt` sebagai berikut:
1. `BEGIN` transaksi GR
2. Lock PO + items
3. Insert `goods_receipts`
4. Update `qty_received` pada PO items
5. **Panggil `inventory.AdjustStock()`** — fungsi ini **membuka transaksi sendiri** (`r.db.Begin(ctx)` di `inventory/repository.go:66`)
6. Recalculate PO status
7. `COMMIT` transaksi GR

Ini berarti ada **dua transaksi database terpisah yang tidak saling melingkari**:
- Transaksi A: GR + PO update
- Transaksi B: Inventory adjustment (dipanggil di tengah-tengah Transaksi A)

**Dampak:**
- **Scenario 1 (GR commit, AdjustStock gagal):** GR tercatat dan PO status terupdate, tetapi stok tidak bertambah. Data tidak konsisten.
- **Scenario 2 (AdjustStock commit, GR rollback):** Stok bertambah tanpa ada GR record. Lebih parah — stok "hilang" tanpa jejak.
- **Scenario 3 (AdjustStock publish event sebelum commit):** Event `stock.adjusted` sudah dibroadcast sebelum transaksi GR commit. Jika GR nanti rollback, WebSocket listeners sudah menerima notifikasi stok berubah yang seharusnya tidak terjadi.

**Risiko:**
- Data integrity violation di produksi
- Stok bisa melompat atau hilang tanpa audit trail yang jelas
- Reconciliation menjadi sangat sulit

**Contoh kasus:**
User melakukan receiving 100 unit. GR berhasil di-insert, qty_received terupdate. Kemudian AdjustStock gagal karena deadlock. GR tetap ada, stok tidak bertambah. PO menampilkan "partial received" tetapi stok tidak mencerminkan itu.

**Rekomendasi:**
**WAJIB diperbaiki sebelum implementasi.**

**Opsi A (Disarankan):** Tambahkan method baru di `inventory/repository.go`:
```go
func (r *Repository) AdjustStockTx(ctx context.Context, tx pgx.Tx, productID int, quantityChange int, userID *int, notes string) error
```
Method ini menjalankan seluruh SQLAdjustStock menggunakan `tx` yang sudah ada, tanpa `BEGIN` baru. Lalu panggil method ini dari `purchase/service.go` di dalam transaksi GR yang sama. Event `stock.adjusted` hanya dipublish **setelah** `tx.Commit()`.

**Opsi B:** Refactor `inventory.Service` untuk menerima `pgx.Tx` opsional. Jika `tx != nil`, gunakan tx tersebut; jika `nil`, buat tx baru (backward compatible untuk endpoint inventory adjustment yang existing).

**Catatan:** Jangan menggunakan PostgreSQL savepoint sebagai solusi karena project pattern saat ini tidak menggunakannya dan akan menambah kompleksitas tanpa solusi yang benar-benar atomic.

---

### CRIT-02: Event Published Inside Transaction Before Commit

**Masalah:**
`inventory.AdjustStock` mempublish event `stock.adjusted` **sebelum** `tx.Commit()` (`inventory/service.go:31-38`). Jika transaksi gagal dan di-rollback, event tetap sudah dipublish ke bus dan diterima oleh WebSocket listeners.

**Dampak:**
- WebSocket clients menerima `stock_update` untuk perubahan yang sebenarnya tidak terjadi
- Dashboard/real-time monitor menampilkan stok yang tidak sesuai database
- State divergence antara cache/realtime dan source of truth

**Risiko:**
- High —直接影响 UI real-time yang menjadi salah satu fitur penting project

**Rekomendasi:**
**WAJIB diperbaiki sebelum implementasi.**

Event harus dipublish **setelah** transaksi commit. Pola yang benar:
```go
// Dalam service atau repository
err = tx.Commit(ctx)
if err != nil { return err }

// Publish event AFTER commit
_ = s.eventBus.Publish(ctx, ...)
```

Jika inventori diubah dari dalam transaksi PO/GR, event bisa dipublish oleh caller setelah commit.

---

## High Priority Findings

### HIGH-01: Missing Composite Indexes untuk Query Pattern

**Masalah:**
Migration hanya membuat single-column indexes. Berdasarkan pola query yang ada di project (`sale/repository.go`, `product/repository.go`), query PO akan melakukan filter kombinasi seperti:
- `WHERE status = $1 AND store_id = $2 AND created_at > $3`
- `WHERE supplier_id = $1 AND status = $2`
- `ORDER BY created_at DESC` untuk list

**Index yang hilang:**
```sql
CREATE INDEX idx_purchase_orders_status_store ON purchase_orders(status, store_id);
CREATE INDEX idx_purchase_orders_supplier_status ON purchase_orders(supplier_id, status);
CREATE INDEX idx_purchase_orders_created_at_desc ON purchase_orders(created_at DESC);
CREATE UNIQUE INDEX idx_purchase_orders_po_number ON purchase_orders(po_number);  -- UNIQUE sudah ada, tapi index untuk lookup
CREATE UNIQUE INDEX idx_goods_receipts_gr_number ON goods_receipts(gr_number);
```

**Dampak:**
- Query list + filter akan melakukan sequential scan saat data bertambah
- Responsivitas halaman menurun seiring bertambahnya data

**Rekomendasi:**
Tambahkan composite indexes di migration. Prioritas: `(status, store_id)` dan `(created_at DESC)`.

---

### HIGH-02: Goods Receipt Item Tidak Menyimpan Snapshot

**Masalah:**
`goods_receipt_items` saat ini hanya menyimpan:
- `purchase_order_item_id`
- `product_id`
- `qty_good`
- `qty_damaged`
- `notes`

Tidak ada: `unit_cost`, `product_name`, `supplier_id`, `uom_id`, `subtotal`

**Dampak:**
- Jika harga di PO item diubah di masa depan (melalui price update), historical GR tidak bisa menampilkan harga saat receiving terjadi
- Report "purchase cost by date" harus join ke `purchase_order_items` dan `products` — tapi jika product di-rename atau supplier berubah, historical data tidak akurat
- Audit trail tidak bisa menampilkan "GR #001 received 100 unit @ Rp 5,000" secara self-contained

**Rekomendasi:**
Tambahkan kolom snapshot di `goods_receipt_items`:
```sql
unit_cost INT NOT NULL,
product_name VARCHAR(255) NOT NULL,
supplier_id INT,
notes TEXT
```
Data diisi dari PO item + product + supplier saat GR dibuat. Ini mirip prinsip "immutable event sourcing" — snapshot pada waktu receiving terjadi.

---

### HIGH-03: Inventory Movement Tidak Mencukupi untuk Audit

**Masalah:**
`inventory_movements` saat ini hanya menyimpan `reference_id` (integer tanpa tipe) dan `reference_table` (string nullable). Tidak ada:
- `document_number` (GR number, PO number)
- `source_module` ("purchase", "sale", "adjustment")
- `reference_type` yang terenkapsulasi

**Dampak:**
- Query "tampilkan semua inventory movement untuk PO #001" harus mencari `reference_id` dan mencocokkan dengan tabel `goods_receipts` dan `purchase_orders` — kompleks dan rapuh
- Audit trail inventory tidak bisa menampilkan nomor dokumen secara langsung
- Perbedaan antar `type = 'adjustment'` untuk sale vs purchase vs manual adjustment tidak jelas

**Rekomendasi:**
Tambahkan kolom:
```sql
reference_type VARCHAR(50),  -- 'purchase_receipt', 'sale', 'adjustment', 'transfer'
reference_number VARCHAR(50), -- GR number, INV number, dll
source_module VARCHAR(50)     -- 'purchase', 'sale', 'inventory'
```

Atau minimal tambahkan `reference_number` untuk memudahkan query audit.

---

### HIGH-04: Status Machine Kurang untuk Approval Workflow

**Masalah:**
Status saat ini: `draft → confirmed → partial_received → fully_received` + `cancelled`.

Real-world purchasing sering memerlukan approval workflow:
- PO dibuat → menunggu approval → approved → confirmed → receiving → ...
- Atau: PO dibuat → rejected (oleh manager) → kembali ke draft

**Dampak:**
- Jika nanti ditambahkan approval, migrasi status akan ribet karena tidak ada state "waiting_approval" atau "rejected"
- Business rule "hanya admin yang bisa confirm" harus diimplementasikan via permission saja, bukan via status

**Rekomendasi:**
Tambahkan status opsional yang tidak mengganggu flow existing:
```go
StatusWaitingApproval = "waiting_approval"
StatusRejected       = "rejected"
```
Implementasi bisa diabaikan di tahap awal, tapi skema DB harus mendukungnya dari awal agar migration berikutnya mudah.

---

### HIGH-05: Number Generation Scope Tidak Jelas

**Masalah:**
Format `PO-2026-000001` menggunakan global sequence (`po_seq`). Tidak ada reset per tahun atau per store.

**Dampak:**
- Sequence akan terus bertambah tanpa batas. `PO-2026-000001` → `PO-2027-000150` → `PO-2030-000500`. Format tahun di prefix menjadi misleading karena nomor urut tidak reset per tahun.
- Jika company multi-store, nomor PO antar store bisa bentrok (duplicate `po_number`).
- Audit/accounting sering mengharapkan nomor yang bisa di-trace per tahun.

**Rekomendasi:**
Pilih salah satu strategi dan konsistenkan:
1. **Per-year reset (disarankan untuk Indonesia):** `PO-2026-000001`, reset sequence setiap Januari. Atau gunakan composite: `SELECT nextval('po_seq')` + format manual.
2. **Global tanpa year prefix:** `PO-000001` — lebih sederhana, tidak ada masalah reset.
3. **Per-store:** `PO-STORE1-2026-000001` — untuk multi-store.

Jika memilih opsi 1, migration harus menambahkan logic di repository untuk generate nomor dengan tahun, bukan cuma `nextval()`.

---

### HIGH-06: Multi-Store Data Leakage Risk

**Masalah:**
Plan menyebutkan `store_id` dari middleware context, tetapi tidak ada jaminan bahwa:
- Semua query list (`GetAllPurchaseOrders`) selalu filter `store_id`
- Query untuk supplier lookup (jika perlu filter supplier per store) sudah benar
- Goods Receipt bisa dilakukan oleh user dari store A ke PO milik store B

**Dampak:**
- User store A bisa melihat/memodifikasi PO store B jika ada bug di query filter
- Data leakage antar store, terutama di multi-store environment

**Rekomendasi:**
- Semua query yang return list harus include `store_id` filter
- Tambahkan validasi di service: `if po.store_id != userStoreID && !userIsSuperadmin { return ErrForbidden }`
- Audit seluruh query di repository untuk memastikan tidak ada yang skip `store_id`

---

## Medium Priority Findings

### MED-01: Money Type — INT vs DECIMAL

**Masalah:**
Project menggunakan `INT` untuk `subtotal`, `grand_total`, `unit_cost`. Plan mengadopsi pola yang sama.

**Dampak:**
- INT bekerja untuk Rupiah (tidak ada desimal) tetapi berisiko:
  - Integer overflow jika total > 2.1 miliar (akan overflow int32). Project sepertinya menggunakan Go `int` yang bisa 64-bit, tapi di PostgreSQL kolom adalah `INT` (32-bit).
  - Perhitungan rounding jika nanti ada diskon persentase atau pajak non-integer
- Tidak ada kolom `currency` untuk multi-currency di masa depan

**Rekomendasi:**
Verifikasi dulu apakah project memang menggunakan INT untuk semua uang. Jika ya, pastikan:
- Kolom PostgreSQL menggunakan `BIGINT` bukan `INT`
- Frontend dan backend tidak pernah melakukan floating point arithmetic
- Tambahkan constraint `CHECK (grand_total >= 0)` dan `CHECK (unit_cost >= 0)`

Jika project terbuka untuk multi-currency di masa depan, pertimbangkan `NUMERIC(19,4)` sekarang agar tidak perlu migrasi nanti.

---

### MED-02: qty_received Denormalization — Consistency Risk

**Masalah:**
`qty_received` di `purchase_order_items` adalah denormalized field yang di-update setiap kali GR dibuat. Plan mengandalkan atomic update di dalam transaksi.

**Risiko:**
- Jika ada bug di `RecalculatePOStatus` atau `UpdatePOItemQtyReceived`, `qty_received` bisa menyimpang dari sum(`goods_receipt_items.qty_good`)
- Rollback transaksi akan meng-rollback `qty_received`, tapi GR items tetap ada (dalam transaction yang sama, jadi seharusnya aman — tapi perlu diverge test)

**Rekomendasi:**
- Tambahkan trigger atau generated column untuk validasi: `qty_received = (SELECT COALESCE(SUM(qty_good), 0) FROM goods_receipt_items WHERE purchase_order_item_id = X)`
- Atau tambahkan periodic reconciliation job
- Unit test harus coverage rollback scenario: GR gagal → qty_received tidak berubah

---

### MED-03: Missing Damaged Goods Tracking

**Masalah:**
Plan mengakui bahwa `qty_damaged` tidak menambah stok jual, tetapi tidak ada tabel `damaged_inventory` untuk melacak barang rusak.

**Dampak:**
- Barang rusak "hilang" tanpa jejak
- Tidak bisa melaporkan damaged goods rate
- Tidak bisa memutuskan apakah barang rusak bisa diperbaiki, di-retur ke supplier, atau di-scrap

**Rekomendasi:**
Jika damaged goods tracking penting bagi operasional, tambahkan tabel `damaged_inventory` sekarang:
```sql
CREATE TABLE damaged_inventory (
    id SERIAL PRIMARY KEY,
    goods_receipt_item_id INT NOT NULL,
    product_id INT NOT NULL,
    qty INT NOT NULL,
    status VARCHAR(20) DEFAULT 'damaged', -- damaged, returned, scrapped, repaired
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```
Jika tidak, tambahkan minimal notes di `goods_receipt_items.notes` untuk mencatat alasan damaged.

---

### MED-04: Frontend Modal UX untuk Large PO

**Masalah:**
Plan mengusulkan `GoodsReceiptModal.svelte` sebagai modal untuk proses receiving.

**Dampak:**
- Jika PO memiliki 200+ item (misalnya PO bulk untuk supermarket), modal akan sangat padat dan sulit digunakan
- Scroll di dalam modal memiliki UX issues (focus trap, escape key, overlay click)
- Validation message sulit ditampilkan dengan jelas untuk banyak item

**Rekomendasi:**
Gunakan **dedicated page** untuk Goods Receiving, bukan modal. Alur:
1. User buka PO detail
2. Klik "Receive Goods"
3. Navigasi ke `/purchase-orders/:id/receive` (dedicated page)
4. Form receiving dengan better layout untuk large datasets (batch input, inline edit, etc)

Modal bisa dipertahankan untuk PO dengan < 10 item sebagai quick action.

---

### MED-05: Audit Log Coverage Gap

**Masalah:**
Plan mencantumkan audit actions seperti `purchase_order.partially_received` dan `purchase_order.fully_received`. Namun:
- Tidak ada audit untuk `qty_received` change pada item level
- Tidak ada audit untuk perubahan `unit_cost` atau `discount_amount` pada draft
- Tidak ada audit untuk perubahan status otomatis oleh sistem (RecalculatePOStatus)

**Rekomendasi:**
- Setiap kali `qty_received` berubah, buat audit log `purchase_order.item_updated`
- Audit log untuk automatic status change: `purchase_order.status_changed` dengan `old_values` dan `new_values`
- Tambahkan `created_by` dan `updated_by` di semua tabel utama (sudah ada di PO, pastikan ada di GR)

---

### MED-06: Search Performance

**Masalah:**
Plan menyebutkan search on `po_number`, `supplier name`. Jika menggunakan ILIKE dengan leading wildcard (`%search%`), query akan lambat saat data bertambah.

**Rekomendasi:**
- Gunakan `pg_trgm` extension untuk trigram similarity search, atau
- Gunakan `ILIKE 'search%'` (prefix search) untuk po_number
- Untuk supplier name, gunakan index di kolom `name` tabel `suppliers`
- Pertimbangkan full-text search (`tsvector`) jika search harus bebas posisi wildcard

---

### MED-07: Frontend State Management Complexity

**Masalah:**
Plan mengusulkan `po-store.svelte.ts` dengan pola yang sama seperti `sales-store.svelte.ts`. Namun PO form memiliki state yang lebih kompleks:
- Items array yang dinamis (add/remove row)
- Calculation real-time (subtotal, discount, tax, grand total)
- Dirty state untuk draft edit

**Rekomendasi:**
- Pisahkan store menjadi dua: `po-list-store` (untuk list page) dan `po-form-store` (untuk create/edit)
- Form state bisa menggunakan reactive `$state` langsung di component untuk item array yang sering berubah
- Konsisten dengan pola `sales-store.svelte.ts` tapi jangan copy-paste blindly

---

## Low Priority Findings

### LOW-01: Sequence Gap Documentation

**Masalah:**
PostgreSQL sequence akan memiliki gap jika transaction rollback setelah `nextval()` dipanggil.

**Dampak:**
- Nomor PO/GR tidak berurutan tanpa celah
- User mungkin bingung melihat `PO-2026-000001`, `PO-2026-000003` (nomor 2 hilang karena rollback)

**Rekomendasi:**
- Dokumentasikan bahwa gap adalah expected behavior
- Jika bisnis menuntut nomor berurutan tanpa celah, implementasi harus完全不同 (menggunakan tabel counter dengan lock explicit), tetapi ini akan menurunkan throughput significantly. **Tidak direkomendasikan.**

---

### LOW-02: Hard Delete Policy untuk Draft

**Masalah:**
Plan menggunakan hard delete untuk draft PO. Ini sesuai dengan "Drafts are mutable" philosophy.

**Rekomendasi:**
- Dokumentasikan policy: "Draft yang di-delete tidak bisa di-recover"
- Tambahkan confirmation dialog dengan text yang jelas: "This will permanently delete the draft PO"
- Pastikan tidak ada foreign key yang reference `purchase_orders.id` dengan `ON DELETE RESTRICT` — plan sudah menggunakan `ON DELETE CASCADE` untuk items, yang benar

---

### LOW-03: Soft Delete untuk Supplier

**Masalah:**
Tabel `suppliers` memiliki `deleted_at` (soft delete). Plan tidak menjelaskan apa yang terjadi jika supplier di-soft-delete setelah PO dibuat.

**Dampak:**
- PO dengan supplier yang sudah di-delete tetap bisa dilihat (karena `supplier_id` masih valid)
- Tapi `JOIN suppliers` akan return NULL

**Rekomendasi:**
- Query PO detail/read-only bisa menggunakan `LEFT JOIN suppliers` agar PO tetap terlihat meskipun supplier dihapus
- Form create/edit PO hanya menampilkan supplier dengan `deleted_at IS NULL`

---

### LOW-04: Import/Export Registration

**Masalah:**
Plan tidak menyebutkan apakah PO perlu didaftarkan di import/export engine (`internal/platform/importexport`).

**Rekomendasi:**
- Jika PO perlu di-export (CSV/Excel), tambahkan schema dan adapter di `internal/platform/importexport`
- Jika tidak perlu di tahap awal, tambahkan ke backlog

---

### LOW-05: WebSocket Event untuk PO

**Masalah:**
Plan tidak menyebutkan apakah PO status change perlu dibroadcast via WebSocket.

**Rekomendasi:**
- Pertimbangkan event `po_created`, `po_confirmed`, `po_status_changed` untuk real-time update di dashboard
- Bisa menggunakan event bus yang sudah ada + listener baru di WebSocket hub

---

## Recommended Design Changes

### 1. Revised Transaction Flow (REQUIRED)

```
CreateGoodsReceipt flow (SINGLE transaction):
1. BEGIN tx
2. LockPurchaseOrderForUpdate(ctx, tx, poID)
3. Load PO + items (dalam tx)
4. Validasi status dan qty
5. Insert goods_receipts (dalam tx)
6. Untuk setiap item:
   a. Insert goods_receipt_items (dalam tx)
   b. UpdatePOItemQtyReceived (dalam tx)
   c. AdjustStockTx(ctx, tx, productID, qty_good, userID, notes)  ← GUNA TX YANG SAMA
7. RecalculatePOStatus(ctx, tx, poID)
8. COMMIT tx
9. Publish events (setelah commit)
10. Audit log (setelah commit)
```

**Perubahan kode yang diperlukan:**

`inventory/repository.go` — tambah:
```go
func (r *Repository) AdjustStockTx(ctx context.Context, tx pgx.Tx, productID int, quantityChange int, userID *int, notes string) error {
    // Sama seperti AdjustStock tapi tanpa tx.Begin() dan tx.Commit()
    // Gunakan tx yang diberikan untuk semua query
}
```

`inventory/service.go` — tambah:
```go
func (s *Service) AdjustStockTx(ctx context.Context, tx pgx.Tx, productID int, quantityChange int, userID int, notes string) error {
    err := s.repo.AdjustStockTx(ctx, tx, productID, quantityChange, &userID, notes)
    if err != nil {
        return fmt.Errorf("adjust stock: %w", err)
    }
    // Jangan publish event di sini — caller yang publish setelah commit
    return nil
}
```

`purchase/service.go` — di `CreateGoodsReceipt`:
```go
// Setelah COMMIT
if err := tx.Commit(ctx); err != nil { return err }

// Publish events AFTER commit
_ = s.eventBus.Publish(ctx, "goods_receipt.created", ...)
_ = s.eventBus.Publish(ctx, "stock.adjusted", ...)  // jika perlu
```

---

### 2. Revised Database Design

Tambahkan kolom berikut di migration:

```sql
-- goods_receipt_items: tambah snapshot
ALTER TABLE goods_receipt_items ADD COLUMN IF NOT EXISTS unit_cost INT NOT NULL DEFAULT 0;
ALTER TABLE goods_receipt_items ADD COLUMN IF NOT EXISTS product_name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE goods_receipt_items ADD COLUMN IF NOT EXISTS supplier_id INT;

-- inventory_movements: tambah reference metadata
ALTER TABLE inventory_movements ADD COLUMN IF NOT EXISTS reference_number VARCHAR(50);
ALTER TABLE inventory_movements ADD COLUMN IF NOT EXISTS source_module VARCHAR(50) DEFAULT 'purchase';

-- purchase_orders: tambah status opsional untuk approval workflow
-- (sudah ada kolom status, tinggal tambahkan konstanta di code)

-- Indexes tambahan
CREATE INDEX idx_purchase_orders_status_store ON purchase_orders(status, store_id);
CREATE INDEX idx_purchase_orders_supplier_status ON purchase_orders(supplier_id, status);
CREATE INDEX idx_purchase_orders_created_at_desc ON purchase_orders(created_at DESC);
CREATE INDEX idx_goods_receipts_po_id ON goods_receipts(purchase_order_id);
```

---

### 3. Revised Number Generation

Pilih salah satu:

**Opsi A (Disarankan — Per-year dengan fallback):**
```go
func (r *Repository) GetNextPONumber(ctx context.Context) (string, error) {
    var seq int
    err := r.db.QueryRow(ctx, `SELECT nextval('po_seq')`).Scan(&seq)
    if err != nil { return "", err }
    year := time.Now().In(shared.JakartaLocation()).Year()
    return fmt.Sprintf("PO-%d-%06d", year, seq), nil
}
```
Catatan: sequence tidak reset per tahun, tetapi prefix tahun membuat nomor tetap readable. Gap antara tahun adalah expected.

**Opsi B (Global tanpa year):**
```go
return fmt.Sprintf("PO-%06d", seq), nil
```

---

### 4. Revised Status Machine

```go
const (
    StatusDraft           = "draft"
    StatusWaitingApproval = "waiting_approval"  // opsional untuk approval workflow
    StatusConfirmed       = "confirmed"
    StatusPartialReceived = "partial_received"
    StatusFullyReceived   = "fully_received"
    StatusCancelled       = "cancelled"
    StatusRejected        = "rejected"          // opsional untuk approval workflow
    StatusClosed          = "closed"            // opsional untuk tutup PO setelah fully received
)
```

Valid transisi:
- `draft → confirmed` (via confirm action)
- `draft → cancelled` (via cancel action)
- `confirmed → partial_received` (via first GR)
- `partial_received → fully_received` (via last GR)
- `confirmed → cancelled` (hanya jika qty_received == 0)
- `draft → waiting_approval` (opsional)
- `waiting_approval → confirmed` (opsional)
- `waiting_approval → rejected` (opsional)
- `fully_received → closed` (opsional, manual close)

---

### 5. Revised API Design

Tambahkan endpoint opsional untuk approval workflow:
```go
r.POST("/purchase-orders/:id/submit-for-approval", auth, perm("purchase_order.update"), h.SubmitForApproval)
r.POST("/purchase-orders/:id/approve", auth, perm("purchase_order.approve"), h.Approve)
r.POST("/purchase-orders/:id/reject", auth, perm("purchase_order.approve"), h.Reject)
```

---

## Future Compatibility Analysis

| Fitur Mendatang | Kesiapan Desain Saat Ini | Catatan |
|-----------------|--------------------------|---------|
| Purchase Return | Perlu tabel baru `purchase_returns` + `purchase_return_items` | GR snapshot akan membantu |
| Supplier Invoice | Bisa menggunakan `goods_receipts` sebagai basis | Perlu tambah kolom `invoice_number` di GR atau tabel baru |
| Backorder | `qty_received < qty_ordered` sudah naturalmente support | Hanya perlu field `backorder_status` |
| Landed Cost | Perlu tabel `po_cost_allocations` | GR snapshot akan membantu |
| Approval Workflow | Butuh tambah status `waiting_approval`/`rejected` | Mudah ditambahkan |
| Multi Warehouse | Perlu kolom `warehouse_id` di GR + stock per warehouse | Desain saat ini belum siap |
| Batch/Lot | Perlu tabel `goods_receipt_batches` | GR items perlu ditambah kolom batch |
| Multi Currency | Perlu kolom `currency` + `exchange_rate` di PO | INT vs DECIMAL perlu ditentukan |
| Purchase Discount per Item | Sudah ada `discount_amount` di PO item | Pastikan service menghitungnya benar |

---

## Final Recommendation

**Jangan implementasi kode sebelum revisi berikut selesai:**

1. **CRIT-01 (Atomic Transaction):** Tambahkan `AdjustStockTx` ke inventory repository. Ini adalah blocker utama.
2. **CRIT-02 (Event After Commit):** Pindah publishing event ke setelah commit transaksi.
3. **HIGH-01 (Indexes):** Tambahkan composite indexes di migration.
4. **HIGH-02 (GR Snapshots):** Tambahkan kolom snapshot di `goods_receipt_items`.
5. **HIGH-03 (Inventory Movement Reference):** Tambahkan `reference_number` dan `source_module`.
6. **HIGH-04 (Status Machine):** Tambahkan status opsional untuk approval workflow.
7. **HIGH-05 (Number Generation):** Konsolidasikan format nomor dan dokumentasikan scope (global/per-year/per-store).
8. **HIGH-06 (Multi-Store):** Audit seluruh query untuk store_id filter.

Setelah revisi desain selesai, baru mulai implementasi bertahap sesuai TDD plan yang ada.

---

## Appendix: Consistency Check dengan Project Existing

| Aspek | Project Existing | Plan Saat Ini | Kesesuaian |
|--------|------------------|---------------|------------|
| Package structure | `internal/{module}/domain.go, repo.go, service.go, handler.go` | ✅ Konsisten | OK |
| Repository pattern | `db shared.DBPool`, `BeginTx(ctx)` | ✅ Konsisten | OK |
| Service pattern | Compose repo + eventBus, business logic | ✅ Konsisten | OK |
| Handler pattern | Bind JSON, extract context, call service, audit log | ✅ Konsisten | OK |
| Transaction pattern | Single `pgx.Tx` per use case | ❌ **Tidak konsisten** — AdjustStock buat tx sendiri | **MUST FIX** |
| Event publishing | After commit in sale service | ❌ **Tidak konsisten** — AdjustStock publish sebelum commit | **MUST FIX** |
| Number generation | `SELECT nextval('seq')` + format | ✅ Konsisten | OK |
| Pagination | `shared.ParsePaginationParams`, `QueryBuilder` | ✅ Konsisten | OK |
| Frontend module | `web/src/modules/{name}/` dengan components, services, stores | ✅ Konsisten | OK |
| Svelte patterns | `$state`, `$derived`, `$effect`, lazy route import | ✅ Konsisten | OK |
| Permissions | Dot notation `.view`, `.create` di backend + frontend | ✅ Konsisten | OK |
| Audit logging | `audit.AuditCreator` after mutation | ✅ Konsisten | OK |
| Money type | INT for currency | ✅ Konsisten | OK |

**Kesimpulan:** Secara arsitektur dan pola kode, plan sangat konsisten dengan project. Yang tidak konsisten — dan berbahaya — adalah **transaction handling di inventory integration**. Itu adalah satu-satunya blocker kritis.
