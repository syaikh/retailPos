# Overlap: Produk Price vs Pricing Rules — Use Case (SOLVED)

Dokumen ini menunjukkan secara detail bagaimana `products.price` dan pricing rules saling tumpang tindih, dengan contoh nyata yang bisa diverifikasi di UI.

> **Status: SOLVED** — Type `default` sudah dihapus. `products.price` menjadi single source of truth. Pricing rules hanya bisa `special_price` atau `promotion`.

---

## Setup Awal

**Produk: Indomie Goreng**
```
products.price = Rp 3.500
products.cost  = Rp 2.800
```

**Form Produk (mode edit) menampilkan:**
```
┌─────────────────────────────────────────────┐
│  Price (IDR)  │  Rp 3.500                   │  ← input field, bisa diedit
│  Cost (IDR)   │  Rp 2.800                   │
├─────────────────────────────────────────────┤
│  PRICING RULES (2)                          │  ← section di bawah form
│  ┌───────────────────────────────────────┐  │
│  │ special_price Harga Grosir Rp 3.000  │  │  ← tampil sebagai informasi
│  │ promotion Flash Sale       Rp 2.800  │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

**Status: RESOLVED** — Dengan type `default` dihapus, `products.price` menjadi satu-satunya harga jual normal. Pricing rules hanya berlaku untuk `special_price` (harga khusus) dan `promotion` (diskon/promosi). Tidak ada lagi overlap yang membingungkan.

---

## Use Case 1: Tidak Ada Pricing Rule (Normal)

```
products.price = Rp 3.500
Pricing Rules  = (kosong)
```

**Apa yang terjadi:**
- Resolver baca base price = Rp 3.500
- Tidak ada rule eligible
- Return Rp 3.500

**Harga jual = Rp 3.500** ✅

**Kebingungan:** Tidak ada. Semua konsisten.

---

## Use Case 2: Default + Diskon (%)

```
products.price = Rp 3.500
Pricing Rule:
  Tipe     : Default
  Metode   : Diskon (%)
  Nilai    : 10%
  Target   : Produk "Indomie Goreng"
```

**Apa yang terjadi:**
- Resolver baca base price = Rp 3.500
- Temukan rule eligible
- computePrice: Rp 3.500 × (1 - 10/100) = Rp 3.150

**Harga jual = Rp 3.150** ✅

**Kebingungan:** User lihat `Price = Rp 3.500` di form produk, tapi harga jual Rp 3.150. Tidak jelas dari mana angka Rp 3.150 berasal kecuali buka Pricing Rules page.

---

## Use Case 3: Default + Diskon (Rp)

```
products.price = Rp 3.500
Pricing Rule:
  Tipe     : Default
  Metode   : Diskon (Rp)
  Nilai    : Rp 500
  Target   : Produk "Indomie Goreng"
```

**Apa yang terjadi:**
- Resolver baca base price = Rp 3.500
- computePrice: Rp 3.500 - Rp 500 = Rp 3.000

**Harga jual = Rp 3.000** ✅

**Kebingungan:** Sama — `products.price` tampilkan Rp 3.500, tapi harga jual Rp 3.000.

---

## Use Case 4: Default + Harga Tetap (OVERLAP BESAR)

```
products.price = Rp 3.500
Pricing Rule:
  Tipe     : Default
  Metode   : Harga Tetap
  Nilai    : Rp 2.800
  Target   : Produk "Indomie Goreng"
```

**Apa yang terjadi:**
- Resolver baca base price = Rp 3.500
- computePrice: fixed_price → langsung pakai Rp 2.800

**Harga jual = Rp 2.800** ✅

**Kebingungan BESAR:**
- Form produk: `Price = Rp 3.500`
- Harga jual aktual: `Rp 2.800`
- User tidak tahu kenapa harga berubah
- User edit produk, ubah price jadi Rp 4.000 → harga jual **tetap Rp 2.800** (karena rule fixed_price override)
- User bingung: "Sudah saya ubah harga kok tidak berubah?"

---

## Use Case 5: Daftar Harga + Harga Tetap

```
products.price = Rp 3.500
Pricing Rule:
  Tipe            : Daftar Harga
  Metode          : Harga Tetap
  Nilai           : Rp 2.500
  Customer Group  : Grosir
  Target          : Produk "Indomie Goreng"
```

**Apa yang terjadi:**
- Customer biasa beli → resolver tidak temukan rule (bukan Grosir) → **Rp 3.500**
- Grosir beli → resolver temukan rule → **Rp 2.500**

**Kebingungan:**
- Di form produk, user lihat `Price = Rp 3.500` → kira itu harga jual untuk semua
- Ternyata Grosir beli Rp 2.500
- User tidak bisa lihat di form produk bahwa ada harga khusus untuk Grosir

---

## Use Case 6: Promosi + Diskon (%) + Stacking

```
products.price = Rp 3.500
Pricing Rule 1:
  Tipe       : Promosi
  Metode     : Diskon (%)
  Nilai      : 15%
  Berlaku    : 2026-07-01 s/d 2026-07-31
  allow_combine : true

Pricing Rule 2:
  Tipe       : Promosi
  Metode     : Diskon (Rp)
  Nilai      : Rp 200
  Berlaku    : 2026-07-15 s/d 2026-07-20
  allow_combine : true
```

**Apa yang terjadi (tanggal 17 Juli, kedua rule aktif):**
- Resolver baca base price = Rp 3.500
- Rule 1: Rp 3.500 × 15% = Rp 525 → harga = Rp 2.975
- Rule 2: Rp 2.975 - Rp 200 = Rp 2.775

**Harga jual = Rp 2.775** ✅

**Kebingungan:**
- Form produk: `Price = Rp 3.500`
- Harga jual aktual: `Rp 2.775` (selama 15-20 Juli)
- Setelah 20 Juli: harga jual = `Rp 2.975` (hanya Rule 1 aktif)
- Setelah 31 Juli: harga jual = `Rp 3.500` (kembali normal)
- User harus cek **3 tempat berbeda** untuk paham: form produk, pricing rules page, dan kalender

---

## Use Case 7: User Edit Produk, Tidak Sadar Ada Rule

```
Sebelum edit:
  products.price = Rp 3.500
  Pricing Rule: Default, Diskon 10% → harga jual Rp 3.150

User edit produk:
  Ubah Price dari Rp 3.500 → Rp 4.000

Setelah edit:
  products.price = Rp 4.000
  Pricing Rule: Default, Diskon 10% → harga jual Rp 3.600

Seharusnya:
  Harga jual = Rp 3.600 (Rp 4.000 - 10%)

Yang user harapkan:
  Harga jual = Rp 4.000 (karena sudah diubah)
```

**Kebingungan:** User mengira mengubah `products.price` akan langsung mengubah harga jual. Ternyata rule masih aktif dan override harga tersebut.

---

## Use Case 8: User Hapus Rule, Harga Berubah

```
Sebelum hapus:
  products.price = Rp 3.500
  Pricing Rule: Default, Harga Tetap Rp 2.800 → harga jual Rp 2.800

User hapus pricing rule:
  products.price = Rp 3.500
  Pricing Rules = (kosong)

Setelah hapus:
  Harga jual = Rp 3.500 (kembali ke base price)
```

**Kebingungan:** User tidak sadar bahwa menghapus rule akan mengembalikan harga ke `products.price`. Jika user sudah lupa bahwa `products.price` = Rp 3.500, harga tiba-tiba naik dari Rp 2.800 ke Rp 3.500.

---

## Use Case 9: Form Produk Tampilkan Rule Tapi Tidak Jelas

Di `ProductFormModal.svelte` (line 226-265), pricing rules ditampilkan di bawah form saat mode edit:

```
┌─────────────────────────────────────────────┐
│  Price (IDR)  │  Rp 3.500                   │
├─────────────────────────────────────────────┤
│  PRICING RULES (1)                          │
│  ┌───────────────────────────────────────┐  │
│  │ default  Harga Normal  Rp 2.800      │  │  ← hanya tampilkan nama & nilai
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

**Masalah:**
1. Rule tampil sebagai **read-only** — user tidak bisa edit dari sini
2. Tidak ada indikasi bahwa rule ini **override** `products.price`
3. Tidak ada warning "Harga jual produk ini berbeda dari harga dasar karena ada pricing rule aktif"
4. User harus buka Pricing Rules page terpisah untuk mengelola rule

---

## Ringkasan Masalah

| Skenario | `products.price` | Harga Jual Aktual | User Sadar? |
|----------|------------------|-------------------|-------------|
| Tidak ada rule | Rp 3.500 | Rp 3.500 | ✅ Ya |
| Default + Diskon 10% | Rp 3.500 | Rp 3.150 | ❌ Tidak |
| Default + Harga Tetap Rp 2.800 | Rp 3.500 | Rp 2.800 | ❌ Tidak |
| Daftar Harga (Grosir) | Rp 3.500 | Rp 2.500 (Grosir) | ❌ Tidak |
| Promosi + Stacking | Rp 3.500 | Rp 2.775 | ❌ Tidak |
| User edit price | Rp 4.000 | Rp 3.600 (10% rule) | ❌ Tidak |
| User hapus rule | Rp 3.500 | Rp 3.500 | ⚠️ Kaget |

---

## Rekomendasi Perbaikan

### Opsi A: Tampilkan "Harga Jual Aktual" di Form Produk

```
┌─────────────────────────────────────────────┐
│  Harga Dasar (IDR)  │  Rp 3.500             │  ← products.price
│  ⚠️ Harga Jual      │  Rp 2.800             │  ← computed dari rules
│     (ada 1 pricing rule aktif)              │
├─────────────────────────────────────────────┤
│  PRICING RULES (1)                          │
│  ┌───────────────────────────────────────┐  │
│  │ default  Harga Normal  Rp 2.800      │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

**Kelebihan:** User langsung tahu harga jual aktual tanpa buka halaman lain.
**Kekurangan:** Butuh additional API call untuk resolve harga.

### Opsi B: Batasi Default + Fixed Price

Tidak izinkan pricing rule dengan `type = default` + `method = fixed_price`. Jika user ingin mengubah harga jual produk, cukup edit `products.price` langsung.

**Kelebihan:** Single source of truth — `products.price` selalu = harga jual normal.
**Kekurangan:** Kehilangan fleksibilitas (mis. harga tetap beda per outlet).

### Opsi C: Rename `products.price` jadi "Harga Dasar"

Ganti label di form dari "Price" jadi "Harga Dasar (Base Price)" + tambahkan tooltip "Ini adalah harga awal sebelum pricing rules diterapkan."

**Kelebihan:** Paling minimal perubahan, user jadi aware bahwa ini bukan harga jual final.
**Kekurangan:** User masih perlu cek pricing rules untuk tahu harga jual aktual.
