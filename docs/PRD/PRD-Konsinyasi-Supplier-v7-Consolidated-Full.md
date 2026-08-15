# PRD — Fitur Konsinyasi Supplier

**Version:** v7 — Consolidated Full PRD  
**Status:** Working PRD / Source of Truth

---

# 1. Overview

Fitur konsinyasi digunakan untuk mengakomodasi supplier yang menitipkan barang kepada toko untuk dijual.

Barang konsinyasi tetap merupakan barang milik supplier sampai terjadi penjualan atau proses return sesuai kondisi bisnis.

Fitur harus memungkinkan toko untuk:
- membedakan stok konsinyasi dari stok toko;
- mengetahui supplier pemilik stok konsinyasi;
- menerima barang setelah pemeriksaan;
- menjual barang melalui POS existing;
- mencatat barang yang perlu diretur;
- melakukan return kepada supplier;
- melakukan settlement atas seluruh barang yang telah terjual dan belum disettlement;
- melakukan payment kepada supplier.

Customer tidak perlu mengetahui status konsinyasi.

# 2. Goals

1. Mendukung supplier yang menitipkan barang kepada toko.
2. Memisahkan ownership stok toko dan stok konsinyasi.
3. Mencegah satu SKU dimiliki oleh lebih dari satu sumber konsinyasi pada waktu yang sama.
4. Mencegah SKU yang sama bercampur antara stok toko dan active consignment stock.
5. Mencatat penerimaan hanya untuk barang yang benar-benar diterima setelah pemeriksaan.
6. Mendukung supplier yang menitipkan lebih dari satu SKU.
7. Mendukung penambahan stok oleh supplier yang sama.
8. Mendukung return barang konsinyasi.
9. Mendukung settlement penuh atas seluruh penjualan konsinyasi yang belum diselesaikan.
10. Mengikuti flow POS, pricing, dan payment existing sejauh memungkinkan.

# 3. Scope

## 3.1 In Scope

- Penyesuaian supplier existing agar dapat ditandai sebagai supplier konsinyasi.
- Consignment arrangement.
- Harga yang disepakati supplier dan toko.
- Hak/potongan toko.
- Penerimaan barang konsinyasi.
- Pemeriksaan barang sebelum pencatatan.
- Ownership stok konsinyasi.
- Conflict SKU.
- Penjualan melalui POS existing.
- Pending Return.
- Consignment Return.
- Consignment Settlement.
- Payment melalui mekanisme existing.
- Perubahan harga.
- Perubahan hak/potongan toko.
- Supplier visit.
- Arrangement termination setelah supplier tidak datang lebih dari 2 minggu.
- Business rules untuk kerusakan, expired, dan eligible customer return.

## 3.2 Out of Scope

- Customer tidak perlu mengetahui status konsinyasi.
- Aplikasi tidak menentukan harga jual berdasarkan hak toko.
- Aplikasi tidak menentukan apakah toko harus menerima barang ketika tidak ada conflict.
- Aplikasi tidak perlu mengetahui alasan bisnis supplier berhenti.
- Detail fisik lokasi penyimpanan Pending Return.
- Proses bisnis yang memang telah disepakati berada di luar aplikasi.

# 4. Terminology

## 4.1 Supplier
Supplier existing yang dapat ditandai sebagai supplier konsinyasi.

## 4.2 Consignment Arrangement
Kesepakatan bisnis antara toko dan supplier terkait barang konsinyasi. Arrangement dapat tetap ada walaupun stok menjadi 0.

## 4.3 Consignment Terms
Kesepakatan mengenai harga dan hak/potongan toko. Hak toko harus berupa salah satu: Percentage atau Fixed Amount. Tidak boleh keduanya.

## 4.4 Consignment Receipt
Dokumen/transaksi pencatatan barang konsinyasi yang benar-benar diterima setelah pemeriksaan. Satu receipt dapat berisi banyak SKU.

## 4.5 Consignment Stock
Stok yang secara ownership merupakan milik supplier dan berada di toko untuk dijual. **Quantity ownership = quantity tersedia + quantity Pending Return.** Quantity ownership inilah yang menentukan apakah SKU dapat dialihkan kepada supplier lain (lihat BR-05b).

## 4.6 Pending Return
Catatan sederhana untuk barang yang sudah ditarik dari display, tidak lagi tersedia untuk dijual, tetapi belum diserahkan kepada supplier.

## 4.7 Consignment Return
Transaksi formal ketika barang benar-benar dikembalikan kepada supplier.

## 4.8 Unsettled Consignment Sale
Penjualan barang konsinyasi yang sudah terjadi tetapi belum masuk settlement.

## 4.9 Consignment Settlement
Penyelesaian seluruh unsettled consignment sales supplier. Partial settlement tidak diperbolehkan.

# 5. Core Business Rules

## 5.1 Supplier

### BR-01 — Supplier Existing
Supplier existing tetap digunakan. Tidak dibuat supplier entity baru khusus konsinyasi. Supplier dapat ditandai sebagai supplier konsinyasi.

## 5.2 SKU Ownership & Conflict

### BR-02 — Satu SKU Tidak Boleh Bercampur dengan Stok Toko
Jika SKU sedang memiliki stok toko, SKU tersebut tidak boleh diterima sebagai consignment stock.

### BR-03 — Satu SKU Tidak Boleh Dimiliki Dua Supplier Konsinyasi
Pada satu waktu, satu SKU tidak boleh memiliki active consignment stock dari dua supplier berbeda.

### BR-04 — Supplier yang Sama Boleh Menambah Stok
Supplier yang sama boleh menambah stok SKU yang sedang menjadi active consignment stock miliknya.

### BR-05 — Supplier Lain Boleh Setelah Available 0 dan Pending Return 0
Jika active consignment stock suatu SKU menjadi 0 **dan tidak ada Pending Return** untuk SKU tersebut, supplier lain boleh menawarkan SKU tersebut. Toko tetap bebas menerima atau menolak.

### BR-05b — Pending Return Tetap Merupakan Ownership
SKU tidak dapat dialihkan kepada supplier konsinyasi lain selama masih terdapat available stock **atau** Pending Return milik supplier sebelumnya. Alih kepemilikan hanya terjadi setelah available stock = 0 **dan** Pending Return = 0.

### BR-06 — Toko Tetap Boleh Menolak
Tidak adanya conflict SKU tidak berarti toko wajib menerima barang.

## 5.3 Receipt

### BR-07 — Pemeriksaan Sebelum Pencatatan
Barang diperiksa terlebih dahulu. Hanya barang yang benar-benar diterima setelah pemeriksaan yang dicatat.

### BR-08 — Barang Ditolak Tidak Masuk Sistem
Barang yang tidak memenuhi standar toko, misalnya rusak, tidak dicatat sebagai consignment stock.

### BR-09 — Multi-SKU Receipt
Satu supplier dapat menitipkan lebih dari satu jenis barang. Satu receipt dapat mencatat beberapa SKU.

### BR-10 — Receipt Quantity
Yang dicatat adalah accepted quantity, bukan quantity yang dibawa supplier.

### BR-11 — Bukti Penerimaan
Receipt/nota/dokumen pencatatan penerimaan dapat diberikan kepada supplier sebagai bukti barang yang diterima konsinyasi.

## 5.4 Price & Store Share

### BR-12 — Harga Berdasarkan Kesepakatan
Supplier menawarkan barang dengan harga. Toko menyepakati harga dan hak/potongan toko. Aplikasi hanya mencatat hasil kesepakatan.

### BR-13 — Aplikasi Tidak Menentukan Harga
Harga jual bukan ditentukan oleh aplikasi berdasarkan formula hak toko.

### BR-14 — Hak Toko Harus Memilih Satu Tipe
Hak toko harus berupa Percentage atau Fixed Amount, tidak keduanya.

### BR-15 — Tidak Ada Diskon Konsinyasi
Tidak ada diskon khusus pada barang konsinyasi sebagai bagian dari fitur konsinyasi. Penjualan mengikuti alur pricing/discount existing.

### BR-16 — Hak Toko Tidak Boleh Berubah Sepihak
Supplier tidak boleh mengubah hak/potongan toko secara sepihak.

### BR-17 — Fixed Store Share Dapat Berubah
Jika hak toko berupa fixed amount, nilainya dapat berubah berdasarkan kesepakatan baru.

### BR-18 — Perubahan Harga Mengikuti Mekanisme Existing
Perubahan harga mengikuti alur perubahan harga yang sudah ada di aplikasi dan berlaku terhadap stok yang ada, bukan hanya supply berikutnya.

### BR-19 — Perubahan Terms Tidak Retroaktif terhadap Sale
Harga dan hak toko yang digunakan untuk settlement adalah yang berlaku ketika penjualan terjadi.

### BR-20 — Perubahan Store Share untuk Stock Belum Terjual
Perubahan hak toko dapat berlaku untuk stok yang belum terjual.

## 5.5 Sales

### BR-21 — Menggunakan POS Existing
Barang konsinyasi dijual melalui flow POS existing.

### BR-22 — Customer Tidak Perlu Aware
Customer tidak perlu mengetahui barang tersebut merupakan barang konsinyasi.

### BR-23 — Sale Mengurangi Consignment Stock
Jika consignment stock terjual, stok konsinyasi berkurang.

### BR-24 — Sale Menjadi Unsettled
Penjualan konsinyasi yang belum disettlement menjadi unsettled consignment sale.

## 5.6 Pending Return

### BR-25 — Barang Rusak Ditarik dari Display
Barang yang sudah tidak layak jual ditarik dari display.

### BR-26 — Pending Return Tidak Bisa Dijual
Barang Pending Return tidak tersedia untuk dijual.

### BR-27 — Pending Return Belum Mengurangi Ownership
Selama barang masih berada di toko dan belum diserahkan kepada supplier, ownership belum berkurang.

### BR-28 — Pending Return Tidak Masuk Settlement
Pending Return tidak menghasilkan kewajiban payment kepada supplier.

### BR-29 — Pending Return Dicatat Sederhana
Pending Return hanya perlu dicatat secara sederhana.

### BR-30 — Supplier yang Sama Tetap Boleh Supply
Supplier yang sama boleh menambah stok SKU walaupun masih ada Pending Return untuk SKU tersebut.

## 5.7 Return

### BR-31 — Return Formal
Return formal terjadi ketika barang benar-benar diserahkan kepada supplier.

### BR-32 — Return Unsold Tidak Menghasilkan Settlement
Barang yang belum terjual dan diretur tidak menghasilkan settlement/payment.

### BR-33 — Supplier Dapat Meminta Return
Supplier dapat meminta return barang yang belum terjual.

### BR-34 — Return dan New Receipt Dapat Terjadi dalam Satu Kunjungan
Tidak perlu memisahkan kunjungan supplier hanya untuk return dan supply.

## 5.8 Damaged / Expired / Customer Return

### BR-35 — Natural Damage
Jika barang rusak dengan sendirinya dan bukan akibat tindakan toko/customer, kerusakan bukan tanggung jawab toko. Barang menjadi Pending Return.

### BR-36 — Customer Damage
Jika barang rusak karena customer, tanggung jawab berada pada toko, kecuali force majeure.

### BR-37 — Expired
Barang expired merupakan tanggung jawab supplier. Barang ditarik dari display dan menjadi Pending Return.

### BR-38 — No Change of Mind
Customer tidak dapat melakukan return tanpa alasan yang memenuhi kebijakan toko.

### BR-39 — Eligible Customer Return
Jika customer return memang eligible, misalnya karena barang rusak, barang tidak kembali menjadi available consignment stock. Barang diarahkan menjadi Pending Return kepada supplier.

## 5.9 Settlement & Payment

### BR-40 — Settlement Hanya untuk Barang Terjual
Settlement hanya mencakup barang yang sudah terjual dan belum disettlement.

### BR-41 — Tidak Ada Partial Settlement
Settlement harus menyelesaikan seluruh unsettled sales supplier.

### BR-42 — Settlement Tidak Mencakup Stock
Settlement tidak mencakup available stock, Pending Return, atau returned unsold goods.

### BR-43 — Settlement Menggunakan Terms Saat Sale
Harga dan hak toko yang digunakan adalah yang berlaku ketika sale terjadi.

### BR-44 — Payment Mengikuti Existing Flow
Setelah settlement, payment dilakukan melalui mekanisme payment existing.

### BR-45 — Settlement Saat Supplier Datang
Settlement dilakukan ketika supplier datang dan toko melakukan settlement.

## 5.10 Supplier Visit & Arrangement

### BR-46 — Satu Kunjungan Dapat Multi-Aktivitas
Satu kunjungan supplier dapat mencakup Receipt, Return, Settlement, dan Payment.

### BR-47 — Arrangement Dapat Aktif Saat Stock 0
Arrangement dapat tetap Active walaupun stok 0.

### BR-48 — Termination >2 Minggu
Jika supplier tidak datang selama lebih dari 2 minggu, arrangement dapat menjadi Ended.

### BR-49 — Ended Tidak Otomatis Return
Arrangement Ended tidak otomatis menyebabkan return.

### BR-50 — Stock Ended Tetap Boleh Dijual
Jika arrangement Ended tetapi masih terdapat stok yang layak jual, stok tetap boleh dijual.

# 6. Business Objects

## 6.1 Supplier
Supplier existing. Perlu penanda bahwa supplier dapat digunakan untuk konsinyasi.

## 6.2 Consignment Arrangement
Mewakili kesepakatan bisnis toko dan supplier. Dapat berhubungan dengan lebih dari satu SKU. Dapat tetap ada ketika stock = 0.

## 6.3 Consignment Terms
Mewakili terms harga dan hak toko. Minimal: harga, tipe hak toko, nilai hak toko. Terms yang digunakan pada sale harus dapat diketahui kembali untuk settlement.

## 6.4 Consignment Receipt
Mencatat accepted goods. Minimal: supplier, tanggal, SKU, accepted quantity, terms yang berlaku, dan dokumen/bukti penerimaan. Satu receipt dapat memiliki banyak SKU.

## 6.5 Consignment Stock
Mewakili stok yang ownership-nya supplier. Minimal dapat diketahui: SKU, supplier owner, quantity tersedia, quantity Pending Return. **Quantity ownership = quantity tersedia + quantity Pending Return**, dan quantity ownership inilah yang dipakai untuk conflict check pengalihan SKU (BR-05b).

## 6.6 Pending Return
Catatan internal sederhana: supplier, SKU, quantity, alasan, tanggal, status.

## 6.7 Consignment Return
Mencatat barang yang benar-benar dikembalikan: supplier, SKU, quantity, tanggal, alasan, referensi Pending Return jika ada.

## 6.8 Sale / Sale Item
Existing POS business object. Sale Item konsinyasi menjadi dasar settlement.

## 6.9 Consignment Settlement
Mencatat penyelesaian seluruh unsettled sales: supplier, tanggal, sale items, total nilai penjualan, total hak toko, total payable supplier, payment reference/status.

## 6.10 Payment
Mengikuti payment existing.

# 7. Domain Relationship

```text
Supplier
   │
   ├───────────────┐
   │               │
   ▼               ▼
Consignment      Consignment
Arrangement       Receipt
   │               │
   │               ▼
   │        Consignment Stock
   │               │
   │       ┌───────┼────────┐
   │       │       │        │
   │       ▼       ▼        ▼
   │     Sale    Pending   Return
   │              Return
   │
   ▼
Consignment Terms

Sale / Sale Item
       │
       ▼
Unsettled Consignment Sales
       │
       ▼
Consignment Settlement
       │
       ▼
Payment
```

# 8. Main Business Process

## 8.1 Supplier Offers Goods

```text
Supplier menawarkan barang
        ↓
Toko mengecek SKU conflict
        ↓
Conflict?
 ┌──────┴──────┐
 Yes           No
  ↓             ↓
Reject       Toko memutuskan
               ↓
          Accept / Reject
```

Tidak adanya conflict tidak membuat toko wajib menerima.

## 8.2 Receipt

```text
Supplier membawa barang
        ↓
Pemeriksaan barang oleh toko
        ↓
Barang sesuai standar?
    ┌──────┴──────┐
   No             Yes
    ↓              ↓
Tidak dicatat   Accepted Qty
                  ↓
            Consignment Receipt
                  ↓
            Consignment Stock
```

## 8.3 Receipt Document

Dokumen penerimaan dapat menunjukkan supplier, barang, quantity diterima, tanggal, dan informasi relevan.

## 8.4 Selling

```text
Consignment Stock
       ↓
Existing POS
       ↓
Sale
       ↓
Stock decreases
       ↓
Unsettled Sale
```

## 8.5 Pending Return / Return

```text
Barang tidak layak jual
       ↓
Tarik dari display
       ↓
Pending Return
       ↓
Supplier visit
       ↓
Return
       ↓
Ownership/stock berkurang
```

## 8.6 Settlement

```text
Supplier visit
       ↓
Toko memilih settlement
       ↓
System identifies ALL unsettled sales
       ↓
Review
       ↓
Confirm Settlement
       ↓
Payment
```

Partial settlement tidak diperbolehkan.

# 9. Detailed Business Scenarios

## Scenario 01 — Initial Receipt

Supplier membawa Product A = 100 dan Product B = 50.

Setelah pemeriksaan:
- Product A accepted = 80, rejected = 20
- Product B accepted = 45, rejected = 5

System records:
- Product A = 80
- Product B = 45

## Scenario 02 — Additional Supply

Product A Supplier A memiliki stock 80. Supplier A membawa 45 accepted. New stock = 125.

## Scenario 03 — Supplier Conflict

Product A Supplier A = 80. Supplier B menawarkan Product A. Result: rejected.

## Scenario 04 — Ownership Becomes Available

Product A Supplier A = 0 **dan Pending Return = 0**. Supplier B menawarkan Product A. Proses dapat dilanjutkan; toko bebas menerima/menolak.

Catatan: selama masih ada Pending Return (walau available stock = 0), SKU belum dapat dialihkan kepada Supplier B (BR-05b).

## Scenario 05 — Sale

```text
Stock = 80
Sale = 10
Stock = 70
Unsettled Sales = 10
```

## Scenario 06 — Natural Damage

```text
Stock = 70
2 pcs damaged naturally
Available = 68
Pending Return = 2
```

## Scenario 07 — Expired

```text
Available Stock
      ↓
Expired
      ↓
Remove from display
      ↓
Pending Return
      ↓
Supplier Return
```

## Scenario 08 — Customer Damage

Customer merusak barang. Kerusakan menjadi tanggung jawab toko, kecuali force majeure.

## Scenario 09 — Eligible Customer Return

Karena no-change-of-mind policy, hanya return yang memenuhi kebijakan toko yang diterima. Jika eligible karena kerusakan:

```text
Sale
 ↓
Eligible Customer Return
 ↓
Item kembali
 ↓
Pending Return
 ↓
Supplier Return
```

## Scenario 10 — Supplier Stops Visiting

Supplier tidak datang >2 minggu → Arrangement Ended. Jika Stock = 30, stock tetap boleh dijual dan tidak ada automatic return.

## Scenario 11 — Price Change

100 received, 30 sold, 70 remain. Harga Rp10.000 → Rp12.000. Sale lama tetap Rp10.000; sale berikutnya dari stock yang tersisa menggunakan harga baru.

## Scenario 12 — Store Share Change

30 pcs sold → 20%. 70 pcs remain → 25%. Sale lama tetap 20%; sale baru menggunakan 25%.

## Scenario 13 — Pending Return + New Supply

Available = 70, Pending Return = 5, new supply = 30. Result: Available = 100, Pending Return = 5.

## Scenario 14 — Supplier Requests Return Before Settlement

Available = 30, supplier requests return = 10. Result: Available = 20, Return = 10, Unsettled Sales unchanged.

## Scenario 15 — One Supplier, Multiple SKU Problems

```text
Supplier A
Product A → Receipt
Product B → Return
Product C → Pending Return
Product D → Settlement
```

Masalah satu SKU tidak menghalangi SKU lain.

# 10. Settlement Rules & Examples

## 10.1 Full Settlement

Jika supplier memiliki Product A = 20, Product B = 10, Product C = 5 unsettled, settlement harus mencakup semuanya.

## 10.2 Percentage Store Share

```text
Sale price = Rp10.000
Store share = 20%
Store = Rp2.000
Supplier = Rp8.000
```

## 10.3 Fixed Store Share

```text
Sale price = Rp10.000
Store share = Rp2.000
Store = Rp2.000
Supplier = Rp8.000
```

## 10.4 Price Change Before Settlement

Sale 10 pcs × Rp10.000, kemudian harga berubah menjadi Rp12.000. Settlement tetap menggunakan Rp10.000.

## 10.5 Store Share Change Before Settlement

Sale 10 pcs dengan store share 20%, kemudian berubah menjadi 25%. Settlement menggunakan 20%.

# 11. Lifecycle

## 11.1 Arrangement

```text
Active
  │
  ├── supplier continues visiting
  │
  └── no visit > 2 weeks
             ↓
           Ended
```

Arrangement dapat Active walaupun stock = 0.

Arrangement Ended tidak otomatis meretur stock.

Jika masih ada stock layak jual, stock tetap boleh dijual.

## 11.2 Consignment Stock

```text
Received
   ↓
Available
   ├── Sold
   │
   ├── Pending Return
   │     ↓
   │  Returned
   │
   └── Remains Available
```

`Unsettled` dan `Settled` adalah status kewajiban/settlement atas sale, bukan status fisik barang.

### Sale / Settlement Lifecycle

```text
Consignment Sale
      ↓
Unsettled
      ↓
Settled
      ↓
Payment
```

# 12. Exception & Edge Case Matrix

| ID | Situation | Expected Business Result |
|---|---|---|
| EC-01 | Supplier sama menambah stok ketika masih ada stok | Boleh |
| EC-02 | Supplier sama menambah stok ketika ada Pending Return | Boleh; Pending Return terpisah |
| EC-03 | Harga berubah sebelum settlement | Sale lama menggunakan harga saat sale |
| EC-04 | Hak toko berubah sebelum settlement | Sale lama menggunakan hak toko saat sale |
| EC-05 | Arrangement Ended tetapi masih ada stok layak jual | Stok tetap boleh dijual |
| EC-06 | Barang rusak sendiri | Pending Return; bukan tanggung jawab toko |
| EC-07 | Barang rusak karena customer | Tanggung jawab toko, kecuali force majeure |
| EC-08 | Barang expired | Pending Return; tanggung jawab supplier |
| EC-09 | Supplier meminta return barang belum terjual | Boleh; tidak menghasilkan settlement |
| EC-10 | Eligible customer return karena kerusakan | Pending Return ke supplier; tidak kembali menjadi available stock |
| EC-11 | Satu supplier banyak SKU dengan kondisi berbeda | Aktivitas SKU independen |
| EC-12 | Available = 0, Pending Return > 0, supplier lain menawarkan SKU | Ditolak sampai Pending Return diselesaikan (BR-05b) |

# 13. Acceptance Criteria

## 13.1 Supplier

### AC-C01
Supplier existing dapat ditandai sebagai supplier konsinyasi dan digunakan untuk transaksi konsinyasi.

## 13.2 Receipt

### AC-C02
Jika supplier membawa 100 pcs dan toko menerima 80 setelah pemeriksaan, hanya 80 yang masuk consignment stock.

### AC-C03
Satu receipt dapat berisi banyak SKU.

### AC-C04
SKU yang sudah menjadi stok toko tidak dapat diterima sebagai consignment stock.

### AC-C05
SKU yang sedang menjadi consignment stock supplier lain tidak dapat diterima.

### AC-C06
Supplier yang sama dapat menambah SKU yang sedang menjadi consignment stock miliknya.

### AC-C07
Jika available consignment stock = 0 **dan tidak ada Pending Return**, supplier lain dapat menawarkan SKU tersebut; toko bebas menerima/menolak.

### AC-C08
Toko dapat menolak barang walaupun tidak ada conflict.

## 13.3 Price & Store Share

### AC-C09
Hak toko harus Percentage atau Fixed Amount, tidak keduanya.

### AC-C10
Harga dan hak toko adalah informasi berbeda.

### AC-C11
Sale lama menggunakan harga ketika sale terjadi.

### AC-C12
Sale lama menggunakan hak toko ketika sale terjadi.

### AC-C13
Perubahan hak toko berlaku untuk penjualan berikutnya dari stock yang belum terjual.

### AC-C14
Supplier tidak dapat mengubah hak toko secara sepihak.

## 13.4 Sales

### AC-C15
Barang konsinyasi dijual melalui POS existing.

### AC-C16
Customer tidak mengetahui status konsinyasi.

### AC-C17
Penjualan mengurangi consignment stock.

### AC-C18
Sale yang belum disettlement tercatat sebagai unsettled consignment sale.

## 13.5 Pending Return

### AC-C19
Barang tidak layak jual dapat menjadi Pending Return.

### AC-C20
Pending Return tidak tersedia untuk dijual.

### AC-C21
Pending Return belum mengurangi ownership.

### AC-C22
Pending Return tidak masuk settlement.

## 13.6 Return

### AC-C23
Formal Return mengurangi consignment stock ketika barang diserahkan kepada supplier.

### AC-C24
Return barang belum terjual tidak menghasilkan payment.

### AC-C25
Pending Return dapat diselesaikan melalui formal Return.

## 13.7 Settlement

### AC-C26
Settlement hanya mencakup barang yang sudah terjual dan belum disettlement.

### AC-C27
Settlement harus mencakup seluruh unsettled sales supplier.

### AC-C28
Partial settlement tidak diperbolehkan.

### AC-C29
Nominal supplier dihitung berdasarkan sale value dikurangi store share sesuai terms saat sale.

### AC-C30
Settlement diikuti payment melalui mekanisme existing.

## 13.8 Supplier Visit

### AC-C31
Satu kunjungan dapat mencakup Receipt, Return, Settlement, dan Payment.

## 13.9 Arrangement

### AC-C32
Arrangement Active dengan stock 0 valid.

### AC-C33
Arrangement dapat Ended setelah supplier tidak datang >2 minggu.

### AC-C34
Ending arrangement tidak otomatis meretur stock; stock layak jual tetap boleh dijual.

## 13.10 Ownership Invariant

### AC-C35
Satu SKU tidak boleh memiliki active consignment stock dari dua supplier berbeda.

### AC-C36
Satu SKU tidak boleh memiliki stok toko dan active consignment stock secara bersamaan.

### AC-C37
Selama masih ada Pending Return milik supplier sebelumnya (walau available stock = 0), SKU tidak dapat dialihkan kepada supplier konsinyasi lain.

# 14. Updated Exception Business Rules

| ID | Rule |
|---|---|
| EC-BR-01 | Arrangement Ended tetap dapat memiliki stok belum terjual. |
| EC-BR-02 | Stok layak jual tetap boleh dijual meskipun arrangement Ended. |
| EC-BR-03 | Barang rusak sendiri menjadi Pending Return dan bukan tanggung jawab toko. |
| EC-BR-04 | Barang rusak karena customer menjadi tanggung jawab toko, kecuali force majeure. |
| EC-BR-05 | Barang expired menjadi tanggung jawab supplier dan diarahkan ke Pending Return. |
| EC-BR-06 | Harga settlement menggunakan harga saat sale. |
| EC-BR-07 | Hak toko settlement menggunakan hak toko saat sale. |
| EC-BR-08 | Supplier dapat meminta return terhadap barang belum terjual. |
| EC-BR-09 | Eligible customer return karena kerusakan menjadi Pending Return kepada supplier dan tidak kembali menjadi available stock. |
| EC-BR-10 | Masalah satu SKU tidak menghalangi pemrosesan SKU lain supplier yang sama. |
| EC-BR-11 | Pending Return tetap merupakan ownership (available + pending return) dan menghalangi pengalihan SKU ke supplier konsinyasi lain sampai pending return selesai. |

# 15. Open / Future Considerations

1. Detail aturan masa kedaluwarsa berdasarkan sisa umur produk ketika diterima.
2. Detail klasifikasi alasan kerusakan.
3. Detail refund/void selain eligible customer return.
4. Detail format final nota/receipt untuk supplier.
5. Detail laporan konsinyasi.
6. Detail hak akses user.
7. Detail audit trail.
8. Detail payment apabila flow existing memiliki keterbatasan.
9. Detail kebijakan settlement jika supplier tidak datang dalam waktu lama. **Konsekuensi BR-05b:** jika supplier berhenti datang (Arrangement Ended) dan meninggalkan Pending Return yang tidak pernah diambil, SKU tersebut tetap terkunci dari supplier lain selama Pending Return belum diselesaikan. Kebijakan release di luar kunjungan (mis. forced return oleh toko dengan permission khusus) ditunda; untuk MVP, skenario ini diterima sebagai konsekuensi ownership-first.
10. Detail rekonsiliasi jika terdapat perbedaan fisik antara catatan dan kondisi barang.

# 16. MVP Business Scope

```text
Supplier Konsinyasi
        ↓
SKU Conflict Check
        ↓
Inspection
        ↓
Receipt
        ↓
Consignment Stock
        ↓
Existing POS Sale
        ↓
┌───────────────┐
│               │
▼               ▼
Settlement    Pending Return
   │               │
   ▼               ▼
Payment         Return
```

Prinsip utama:

> **Satu SKU tidak boleh memiliki stok toko atau active consignment stock (termasuk Pending Return) dari supplier konsinyasi lain pada waktu yang sama.**

Dan:

> **Settlement selalu menyelesaikan seluruh penjualan konsinyasi yang masih outstanding, bukan sebagian.**

# 17. Document Status

Dokumen ini adalah consolidated PRD yang menggabungkan keputusan bisnis dari versi sebelumnya.

v7 mempertahankan detail acceptance criteria dan exception/business rules dari v5, sekaligus mengelompokkannya ke dalam struktur PRD yang lebih teratur.

**v7 menjadi source of truth untuk pembahasan business process selanjutnya.**

Versi sebelumnya tetap dapat dipertahankan sebagai history, tetapi tidak digunakan sebagai sumber requirement utama.

### Amendment Record

- **v7.1 (14-08-2026)** — Menambahkan **BR-05b (Pending Return Tetap Merupakan Ownership)**: SKU tidak dapat dialihkan ke supplier konsinyasi lain selama masih ada available stock atau Pending Return milik supplier sebelumnya. Amendemen terkait agar dokumen koheren: BR-05 dan AC-C07 (release condition = available 0 **dan** pending return 0), definisi ownership di §4.5 dan §6.5 (ownership = available + pending return), Scenario 04, EC-12, AC-C37, EC-BR-11, prinsip di §16, dan catatan konsekuensi di §15.9.
