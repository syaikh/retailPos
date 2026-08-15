# Audit & Fix: Dropdown/Kebab Menu Clipping Bug

## Root Cause

Semua dropdown yang menggunakan `position: absolute` di-clip oleh ancestor dengan `overflow: hidden`, `overflow: auto`, atau `overflow: scroll`. Chain clipping pada Purchase Orders table:

```
PurchaseOrdersPage.svelte:104  →  <div class="card overflow-hidden">       // clipping
PurchaseOrdersTable.svelte:101 →  <div class="overflow-x-auto">            // clipping
  <td>
    <div class="relative inline-block">                                     // positioning anchor
      <div class="absolute z-50 ...">                                      // menu ter-clip
```

`overflow: hidden` dan `overflow: auto` membuat **Block Formatting Context** (BFC) yang memotong semua child content yang overflow, termasuk `position: absolute` children.

## Solusi

**Ganti `position: absolute` dengan `position: fixed` + computed coordinates via `getBoundingClientRect()`.**

`position: fixed` memposisikan elemen relatif terhadap **viewport**, bukan terhadap parent container. Ini secara fundamental lolos dari clipping oleh `overflow: hidden`/`auto` ancestor mana pun selama tidak ada `transform`, `filter`, atau `perspective` pada ancestor (yang mengubah containing block untuk fixed positioning).

Pattern yang sama sudah dipakai oleh `Tooltip.svelte` di project yang sama.

### Keuntungan
- Tidak ada perubahan API, behavior, atau interface component
- Semua komponen yang menggunakan component yang di-fix langsung mendapat benefit
- Tidak perlu menaikkan z-index secara acak
- Layout tetap stabil — `position: fixed` tidak mempengaruhi flow dokumen
- Scroll/resize listener memastikan posisi tetap akurat

## Files Changed

### 1. `web/src/shared/ui/Dropdown.svelte`
- **Hapus** `placementClasses` (Tailwind placement statis)
- **Hapus** `absolute` + placement classes dari menu `<div>`
- **Tambah** `menuStyle` state, `computePosition()` menggunakan `getBoundingClientRect()`
- **Ganti** dengan `class="fixed"` + `style={menuStyle}`
- **Tambah** scroll/resize listener (passive) saat open
- **Pattern:** `position: fixed` dengan computed top/left/right/bottom viewport-relative

### 2. `web/src/shared/ui/SelectSearch.svelte`
- **Hapus** inline `position: absolute; top: calc(100% + 6px); left: 0; width: 100%;`
- **Tambah** `menuStyle` state, `computePosition()` menggunakan `getBoundingClientRect()`
- **Ganti** dengan `class="fixed"` + `style={menuStyle}`
- **Tambah** scroll/resize listener (passive) saat open
- **Pattern:** Sama dengan Dropdown — `position: fixed` untuk menghindari clipping oleh Modal body `overflow-y-auto`

### 3. `web/src/modules/product/components/ProductActionsDropdown.svelte`
- **Hapus** `absolute right-0 top-full mt-1` dari dropdown `<div>`
- **Tambah** `menuStyle` state, `computePosition()` menggunakan `buttonRef.getBoundingClientRect()`
- **Ganti** dengan `class="fixed z-50 w-48"` + `style={menuStyle}`
- **Tambah** scroll/resize listener (passive) via `$effect()`
- **Pattern:** Sama dengan shared Dropdown. Sebelumnya standalone custom implementation dengan `position: absolute` di dalam card `overflow-hidden`.

### 4. `web/src/modules/product/components/ProductFormModal.svelte`
- **Tambah** `categoryContainer: HTMLDivElement` + `categoryMenuStyle = $state('')`
- **Tambah** `computeCategoryPosition()` function
- **Tambah** `bind:this={categoryContainer}` ke container `<div class="relative">`
- **Ganti** `class="absolute top-full mt-2 w-full"` dengan `style={categoryMenuStyle}` + `class="fixed"`
- **Tambah** scroll/resize effect

### 5. `web/src/modules/pricing/components/PriceSimulationModal.svelte`
- **Tambah** `productSearchContainer: HTMLDivElement` + `productMenuStyle = $state('')`
- **Tambah** `computeProductSearchPosition()` function
- **Tambah** `bind:this={productSearchContainer}` ke container `<div class="relative">`
- **Ganti** `class="absolute z-20 top-full mt-1 w-full"` dengan `style={productMenuStyle}` + `class="fixed"`
- **Tambah** scroll/resize effect

### New File: `tests/e2e/purchase-orders-dropdown.spec.ts`
- Test 1: Klik kebab menu, verifikasi dropdown muncul penuh, semua item visible dan clickable, bounding box dalam viewport
- Test 2: Verifikasi z-stacking — dropdown di atas container, tutup dengan Escape

## Audit Summary: Semua Dropdown/Kebab/Popover di Aplikasi

| # | Komponen | Positioning | Ancestor Overflow | Clipping Risk | Status |
|---|---|---|---|---|---|
| 1 | **Dropdown** (shared) | `fixed` | `overflow-hidden`, `overflow-x-auto` | **None** | ✅ Fixed |
| 2 | **SelectSearch** (shared) | `fixed` | Modal body `overflow-y-auto` | **None** | ✅ Fixed |
| 3 | **SuppliersTable kebab** | `fixed` (via shared Dropdown) | `overflow-x-auto` | **None** | ✅ Aman |
| 4 | **PricingRulesTable kebab** | `fixed` (via shared Dropdown) | `overflow-x-auto` | **None** | ✅ Aman |
| 5 | **CustomerGroupsTable kebab** | `fixed` (via shared Dropdown) | `overflow-x-auto` | **None** | ✅ Aman |
| 6 | **RolesPage kebab** | `fixed` (via shared Dropdown) | `overflow-hidden`, `overflow-x-auto` | **None** | ✅ Aman |
| 7 | **ProductActionsDropdown** | `fixed` | `overflow-hidden`, `overflow-x-auto` | **None** | ✅ Fixed |
| 8 | **ProductFormModal category** | `fixed` | Modal body `overflow-y-auto` | **None** | ✅ Fixed |
| 9 | **PriceSimulationModal product search** | `fixed` | Modal body `overflow-y-auto` | **None** | ✅ Fixed |
| 10 | **NotificationBell** | `absolute` | None (navbar, no overflow ancestor) | **Low** | ⏸️ Lewat |
| 11 | **BulkActionDropdown** | `fixed` (via shared Dropdown) | Toolbar areas | **None** | ✅ Aman |
| 12 | **ShiftsPage filters** | `fixed` (via shared Dropdown) | None | **None** | ✅ Aman |
| 13 | **Date pickers (toolbar)** | `absolute` | Toolbar areas | **Low** | ⏸️ Lewat |

### Notes

- **NotificationBell** (`app/layouts/NotificationBell.svelte:139`) menggunakan `position: absolute` tapi parent nya adalah page-level header tanpa overflow ancestor. Risiko clipping rendah.
- **Date pickers** (`PurchaseOrdersToolbar`, `TransactionFilters`, `AuditLogsFilterToolbar`, `PeriodSelector`) menggunakan `position: absolute` tapi berada di toolbar (tidak di dalam overflow container). Risiko clipping rendah.
- Semua komponen toolbar filter sudah menggunakan shared `Dropdown` (`position: fixed`) dan aman.

## Dampak ke Komponen Lain

Tidak ada. Semua perubahan bersifat internal — hanya mengganti CSS positioning dari `absolute` ke `fixed` dengan computed coordinates. API component, props, dan behavior tidak berubah.

## Testing

```
21 passed (51.2s)
  ✓ 2 dropdown visibility tests (new)
  ✓ 18 API flow tests
  ✓ 1 UI flow test
```
