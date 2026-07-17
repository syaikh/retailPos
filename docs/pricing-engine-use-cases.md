# Pricing Engine — Use Case & Contoh Penggunaan

Dokumen ini menjelaskan secara detail masing-masing **Tipe Harga**, **Metode Harga**, dan bagaimana **Stacking** (penggabungan rule) bekerja dalam Pricing Engine v2.

---

## Daftar Isi

1. [Tipe Harga](#1-tipe-harga)
2. [Metode Harga](#2-metode-harga)
3. [Kombinasi Tipe × Metode — Contoh Penggunaan](#3-kombinasi-tipe--metode--contoh-penggunaan)
4. [Stacking (Penggabungan Rule)](#4-stacking-penggabungan-rule)
5. [Algoritma Prioritas & Resolusi](#5-algoritma-prioritas--resolusi)
6. [Edge Cases](#6-edge-cases)

---

## 1. Tipe Harga

Tipe harga menentukan **konteks kapan** sebuah pricing rule aktif.

### 1.1 Default

Harga yang berlaku sebagai **harga normal** produk. Tidak ada syarat khusus — berlaku untuk semua customer di semua outlet, kecuali ada aturan lain yang lebih spesifik.

**Karakteristik:**
- Berlaku sepanjang waktu (kecuali ada batasan jadwal)
- Berlaku untuk semua customer dan outlet
- Menjadi **baseline** — rule lain yang lebih spesifik akan menimpa (override) rule default

**Contoh:**
```
Nama     : Harga Normal Indomie Goreng
Tipe     : Default
Metode   : Harga Tetap
Nilai    : Rp 3.500
Target   : Produk "Indomie Goreng"
Outlet   : Semua Outlet
```
→ Indomie Goreng dijual Rp 3.500 di semua outlet untuk semua customer.

---

### 1.2 Daftar Harga (Price List)

Harga khusus yang berlaku untuk **kelompok customer atau outlet tertentu**. Berguna untuk membedakan harga grosir, member, reseller, atau antar cabang.

**Karakteristik:**
- Hanya berlaku jika customer/outlet sesuai kondisi
- Bisa dibedakan per customer group (VIP, Grosir, Reseller)
- Bisa dibedakan per outlet (Cabang A lebih mahal dari Cabang B)

**Contoh:**
```
Nama            : Harga Grosir Indomie
Tipe            : Daftar Harga
Metode          : Harga Tetap
Nilai           : Rp 2.800
Target          : Produk "Indomie Goreng"
Customer Group  : Grosir
Outlet          : Semua Outlet
```
→ Hanya customer group "Grosir" yang mendapat harga Rp 2.800. Customer lain tetap harga normal.

---

### 1.3 Promosi (Promotion)

Harga diskon/promo yang **terbatas waktu**. Cocok untuk flash sale, event musiman, obral akhir tahun, atau promosi hari raya.

**Karakteristik:**
- Punya periode berlaku (tanggal mulai & selesai)
- Bisa dibatasi hari dan jam tertentu
- Bersifat sementara — otomatis nonaktif setelah periode berakhir
- Bisa digabung dengan rule lain (jika `allow_combine = true`)

**Contoh:**
```
Nama       : Flash Sale Indomie
Tipe       : Promosi
Metode     : Diskon (%)
Nilai      : 15%
Target     : Produk "Indomie Goreng"
Outlet     : Semua Outlet
Jadwal     : Senin - Jumat
Jam        : 09:00 - 12:00
Berlaku    : 2026-07-01 s/d 2026-07-31
```
→ Indomie Goreng diskon 15% hanya Senin-Jumat, jam 9-12, selama bulan Juli 2026.

---

## 2. Metode Harga

Metode harga menentukan **bagaimana** harga jual dihitung dari harga normal.

### 2.1 Harga Tetap (Fixed Price)

**Mengganti** harga normal langsung dengan harga pasti. Harga normal diabaikan.

```
Harga Normal : Rp 50.000
Metode       : Harga Tetap
Nilai        : Rp 45.000

→ Harga Jual = Rp 45.000
```

**Kapan digunakan:**
- Harga grosir tetap yang berbeda dari harga ecer
- Harga khusus untuk event tertentu
- Harga promosi dengan nominal pasti

---

### 2.2 Diskon Persen (Discount Percent)

Potongan harga berupa **persentase** dari harga normal.

```
Harga Normal : Rp 50.000
Metode       : Diskon (%)
Nilai        : 20%

→ Diskon    = Rp 50.000 × 20% = Rp 10.000
→ Harga Jual = Rp 50.000 - Rp 10.000 = Rp 40.000
```

**Kapan digunakan:**
- Flash sale ("Diskon 30% semua elektronik")
- Happy hour ("Diskon 10% jam 14:00-16:00")
- Diskon member VIP

---

### 2.3 Diskon Nominal (Discount Amount)

Potongan harga berupa **nominal tetap** (Rupiah).

```
Harga Normal : Rp 50.000
Metode       : Diskon (Rp)
Nilai        : Rp 15.000

→ Diskon    = Rp 15.000
→ Harga Jual = Rp 50.000 - Rp 15.000 = Rp 35.000
```

**Kapan digunakan:**
- Diskon flat ("Potongan Rp 20.000 untuk pembelian di atas Rp 100.000")
- Cashback nominal
- Voucher diskon dengan nominal tetap

---

### 2.4 Markup Persen (Markup Percent)

Penambahan harga berupa **persentase** dari harga normal.

```
Harga Normal : Rp 50.000
Metode       : Markup (%)
Nilai        : 10%

→ Markup    = Rp 50.000 × 10% = Rp 5.000
→ Harga Jual = Rp 50.000 + Rp 5.000 = Rp 55.000
```

**Kapan digunakan:**
- Markup otomatis untuk outlet premium
- Surcharge untuk layanan tambahan
- Harga khusus daerah dengan biaya operasional lebih tinggi

---

## 3. Kombinasi Tipe × Metode — Contoh Penggunaan

Berikut contoh nyata untuk setiap kombinasi tipe dan metode.

### 3.1 Default × Harga Tetap

**Skenario:** Menetapkan harga jual dasar untuk semua produk.

```
Produk      : Sabun Lifebuoy 100g
Harga Normal : Rp 8.500
Tipe        : Default
Metode      : Harga Tetap
Nilai       : Rp 8.500
Target      : Produk "Sabun Lifebuoy 100g"
```

**Hasil:**
- Customer biasa beli → Rp 8.500
- Member VIP beli → Rp 8.500 (tidak ada harga lain)
- Grosir beli → Rp 8.500 (tidak ada harga lain)

---

### 3.2 Default × Diskon (%)

**Skenario:** Diskon permanen untuk produk tertentu (clearance/stok lama).

```
Produk      : Kemeja Batik Lama
Harga Normal : Rp 250.000
Tipe        : Default
Metode      : Diskon (%)
Nilai       : 25%
Target      : Produk "Kemeja Batik Lama"
```

**Hasil:**
- Harga jual = Rp 250.000 - (Rp 250.000 × 25%) = **Rp 187.500**
- Berlaku untuk semua customer, semua outlet, sepanjang waktu

---

### 3.3 Default × Diskon (Rp)

**Skenario:** Diskon flat untuk produk tertentu.

```
Produk      : Minyak Goreng 2L
Harga Normal : Rp 45.000
Tipe        : Default
Metode      : Diskon (Rp)
Nilai       : Rp 5.000
Target      : Produk "Minyak Goreng 2L"
```

**Hasil:**
- Harga jual = Rp 45.000 - Rp 5.000 = **Rp 40.000**

---

### 3.4 Default × Markup (%)

**Skenario:** Outlet premium menambah markup otomatis.

```
Produk      : Kopi Arabica 250g
Harga Normal : Rp 85.000
Tipe        : Default
Metode      : Markup (%)
Nilai       : 15%
Target      : Produk "Kopi Arabica 250g"
Outlet      : Mall Central (outlet premium)
```

**Hasil:**
- Harga jual = Rp 85.000 + (Rp 85.000 × 15%) = **Rp 97.750**
- Hanya berlaku di outlet Mall Central
- Outlet lain tetap Rp 85.000

---

### 3.5 Daftar Harga × Harga Tetap

**Skenario:** Harga grosir khusus untuk reseller.

```
Produk      : Teh Pucuk 350ml (per dus, 36 pcs)
Harga Normal : Rp 120.000
Tipe        : Daftar Harga
Metode      : Harga Tetap
Nilai       : Rp 95.000
Target      : Produk "Teh Pucuk 350ml"
Customer Group : Reseller
```

**Hasil:**
- Customer biasa beli → Rp 120.000 (harga normal)
- Reseller beli → **Rp 95.000** (harga grosir)

---

### 3.6 Daftar Harga × Diskon (%)

**Skenario:** Member VIP dapat diskon eksklusif.

```
Produk      : Serum Vitamin C
Harga Normal : Rp 185.000
Tipe        : Daftar Harga
Metode      : Diskon (%)
Nilai       : 20%
Target      : Kategori "Skincare"
Customer Group : VIP
```

**Hasil:**
- Customer biasa beli serum → Rp 185.000
- Member VIP beli serum → Rp 185.000 - (Rp 185.000 × 20%) = **Rp 148.000**
- Member VIP beli produk skincare lain → juga dapat diskon 20% (target kategori)

---

### 3.7 Daftar Harga × Diskon (Rp)

**Skenario:** Diskon flat untuk grosir.

```
Produk      : Gula Pasir 1kg
Harga Normal : Rp 18.000
Tipe        : Daftar Harga
Metode      : Diskon (Rp)
Nilai       : Rp 3.000
Target      : Produk "Gula Pasir 1kg"
Customer Group : Grosir
```

**Hasil:**
- Grosir beli → Rp 18.000 - Rp 3.000 = **Rp 15.000**

---

### 3.8 Daftar Harga × Markup (%)

**Skenario:** Outlet di daerah remote menambah biaya transportasi.

```
Produk      : Susu Formula Bayi
Harga Normal : Rp 350.000
Tipe        : Daftar Harga
Metode      : Markup (%)
Nilai       : 8%
Target      : Kategori "Bayi & Anak"
Outlet      : Cabang Papua
```

**Hasil:**
- Harga jual di Cabang Papua = Rp 350.000 + (Rp 350.000 × 8%) = **Rp 378.000**
- Outlet lain tetap Rp 350.000

---

### 3.9 Promosi × Harga Tetap

**Skenario:** Flash sale dengan harga miring.

```
Produk      : T-shirt Oversize
Harga Normal : Rp 150.000
Tipe        : Promosi
Metode      : Harga Tetap
Nilai       : Rp 89.000
Target      : Produk "T-shirt Oversize"
Outlet      : Semua Outlet
Berlaku     : 2026-08-01 s/d 2026-08-07
Jadwal      : Senin - Minggu
```

**Hasil:**
- Selama periode 1-7 Agustus → harga jual **Rp 89.000**
- Setelah 7 Agustus → kembali ke harga normal Rp 150.000

---

### 3.10 Promosi × Diskon (%)

**Skenario:** Happy hour — diskon untuk semua minuman.

```
Produk      : (semua produk di kategori "Minuman")
Harga Normal : Rp 25.000 (rata-rata)
Tipe        : Promosi
Metode      : Diskon (%)
Nilai       : 30%
Target      : Kategori "Minuman"
Outlet      : Semua Outlet
Jadwal      : Senin - Jumat
Jam         : 14:00 - 17:00
Berlaku     : 2026-07-01 s/d 2026-12-31
```

**Hasil:**
- Jus Jeruk (Rp 25.000) → Rp 25.000 - 30% = **Rp 17.500**
- Teh Tarik (Rp 18.000) → Rp 18.000 - 30% = **Rp 12.600**
- Kopi Susu (Rp 30.000) → Rp 30.000 - 30% = **Rp 21.000**
- Hanya berlaku Senin-Jumat, jam 14:00-17:00

---

### 3.11 Promosi × Diskon (Rp)

**Skenario:** Voucher diskon untuk event tertentu.

```
Produk      : Sepatu Running Nike
Harga Normal : Rp 1.200.000
Tipe        : Promosi
Metode      : Diskon (Rp)
Nilai       : Rp 200.000
Target      : Produk "Sepatu Running Nike"
Outlet      : Semua Outlet
Berlaku     : 2026-08-17 s/d 2026-08-20 (HUT RI)
```

**Hasil:**
- Harga jual = Rp 1.200.000 - Rp 200.000 = **Rp 1.000.000**

---

### 3.12 Promosi × Markup (%)

**Skenario:** Event khusus — harga premium untuk produk limited edition.

```
Produk      : Parfum Limited Edition
Harga Normal : Rp 2.500.000
Tipe        : Promosi
Metode      : Markup (%)
Nilai       : 20%
Target      : Produk "Parfum Limited Edition"
Outlet      : Semua Outlet
Berlaku     : 2026-12-20 s/d 2026-12-31 (Natal/Tahun Baru)
```

**Hasil:**
- Harga jual = Rp 2.500.000 + (Rp 2.500.000 × 20%) = **Rp 3.000.000**
- Harga naik karena demand tinggi di musim liburan

---

## 4. Stacking (Penggabungan Rule)

Stacking memungkinkan **beberapa rule dihitung secara berurutan** pada satu produk. Diaktifkan dengan mengaktifkan checkbox **"Boleh digabung (stacking)"** (`allow_combine = true`).

### 4.1 Aturan Stacking

1. **Default + Default** → TIDAK bisa stacking. Hanya 1 rule default yang berlaku (berdasarkan prioritas).
2. **Daftar Harga + Daftar Harga** → TIDAK bisa stacking. Hanya 1 rule price_list yang berlaku.
3. **Promosi + Promosi** → BISA stacking jika kedua rule `allow_combine = true`.
4. **Default + Promosi** → BISA stacking jika rule promosi `allow_combine = true`.
5. **Daftar Harga + Promosi** → BISA stacking jika kedua rule `allow_combine = true`.

### 4.2 Cara Kerja Stacking

Rule dihitung **berurutan berdasarkan urutan resolution engine**:
1. Pertama, tentukan **harga dasar** (dari rule Default atau Daftar Harga)
2. Kemudian, **terapkan rule Promosi** satu per satu (jika allow_combine = true)
3. Setiap promosi dihitung dari **harga hasil sebelumnya**, bukan dari harga normal

> **Penting:** Stacking selalu dimulai dari harga normal/awal produk. Diskon/promosi berikutnya dihitung dari harga yang sudah dimodifikasi oleh rule sebelumnya.

---

### 4.3 Contoh Stacking — Promosi + Promosi

**Skenario:** Flash sale + happy hour bersamaan.

```
Harga Normal Kaos Polos : Rp 100.000

Rule 1 — Flash Sale Akhir Bulan:
  Tipe       : Promosi
  Metode     : Diskon (%)
  Nilai      : 20%
  allow_combine : true
  Berlaku    : 2026-07-28 s/d 2026-07-31
  Jadwal     : Setiap hari

Rule 2 — Happy Hour:
  Tipe       : Promosi
  Metode     : Diskon (Rp)
  Nilai      : Rp 10.000
  allow_combine : true
  Jadwal     : Senin - Jumat
  Jam        : 14:00 - 17:00
```

**Perhitungan (pada hari Rabu, jam 15:00, tanggal 29 Juli):**

```
Harga Normal          : Rp 100.000
+ Rule 1 (Diskon 20%) : Rp 100.000 × 20% = Rp 20.000
  Harga setelah Rule 1 : Rp 80.000
+ Rule 2 (Diskon Rp)   : Rp 10.000
  Harga setelah Rule 2 : Rp 70.000

→ Harga Jual = Rp 70.000
```

**Perhitungan (pada hari Sabtu, jam 15:00, tanggal 29 Juli):**

```
Harga Normal          : Rp 100.000
+ Rule 1 (Diskon 20%) : Rp 100.000 × 20% = Rp 20.000
  Harga setelah Rule 1 : Rp 80.000
+ Rule 2              : TIDAK aktif (Sabtu, jadwal hanya Sen-Jum)

→ Harga Jual = Rp 80.000
```

**Perhitungan (pada hari Rabu, jam 13:00, tanggal 29 Juli):**

```
Harga Normal          : Rp 100.000
+ Rule 1 (Diskon 20%) : Rp 100.000 × 20% = Rp 20.000
  Harga setelah Rule 1 : Rp 80.000
+ Rule 2              : TIDAK aktif (jam 13:00, jadwal jam 14-17)

→ Harga Jual = Rp 80.000
```

---

### 4.4 Contoh Stacking — Daftar Harga + Promosi

**Skenario:** Member VIP mendapat diskon tambahan saat flash sale.

```
Harga Normal Serum Vitamin C : Rp 185.000

Rule 1 — Harga Member VIP:
  Tipe            : Daftar Harga
  Metode          : Diskon (%)
  Nilai           : 15%
  Customer Group  : VIP
  allow_combine   : false
  Berlaku         : Sepanjang tahun

Rule 2 — Flash Sale Serum:
  Tipe            : Promosi
  Metode          : Diskon (Rp)
  Nilai           : Rp 25.000
  Target          : Produk "Serum Vitamin C"
  allow_combine   : true
  Berlaku         : 2026-07-25 s/d 2026-07-27
```

**Perhitungan (Customer VIP, saat flash sale aktif):**

```
Harga Normal              : Rp 185.000
+ Rule 1 (Member 15%)     : Rp 185.000 × 15% = Rp 27.750
  Harga setelah Rule 1    : Rp 157.250
+ Rule 2 (Flash Sale -Rp) : Rp 25.000
  Harga setelah Rule 2    : Rp 132.250

→ Harga Jual = Rp 132.250
```

**Perhitungan (Customer biasa, saat flash sale aktif):**

```
Harga Normal              : Rp 185.000
+ Rule 1                  : TIDAK aktif (bukan VIP)
+ Rule 2 (Flash Sale -Rp) : Rp 25.000
  Harga setelah Rule 2    : Rp 160.000

→ Harga Jual = Rp 160.000
```

**Perhitungan (Customer VIP, flash sale TIDAK aktif):**

```
Harga Normal              : Rp 185.000
+ Rule 1 (Member 15%)     : Rp 185.000 × 15% = Rp 27.750
  Harga setelah Rule 1    : Rp 157.250
+ Rule 2                  : TIDAK aktif (belum/selesai periode)

→ Harga Jual = Rp 157.250
```

---

### 4.5 Contoh Stacking — 3 Promosi Bersamaan

**Skenario:** Tiga promosi berbeda aktif di waktu yang sama.

```
Harga Normal Jaket Hoodie : Rp 350.000

Rule 1 — Clearance Sale:
  Tipe       : Promosi
  Metode     : Diskon (%)
  Nilai      : 30%
  allow_combine : true
  Berlaku    : 2026-07-01 s/d 2026-07-31

Rule 2 — Voucher Pelanggan Setia:
  Tipe       : Promosi
  Metode     : Diskon (Rp)
  Nilai      : Rp 50.000
  Customer Group : Pelanggan Setia
  allow_combine : true
  Berlaku    : 2026-07-15 s/d 2026-08-15

Rule 3 — Weekend Bonus:
  Tipe       : Promosi
  Metode     : Diskon (%)
  Nilai      : 5%
  allow_combine : true
  Jadwal     : Sabtu, Minggu
```

**Perhitungan (Customer "Pelanggan Setia", hari Minggu, bulan Juli):**

```
Harga Normal                : Rp 350.000
+ Rule 1 (Clearance 30%)    : Rp 350.000 × 30% = Rp 105.000
  Harga setelah Rule 1      : Rp 245.000
+ Rule 2 (Voucher -Rp50k)   : Rp 50.000
  Harga setelah Rule 2      : Rp 195.000
+ Rule 3 (Weekend 5%)       : Rp 195.000 × 5% = Rp 9.750
  Harga setelah Rule 3      : Rp 185.250

→ Harga Jual = Rp 185.250 (hemat Rp 164.750 / 47%)
```

**Perhitungan (Customer biasa, hari Rabu, bulan Juli):**

```
Harga Normal                : Rp 350.000
+ Rule 1 (Clearance 30%)    : Rp 350.000 × 30% = Rp 105.000
  Harga setelah Rule 1      : Rp 245.000
+ Rule 2                    : TIDAK aktif (bukan Pelanggan Setia)
+ Rule 3                    : TIDAK aktif (Rabu, bukan weekend)

→ Harga Jual = Rp 245.000
```

---

### 4.6 Contoh TANPA Stacking (allow_combine = false)

**Skenario:** Rule promosi tidak bisa digabung.

```
Harga Normal Headphone : Rp 750.000

Rule 1 — Promo Lebaran:
  Tipe       : Promosi
  Metode     : Diskon (%)
  Nilai      : 25%
  allow_combine : false
  Berlaku    : 2026-04-01 s/d 2026-04-10

Rule 2 — Cashback Member:
  Tipe       : Promosi
  Metode     : Diskon (Rp)
  Nilai      : Rp 100.000
  allow_combine : false
  Customer Group : Member
```

**Perhitungan (Customer Member, saat Lebaran aktif):**

```
Harga Normal            : Rp 750.000
+ Rule 1 (Diskon 25%)   : Rp 750.000 × 25% = Rp 187.500
  Harga setelah Rule 1  : Rp 562.500
+ Rule 2                : TIDAK bisa stacking (allow_combine = false)

→ Harga Jual = Rp 562.500
→ Rule 2 diabaikan karena Rule 1 sudah diterapkan dan stacking tidak diizinkan
```

---

## 5. Algoritma Prioritas & Resolusi

### 5.1 Urutan Resolusi (8 Langkah)

Resolution engine menentukan harga jual dengan algoritma berikut:

```
Langkah 1: Kumpulkan semua rule yang aktif (is_active = true)
Langkah 2: Filter berdasarkan jadwal (recurrence_days, time_from, time_to)
Langkah 3: Filter berdasarkan periode (effective_from, effective_until)
Langkah 4: Filter berdasarkan target (product_id, category_id, brand_id)
Langkah 5: Filter berdasarkan customer_group_id dan store_id
Langkah 6: Filter berdasarkan quantity range (minimum_quantity, maximum_quantity)
Langkah 7: Urutkan berdasarkan spesifitas target (product > category > brand)
Langkah 8: Terapkan rule secara berurutan (jika allow_combine = true)
```

### 5.2 Spesifitas Target

Rule yang lebih spesifik **diutamakan**:

| Urutan | Spesifitas | Contoh |
|--------|-----------|--------|
| 1 | Produk spesifik | `product_id = 42` |
| 2 | Kategori | `category_id = 5` |
| 3 | Brand | `brand_id = 3` |
| 4 | Semua (tidak ada target) | Tidak ada product/category/brand |

**Contoh:**
```
Rule A: Target = Produk "Indomie" (spesifik) → prioritas lebih tinggi
Rule B: Target = Kategori "Makanan" (kategori) → prioritas lebih rendah

→ Rule A diterapkan duluan, Rule B diabaikan jika Rule A sudah match.
```

### 5.3 Prioritas Angka

Jika dua rule memiliki spesifitas yang sama, **angka prioritas** menentukan urutan:

```
Rule X: Prioritas = 10 (lebih tinggi = lebih diutamakan)
Rule Y: Prioritas = 5  (lebih rendah = kurang diutamakan)

→ Rule X diterapkan duluan.
```

---

## 6. Edge Cases

### 6.1 Harga Jual Negatif

Jika stacking menyebabkan harga jual negatif, engine akan **membatasi ke Rp 0** (tidak ada harga negatif).

**Contoh:**
```
Harga Normal  : Rp 50.000
Rule 1        : Diskon 80%  → Rp 10.000
Rule 2        : Diskon Rp 20.000 → Rp 10.000 - Rp 20.000 = -Rp 10.000

→ Harga Jual = Rp 0 (dibatasi minimum Rp 0)
```

### 6.2 Tidak Ada Rule Aktif

Jika tidak ada rule yang match untuk sebuah produk, **harga normal produk** digunakan sebagai harga jual.

### 6.3 Rule dengan Target Kosong

Jika rule tidak punya `product_id`, `category_id`, atau `brand_id`, rule tersebut berlaku untuk **semua produk**. Ini berguna untuk promosi general ("Diskon 10% untuk semua produk").

### 6.4 Jadwal Kosong

Jika `recurrence_days` kosong dan tidak ada `time_from`/`time_to`, rule berlaku **sepanjang hari setiap hari** (kecuali dibatasi oleh periode berlaku).

### 6.5 Periode Kosong

Jika `effective_from` dan `effective_until` kosong, rule berlaku **selamanya** (sampai di-nonaktifkan atau dihapus).

---

## Ringkasan Cepat

| # | Tipe | Metode | Contoh Hasil |
|---|------|--------|-------------|
| 1 | Default | Harga Tetap | Harga jual = nilai tetap |
| 2 | Default | Diskon % | Harga jual = normal - (normal × %) |
| 3 | Default | Diskon Rp | Harga jual = normal - nominal |
| 4 | Default | Markup % | Harga jual = normal + (normal × %) |
| 5 | Daftar Harga | Harga Tetap | Harga khusus untuk group/outlet |
| 6 | Daftar Harga | Diskon % | Diskon eksklusif untuk group |
| 7 | Daftar Harga | Diskon Rp | Potongan flat untuk group |
| 8 | Daftar Harga | Markup % | Harga lebih mahal untuk outlet tertentu |
| 9 | Promosi | Harga Tetap | Flash sale dengan harga pasti |
| 10 | Promosi | Diskon % | Diskon persentase terbatas waktu |
| 11 | Promosi | Diskon Rp | Potongan nominal terbatas waktu |
| 12 | Promosi | Markup % | Harga naik saat peak season |

**Stacking:** Aktif jika `allow_combine = true`. Promosi berikutnya dihitung dari harga hasil rule sebelumnya, bukan dari harga normal.
