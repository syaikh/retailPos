# Purchase Order & Goods Receiving — Product Management Review

**Document Version:** 1.0  
**Review Date:** 2026-07-27  
**Reviewer:** Senior Product Manager — ERP/Retail Systems  
**Scope:** `.kilo/plans/1785159133766-purchase-order-goods-receiving.md`  
**Status:** REQUIRES PRODUCT REVISION BEFORE IMPLEMENTATION

---

## Executive Summary

Desain Purchase Order & Goods Receiving saat ini adalah **MVP yang solid untuk operasional dasar**, tetapi **tidak cukup untuk menghadapi real-world purchasing complexity** di segment target: minimarket, supermarket, grosir, toko bangunan, apotek, dan multi-cabang.

Desain sekarang hanya mencakup **20% dari procurement lifecycle yang sesungguhnya**. Sisanya adalah standard requirement di ERP retail. Jika tidak disiapkan sejak awal, kita akan menghadapi:
- **Migration cost mahal** saat menambahkan approval workflow, invoice matching, atau return
- **Operational friction** karena user harus keluar dari sistem untuk proses yang tidak tercover
- **Data integrity issues** karena missing snapshot/audit yang seharusnya ada sejak hari pertama
- **Scalability limits** saat jumlah PO/GR mencapai ribuan per bulan

**Rekomendasi utama:** Pisahkan menjadi **MVP Core** (yang sekarang) dan **Phase 2** (yang tidak boleh diabaikan sejak awal). Beberapa field dan status harus ditambahkan **sekarang** meski belum dipakai, untuk menghindari breaking change migration nanti.

---

## 1. Business Workflow Review

### 1.1 Current Workflow vs Real-World

**Yang sudah tercakup:**
- Draft PO → Confirm → Partial Receive → Full Receive → Cancel

**Yang BELUM tercakup (critical gap):**

| Business Process | Status | Keterangan |
|------------------|--------|------------|
| **Purchase Requisition** | Missing | Di dunia nyata, PO tidak dibuat dari thin air. Ada permintaan pembelian dari kasir/gudang yang melalui approval. Tanpa ini, sulit melacak "siapa yang minta beli apa dan mengapa". |
| **Approval Workflow** | Missing | Manager harus approve PO sebelum confirm. Saat ini Confirm = create. Tidak ada layer kontrol. |
| **Request For Quotation (RFQ)** | Missing | Supplier belum bisa mengirim quotation melalui sistem. Semua dilakukan di luar (WhatsApp/email). |
| **Supplier Confirmation** | Missing | Supplier belum bisa konfirmasi delivery date, partial shipment, atau rejection. |
| **Shipment Tracking** | Missing | Tidak ada tracking "PO sudah dikirim, Estimasi tanggal sampai". |
| **Partial Shipment Handling** | Missing | Supplier bisa kirim 50 unit hari ini, 50 unit minggu depan. Plan hanya mendukung partial receiving, bukan partial shipment tracking. |
| **Backorder** | Missing | Jika supplier hanya kirim 80 dari 100, 20 harus masuk backorder. Plan tidak mendukung ini. |
| **Late Delivery Management** | Missing | Tidak ada flag "late" atau perhitungan on-time delivery. |
| **Supplier Cancellation/Rejection** | Missing | Supplier bisa menolak PO? Bagaimana sistem menanganinya? |

### 1.2 Procurement Lifecycle Alignment

**Current state:**
```
Draft → Confirmed → Goods Receipt → Fully Received
```

**Standard ERP Procurement Lifecycle:**
```
Purchase Request
    ↓
Purchase Requisition (internal approval)
    ↓
Request For Quotation (RFQ)
    ↓
Supplier Quotation
    ↓
Supplier Selection + Negotiation
    ↓
Purchase Order (sent to supplier)
    ↓
Supplier Acknowledgment / Confirmation
    ↓
Shipment / Partial Shipment
    ↓
Goods Receiving (可能 partial)
    ↓
Quality Inspection (QC)
    ↓
Supplier Invoice
    ↓
Invoice Matching (3-way: PO + GR + Invoice)
    ↓
Payment
    ↓
Purchase Return (jika ada)
    ↓
Close
```

**Gap analysis:**
- Plan hanya mencover tahap **Purchase Order → Goods Receiving → Close**.
- Tahap **Purchase Request/Requisition** dan **Approval** sangat dibutuhkan untuk supermarket/grosir.
- Tahap **Supplier Invoice + Payment** adalah modul accounting yang mungkin terpisah, tetapi PO dan GR harus menyediakan data yang cukup untuk integrasi.
- **3-way matching** (PO vs GR vs Invoice) tidak bisa dilakukan tanpa data invoice dan tanpa snapshot harga di GR.

### 1.3 Business Process Recommendations

**Yang PERLU dimasukkan sekarang (MVP):**
1. **Approval workflow minimal:** Setidaknya status `waiting_approval` dan field `approved_by`. Tanpa ini, purchasing di perusahaan dengan aturan tidak bisa jalan.
2. **Supplier Acknowledgment:** Field `acknowledged_at` dan `acknowledged_by` di PO. Supplier bisa "terima" PO sebelum mulai deliver.
3. **Shipment tracking:** Field `shipped_at`, `tracking_number`, `shipping_method` di PO. Walaupun sederhana, ini needed untuk grosir yang tracking delivery.

**Yang bisa ditunda (Phase 2):**
1. RFQ module — kompleks, bisa menggunakan form email manual dulu.
2. Supplier quotation comparison — bisa dilakukan di luar sistem dulu.
3. Invoice matching — tergantung apakah sistem ini ada module accounting atau tidak.
4. Purchase Requisition — bisa menggunakan notes/approval notes di PO dulu.

---

## 2. Procurement Lifecycle

### 2.1 Current Limitation

Desain sekarang **hanya Point of Purchase dan Receiving**. Ini adalah **salah satu tahap** dari procurement yang panjang.

**Konsekuensi:**
- PO yang dibuat tidak bisa ditrack "apakah sudah dibayar?"
- Tidak ada hubungan antara PO dan Supplier Invoice
- Tidak ada Purchase Return
- Tidak ada concepsi "PO Open/Outstanding" vs "PO Closed"

### 2.2 Future Compatibility

Desain saat ini **tidak siap** untuk procurement lifecycle penuh tanpa migration besar:

| Future Feature | Blocker |
|----------------|---------|
| Supplier Invoice | Tidak ada tabel invoices; GR tidak menyimpan harga saat received |
| 3-way Matching | Tidak ada invoice table; harga bisa berubah di master product |
| Purchase Return | Tidak ada return table; tidak ada konsep "returnable" di GR |
| Payment | Tidak ada link ke accounting/payment module |
| Purchase Requisition | Tidak ada requisition table; status machine terlalu sederhana |
| Approval Workflow | Status machine kurang state; tidak ada approval trail |

**Rekomendasi:**
- Tambahkan kolom `reference_number` di PO untuk linking ke invoice/requisition
- Tambahkan status opsional `waiting_approval` dan `rejected` sekarang
- Simpan snapshot harga di GR items untuk enabling 3-way matching nanti
- Tambahkan flag `is_fully_paid` atau `outstanding_amount` di PO jika integrate dengan accounting module

---

## 3. Business Rule Review

### 3.1 Critical Missing Business Rules

**3.1.1 Approval Rules**
- Apakah semua PO harus di-approve? Atau hanya di atas certain amount?
- Apakah user yang membuat PO bisa approve PO sendiri?
- Apakah ada approval matrix (amount-based, category-based)?
- **Saat ini:** Tidak ada. Plan hanya ada Confirm action yang bisa dilakukan oleh siapa saja punya permission `purchase_order.confirm`.

**3.1.2 Supplier Change Policy**
- Apakah supplier boleh diubah setelah PO confirmed?
- **Saat ini:** Tidak ada business rule untuk ini. Jika Confirmed PO items immutable, apakah supplier_id immutable?
- **Rekomendasi:** Supplier tidak boleh diubah setelah confirmed. Tambahkan validasi.

**3.1.3 Price Change Policy**
- Apakah `unit_cost` boleh diubah setelah confirmed?
- **Saat ini:** Confirmed items immutable — tidak bisa ubah unit_cost. Ini **benar** secara bisnis.
- **Catatan:** Pastikan frontend menyembunyikan edit harga untuk confirmed PO.

**3.1.4 Receiving Timing**
- Apakah receiving boleh dilakukan sebelum `expected_date`?
- Apakah receiving boleh dilakukan setelah `expected_date`? Jika ya, apakah ada tolerance (misal 3 hari)?
- **Saat ini:** Tidak ada validasi untuk ini. Real-world: receiving sebelum expected_date adalah normal (early delivery), receiving setelah expected_date perlu di-record dan di-report.
- **Rekomendasi:** Tambahkan flag `is_late_delivery` di GR jika `received_at > expected_date`.

**3.1.5 Receiving Quantity Rules**
- Apakah over-delivery diperbolehkan? (Receive lebih dari qty_ordered)
- **Saat ini:** Tidak — `qty_good + qty_damaged <= remaining`. Ini **benar** untuk strict receiving.
- **Catatan:** Beberapa retailer menerima over-delivery dengan approval. Plan bisa ditambahkan flag `allow_over_delivery` di PO atau global config.

**3.1.6 Partial Receiving Authorization**
- Apakah semua user bisa partial receive? Atau hanya user tertentu?
- **Saat ini:** Permission `purchase_order.receive` untuk semua. Tidak ada distinction partial vs full receive.
- **Rekomendasi:** Tambahkan permission opsional `purchase_order.receive_partial` jika perlu kontrol.

**3.1.7 Receiving Documentation**
- Apakah receiving harus menyimpan nomor Surat Jalan (Delivery Order)?
- Apakah receiving harus menyimpan nomor Invoice Supplier?
- **Saat ini:** Tidak ada. Plan hanya `notes` di GR.
- **Rekomendasi:** Tambahkan `delivery_order_number` dan `supplier_invoice_number` di GR header.

**3.1.8 Receiving Multi-User**
- Apakah satu GR boleh dilakukan oleh beberapa user secara concurrent?
- **Saat ini:** Tidak ada "draft receiving" atau "session receiving". SATU GR = SATU user submit.
- **Rekomendasi:** Tambahkan konsep "Receiving Session" atau minimal `draft_receipts` untuk receiving yang memakan waktu lama.

**3.1.9 One PO - Multiple Invoices**
- Apakah satu PO boleh memiliki banyak invoice dari supplier?
- **Saat ini:** Tidak ada konsep invoice di module ini.
- **Rekomendasi:** Jika nanti integrasi dengan accounting, PO perlu menyimpan total invoiced amount untuk matching.

**3.1.10 Cancellation After Receiving**
- **Saat ini:** Plan melarang cancel jika ada receipts. Ini **benar**.
- **Catatan:** Bagaimana dengan "return all received items" lalu cancel? Perlu PO Return untuk itu.

---

## 4. Goods Receiving Review

### 4.1 Current State

Plan hanya menyimpan:
- Header: `gr_number`, `purchase_order_id`, `store_id`, `received_by`, `received_at`, `notes`
- Items: `purchase_order_item_id`, `product_id`, `qty_good`, `qty_damaged`, `notes`

### 4.2 Missing Receiving Metadata

**Yang PERLU ada di GR Header (MVP):**
| Field | Alasan |
|-------|--------|
| `delivery_order_number` | Nomor surat jalan supplier — required untuk tracing |
| `received_at` | Sudah ada — good |
| `received_by` | Sudah ada — good |
| `shipping_method` | Truck/ courier/ self-pickup — untuk analytics |
| `driver_name` | Untuk grosir/toko bangunan yang tracking supplier performance |
| `vehicle_plate_number` | Untuk grosir/apotek |
| `is_late_delivery` | Derived dari `received_at > expected_date`, tapi bisa juga manual flag |
| `late_reason` | Jika late, simpan alasan |

**Yang PERLU ada di GR Items (MVP):**
| Field | Alasan |
|-------|--------|
| `unit_cost` | Snapshot harga saat receive — **CRITICAL** untuk 3-way matching |
| `product_name` | Snapshot nama produk saat receive — historical accuracy |
| `supplier_id` | Snapshot supplier — useful jika PO supplier berubah |
| `batch_number` | Untuk apotek/toko bangunan yang batch tracking |
| `expired_date` | Untuk apotek — expired date tracking |
| `manufacture_date` | Untuk toko bangunan/apotek |
| `serial_number` | Untuk electronics/expensive items |
| `storage_location` | Untuk warehouse management |

**Yang bisa ditunda (Phase 2):**
- Barcode scan result (bisa via import Excel)
- Attachment/foto receiving (bisa di external app)
- Receiver signature (bisa paper-based dulu)
- Quality inspection result (QC pass/fail)
- Pallet/container info
- Receiving duration

### 4.3 Receiving Workflow Gaps

**Current:** User buka modal/PO detail → input qty → submit.

**Real-world receiving scenarios yang belum tercover:**
1. **Batch receiving:** User scan barcode barang yang datang, sistem auto-fill product + qty.
2. **Draft receiving:** User mulai receiving 50 item, simpan sebagai draft, lanjutkan besok.
3. **Receiving by another user:** User A mulai receiving, tidak selesai. User B lanjutkan.
4. **QC hold:** Barang diterima tapi masuk zona "pending QC", belum masuk stok jual.
5. **Return immediately:** Barang rusak saat dibuka, langsung return tanpa masuk stok.

**Rekomendasi:**
- **Batch receiving:** Pertimbangkan Excel import untuk large PO.
- **Draft receiving:** Tambahkan tabel `draft_receipts` atau `receiving_sessions`.
- **QC hold:** Tambahkan status `pending_qc` di GR items, dan `qc_status` (pass/fail/hold).

---

## 5. Purchase Order Review

### 5.1 Current Header Fields

`po_number`, `supplier_id`, `store_id`, `status`, `expected_date`, `subtotal`, `discount_amount`, `tax_amount`, `grand_total`, `notes`

### 5.2 Missing Header Fields

**Yang PERLU ada untuk MVP:**
| Field | Alasan |
|-------|--------|
| `payment_term` | Net 30, Net 60, COD — standard procurement |
| `shipping_method` | Truck, courier, pickup — untuk tracking |
| `delivery_address` | Beda dari store address (misal ke gudang) |
| `billing_address` | Untuk invoice matching |
| `tax_type` | PPN, PPh, non-tax — untuk apotek/supermarket |
| `internal_memo` | Untuk manager notes yang tidak perlu di-expose ke supplier |
| `priority` | Low, normal, high, urgent — untuk采购排序 |
| `requested_by` | Siapa yang meminta pembelian (bisa beda dengan created_by) |
| `approved_by` | Siapa yang approve PO |
| `approved_at` | Kapan approval terjadi |

**Yang bisa ditunda (Phase 2):**
- `currency` + `exchange_rate` — untuk multi-currency
- `incoterm` — untuk internasional
- `supplier_contact` — bisa ambil dari supplier master
- `department` / `cost_center` / `project` — untuk enterprise accounting
- `warehouse_id` — untuk multi-warehouse
- `reference_number` — untuk linking ke PR/invoice

**Yang TIDAK perlu di MVP:**
- `attachment` — bisa email dulu
- `billing_address` — bisa samakan dengan store address dulu

### 5.3 Missing PO-Level Business Rules

1. **Minimum Order Value (MOV):** Apakah supplier punya minimum order? Plan tidak menyimpan ini.
2. **Lead Time:** Supplier memiliki lead time yang berbeda-beda. Plan tidak menyimpan `expected_delivery_date` yang dihitung dari `order_date + lead_time`.
3. **Freight/Cost Allocation:** Apakah ongkir included/excluded? Plan tidak ada kolom `freight_cost`.
4. **Discount per PO vs per Item:** Plan sudah ada `discount_amount` di PO header dan per item. Pastikan perhitungan jelas: `grand_total = sum(item_subtotal) + tax - header_discount`.

---

## 6. Purchase Order Item Review

### 6.1 Current Item Fields

`purchase_order_item_id`, `product_id`, `qty_ordered`, `qty_received`, `unit_cost`, `discount_amount`, `subtotal`, `notes`

### 6.2 Missing Item Snapshots

**Yang PERLU ada di PO Item (MVP):**
| Field | Alasan |
|-------|--------|
| `product_name` | Jika product di-rename, PO history harus tetap menunjukkan nama saat PO dibuat |
| `sku` | Untuk barcode/scanning — product SKU bisa berubah |
| `supplier_sku` | Supplier punya kode produk sendiri — diperlukan untuk receiving dan matching |
| `uom_id` | Unit of Measure — kg, liter, pcs, box |
| `unit_cost` | Sudah ada — good |
| `tax_rate` | PPN 11% atau 0% — untuk menghitung tax per item |
| `last_purchase_price` | Reference harga sebelumnya — untuk approval/perbandingan |
| `catalog_price` | Harga katalog supplier — untuk cek discount |

**Yang bisa ditunda (Phase 2):**
- `brand_id`, `category_id` — untuk reporting by brand/category
- `lead_time_days` — untuk auto-suggest reorder
- `moq` — minimum order quantity
- `conversion` — jika beli per karton, jual per pcs

### 6.3 Snapshot Philosophy

Semua field yang **bisa berubah di master data** harus di-snapshot di PO item:
- `product_name`
- `sku`
- `supplier_sku`
- `uom_name`

Ini prinsip **immutable event sourcing**: PO adalah "kontrak" yang tidak boleh berubah meskipun master data berubah.

---

## 7. Receiving Flexibility

### 7.1 Current Support

Plan mendukung:
- Partial receiving (multiple GR per PO)
- Split shipment (implied — partial receiving)
- Damaged goods (`qty_damaged`)
- Over-receiving prevention

### 7.2 Missing Flexibility

| Scenario | Support | Keterangan |
|----------|---------|------------|
| **Partial Shipment** | Partial | Hanya partial receiving, tidak ada tracking "shipment 1 of 3" |
| **Backorder** | ❌ | Tidak ada field `backordered_qty` atau status `backordered` |
| **Over Delivery** | ❌ | Strict `qty_good + qty_damaged <= remaining` — tidak ada override |
| **Under Delivery** | ✅ | Partial receiving cover ini |
| **Replacement Item** | ❌ | Tidak ada konsep "barang pengganti" dari supplier |
| **Bonus Item / Free Goods** | ❌ | Tidak ada `is_bonus` flag atau `bonus_qty` |
| **Missing Goods** | ❌ | Tidak ada tracking "seharusnya 100, datang 80, mana 20?" |
| **Supplier Substitution** | ❌ | Tidak ada field `substituted_product_id` |
| **Return Immediately** | ❌ | Tidak ada Purchase Return module |
| **Pending Inspection** | ❌ | Tidak ada QC hold |
| **Quality Control** | ❌ | Tidak ada QC pass/fail/hold |
| **Hold Stock** | ❌ | Tidak ada `hold_qty` atau `available_qty` di GR |

### 7.3 Recommendations

**MVP:**
- Tambahkan `backordered_qty` di PO item (calculated: `qty_ordered - qty_received`)
- Tambahkan flag `is_bonus` di GR item untuk free goods

**Phase 2:**
- QC hold: tambahkan `qc_status` di GR items (pending/pass/fail/hold)
- Return: module Purchase Return terpisah
- Replacement: tambahkan `original_item_id` di GR items

---

## 8. Inventory Process

### 8.1 Current Assumption

Plan mengasumsikan receiving langsung menambah stok jual:
```
Receive → AdjustStock → stok bertambah → available for sale
```

### 8.2 Real-World Inventory Flow

Di supermarket/grosir/apotek, biasanya:
```
Receive
    ↓
Receiving Area (dock)
    ↓
Quality Inspection (QC)
    ↓
Putaway ke rak/gudang
    ↓
Available Stock (bisa dijual/dipakai)
```

### 8.3 Gap Analysis

| Inventory Stage | Current Plan | Gap |
|-----------------|--------------|-----|
| Receiving Area | ❌ Tidak ada | Semua barang langsung masuk stok jual |
| QC Hold | ❌ Tidak ada | Barang rusak bisa masuk stok jika qty_good salah input |
| Warehouse Location | ❌ Tidak ada | Tidak ada tracking lokasi penyimpanan |
| Batch/Lot Tracking | ❌ Tidak ada | Tidak ada batch_number, expired_date |
| Available vs Reserved | ❌ Tidak ada | Tidak ada konsep reserved stock untuk PO lain |

### 8.4 Recommendations

**Untuk MVP:**
- Minimal tambahkan `qc_status` di GR items (qc_pending/qc_pass/qc_fail). Jika `qc_fail`, jangan AdjustStock.
- Tambahkan `storage_location` di GR items.

**Untuk Enterprise:**
- Tambahkan tabel `inventory_locations` untuk warehouse management.
- Tambahkan tabel `inventory_batches` untuk batch/lot tracking.

**Desain sekarang harus MUDah dikembangkan:**
- Jangan langsung AdjustStock di dalam GR transaction. Buat `stock_status = 'pending_qc'` di GR items.
- Tambahkan step kedua: "Confirm Receiving" yang memindahkan dari `pending_qc` ke `available`.
- Atau buat flag `is_available = false` di GR items, dengan job/background process yang auto-approve setelah X jam.

---

## 9. Supplier Performance

### 9.1 Current Data Availability

Apakah data yang disimpan cukup untuk menghitung KPI supplier?

| KPI | Data Tersedia? | Catatan |
|-----|---------------|---------|
| **On Time Delivery** | ❌ | Tidak ada `expected_delivery_date` dan flag `is_late_delivery` |
| **Average Lead Time** | ❌ | Tidak ada `order_date` vs `actual_delivery_date` |
| **Supplier Fill Rate** | ⚠️ | Bisa dihitung dari `qty_received / qty_ordered`, tapi tidak ada tracking per shipment |
| **Supplier Defect Rate** | ⚠️ | Ada `qty_damaged`, tapi tidak ada severity level (minor/major/critical) |
| **Supplier Response Time** | ❌ | Tidak ada tracking komunikasi dengan supplier |
| **Purchase Accuracy** | ⚠️ | Bisa dari `qty_ordered - qty_received`, tapi tidak ada reason code |
| **Supplier Ranking** | ❌ | Tidak ada score/rating |
| **Purchase Cost Trend** | ⚠️ | Ada `unit_cost` di PO items, tapi tidak ada cost history yang terpisah |
| **Purchase Frequency** | ✅ | Bisa dihitung dari count PO per supplier |
| **Supplier Reliability** | ❌ | Tidak ada composite score |

### 9.2 Missing Data for Supplier Performance

**Yang PERLU ditambahkan sekarang:**
- `expected_date` → `received_at` comparison untuk on-time delivery
- `is_late_delivery` flag + `late_reason`
- `supplier_confirmed_date` — kapan supplier konfirmasi bisa deliver
- `shipped_date` — kapan supplier kirim
- `delivered_date` — kapan barang sampai (bisa beda dengan `received_at`)

**Yang bisa ditunda:**
- Supplier rating/score
- Defect severity classification
- Response time tracking (bisa di module komunikasi terpisah)

---

## 10. Reporting Readiness

### 10.1 Current Data Availability

| Report | Data Tersedia? | Gap |
|--------|---------------|-----|
| **Outstanding PO** | ⚠️ | Bisa dari `status != 'fully_received' AND status != 'cancelled'`, tapi tidak ada `outstanding_value` |
| **Open PO** | ⚠️ | Sama dengan outstanding |
| **Late PO** | ❌ | Tidak ada flag late delivery |
| **Late Delivery** | ❌ | Tidak ada `expected_date` vs `actual_delivery_date` tracking |
| **Receiving Report** | ✅ | GR sudah ada |
| **Supplier Report** | ⚠️ | Bisa dari PO + GR, tapi kurang metadata |
| **Purchase Trend** | ✅ | Dari PO created_at |
| **Purchase by Category** | ❌ | Tidak ada category di PO item |
| **Purchase by Brand** | ❌ | Tidak ada brand di PO item |
| **Purchase by User** | ✅ | `created_by`, `received_by` sudah ada |
| **Purchase by Store** | ✅ | `store_id` sudah ada |
| **Purchase Cost Analysis** | ⚠️ | Ada `unit_cost`, tapi tidak ada cost trend yang terpisah |
| **Purchase Price History** | ❌ | Tidak ada price history table |
| **Damaged Goods Report** | ⚠️ | Ada `qty_damaged`, tapi tidak ada severity/reason |
| **Supplier Performance** | ❌ | See section 9 |
| **Backorder Report** | ❌ | Tidak ada backorder tracking |

### 10.2 Recommendations

**Untuk reporting, yang PERLU ditambahkan sekarang:**
- `category_id`, `brand_id` di PO items (untuk reporting by category/brand)
- `is_late_delivery` + `late_reason` di GR (untuk late delivery report)

**Yang bisa ditunda:**
- Price history table — bisa dihitung dari PO items history
- Damage severity — bisa ditambah nanti
- Backorder report — menunggu backorder feature

---

## 11. Multi Store

### 11.1 Current Multi-Store Support

Plan memiliki:
- `store_id` di PO header
- `store_id` di GR header
- Middleware context untuk `store_id`

### 11.2 Multi-Store Scenarios

| Scenario | Current Support | Gap |
|----------|----------------|-----|
| **Central Warehouse + Branch** | ❌ | Tidak ada `warehouse_id`, tidak ada konsep transfer dari central ke branch |
| **Inter Store Transfer** | ❌ | Tidak ada module transfer |
| **Supplier per Store** | ❌ | Supplier mungkin berbeda per store, tapi tidak ada relasi supplier-store |
| **Supplier per Warehouse** | ❌ | Tidak ada warehouse concept |
| **Central Purchasing** | ❌ | Tidak ada "purchasing by head office untuk semua cabang" |
| **Branch Purchasing** | ✅ | `store_id` sudah ada |
| **Warehouse Receiving** | ❌ | Tidak ada `warehouse_id` |
| **Cross Store Receiving** | ❌ | GR `store_id` tied to PO `store_id`? Belum ada validasi |

### 11.3 Recommendations

**Untuk multi-cabang yang basic:**
- Tambahkan `warehouse_id` opsional di PO + GR. Jika NULL, gunakan `store_id`.
- Validasi: `gr.store_id == po.store_id` atau user punya permission cross-store.

**Untuk enterprise:**
- Tabel `warehouses` (sudah ada? perlu dicek)
- Tabel `supplier_store_assignments` untuk supplier yang bisa digunakan per store
- Module `inter_store_transfer` terpisah

**Critical:** `store_id` filter harus **WAJIB** di semua query list. Jangan pernah return data antar store tanpa permission explicit.

---

## 12. Future Modules

### 12.1 Dependency Matrix

| Future Module | Dependency ke PO/GR | Kesiapan Desain |
|---------------|---------------------|-----------------|
| **Purchase Return** | GR items perlu reference untuk return | ❌ Tidak siap — tidak ada return table, tidak ada `returnable_qty` |
| **Supplier Credit Note** | GR + PO untuk matching | ⚠️ Parsial — GR ada, tapi tidak ada invoice |
| **Supplier Debit Note** | PO + GR | ⚠️ Parsial |
| **Supplier Invoice** | PO untuk matching | ❌ Tidak ada invoice table |
| **Invoice Matching** | PO + GR + Invoice | ❌ Butuh ketiga tabel |
| **Payment Voucher** | Invoice | ❌ Tidak ada |
| **Approval Workflow** | PO status + user | ⚠️ Parsial — status kurang, tidak ada approval trail terpisah |
| **Budget Control** | PO + budget table | ❌ Tidak ada budget concept |
| **Purchase Budget** | PO | ❌ |
| **Forecast Purchasing** | Historical PO + sales | ⚠️ Data ada, tapi tidak ada forecasting engine |
| **Auto Reorder** | Product stock + PO | ⚠️ Bisa di-build di module inventory |
| **Suggested Purchase** | Stock + PO history | ⚠️ Bisa di-built |
| **MRP** | Sales + Inventory + PO | ❌ Too far — butuh module manufacturing/planning |
| **Demand Planning** | Sales + Seasonality | ❌ Out of scope |
| **Warehouse Management** | GR + stock | ❌ Tidak ada warehouse/location |

### 12.2 Key Gaps for Future Modules

1. **No Invoice Concept:** Sistem tidak bisa melakukan 3-way matching tanpa tabel invoices.
2. **No Return Concept:** Tidak bisa return ke supplier tanpa module Purchase Return.
3. **No Approval Trail:** Hanya `confirmed_by` — tidak ada approval chain, rejection reason, atau delegation.
4. **No Budget Concept:** Tidak ada batas pengeluaran per department/category.
5. **No Batch/Lot:** Tidak bisa track expiry untuk apotek/food.

### 12.3 Recommendations

**Yang harus dipersiapkan SEKARANG (tanpa implementasi):**
- Tambahkan kolom `reference_number` di PO untuk linking ke PR/invoice/future modules
- Tambahkan status `waiting_approval` dan `rejected` — meskipun tidak dipakai sekarang, migration nanti lebih mudah
- Tambahkan `approved_by` dan `approved_at` di PO — minimal support approval later
- Snapshot di GR items (`unit_cost`, `product_name`, `supplier_id`) — essential untuk invoice matching nanti

**Yang bisa ditunggu:**
- Invoice module — bisa ditambah 6 bulan kemudian
- Return module — bisa ditambah saat ada permintaan
- Budget control — butuh module accounting

---

## 13. UX Review (Operational Perspective)

### 13.1 User Personas & Primary Workflow

| Persona | Primary Task | Current UX Fit |
|---------|--------------|----------------|
| **Purchasing Staff** | Create draft PO, send to supplier | ✅ Form cukup |
| **Manager** | Approve PO | ❌ Tidak ada approval UI |
| **Warehouse Staff** | Receive goods | ⚠️ Modal OK untuk < 20 item, tapi tidak untuk 200+ item |
| **Cashier** | Tidak berhubungan | N/A |
| **Accountant** | Match PO + GR + Invoice, pay supplier | ❌ Tidak ada |

### 13.2 Goods Receiving UX

**Current plan: Modal**

**Operational reality:**
- Minimarket: PO rata-rata 10-30 item → Modal OK
- Grosir/Supermarket: PO bisa 100-500 item → Modal TIDAK cukup
- Apotek: PO bisa 50-100 item dengan batch/expired tracking → Modal TIDAK cukup

**Rekomendasi:**
- **MVP:** Modal untuk PO < 30 item. Jika > 30 item, redirect ke dedicated page.
- **Better UX:** Receiving Queue page — user melihat semua PO yang confirmed/partial, pilih yang ingin direceive.
- **Best UX:** Wizard receiving:
  1. Pilih PO
  2. Scan/list items
  3. Input qty + damaged + batch
  4. Review + submit

### 13.3 PO List UX

**Current:** Table dengan kolom: PO Number, Supplier, Status, Expected Date, Grand Total, Created By, Created At, Actions.

**Missing operational needs:**
- **Quick filter:** "Show only POs awaiting my approval"
- **Bulk action:** Confirm multiple POs sekaligus (untuk manager)
- **Export:** Export PO to CSV/Excel untuk dikirim ke supplier
- **Print:** Print PO format A4 untuk supplier
- **Email/Send:** Kirim PO ke supplier via email (bisa integrasi dengan email module)
- **Duplicate PO:** "Create new PO based on this PO" — sangat sering di purchasing

### 13.4 Draft Receiving

**Current:** Tidak ada.

**Operational need:**
User warehouse menerima barang pagi jam 8, tapi tidak selesai karena ada 500 item. Jam 10 ada PO lain yang lebih urgent. User perlu:
- Save progress receiving
- Lanjutkan receiving nanti
- Assign receiving ke user lain

**Rekomendasi:**
- Tambahkan tabel `receiving_sessions` atau minimal `draft_goods_receipts`
- Status `draft_receiving` di PO untuk tracking "sedang receiving"

### 13.5 Keyboard & Scanning UX

- **Barcode scanner** biasanya menghasilkan input seperti keyboard. Pastikan form receiving bisa handle rapid input (scan barcode → auto-fill product → move to next field).
- **Batch scan mode:** User scan barang, sistem hitung qty otomatis.
- **Keyboard shortcut:** F2 = new PO, F5 = refresh, Ctrl+Enter = submit receiving.

---

## 14. MVP Scope

### 14.1 Feature Triage

**MVP Core (Wajib diluncurkan):**
1. Create Draft PO
2. Edit Draft PO
3. Delete Draft PO
4. Confirm PO (tanpa approval — untuk simple case)
5. Cancel PO (hanya draft atau confirmed tanpa receiving)
6. List PO + filter + search + pagination
7. PO Detail
8. Goods Receiving (partial + multiple)
9. Receiving History
10. Inventory auto-update (qty_good only)
11. Basic audit log
12. Basic permission
13. Status badges

**MVP Plus (Sangat disarankan ditambah sebelum launch):**
1. Approval status (`waiting_approval`, `rejected`) — minimal untuk approval matrix
2. Snapshot di GR items (`unit_cost`, `product_name`) — untuk historical accuracy
3. GR header fields (`delivery_order_number`, `shipping_method`) — untuk operational necessity
4. Composite indexes untuk query performance
5. `approved_by` field di PO — untuk approval tracking
6. `is_late_delivery` flag — untuk late reporting

**Phase 2 (Setelah MVP stabil):**
1. Approval workflow engine (matrix-based)
2. Purchase Requisition module
3. RFQ + Supplier Quotation
4. Supplier Invoice + 3-way Matching
5. Purchase Return
6. Backorder management
7. QC Hold + quality inspection
8. Batch/Lot + Expired Date tracking
9. Multi-warehouse support
10. Barcode scan receiving
11. Draft receiving sessions
12. Supplier performance dashboard
13. Advanced reporting (cost trend, fill rate, etc)
14. Budget control

**Enterprise (Untuk grosir/multi-cabang):**
1. Central purchasing + branch distribution
2. Inter-store transfer
3. Multi-currency
4. Incoterm + international shipping
5. MRP + demand planning
6. Advanced approval workflow with delegation
7. Supplier portal (supplier bisa lihat PO, confirm, kirim invoice)
8. EDI integration (supplier integration via EDI/API)

### 14.2 Why This Scope?

**MVP Core** hanya 10 fitur. Ini adalah **minimum yang bisa dipakai di toko kecil**. Tanpa ini, sistem tidak bisa digunakan.

**MVP Plus** ditambahkan karena:
- **Approval** — needed untuk supermarket/grosir yang punya manager
- **Snapshot** — needed untuk compliance dan audit
- **GR metadata** — needed untuk operational tracking

Tanpa MVP Plus, kita akan dapat ticket "tidak bisa approve PO", "tidak bisa track delivery", "data receiving tidak akurat" dalam 2 bulan pertama production.

**Phase 2** adalah fitur-fitur yang kompleks dan bisa di-build bertahap setelah core stabil.

---

## 15. Recommended Product Roadmap

### Phase 0: Foundation (Sekarang — sebelum coding)
- [ ] Fix transaction atomicity (CRIT-01 dari technical review)
- [ ] Fix event publishing after commit (CRIT-02)
- [ ] Tambah composite indexes
- [ ] Tambah snapshot fields di GR items
- [ ] Tambah `delivery_order_number`, `shipping_method` di GR
- [ ] Tambah status opsional `waiting_approval`, `rejected`
- [ ] Tambah `approved_by`, `approved_at` di PO
- [ ] Audit semua query untuk store_id filter

### Phase 1: MVP Launch (4-6 minggu)
- Semua fitur MVP Core + MVP Plus
- Testing: unit, integration, API, basic E2E
- Deploy ke 1-2 store untuk pilot

### Phase 2: Operational Excellence (2-3 bulan setelah MVP)
- Approval workflow
- Purchase Requisition
- RFQ module
- Supplier Invoice + basic matching
- Purchase Return
- Backorder management
- Advanced reporting

### Phase 3: Enterprise Features (6-12 bulan)
- Multi-warehouse
- QC Hold + quality inspection
- Batch/Lot/Serial tracking
- Barcode scan receiving
- Supplier portal
- EDI integration
- Budget control

---

## Final Recommendation

**Jangan mulai implementasi kode sebelum revisi berikut selesai:**

1. **Fix transaction atomicity** — `AdjustStockTx` pattern (CRITICAL)
2. **Fix event publishing** — after commit only (CRITICAL)
3. **Tambah composite indexes** — performance dari hari pertama
4. **Tambah snapshot fields di GR items** — `unit_cost`, `product_name` (HIGH)
5. **Tambah GR metadata** — `delivery_order_number`, `shipping_method` (HIGH)
6. **Tambah status opsional** — `waiting_approval`, `rejected` (HIGH)
7. **Tambah approval fields** — `approved_by`, `approved_at` (HIGH)
8. **Audit store_id filter** — data leakage prevention (HIGH)
9. **Konsolidasikan format nomor** — pilih per-year vs global (HIGH)
10. **Tambah `qc_status` planning** — meskipun tidak di-MVP, desain harus support QC hold

Setelah revisi desain selesai, implementasi bisa diluncurkan dengan **MVP Core + MVP Plus**. Jangan tunggu semua fitur enterprise siap — tapi pastikan fondasi tidak harus di-rewrite saat fitur ditambahkan.

**Prinsip yang harus dipegang:**
- **Immutable records:** PO, GR, dan semua transaksi pembelian tidak boleh diubah setelah commit.
- **Snapshot everything:** Harga, nama produk, supplier info di snapshot di waktu transaksi.
- **Atomic operations:** GR + stock update + audit dalam satu transaksi.
- **Store-aware by default:** Semua query harus filter store_id.
- **Extensible status machine:** Tambah status baru tanpa migrasi massal.
- **Real-world compatibility:** Desain harus mendukung approval, invoice, return tanpa breaking change.
