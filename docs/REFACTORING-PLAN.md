# Refactoring Plan: Modular Monolith Frontend

## Overview

Refactoring Svelte 5 SPA dari struktur flat `src/lib/` menjadi Modular Monolith dengan domain boundaries yang jelas.

| Item | Detail |
|------|--------|
| **Codebase** | Svelte 5 SPA, ~42 komponen, ~69 file sumber |
| **Target** | Modular Monolith dengan 10 modul + shared + app layer |
| **Total fase** | 12 fase |
| **Urutan** | Bottom-up: modul tanpa dependensi didahulukan |
| **Estimasi** | 3-5 hari kerja total |

---

## Target Structure

```
web/src/
├── modules/
│   ├── auth/              ← Login, session, token management
│   ├── product/           ← Product CRUD, categories, brands
│   ├── inventory/         ← Stock adjustment, thresholds
│   ├── customers/         ← Customer CRUD
│   ├── sales/             ← Transaction history, sales export
│   ├── pos/               ← Cart, checkout, payment, receipt
│   ├── reporting/         ← Charts, KPI, period comparison
│   ├── dashboard/         ← Live stats, real-time
│   ├── admin/             ← Users, roles, audit logs
│   └── settings/          ← Categories, brands, tax, UOM, payment methods
├── shared/
│   ├── api/               ← http-client.ts, websocket.ts
│   ├── ui/                ← Button, Modal, DataTable, etc.
│   ├── stores/            ← toast, notifications, printReceipt
│   ├── utils/             ← jakartaTime, debounce, cn
│   ├── actions/           ← chart action
│   └── types/             ← truly shared types
├── app/
│   ├── router/            ← router implementation + route config
│   ├── layouts/           ← Layout, Sidebar, Topbar
│   └── providers/         ← auth-init, websocket lifecycle
└── App.svelte             ← slim orchestrator (~50 baris)
```

## Dependency Direction

```
      auth ─────────────────────────────────→ semua modul
        │
  product ◄─── pos
        │
 customers ◄── pos
        │
   sales ◄─── reporting
        │    ◄─── dashboard
        │
    admin
        │
 settings
```

Hanya 4 jalur dependency antar modul yang berbeda domain:
1. `pos → product` (data produk)
2. `pos → customers` (data customer)
3. `reporting → sales` (data penjualan)
4. `dashboard → sales` (data realtime)

Setiap jalur hanya melalui `index.ts` — tidak ada akses ke internal modul lain.

---

## Phase 0: Foundation

### 0.1 Setup Path Alias

**File diubah:** `vite.config.js`, `tsconfig.json`

```js
// vite.config.js
resolve: {
  alias: {
    '$lib':       fileURLToPath(new URL('./src/lib', import.meta.url)),
    '$modules':   fileURLToPath(new URL('./src/modules', import.meta.url)),
    '$shared':    fileURLToPath(new URL('./src/shared', import.meta.url)),
    '$app':       fileURLToPath(new URL('./src/app', import.meta.url)),
  }
}
```

```json
// tsconfig.json
"paths": {
  "$lib/*":      ["./src/lib/*"],
  "$modules/*":  ["./src/modules/*"],
  "$shared/*":   ["./src/shared/*"],
  "$app/*":      ["./src/app/*"]
}
```

### 0.2 Buat Struktur Folder

```bash
mkdir -p src/modules/{auth,product,inventory,sales,pos,reporting,dashboard,admin,customers,settings}/{components,services,stores,lib,types}
mkdir -p src/shared/{ui,api,stores,utils,actions,types,constants}
mkdir -p src/app/{router,layouts,providers}
```

### 0.3 Pindahkan Shared Files

| File Asal | File Tujuan |
|-----------|-------------|
| `lib/utils/jakartaTime.ts` | `shared/utils/jakartaTime.ts` |
| `lib/utils/debounce.ts` | `shared/utils/debounce.ts` |
| `lib/utils/cn.ts` | `shared/utils/cn.ts` |
| `lib/actions/chart.js` | `shared/actions/chart.ts` (migrasi JS→TS) |
| `lib/stores/toast.ts` | `shared/stores/toast.svelte.ts` (migrasi ke runes) |
| `lib/stores/notifications.ts` | `shared/stores/notifications.svelte.ts` (migrasi ke runes) |
| `lib/stores/printReceipt.ts` | `shared/stores/printReceipt.svelte.ts` (migrasi ke runes) |

### 0.4 HttpClient Consolidation

Satukan `apiClient` (axios) dan `apiFetch` (native) menjadi satu service:

```ts
// shared/api/http-client.ts
// Satu public API dengan dua implementasi internal
export async function apiGet<T>(url: string, params?: Record<string, string>): Promise<T>;
export async function apiPost<T>(url: string, body?: unknown): Promise<T>;
export async function apiPut<T>(url: string, body?: unknown): Promise<T>;
export async function apiDelete<T>(url: string): Promise<T>;
export async function apiFetch(url: string, options?: RequestInit): Promise<Response>;
```

**Risiko:** Rendah.

---

## Phase 1: Auth Module

Dependency: **none** (leaf module)

### Files to Create

```
modules/auth/
├── services/
│   └── auth-service.ts             ← login, logout, refreshAccessToken, restoreSession, checkAuth
├── stores/
│   └── auth-store.svelte.ts        ← user, isAuthenticated, loading (runes)
├── lib/
│   └── session.ts                  ← sessionStorage helpers, getAuthToken
├── types/
│   └── index.ts                    ← User, LoginResult, AuthState
├── components/
│   └── LoginPage.svelte            ← dipindah dari pages/LoginPage.svelte + tambah lang="ts"
└── index.ts                        ← public API
```

### Public API (index.ts)

```ts
export { login, logout, restoreSession, refreshAccessToken, refreshTokenSilently, checkAuth } from './services/auth-service';
export { useAuthStore } from './stores/auth-store.svelte';
export { getAuthToken } from './lib/session';
export type { User, LoginResult } from './types';
```

### Import Updates

| File Asal | Import Lama | Import Baru |
|-----------|-------------|-------------|
| `shared/api/http-client.ts` | `import { getAuthToken } from '$lib/stores/auth'` | `import { getAuthToken } from '$modules/auth'` |
| `shared/api/websocket.ts` | `import { refreshTokenSilently } from '$lib/api/auth'` | `import { refreshTokenSilently } from '$modules/auth'` |
| `app/providers/auth-init.ts` | (inline di App.svelte) | `import { restoreSession, useAuthStore } from '$modules/auth'` |
| Every page using `auth` store | `import { auth } from '$lib/stores/auth'` | `import { useAuthStore } from '$modules/auth'` |

### Verification

1. Login → success
2. Refresh page → session restored
3. Token expired → auto refresh
4. Logout → redirect ke /login

**Risiko:** Sedang.

---

## Phase 2: Product Module

Dependency: **none** (leaf module, except auth)

### Files to Create

```
modules/product/
├── components/
│   ├── ProductPage.svelte          ← orkestrator (dari ProductsPage.svelte, ~80 baris)
│   ├── ProductTable.svelte         ← tabel + sorting + bulk select (~300 baris)
│   ├── ProductFilterBar.svelte     ← search + filter + chips (~120 baris)
│   ├── ProductDetailDrawer.svelte  ← side panel detail (~200 baris)
│   ├── ProductFormModal.svelte     ← pindah dari components/inventory/
│   ├── ProductDeleteModal.svelte   ← konfirmasi hapus (~50 baris)
│   ├── BulkStatusModal.svelte      ← bulk update status (~60 baris)
│   └── CategoryFilterModal.svelte  ← pindah dari components/ui/
├── services/
│   └── product-service.ts          ← CRUD + bulk status + fetch master data
├── stores/
│   └── product-store.svelte.ts     ← products, filters, pagination, selection
├── lib/
│   └── product-utils.ts            ← statusInfo, formatCurrency, validateProductForm
├── types/
│   └── index.ts                    ← Product, ProductForm, ProductFilters, Category, Brand, TaxClass, UnitOfMeasure
└── index.ts
```

### Component Boundaries

| Component | Props (input) | Events (output) |
|-----------|--------------|-----------------|
| ProductTable | `products`, `selectedIds`, `sortBy`, `sortDir`, `loading`, `canManageInventory` | `onselect`, `onselectAll`, `onsort`, `onedit`, `ondelete`, `onadjustStock`, `ondetail` |
| ProductFilterBar | `searchQuery`, `selectedCategories`, `filterStatus`, `lowStockOnly`, `categories` | `onsearch`, `onfilterChange`, `onclearFilters` |
| ProductDetailDrawer | `product`, `open`, `canEdit` | `onclose`, `onedit`, `ondelete` |
| BulkStatusModal | `open`, `products`, `selectedIds` | `onconfirm`, `oncancel` |

### Public API (index.ts)

```ts
export { getProducts, getProductById, createProduct, updateProduct, deleteProduct, bulkUpdateStatus } from './services/product-service';
export { getCategories, getBrands, getTaxClasses, getUnitsOfMeasure } from './services/product-service';
export { useProductStore } from './stores/product-store.svelte';
export { statusInfo, formatCurrency } from './lib/product-utils';
export type { Product, ProductForm, ProductFilters, Category, Brand, TaxClass, UnitOfMeasure } from './types';
```

### Import Updates

| File | Import Lama | Import Baru |
|------|-------------|-------------|
| `app/router/routes.ts` | `import ProductsPage from '$lib/pages/ProductsPage.svelte'` | `() => import('$modules/product')` |
| `modules/pos/pos-service.ts` | `apiClient.get('/products?...')` | `import { getProducts } from '$modules/product'` |
| `modules/pos/PosPage.svelte` | inline fetch | lewat pos-service → $modules/product |

### Verification

1. List produk → muncul
2. Search → filter sesuai query
3. Filter kategori → produk terfilter
4. Add produk → muncul di list
5. Edit produk → data berubah
6. Delete produk → hilang dari list
7. Bulk status → status berubah
8. Stock adjust → stok berubah
9. Detail drawer → informasi lengkap

**Risiko:** Tinggi. ProductsPage adalah 1188 baris. Potong bertahap: service dulu → store → template.

---

## Phase 3: Inventory Module

Dependency: **product** (types)

```
modules/inventory/
├── components/
│   └── StockAdjustModal.svelte     ← pindah dari components/inventory/
├── services/
│   └── inventory-service.ts        ← adjustStock, getStockThresholds
├── stores/
│   └── inventory-store.svelte.ts
├── types/
│   └── index.ts                    ← StockAdjustment, StockThreshold
└── index.ts
```

```ts
// Public API
export { adjustStock, getStockThresholds } from './services/inventory-service';
export type { StockAdjustment, StockThreshold } from './types';
```

**Risiko:** Rendah.

---

## Phase 4: Customers Module

Dependency: **none** (leaf module)

```
modules/customers/
├── components/
│   ├── CustomersPage.svelte        ← pindah dari pages/
│   ├── CustomerTable.svelte
│   ├── CustomerFormModal.svelte
│   └── CustomerDeleteModal.svelte
├── services/
│   └── customer-service.ts
├── stores/
│   └── customer-store.svelte.ts
├── types/
│   └── index.ts
└── index.ts
```

```ts
// Public API
export { getCustomers, createCustomer, updateCustomer, deleteCustomer, searchCustomers } from './services/customer-service';
export type { Customer } from './types';
```

**Risiko:** Rendah.

---

## Phase 5: Sales Module

Dependency: **none** (leaf module)

```
modules/sales/
├── components/
│   ├── TransactionsPage.svelte     ← pindah dari pages/ (orkestrator)
│   ├── SaleTable.svelte
│   ├── SaleDetailModal.svelte
│   └── SaleExportButton.svelte
├── services/
│   └── sales-service.ts            ← getSalesHistory, getSaleByID, exportSales
├── stores/
│   └── sales-store.svelte.ts
├── types/
│   └── index.ts                    ← Sale, SaleItem, SaleFilters, SaleExportRow
└── index.ts
```

```ts
// Public API
export { getSalesHistory, getSaleById, exportSalesToExcel, exportSalesToPdf } from './services/sales-service';
export type { Sale, SaleItem, SaleFilters } from './types';
```

**Risiko:** Rendah-Sedang.

---

## Phase 6: POS Module

Dependency: **product**, **customers**, **sales**, **auth** (user info)

```
modules/pos/
├── components/
│   ├── PosPage.svelte              ← pindah dari pages/ (orkestrator, ~60 baris)
│   ├── ProductSearchPanel.svelte   ← panel kiri: search + grid (~200 baris)
│   ├── CartPanel.svelte            ← panel kanan: cart + summary (~200 baris)
│   ├── CheckoutModal.svelte        ← payment modal (~120 baris)
│   ├── CustomerSelectModal.svelte  ← search customer (~80 baris)
│   └── ReceiptPrintOverlay.svelte  ← dari App.svelte thermal receipt (~100 baris)
├── services/
│   └── pos-service.ts              ← createSale, printReceipt, loadProducts
├── stores/
│   └── pos-store.svelte.ts         ← cart, payment, selectedCustomer, lastSale
├── lib/
│   └── cart-utils.ts               ← calcSubtotal, calcTax, calcChange, calcTotalItems
├── types/
│   └── index.ts                    ← CartItem, CheckoutPayload, ReceiptData
└── index.ts
```

### Component Boundaries

| Component | Props (input) | Events (output) |
|-----------|--------------|-----------------|
| ProductSearchPanel | `products`, `searchQuery`, `loading`, `pagination` | `onsearch`, `onpageChange`, `onaddToCart` |
| CartPanel | `cart`, `subtotal`, `tax`, `total`, `itemCount` | `onupdateQty`, `onremove`, `oncheckout`, `onclear` |
| CheckoutModal | `open`, `total`, `paymentMethods` | `onconfirm(payload)`, `oncancel` |
| CustomerSelectModal | `open`, `customers` | `onselect(customer)`, `oncancel` |

```ts
// Public API
export { createSale, calcCartTotal, printThermalReceipt } from './services/pos-service';
export { usePosStore } from './stores/pos-store.svelte';
export { calcSubtotal, calcTax, calcChange } from './lib/cart-utils';
export type { CartItem, CheckoutPayload } from './types';
```

**Risiko:** Tinggi. Fitur inti bisnis. Checkout flow harus identik.

---

## Phase 7: Reporting Module

Dependency: **sales** (chart data)

```
modules/reporting/
├── components/
│   ├── ReportsPage.svelte          ← pindah dari pages/ (orkestrator, ~60 baris)
│   ├── PeriodSelector.svelte       ← dropdown + calendar (~280 baris)
│   ├── KpiCards.svelte             ← 5 KPI cards (~150 baris)
│   ├── RevenueChart.svelte         ← Chart.js canvas (~100 baris)
│   ├── ComparisonDataTable.svelte  ← sortable table (~190 baris)
│   └── BestWorstBadges.svelte      ← best/worst period (~40 baris)
├── services/
│   └── reporting-service.ts        ← fetchChartData, fetchPeriodComparison, exportExcel, exportPdf
├── stores/
│   └── reporting-store.svelte.ts   ← kpiData, chartData, chartType, sortedRows, tableRows
├── lib/
│   └── period-utils.ts             ← getPeriodDateRange, formatDate, getPeriodLabel, formatCurrencyShort
├── types/
│   └── index.ts                    ← ChartDataRow, KpiData, PeriodType, PeriodOption
└── index.ts
```

### Derived State Extraction

Semua `$derived` dan `$derived.by()` dari ReportsPage dipindah ke `reporting-store.svelte.ts`:
- `chartType`, `statCardLabels`, `comparisonDateRange`, `peakChartValue`
- `chartTotalRevenue`, `chartYear`, `daysInMonth`, `projectedRevenue`
- `tableRows`, `sortedRows`, `bestPeriod`, `worstPeriod`

### Helper Functions Extraction

Dipindah ke `period-utils.ts`:
- `getPeriodDateRange(periodType)` — logic kalkulasi tanggal
- `formatDate(dateString)` — format display
- `getPeriodLabel(item)` — label chart axis
- `formatCurrencyShort(value)` — format Rp
- `formatLargeNumber(value)` — format angka

### Public API

```ts
export { fetchChartData, fetchPeriodComparison, exportExcel, exportPdf } from './services/reporting-service';
export { useReportingStore } from './stores/reporting-store.svelte';
export { formatCurrencyShort, formatDate, getPeriodLabel, getPeriodDateRange } from './lib/period-utils';
export type { ChartDataRow, KpiData, PeriodType } from './types';
```

**Risiko:** Tinggi. ReportPage adalah 2191 baris — file terbesar. Banyak derived state yang saling referensi. Ekstrak store dulu sebelum template.

---

## Phase 8: Dashboard Module

Dependency: **sales** (real-time stats), **websocket**

```
modules/dashboard/
├── components/
│   └── DashboardPage.svelte        ← pindah dari pages/Home.svelte + rename
├── services/
│   └── dashboard-service.ts        ← fetchLiveStats, fetchDashboardYears
├── stores/
│   └── dashboard-store.svelte.ts   ← liveStats, availableYears, ws listeners
├── types/
│   └── index.ts                    ← LiveStats, DashboardStat
└── index.ts
```

```ts
// Public API
export { fetchLiveStats, fetchDashboardYears } from './services/dashboard-service';
export { useDashboardStore } from './stores/dashboard-store.svelte';
export type { LiveStats } from './types';
```

**Risiko:** Rendah. Home.svelte hanya ~300 baris.

---

## Phase 9: Admin Module

Dependency: **auth** (role check)

```
modules/admin/
├── components/
│   ├── UsersPage.svelte            ← pindah dari pages/admin/
│   ├── UserFormModal.svelte
│   ├── RolesPage.svelte            ← pindah dari pages/admin/
│   ├── RoleFormModal.svelte
│   ├── PermissionMatrix.svelte
│   ├── AuditLogsPage.svelte        ← pindah dari pages/admin/
│   └── AuditLogTable.svelte
├── services/
│   └── admin-service.ts            ← users CRUD, roles CRUD, permissions, audit logs
├── stores/
│   └── admin-store.svelte.ts
├── types/
│   └── index.ts                    ← AdminUser, AdminRole, AdminPermission, AuditLog
└── index.ts
```

**Note:** CategoriesPage (`pages/admin/CategoriesPage.svelte`) dipindah ke `modules/settings/` karena kategori adalah master data, bukan user management.

```ts
// Public API
export { getUsers, createUser, updateUser, deleteUser } from './services/admin-service';
export { getRoles, createRole, updateRole, deleteRole, updateRolePermissions, listPermissions } from './services/admin-service';
export { getAuditLogs, exportAuditLogs } from './services/admin-service';
export type { AdminUser, AdminRole, AdminPermission } from './types';
```

**Risiko:** Rendah.

---

## Phase 10: Settings Module

Dependency: **none** (leaf module)

```
modules/settings/
├── services/
│   └── settings-service.ts         ← CRUD kategori, brand, tax class, UOM, payment methods
├── types/
│   └── index.ts                    ← Category, Brand, TaxClass, UnitOfMeasure, PaymentMethod
└── index.ts
```

Category CRUD page (`CategoriesPage.svelte`) tetap tampil di route `/admin/categories` tapi data kategori disimpan di sini.

```ts
// Public API
export { getCategories, createCategory, updateCategory, deleteCategory } from './services/settings-service';
export type { MasterCategory, MasterBrand, MasterTaxClass, MasterUnitOfMeasure, MasterPaymentMethod } from './types';
```

**Risiko:** Rendah.

---

## Phase 11: App Layer Refactor

### 11.1 Router Extraction

File `lib/router/index.ts` → pindah ke `app/router/index.ts` (tidak berubah).

File baru `app/router/routes.ts`:

```ts
export interface RouteConfig {
  path: string;
  title: string;
  guard?: 'auth' | 'guest' | 'admin';
}

export const routes: RouteConfig[] = [
  { path: '/login',              title: 'Login',            guard: 'guest' },
  { path: '/',                   title: 'Dashboard',        guard: 'auth'  },
  { path: '/pos',                title: 'POS',              guard: 'auth'  },
  { path: '/inventory/products', title: 'Products',         guard: 'auth'  },
  { path: '/reports',            title: 'Reports',          guard: 'auth'  },
  { path: '/transactions',       title: 'Transactions',     guard: 'auth'  },
  { path: '/customers',          title: 'Customers',         guard: 'auth'  },
  { path: '/admin',              title: 'Administration',   guard: 'admin' },
  { path: '/admin/users',        title: 'User Management',  guard: 'admin' },
  { path: '/admin/roles',        title: 'Role Management',  guard: 'admin' },
  { path: '/admin/audit-logs',   title: 'Audit Logs',       guard: 'admin' },
  { path: '/admin/categories',   title: 'Categories',       guard: 'admin' },
];
```

### 11.2 Layouts

| File Asal | File Tujuan |
|-----------|-------------|
| `lib/components/Layout.svelte` | `app/layouts/Layout.svelte` |
| `lib/components/Sidebar.svelte` | `app/layouts/Sidebar.svelte` |
| `lib/components/Topbar.svelte` | `app/layouts/Topbar.svelte` |
| `lib/components/Navbar.svelte` | HAPUS (tidak dipakai) |
| `lib/components/NotificationBell.svelte` | `app/layouts/NotificationBell.svelte` |

### 11.3 Providers

**File baru `app/providers/auth-init.ts`:**

```ts
import { restoreSession, useAuthStore } from '$modules/auth';

export async function initAuth(path: string): Promise<void> {
  const store = useAuthStore();
  const result = await restoreSession();
  if (result.success && result.user) {
    store.user = result.user;
    store.isAuthenticated = true;
  }
  store.loading = false;
}
```

**File baru `app/providers/websocket.ts`:**

```ts
import { useWebSocket } from '$shared/api/websocket';
import { useAuthStore } from '$modules/auth';

export function initWebSocket(): void {
  const auth = useAuthStore();
  // Koneksi WS tergantung auth state
}
```

### 11.4 App.svelte Target (~50 baris)

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { router, getPath } from '$app/router';
  import { initAuth } from '$app/providers/auth-init';
  import { initWebSocket } from '$app/providers/websocket';
  import Layout from '$app/layouts/Layout.svelte';
  import Toast from '$shared/ui/Toast.svelte';
  import ReceiptPrintOverlay from '$modules/pos/components/ReceiptPrintOverlay.svelte';

  let Component = $state();
  let currentPath = $state(getPath());
  let isInitializing = $state(true);

  onMount(async () => {
    await initAuth(currentPath);
    initWebSocket();
    isInitializing = false;
  });

  let isLogin = $derived(currentPath === '/login');
</script>

{#if isInitializing}
  <!-- splash screen -->
{:else if isLogin}
  <Component />
{:else}
  <Layout {currentPath}>
    <Component />
  </Layout>
{/if}

<Toast />
<ReceiptPrintOverlay />
```

**Risiko:** Sedang. App.svelte adalah root. Pastikan init order benar (auth → route → ws).

---

## Phase 12: Shared UI Consolidation

### 12.1 Pindahkan Shared Components

Dari `lib/components/ui/` ke `shared/ui/`:

```
shared/ui/
├── index.ts                  ← barrel export SEMUA komponen
├── Button.svelte
├── Badge.svelte
├── Card.svelte
├── Input.svelte
├── Modal.svelte
├── Pagination.svelte
├── SearchBar.svelte
├── Skeleton.svelte
├── Toast.svelte
├── DataTable.svelte          ← baru, ekstrak pola tabel umum
├── PageHeader.svelte
├── StatCard.svelte
├── RpIcon.svelte
├── ActionBadge.svelte
├── ExpandableRow.svelte
├── FilterChip.svelte
└── ProductActionsDropdown.svelte  ← pindah ke modules/product/
```

### 12.2 Barrel Export

```ts
// shared/ui/index.ts
export { default as Button } from './Button.svelte';
export { default as Modal } from './Modal.svelte';
export { default as Input } from './Input.svelte';
export { default as Badge } from './Badge.svelte';
export { default as Card } from './Card.svelte';
export { default as Pagination } from './Pagination.svelte';
export { default as SearchBar } from './SearchBar.svelte';
export { default as Skeleton } from './Skeleton.svelte';
export { default as Toast } from './Toast.svelte';
export { default as DataTable } from './DataTable.svelte';
export { default as PageHeader } from './PageHeader.svelte';
export { default as StatCard } from './StatCard.svelte';
export { default as RpIcon } from './RpIcon.svelte';
export { default as ActionBadge } from './ActionBadge.svelte';
export { default as ExpandableRow } from './ExpandableRow.svelte';
export { default as FilterChip } from './FilterChip.svelte';
```

### 12.3 Import Pattern Setelah Refactor

```ts
// Sebelum:
import Button from '$lib/components/ui/Button.svelte';
import Modal from '$lib/components/ui/Modal.svelte';

// Sesudah:
import { Button, Modal } from '$shared/ui';
```

### 12.4 Komponen yang TIDAK Dipindahkan ke Shared

| Komponen | Tetap di | Alasan |
|----------|----------|--------|
| ProductFormModal | `modules/product/` | hanya dipakai 1 modul |
| StockAdjustModal | `modules/inventory/` | hanya dipakai 1 modul |
| ProductActionsDropdown | `modules/product/` | domain-specific |
| CategoryFilterModal | `modules/product/` | domain-specific |
| Calendar components | `modules/reporting/` | hanya dipakai reporting |
| PeriodSelector | `modules/reporting/` | domain-specific |

**Risiko:** Rendah. Hanya pindah folder + update import.

---

## Verification Per Fase

| Fase | Test | Kriteria |
|------|------|----------|
| 0 | `npm run dev` | App jalan tanpa error alias |
| 1 | Login → logout → refresh | Session restore berfungsi |
| 2 | CRUD produk, filter, search | Semua fitur produk berfungsi |
| 3 | Adjust stock | Stok berubah |
| 4 | CRUD customer | Customer tersimpan |
| 5 | List transaksi, export | Data transaksi tampil |
| 6 | Checkout, print receipt | Transaksi sukses, receipt muncul |
| 7 | Ganti periode, chart, export | Chart dan data berubah sesuai periode |
| 8 | Dashboard realtime | Live stats muncul |
| 9 | CRUD user, role, audit | Admin functions berfungsi |
| 10 | CRUD category | Kategori tersimpan |
| 11 | Navigasi semua route | Setiap halaman render |
| 12 | `npm run build` | Build sukses |

---

## Risk Matrix

| Fase | Risiko | Mitigasi |
|------|--------|----------|
| 0 | Rendah | Backup vite.config.js sebelum edit |
| 1 | Sedang | Test session restore manual; jangan hapus file lama |
| 2 | **Tinggi** | ProductsPage 1188 baris; potong service → store → template |
| 3 | Rendah | Hanya pindahan |
| 4 | Rendah | CustomersPage ~400 baris |
| 5 | Rendah-Sedang | TransactionsPage ~600 baris |
| 6 | **Tinggi** | Fitur inti bisnis; test checkout flow setelah setiap potong |
| 7 | **Tinggi** | ReportsPage 2191 baris; banyak derived state saling referensi |
| 8 | Rendah | Home.svelte ~300 baris |
| 9 | Rendah | Halaman admin kecil |
| 10 | Rendah | Settings minimal |
| 11 | Sedang | App.svelte adalah root; init order penting |
| 12 | Rendah | Hanya pindah + barrel export |

---

## Architecture Decision Records

### ADR-1: Category Domain

**Keputusan:** Category types di `modules/product/types/`, Category CRUD service di `modules/settings/`.

**Alasan:** Produk membutuhkan category untuk display (relasi). Manajemen kategori adalah fungsi settings/master data. Tipe `Category` diperlukan oleh module product, jadi ditaruh di sana.

### ADR-2: Calendar Components

**Keputusan:** Calendar tetap di `modules/reporting/components/`, tidak dipindahkan ke shared.

**Alasan:** Saat ini hanya dipakai oleh ReportsPage. Jika ada modul lain (misal dashboard) yang butuh, baru dipindahkan. Favor duplication over wrong abstraction.

### ADR-3: Single HttpClient

**Keputusan:** Satukan `apiClient` (axios) dan `apiFetch` (fetch) menjadi satu service `shared/api/http-client.ts`.

**Alasan:** Dua pendekatan API meningkatkan inconsistency. Service layer di atasnya akan memanggil satu fungsi (`apiGet`, `apiPost`, dll) tanpa peduli implementasi bawahannya.

### ADR-4: Module Pages sebagai Orkestrator

**Keputusan:** Setiap modul memiliki `<ModulName>Page.svelte` yang bertindak sebagai orkestrator. Isinya minimal: import sub-komponen + state minimal + event handler delegasi.

**Alasan:** Memisahkan tanggung jawab. Page mengatur alur (kapan fetch, kapan modal muncul). Sub-komponen fokus pada rendering dan interaksi UI.

### ADR-5: No Cross-Module Component Imports

**Keputusan:** Modul A tidak boleh mengimpor komponen Svelte dari modul B. Jika butuh data dari modul B, panggil fungsi service melalui `index.ts`.

**Alasan:** Mencegah tight coupling UI. Jika modul `pos` butuh menampilkan daftar produk, dia panggil `getProducts()` dari `$modules/product`, bukan mengimpor `ProductTable` dari `$modules/product/components/ProductTable.svelte`.
