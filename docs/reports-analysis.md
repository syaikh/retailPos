# Reports Page — Analisis & Roadmap

## Overview

Halaman Reports (`web/src/lib/pages/ReportsPage.svelte`) menampilkan analisis penjualan retail dengan dukungan multiple period, chart interaktif, KPI cards, dan export.

---

## ✅ Yang SUDAH Ada

### Period Selector
- **Real-time**: Hari ini (00:00 - jam sekarang), dibandingkan dengan yesterday di jam yang sama
- **Yesterday**: Full day kemarin, dibandingkan dengan same day last week
- **7 Days**: 7 hari terakhir, dibandingkan dengan previous 7 days
- **30 Days**: 30 hari terakhir, dibandingkan dengan previous 30 days
- **Daily**: Pilih tanggal spesifik, dibandingkan dengan H-7
- **Weekly**: Pilih minggu, dibandingkan dengan previous week
- **Monthly**: Pilih bulan, dibandingkan dengan previous month (MTD untuk bulan berjalan)
- **Yearly**: Pilih tahun, dibandingkan dengan previous year

### KPI Cards (5 cards)
| KPI | Value | Comparison |
|-----|-------|------------|
| Total Revenue | `formatCurrencyShort()` | vs previous revenue |
| Total Orders | `formatLargeNumber()` | vs previous orders |
| Avg Order Value | `formatCurrencyShort()` + trend icon | vs previous AOV |
| Peak Revenue / Revenue Per Day | Dinamis sesuai period | vs previous peak/revenue per day |
| % Change | Persentase ± trend icon | growth vs previous period |

### Chart
- Tipe line/bar menyesuaikan period
- Dual dataset (current + previous) untuk comparison
- Tooltip dengan difference calculation
- Label X-axis menampilkan current & previous period dates

### Export
- Export to Excel (.xlsx)
- Export to PDF

---

## 📊 Data API yang Tersedia

| Endpoint | Method | Auth | Data |
|----------|--------|------|------|
| `/api/dashboard/years` | GET | Public | Years dengan sales data |
| `/api/dashboard/stats` | GET | dashboard:read | Today's revenue, yesterday revenue, sales count, products, low stock |
| `/api/dashboard/live` | GET | dashboard:read | Live: today's revenue, sales, products, low stock |
| `/api/dashboard/chart` | GET | report:read | Hourly (single day) / Daily (range) chart data, dual-period |
| `/api/dashboard/chart/weekly` | GET | report:read | Weekly aggregated revenue + order_count |
| `/api/dashboard/chart/monthly` | GET | report:read | Monthly aggregated revenue + order_count, dual-period |
| `/api/dashboard/comparison` | GET | report:read | Period comparison: revenue, orders, AOV, rev/day, peak hour |
| `/api/sales/export` | GET | report:read | CSV/XLSX export of sales transactions |

### Field Tersedia di Sale Model
- `invoice_number`, `cashier_id`, `customer_id`, `customer_name`
- `subtotal`, `discount`, `tax`, `total_amount`
- `payment_method` (Cash/Card/E-Wallet/dll)
- `status` (completed/cancelled/refunded)
- `items[]`: `product_id`, `name`, `quantity`, `unit_price`, `subtotal`
- `items_count` (jumlah item per transaksi)

---

## 🚀 Roadmap Implementasi

### Phase 1 — Low Effort (Frontend Only)
| # | Feature | Status |
|---|---------|--------|
| 1 | **Data Table** — Tabel data period di bawah chart | ✅ DONE |
| 2 | **Order Count di chart tooltip** | 📋 TODO |
| 3 | **Cumulative Revenue line** — Overlay di chart | 📋 TODO |
| 4 | **Best/Worst Day highlight** — Card best & worst performing day | ✅ DONE |
| 5 | **Total Items Sold** — KPI card items terjual | 📋 TODO |

### Phase 2 — Medium Effort (Backend + Frontend)
| # | Feature | Data Source | Status |
|---|---------|-------------|--------|
| 6 | **Revenue by Payment Method** — Pie chart breakdown | `payment_method` field | 📋 TODO |
| 7 | **Revenue by Category** — Bar chart per kategori | Product `category_name` | 📋 TODO |
| 8 | **Top 10 Products** — Tabel produk terlaris | `sale_items` table | 📋 TODO |
| 9 | **Discount & Tax Summary** — KPI card | `discount`, `tax` fields | 📋 TODO |
| 10 | **Hourly Transaction Count** — Dual-axis chart | Aggregasi count per hour | 📋 TODO |
| 11 | **Order Status Breakdown** — Completed vs cancelled | `status` field | 📋 TODO |
| 12 | **Low Stock Report** — Pindah dari Dashboard | `product_stock` table | 📋 TODO |

### Phase 3 — High Effort
| # | Feature | Notes |
|---|---------|-------|
| 13 | **Gross Profit / Margin** | Butuh COGS tracking (belum ada) |
| 14 | **Customer Analytics** | Repeat rate, new vs returning customers |
| 15 | **Cashier Performance** | Aggregasi per `cashier_id` |
| 16 | **Store Comparison** | Multi-store rollup |

---

## Detail Implementasi

### #1 Data Table
- Tabel collapsible di bawah chart
- Kolom dinamis sesuai period type (hourly → jam, daily → tanggal, yearly → bulan)
- Setiap baris: label period, revenue, prev period revenue, change %, order count
- Summary row: total revenue, total prev, overall change %
- Sortable per kolom (asc/desc)
- Virtual scroll untuk period panjang (30+ baris)

### #4 Best/Worst Day
- Dua KPI card kecil di samping chart atau di atas chart
- Best day: tanggal + revenue tertinggi
- Worst day: tanggal + revenue terendah (skip hari dengan Rp 0 jika period penuh)
- Perbandingan dengan prev period day yang sama

### #2 Order Count Tooltip
- Untuk weekly/monthly/yearly: API return `order_count`
- Tambahkan di chart tooltip: "Orders: X"
- Untuk realtime/yesterday/daily: hitung dari data response

### #3 Cumulative Revenue
- Derived dari `chartData` → hitung running total
- Tampilkan sebagai line overlay kedua di chart
- Bisa toggle visibility

---

## Notes Teknis

- **Timezone**: Semua query pake `Asia/Jakarta` (UTC+7)
- **Chart alignment**: Current vs Previous period di-align by index (bukan tanggal absolut)
- **Partial period**: Monthly/Weekly partial periods pakai comparison like-for-like (same number of days)
- **Realtime**: Filter hours sampai current hour (inclusive partial hour)
- **Format mata uang**: Rp dengan suffix (jt untuk juta, M untuk milyar, k untuk ribuan)
