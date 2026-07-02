# Test Plan: Modular Monolith Refactoring

## Baseline (Before Refactoring)

| Test File | Type | Tests | Status |
|-----------|------|-------|--------|
| `utils/jakartaTime.test.ts` | Pure function | 15 ✅ | **Survives** refactoring |
| `composables/useWebSocket.test.ts` | Simulated logic | 3 ✅ | **Will break** (file moves) |
| `components/Sidebar.svelte.test.ts` | Structure guard | 24 ✅ | **Will break** (file moves) |
| `pages/Home.svelte.test.ts` | Structure guard | 10 ✅ | **Will break** (component splits) |
| `pages/PosPage.svelte.test.ts` | Mixed (structure + logic) | 46 ✅ | **Will break** (component splits) |
| `pages/ProductsPage.svelte.test.ts` | Structure guard | 78 ✅ | **Will break** (component splits) |
| `pages/ReportsPage.svelte.test.ts` | Structure guard | 31 ✅ / 8 ❌ | **Already failing, will break** |
| `pages/admin/Users.svelte.test.ts` | Structure guard | 19 ✅ | **Will break** (file moves) |
| **Total** | | **226 ✅ / 8 ❌** | **Only 15 tests survive as-is** |

### Why Most Tests Will Break

The existing tests use **source-structure guarding**:

```ts
// ProductsPage.svelte.test.ts — pattern that WILL break
it('declares showModal for add/edit product', () => {
  expect(src).toContain('let showModal');  // ← checks source text!
});
```

When `ProductsPage.svelte` is split into `ProductTable`, `ProductFilterBar`, `ProductFormModal`, etc., these variables move to sub-components. The source guard fails.

**This is the right time to replace them with proper tests.**

---

## Test Migration Strategy

### Three Categories of Tests

| Category | Survival | Action |
|----------|----------|--------|
| **Pure function** | ✅ Survives | Keep as-is. Move with the file. |
| **Source-structure guard** | ❌ Breaks | Replace with service/store/component tests |
| **Simulated runtime** | ⚠️ Partially | Refactor into proper unit tests |

### Test Pyramid (Target)

Setelah refactoring, test pyramid idealnya:

```
        ╱╲
       ╱  ╲          E2E (Playwright) — smoke test
      ╱    ╲
     ╱──────╲        Integration — component test
    ╱        ╲
   ╱──────────╲      Unit — service, store, utils logic
  ╱────────────╲
```

Saat ini: **hampir tidak ada unit test untuk business logic.**

Service layer yang baru diekstrak adalah **kesempatan emas** untuk menambah unit test yang sebenarnya.

---

## Per-Phase Test Plan

### Phase 0: Foundation

**File pindah:** `utils/jakartaTime.ts`, `utils/debounce.ts`, `utils/cn.ts`, `actions/chart.js`, `stores/toast.ts`, `stores/notifications.ts`, `stores/printReceipt.ts`

**Tests yang terpengaruh:**

| Test | Perubahan |
|------|-----------|
| `utils/jakartaTime.test.ts` | Update import dari `$lib/utils/jakartaTime` → `$shared/utils/jakartaTime` |
| `composables/useWebSocket.test.ts` | File pindah ke `shared/api/websocket.ts` — update path |

**Yang perlu dilakukan:**

1. Pindahkan `jakartaTime.test.ts` ke `shared/utils/jakartaTime.test.ts`
2. Update path import di test file
3. Pindahkan `useWebSocket.test.ts` ke `shared/api/websocket.test.ts`

**Smoke test manual:**
```
□ npm run dev → app jalan
□ npm run build → build sukses
□ npm test → 226 passed (sama dengan baseline)
```

**Jangan hapus file asli.** Buat file baru di target, test dulu, baru hapus yang lama.

---

### Phase 1: Auth Module

**Yang berubah:**
- `stores/auth.ts` → `modules/auth/stores/auth-store.svelte.ts`
- `api/auth.ts` → `modules/auth/services/auth-service.ts`
- `pages/LoginPage.svelte` → `modules/auth/components/LoginPage.svelte`

**Tests yang terpengaruh:**

| Test File | Tests | Action |
|-----------|-------|--------|
| `PosPage.svelte.test.ts` | 2 tes ref `auth` | Update import |
| `ProductsPage.svelte.test.ts` | 3 tes ref `canManageInventory` | Update import |
| `Users.svelte.test.ts` | 1 tes ref `auth` | Update import |

**Tidak ada existing test untuk auth logic.** Ini kesempatan buat nambah:

**New tests to create:**
```
modules/auth/services/__tests__/auth-service.test.ts
modules/auth/stores/__tests__/auth-store.test.ts
modules/auth/lib/__tests__/session.test.ts
```

**Contoh test auth-service:**
```ts
import { describe, it, expect, vi } from 'vitest';

// Test login flow without rendering
describe('login', () => {
  it('stores token on successful login', async () => {
    // mock api response
    // call login()
    // expect sessionStorage.getItem('access_token').toBe('...')
  });

  it('returns false on failed login', async () => {
    // mock 401 response
    // expect(await login('wrong', 'wrong')).toBe(false)
  });
});
```

**Contoh test auth-store:**
```ts
import { describe, it, expect } from 'vitest';

describe('auth-store', () => {
  it('starts with loading=true and isAuthenticated=false', () => {
    const store = useAuthStore();
    expect(store.loading).toBe(true);
    expect(store.isAuthenticated).toBe(false);
  });

  it('sets user on setUser call', () => {
    const store = useAuthStore();
    store.setUser({ id: 1, username: 'admin' });
    expect(store.user?.username).toBe('admin');
    expect(store.isAuthenticated).toBe(true);
  });
});
```

**Smoke test manual:**
```
□ Login dengan kredensial valid → sukses
□ Login dengan kredensial invalid → gagal + error message
□ Refresh halaman → session restore, tetap login
□ Logout → redirect ke /login
□ Akses / (dashboard) tanpa login → redirect ke /login
□ Akses /login setelah login → redirect ke /
```

---

### Phase 2: Product Module

**Yang berubah:**
- `ProductsPage.svelte` (1188 baris) → dipecah jadi 8 komponen baru
- API calls inline → pindah ke `product-service.ts`
- State inline → pindah ke `product-store.svelte.ts`

**Existing test yang akan BREAK:**
`ProductsPage.svelte.test.ts` — **78 tests semuanya akan fail** karena source-structure guard.

**Action:** Tulis ulang test ini menjadi:

1. **Service tests** (baru):
```
modules/product/services/__tests__/product-service.test.ts
```
Test:
```ts
// Test URL construction
it('getProducts builds correct query params', async () => {
  // mock apiClient
  // call getProducts({ search: 'kopi', status: 'active' })
  // expect apiClient.get called with '/products?search=kopi&status=active&limit=20&offset=0'
});

// Test data transformation
it('getProducts transforms response correctly', async () => {
  // mock response { data: [...], total: 5 }
  // expect result = { products: [...], total: 5 }
});
```

2. **Store tests** (baru):
```
modules/product/stores/__tests__/product-store.test.ts
```
```ts
it('toggleSelect adds/removes id from selectedIds', () => {
  const store = useProductStore();
  store.toggleSelect(1);
  expect(store.selectedIds.has(1)).toBe(true);
  store.toggleSelect(1);
  expect(store.selectedIds.has(1)).toBe(false);
});
```

3. **Utility tests** (baru):
```
modules/product/lib/__tests__/product-utils.test.ts
```
```ts
it('statusInfo returns correct variant for each status', () => {
  expect(statusInfo('active').variant).toBe('success');
  expect(statusInfo('draft').variant).toBe('muted');
  expect(statusInfo('archived').variant).toBe('destructive');
});
```

**Untuk ProductsPage.svelte.test.ts — jangan hapus dulu.** Buat test baru dulu, konfirmasi passing, baru hapus yang lama.

**Smoke test manual:**
```
□ List produk → muncul dengan pagination
□ Search produk → filter sesuai query
□ Filter kategori → produk terfilter
□ Filter status → produk terfilter
□ Low stock toggle → hanya tampil stok rendah
□ Sort name, price, stock → urutan berubah
□ Bulk select → checkbox tercentang
□ Bulk status → status berubah massal
□ Add produk → muncul di list
□ Edit produk → data berubah
□ Delete produk → hilang dari list
□ Stock adjust → stok berubah
□ Detail drawer → informasi lengkap
□ Copy SKU → clipboard
□ Permission check → tombol add/edit sesuai role
```

---

### Phase 3: Inventory Module

**Hanya pindahan.** Tidak perlu test baru untuk StockAdjustModal.

**Smoke test:**
```
□ Stock adjust → stok berubah
□ Notes required → validasi
□ Zero quantity → error
```

---

### Phase 4: Customers Module

CustomersPage ~400 baris. Ekstrak service + store.

**New test:**
```
modules/customers/services/__tests__/customer-service.test.ts
```

**Smoke test:**
```
□ List customer → muncul
□ Search customer → filter
□ Add customer → tersimpan
□ Edit customer → data berubah
□ Delete customer → hilang
□ Customer dengan transaksi → bisa dihapus? (validasi backend)
```

---

### Phase 5: Sales Module

TransactionsPage ~600 baris. Ekstrak service + store + export.

**New tests:**
```
modules/sales/services/__tests__/sales-service.test.ts
modules/sales/lib/__tests__/sales-utils.test.ts
```

**Smoke test:**
```
□ List transaksi → muncul dengan pagination
□ Filter tanggal → data terfilter
□ Filter payment method → terfilter
□ Search invoice → filter
□ Click transaksi → detail modal
□ Export Excel → file download
□ Print receipt dari transaksi → print dialog
```

---

### Phase 6: POS Module

PosPage 935 baris. Ekstrak service + store + cart-utils + sub-komponen.

**New tests:**
```
modules/pos/lib/__tests__/cart-utils.test.ts    ← PRIORITAS TERTINGGI
modules/pos/services/__tests__/pos-service.test.ts
modules/pos/stores/__tests__/pos-store.test.ts
```

**cart-utils.test.ts:**
```ts
import { describe, it, expect } from 'vitest';

describe('calcSubtotal', () => {
  it('returns 0 for empty cart', () => {
    expect(calcSubtotal([])).toBe(0);
  });

  it('sums price * quantity for each item', () => {
    const cart = [
      { price: 10000, quantity: 2 },
      { price: 5000, quantity: 3 },
    ];
    expect(calcSubtotal(cart)).toBe(35000);
  });
});

describe('calcTax', () => {
  it('calculates PPN 11% correctly', () => {
    // DPP = total * 100/111
    // PPN = total - DPP
    const total = 11100;
    // DPP = 10000, PPN = 1100
    expect(calcTax(total, 11)).toBe(1100);
  });

  it('returns 0 for non-taxable items', () => {
    expect(calcTax(50000, 0)).toBe(0);
  });
});

describe('calcChange', () => {
  it('returns positive change when overpaid', () => {
    expect(calcChange(50000, 45000)).toBe(5000);
  });

  it('returns 0 when exact amount', () => {
    expect(calcChange(45000, 45000)).toBe(0);
  });

  it('returns negative when underpaid', () => {
    expect(calcChange(40000, 45000)).toBe(-5000);
  });
});
```

**Smoke test manual (KRITIS — urutan tepat):**
```
□ Halaman POS load → product grid muncul
□ Search produk → filter
□ Click add to cart → item masuk cart
□ Update qty (+/-) → quantity berubah
□ Qty = 0 → item hilang dari cart
□ Total, subtotal, tax → kalkulasi benar
□ Select customer → nama customer tercantum
□ Checkout modal → muncul
□ Pilih payment method → (Cash/Card/E-Wallet)
□ Cash payment → cash received input + change due
□ Cash < total → tombol bayar disabled
□ Cash >= total → tombol bayar enabled
□ Cash presets (50k, 100k, 150k, 200k) → berfungsi
□ Konfirmasi checkout → sale sukses, cart kosong
□ Print receipt → preview muncul
□ Keyboard shortcuts:
  □ F2 → focus search
  □ F4 → buka checkout modal (hanya jika cart tidak kosong)
  □ Alt+Del → clear cart + konfirmasi
  □ Escape → tutup modal
  □ Enter di modal checkout → finalize
  □ F3 → tutup modal checkout
  □ F6 → isi cashReceived dengan totalAmount
```

---

### Phase 7: Reporting Module

ReportsPage 2191 baris → paling kompleks. Ekstrak service + store + period-utils + 5 komponen.

**New tests (PRIORITAS):**
```
modules/reporting/lib/__tests__/period-utils.test.ts    ← #1
modules/reporting/services/__tests__/reporting-service.test.ts
modules/reporting/stores/__tests__/reporting-store.test.ts
```

**period-utils.test.ts:**
```ts
import { describe, it, expect } from 'vitest';
import { formatCurrencyShort, formatDate, getPeriodLabel, getPeriodDateRange } from '../period-utils';

describe('formatCurrencyShort', () => {
  it('formats thousands as Rp Xk', () => {
    expect(formatCurrencyShort(5000)).toBe('Rp 5k');
    expect(formatCurrencyShort(999000)).toBe('Rp 999k');
  });

  it('formats millions as Rp Xjt', () => {
    expect(formatCurrencyShort(1500000)).toBe('Rp 1.5jt');
    expect(formatCurrencyShort(2000000)).toBe('Rp 2jt');
  });

  it('formats billions as Rp XM', () => {
    expect(formatCurrencyShort(1000000000)).toBe('Rp 1M');
  });

  it('handles zero', () => {
    expect(formatCurrencyShort(0)).toBe('Rp 0');
  });
});

describe('getPeriodDateRange', () => {
  // Test with fixed date (mock getTodayInJakarta)
  it('realtime returns today-today', () => {
    const range = getPeriodDateRange('realtime', '2026-06-22');
    expect(range).toEqual({ start: '2026-06-22', end: '2026-06-22' });
  });

  it('7days returns correct range', () => {
    const range = getPeriodDateRange('7days', '2026-06-22');
    expect(range.start).toBe('2026-06-15');
    expect(range.end).toBe('2026-06-21');
  });
});
```

**Smoke test manual (KRITIS — testing semua periode):**
```
□ Real-time → chart per jam dari 00:00 sampai sekarang
□ Yesterday → chart per jam kemarin
□ 7 Days → chart per hari 7 hari terakhir
□ 30 Days → chart per hari 30 hari terakhir
□ Daily (pilih tanggal) → chart per jam tanggal tertentu
□ Weekly (pilih minggu) → chart per hari seminggu
□ Monthly (pilih bulan) → chart per hari sebulan
□ Yearly (pilih tahun) → chart per bulan setahun
□ Setiap periode → 5 KPI cards muncul dengan data benar
□ Best/worst period → badge muncul
□ Data table → sortable, total row benar
□ Export Excel → file download dengan data sesuai periode
□ Export PDF → file download dengan chart + tabel
□ Partial month → projected revenue tampil
□ Period comparison → "vs Previous" label benar
□ Dropdown → open/close, pilih periode
□ Calendar → pilih tanggal di masa lalu hanya
```

---

### Phase 8: Dashboard Module

Home.svelte ~300 baris. Ekstrak service + store.

**New tests:**
```
modules/dashboard/services/__tests__/dashboard-service.test.ts
modules/dashboard/stores/__tests__/dashboard-store.test.ts
```

**Smoke test:**
```
□ Dashboard load → 4 KPI cards muncul
□ Real-time update (30s polling) → data berubah
□ WebSocket sale_created → todaysRevenue/todaysSales update
□ Connection status → Live/Offline indicator
□ Quick Access modules → link navigasi benar
```

---

### Phase 9: Admin Module

Halaman admin relatif kecil (200-400 baris per halaman).

**New tests:**
```
modules/admin/services/__tests__/admin-service.test.ts
```

**Smoke test:**
```
□ User list → muncul
□ Add user → tersimpan
□ Edit user → data berubah
□ Delete user → hilang
□ Role list → muncul
□ Add role → tersimpan
□ Edit role permissions → berubah
□ Audit log → list muncul
□ Audit log filter → terfilter
```

---

### Phase 10: Settings Module

Hanya service, tidak ada komponen baru yang signifikan.

**Smoke test:**
```
□ Category CRUD → berfungsi (dari halaman admin/categories)
```

---

### Phase 11: App Layer Refactor

**Yang berubah:**
- `App.svelte` dari 254 baris → ~50 baris
- Router → `app/router/`
- Auth init → `app/providers/auth-init.ts`
- WS init → `app/providers/websocket.ts`
- Layout → `app/layouts/`

**New tests:**
```
app/router/__tests__/routes.test.ts
app/providers/__tests__/auth-init.test.ts
```

**Smoke test manual:**
```
□ Setiap route → halaman sesuai
  □ /login → LoginPage
  □ / → Dashboard
  □ /pos → POS
  □ /inventory/products → Products
  □ /reports → Reports
  □ /transactions → Transactions
  □ /customers → Customers
  □ /admin/users → Users
  □ /admin/roles → Roles
  □ /admin/audit-logs → Audit Logs
  □ /admin/categories → Categories
□ Redirect:
  □ /inventory → /inventory/products
  □ /inventory/stock → /inventory/products
  □ /admin → /admin/users
□ Guard:
  □ Without login → semua redirect ke /login
  □ After login → landing di /
□ Loading splash → muncul saat init
□ Thermal receipt → muncul setelah checkout
□ Toast notification → muncul + auto-hide
□ Responsive layout → sidebar/topbar rapi
```

---

### Phase 12: Shared UI Consolidation

**Hanya pindah folder + barrel export.**

**Smoke test:**
```
□ Build → npm run build sukses
□ Setiap halaman render → tidak ada broken import
□ UI components → Button, Modal, Input, dll masih berfungsi
```

---

## Test Migration Summary Table

| Existing Test File | Tests | Phase Breaks | Migrate To |
|-------------------|-------|-------------|------------|
| `jakartaTime.test.ts` | 15 ✅ | Phase 0 | Pindah ke `shared/utils/jakartaTime.test.ts` |
| `useWebSocket.test.ts` | 3 ✅ | Phase 0 | Pindah ke `shared/api/websocket.test.ts` |
| `Sidebar.svelte.test.ts` | 24 ✅ | Phase 11 | Pindah ke `app/layouts/Sidebar.test.ts` |
| `Home.svelte.test.ts` | 10 ✅ | Phase 8 | HAPUS + ganti `modules/dashboard/` service tests |
| `PosPage.svelte.test.ts` | 46 ✅ | Phase 6 | HAPUS + ganti `modules/pos/` service/store/utils tests |
| `ProductsPage.svelte.test.ts` | 78 ✅ | Phase 2 | HAPUS + ganti `modules/product/` service/store/utils tests |
| `ReportsPage.svelte.test.ts` | 31 ✅/8 ❌ | Phase 7 | HAPUS + ganti `modules/reporting/` service/store/utils tests |
| `Users.svelte.test.ts` | 19 ✅ | Phase 9 | HAPUS + ganti `modules/admin/` service tests |
| **Total existing** | **226 ✅/8 ❌** | | → **~100 NEW unit tests** (service, store, utils) |

---

## Target Test Count After Refactoring

| Module | New Unit Tests | Description |
|--------|---------------|-------------|
| `shared/utils/jakartaTime` | 15 (existing) | Pure date utils |
| `shared/api/websocket` | 3 (existing, updated) | WebSocket |
| `modules/auth/` | 8-10 | Login, session, token |
| `modules/product/` | 15-20 | CRUD, filter, status, validation |
| `modules/inventory/` | 3-5 | Stock adjust |
| `modules/customers/` | 5-8 | Customer CRUD |
| `modules/sales/` | 8-10 | Sales queries, export |
| `modules/pos/` | 15-20 | **Cart math**, checkout |
| `modules/reporting/` | 10-15 | **Period utils**, formatting |
| `modules/dashboard/` | 3-5 | Live stats |
| `modules/admin/` | 8-10 | Users, roles |
| `app/router/` | 3-5 | Route config |
| `app/layouts/` | 24 (existing) | Sidebar |
| **Target total** | **~120-150** | |

---

## Test Execution Per Phase Checklist

```
Fase 0:
  □ npm test → 226 passed (sama dengan baseline)
  □ Tidak boleh ada test baru gagal

Fase 1:
  □ npm test → min 234 passed (+8 auth tests baru)
  □ Login manual → sukses

Fase 2:
  □ npm test → min 15 product tests baru passing
  □ ProductsPage.svelte.test.ts masih passing (78 tests)
    → karena kita belum hapus file asli
  □ Manual test semua fitur produk

Fase 3-5:
  □ npm test → tidak ada regression
  □ Manual test masing-masing modul

Fase 6:
  □ npm test → min 15 pos tests baru (cart-utils)
  □ Manual test FULL flow POS:
    add → cart → checkout → receipt
  □ Test dengan Cash, Card, E-Wallet

Fase 7:
  □ npm test → min 10 reporting tests baru
  □ Manual test SEMUA periode
  □ Bandingkan angka chart dengan real data backend

Fase 8-10:
  □ npm test → tidak ada regression
  □ Manual test masing-masing admin

Fase 11:
  □ npm test → app router tests baru
  □ Manual test SEMUA route + guard

Fase 12:
  □ npm run build → sukses
  □ npm test → semua passing
```

---

## Cara Menjalankan Test

```bash
# Semua test
npm test

# Satu file test
npx vitest run src/modules/pos/lib/__tests__/cart-utils.test.ts

# Watch mode (development)
npx vitest

# Coverage
npx vitest run --coverage
```

---

## Catatan Penting

1. **Jangan hapus existing test file sampai modul baru sudah stabil.**
   - File lama masih bisa dijadikan referensi fitur apa saja yang perlu ditest
   - Setelah service/store test baru dibuat dan passing, hapus structure guard test

2. **Source-structure guard test bukan "salah"** — hanya tidak cocok untuk refactoring.
   Fungsinya adalah regression guard untuk developer tunggal. Setelah modular monolith, struktur file lebih stabil sehingga structure guard lebih jarang perlu diubah.

3. **Prioritas test baru: cart-utils → period-utils → product-service → auth-service.**
   Urut berdasarkan jumlah logic processing (cart math paling banyak kalkulasi, paling riskan regression).

4. **Manual test tetap penting** — terutama untuk UI behavior yang susah di-test tanpa browser (animasi, transisi, layout responsif).
