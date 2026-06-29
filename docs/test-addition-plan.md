# Test Addition Plan

Berdasarkan hasil audit, berikut gap yang ditemukan beserta prioritas implementasi yang direkomendasikan.

---

## Tier 1 — Stores
**Prioritas:** Tinggi  
**Effort:** Rendah (~2–3 jam total)  
**Value:** Tinggi

Karakteristik:
- Pure JavaScript/TypeScript logic.
- Tidak membutuhkan DOM rendering.
- Dapat mengikuti pola yang sudah ada pada `auth-store.test.ts`.

| Store | Lines | Key Logic to Test |
|---------|------:|------------------|
| `pos/stores/pos-store.test.ts` | 90 | `addToCart`, `removeFromCart`, `updateQty`, `$derived` (`subtotal`, `tax`, `total`, `change`) |
| `customers/stores/customer-store.test.ts` | 70 | `load()`, `searchQuery`, `statusFilter`, `selectedIds`, `clearSelection()` |
| `sales/stores/sales-store.test.ts` | 55 | `load(filters)`, `sortBy`, `sortDir`, `searchQuery` |
| `product/stores/product-store.test.ts` | 130 | `initialize()`, `loadProducts()`, `loadMasterData()`, master data state, `selectedIds` |
| `inventory/stores/inventory-store.test.ts` | 20 | `setThresholds()` |
| `shared/stores/toast.test.ts` | 60 | `success()`, `error()`, `warning()`, `info()`, `remove()`, auto-dismiss |
| `shared/stores/notifications.test.ts` | 65 | `push()`, `markAsRead()`, `markAllRead()`, `clear()`, derived `unreadCount` |
| `shared/stores/printReceipt.test.ts` | 18 | Basic writable store behavior |

## Tier 2 — Key UI Components
**Prioritas:** Tinggi  
**Effort:** Sedang (~4–6 jam total)  
**Value:** Tinggi

| Component | Lines | Key Behavior to Test |
|------------|------:|---------------------|
| `shared/ui/Modal.svelte` | 130 | Render saat `open`, close saat backdrop click / ESC, focus trap |
| `shared/ui/Pagination.svelte` | 148 | Perhitungan halaman, prev/next/first/last, inline page input |
| `shared/ui/SearchBar.svelte` | 67 | Perubahan input, tombol clear, loading spinner |
| `shared/ui/StatCard.svelte` | 52 | Render value/label, indikator trend naik/turun |
| `shared/ui/Toast.svelte` | 39 | Render toast dari store, dismiss saat diklik |
| `app/layouts/Sidebar.svelte` | 435 | Role-based navigation visibility, collapse/expand, logout |
| `app/components/ReceiptPrintOverlay.svelte` | 66 | Render saat `$printReceipt` memiliki nilai |

## Tier 3 — Entry Pages
**Prioritas:** Menengah  
**Effort:** Sedang (~2–3 jam total)

| Page | Lines | Key Behavior |
|--------|------:|-------------|
| `auth/components/LoginPage.svelte` | 163 | Submit form memanggil `login()`, menampilkan error saat gagal |
| `dashboard/components/Home.svelte` | 182 | Render stat cards dari live data, koneksi WebSocket |

## Tier 4 — Large Pages
**Prioritas:** Rendah (ditunda sementara)  
**Effort:** Tinggi

| Page | Lines |
|--------|------:|
| `PosPage.svelte` | 932 |
| `ProductsPage.svelte` | 1,187 |
| `TransactionsPage.svelte` | 843 |
| `ReportsPage.svelte` | 2,189 |
| `CustomersPage.svelte` | 669 |
| `AuditLogsPage.svelte` | 1,135 |
| `RolesPage.svelte` | 744 |
| `UsersPage.svelte` | 693 |
| `CategoriesPage.svelte` | 419 |

## Recommended Execution Order

### Phase 1 — Quick Wins
- pos-store
- customer-store
- sales-store
- product-store
- inventory-store
- toast
- notifications
- printReceipt

### Phase 2 — Core Shared Components
- Modal
- Pagination
- SearchBar
- Toast
- StatCard

### Phase 3 — Critical User Flows
- Sidebar
- ReceiptPrintOverlay
- LoginPage
- Dashboard/Home

### Phase 4 — Large Feature Pages
- POS
- Products
- Transactions
- Reports
- Customers
- Audit Logs
- Roles
- Users
- Categories

## Estimasi Total

| Tier | Files | Estimasi |
|--------|------:|----------|
| Tier 1 | 8 | 2–3 jam |
| Tier 2 | 7 | 4–6 jam |
| Tier 3 | 2 | 2–3 jam |
| Tier 4 | 9 | 1–2 minggu |
