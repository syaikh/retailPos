# Product Requirement Document (PRD) — Reports Restructuring

**Project:** Restrukturisasi Komponen Reports & Smart Dropdown Filter Terpadu
**Platform:** Retail POS System (Svelte 5 Runes + Tailwind CSS + Golang Backend)
**Status:** Approved
**Timezone:** Asia/Jakarta (GMT+07)

---

## 1. Ringkasan Proyek & Tujuan (Overview)

Proyek ini bertujuan untuk merestrukturisasi total komponen filter tanggal, manajemen state grafik, dan visualisasi kartu statistik (Stat Cards) pada halaman Laporan (Reports.svelte).

Sistem baru mengeliminasi sistem filter lama yang redundan dan meleburnya menjadi Satu Kesatuan Dropdown Terpadu 2-Kolom yang cerdas. Sistem ini mendefinisikan ulang cara pengusaha retail menganalisis data melalui fungsionalitas Timezone Awareness (GMT+07) dan Auto-Switch Agregasi Data secara dinamis guna meminimalkan bias data operasional (seperti menukiknya tren grafik akibat mencampur data hari berjalan yang belum selesai).

## 2. Pengguna & Ruang Lingkup (Scope)

- **Aktor Utama:** Pemilik Toko (Owner), Manajer Operasional Toko
- **Ruang Lingkup Sistem:** Modul antarmuka analitik frontend pada halaman Reports.svelte yang terintegrasi penuh dengan REST API backend berbasis Golang

## 3. Spesifikasi Fungsional (Functional Requirements)

### 3.1 Resolusi Zona Waktu Lokal (Timezone Aware)

- Sistem wajib mendeteksi dan menggunakan zona waktu lokal mesin/browser pengguna secara dinamis (Default: GMT+07 atau WIB)
- Header utama atau tombol pemicu (trigger button) dropdown wajib menampilkan teks keterangan yang presisi, contoh: "30 hari sebelumnya. 24-04-2026 - 23-05-2026 (GMT+07)"
- Semua parameter tanggal yang dilemparkan ke API backend Golang wajib menyertakan atau memperhitungkan offset zona waktu lokal agar data transaksi di akhir hari (23:00 - 23:59) tidak bergeser ke hari berikutnya akibat standarisasi UTC server

### 3.2 Pembersihan Komponen Lama (Breaking Changes)

- [DEPRECATED] Komponen input kalender tanggal manual eceran beserta tombol Apply manual dihapus total
- [DEPRECATED] Tombol tab manual penukar grafik "Daily | Weekly | Monthly" dihapus total dari tampilan atas grafik

### 3.3 Logika Dropdown Komponen Dua Kolom (Smart Selector)

Komponen berupa panel popover dropdown yang terbagi menjadi dua kolom fungsional:

**A. Kolom Kiri: Menu Pilihan Cepat**
**B. Kolom Kanan: Area Selector Adaptif**

**Matriks Perilaku Pilihan Dropdown:**

| Pilihan | Kolom Kanan | Rentang |
|---------|-------------|---------|
| Real-time | Teks informasi statis | Hari ini dari jam 00:00 hingga jam berjalan |
| Kemarin | Teks informasi statis | 1 Hari penuh kemarin (00:00 - 23:59) |
| 7 Hari Sebelumnya | Teks informasi statis | 7 hari ke belakang, exclude hari ini |
| 30 Hari Sebelumnya | Teks informasi statis | 30 hari ke belakang, exclude hari ini |
| Per Minggu | Kalender Bulanan (Snap-to-week) | Hover menyorot satu baris minggu utuh |
| Per Bulan | Grid 12 Bulan (Adaptive Grid) | Satu bulan utuh untuk tahun aktif |
| Berdasarkan Tahun | Grid Angka Tahun | Tahun 2024, 2025, 2026, dst. |

### 3.4 Manajemen Matriks Otomatisasi Grafik & Stat Cards (Auto-Switch)

Sistem wajib melakukan pembaruan reaktif (re-render) secara otomatis pada tipe grafik dan struktur data Stat Card nomor 4 dan 5 berdasarkan opsi waktu yang dipilih.

**Alur State Reaktif:**
```
[Pilihan Filter Dropdown] --> Konversi Tanggal (GMT+07) --> Set Tipe Grafik --> Mutasi Label Stat Cards
```

## 4. Desain Antarmuka & Aturan UX (UI/UX Guidelines)

- **Prinsip Desain:** Elegan, dark-mode native, menyatu dengan skema visual POS eksisting
- **Warna Panel Dropdown:** `bg-slate-900` dengan border `border-slate-800`
- **Warna Teks:** `text-slate-100` (data utama), `text-slate-400` (label sekunder)
- **Aksen State Aktif:** `bg-purple-600` / `text-purple-400`
- **Indikator Pertumbuhan:** `text-emerald-400`

## 5. Cetak Biru Implementasi (Svelte 5 Source Code)

```svelte
<script>
  import { onMount } from 'svelte';

  // --- STATE MANAGEMENT (SVELTE 5 RUNES) ---
  let selectedPeriodType = $state('7days');
  let dropdownOpen = $state(false);
  let timezoneString = $state('GMT+07');

  let dateRange = $state({ start: '', end: '' });
  let hoveredWeekStart = $state(null);

  // --- DERIVED STATES ---
  let chartType = $derived(
    ['realtime', 'yesterday'].includes(selectedPeriodType) ? 'hourly' :
    ['7days', '30days'].includes(selectedPeriodType) ? 'daily' : 'high-unit'
  );

  let statCardLabels = $derived({
    card4:
      chartType === 'hourly' ? 'Peak Revenue Hour' :
      selectedPeriodType === 'yearly' ? 'Avg. Revenue / Month' :
      selectedPeriodType === 'monthly' ? 'Avg. Revenue / Week' : 'Avg. Revenue / Day',
    card5:
      selectedPeriodType === 'realtime' ? 'vs Same Hours Yesterday' :
      selectedPeriodType === 'yesterday' ? 'vs Same Day Last Week' : 'vs Previous Period'
  });

  // --- UTILITY LOGIC & TIMEZONE FUNCTIONS ---
  function calculateTimezoneOffset() {
    const offset = new Date().getTimezoneOffset();
    const absOffset = Math.abs(offset);
    const hours = String(Math.floor(absOffset / 60)).padStart(2, '0');
    const sign = offset <= 0 ? '+' : '-';
    timezoneString = `GMT${sign}${hours}`;
  }

  function formatDisplayDate(date) {
    if (!date) return '';
    return date.toLocaleDateString('id-ID', { day: '2-digit', month: '2-digit', year: 'numeric' });
  }

  function setPeriod(type) {
    selectedPeriodType = type;
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(today.getDate() - 1);

    if (type === 'realtime') {
      dateRange.start = formatDisplayDate(today);
      dateRange.end = `Hari Ini - Pk ${String(today.getHours()).padStart(2, '0')}:00`;
    } else if (type === 'yesterday') {
      dateRange.start = formatDisplayDate(yesterday);
      dateRange.end = formatDisplayDate(yesterday);
    } else if (type === '7days') {
      const start7 = new Date();
      start7.setDate(today.getDate() - 7);
      dateRange.start = formatDisplayDate(start7);
      dateRange.end = formatDisplayDate(yesterday);
    } else if (type === '30days') {
      const start30 = new Date();
      start30.setDate(today.getDate() - 30);
      dateRange.start = formatDisplayDate(start30);
      dateRange.end = formatDisplayDate(yesterday);
    } else if (type === 'weekly') {
      dateRange.start = "2026-05-18";
      dateRange.end = "2026-05-24 (Partial)";
    } else if (type === 'monthly') {
      dateRange.start = "Mei 2026";
      dateRange.end = "Satu Bulan Penuh";
    } else if (type === 'yearly') {
      dateRange.start = "Tahun 2026";
      dateRange.end = "Satu Tahun Penuh";
    }

    if (!['weekly', 'monthly', 'yearly'].includes(type)) {
      dropdownOpen = false;
    }
  }

  onMount(() => {
    calculateTimezoneOffset();
    setPeriod('7days');
  });
</script>
```
