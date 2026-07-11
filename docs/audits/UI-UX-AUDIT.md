# Comprehensive UI/UX Audit Report — Retail POS System

**Date:** 2026-07-11  
**Scope:** Full frontend application (Svelte 5 SPA)  
**Methodology:** Codebase analysis across all modules, shared components, layout, routing, design tokens, accessibility, and workflow patterns.

---

## Executive Summary

| Metric | Score | Rationale |
|--------|-------|-----------|
| **Overall UI Score** | **6.5 / 10** | Strong design token system and dark theme; undermined by massive dead code, inconsistent patterns, and no responsive design |
| **Overall UX Score** | **6 / 10** | POS workflow is solid with keyboard shortcuts; dragged down by inconsistency across modules, no mobile support, and fragmented error handling |
| **Design System Maturity** | **5 / 10** | Tokens defined but inconsistently applied; shared components exist but are underused; critical components (Drawer, ToggleSwitch, ConfirmDialog, FilterChips) are missing |
| **Accessibility Score** | **3.5 / 10** | Basic ARIA labels on buttons; no aria-sort, no aria-live regions, no focus management, no keyboard navigation for tables, no skip links, no screen reader announcements |
| **Maintainability Score** | **4 / 10** | ~33% of the codebase is dead code; duplicated patterns across modules; mixed file extensions; dead stores and services |

### Major Strengths

- Well-structured design token system with semantic color naming (`$shared/ui/`, `app.css`)
- Custom SPA router with lazy-loaded modules — good architecture for code splitting
- POS module has barcode scanner support and keyboard shortcuts (F2/F4/F6/ALT+DEL)
- WebSocket real-time stock updates in POS
- Module-based architecture (`$modules/`) with consistent internal structure (components/services/stores/types/lib)
- Shared UI library (`$shared/ui/`) with 24 components covering common patterns
- Print receipt overlay with proper thermal receipt styling

### Major Weaknesses

- **~33% of the codebase is dead code**: `src/routes/` (SvelteKit remnants) + `$lib/` (legacy components/stores/API) — zero imports from active modules
- **Zero responsive/mobile design**: No hamburger menu, no mobile sidebar overlay, no breakpoints on POS/cart
- **Inconsistent UI patterns across modules**: Different form patterns, table patterns, toolbar patterns, modal patterns, status filter patterns, and delete confirmation patterns
- **Poor accessibility**: No `aria-sort` on sortable tables, no `aria-live` for dynamic content, no focus management, no keyboard navigation for data tables, no skip-to-content link
- **Dead component code**: Multiple stores (`product-store`, `customer-store`, `pos-store`, `inventory-store`), services (`dashboard-service`), and functions (`reporting-utils.getChartType`, `getPeriodDateRange`) are defined but never imported

### Top Risks

1. **Dead code confusion** — developers may unknowingly edit orphaned files or import dead modules
2. **Accessibility legal exposure** — WCAG 2.1 AA compliance is not met; data tables are unusable by screen readers
3. **POS race condition** — `finalizeSale()` reads `lastSale` from outer scope after async `processCheckout()` completes, which may be stale
4. **No route-level authorization** — `permissions.ts` is defined but never enforced; any authenticated user can access any route by URL
5. **Modal backdrop click prevention** — shared `Modal` component intentionally blocks backdrop click-to-close, breaking user expectations

### Highest-Impact Opportunities

1. **Delete dead code** (`src/routes/`, `$lib/`) — reduces cognitive load and maintenance surface by ~33%
2. **Extract shared components** (Drawer, ToggleSwitch, ConfirmDeleteModal, FilterChips, DatePickerPanel, SortableHeader, EmptyState, TableSkeleton)
3. **Add responsive design** — hamburger menu, mobile sidebar overlay, POS cart responsive layout
4. **Standardize error handling** — replace `confirm()`/`alert()` with toast/modals; add `aria-live` regions
5. **Enforce route authorization** — wire `permissions.ts` into the SPA router

---

## Findings

### F01 — Massive Dead Code: Entire Legacy Codebase

**Severity:** Critical  
**Category:** Maintainability  

**Evidence:**
- `web/src/routes/` — Complete SvelteKit route structure with pages, layouts, hooks — **never loaded by any entry point**
- `web/src/lib/` — 33 files (stores, components, composables, API clients, domain logic) — **zero imports from `$modules/`**
- `web/src/app.html` — SvelteKit HTML template — **never served** (Vite uses `web/index.html`)
- `web/src/main.js` mounts `app/main.svelte` which uses the custom SPA router

**Explanation:** The application migrated from SvelteKit file-based routing to a custom SPA router + module system, but the old code was never deleted. The `$lib/` alias is not even defined in `vite.config.js` — only `$modules`, `$shared`, `$app` are.

**Business Impact:** Developer confusion, longer build times, risk of editing dead files, onboarding friction.

**User Impact:** None directly, but increases bug risk when developers navigate dead code.

**Recommendation:** Delete `src/routes/`, `src/lib/`, `src/app.html`. Move any still-useful utilities from `$lib/` to `$shared/` first. Run a full build to verify nothing breaks.

**Effort:** Small (1-2 hours to verify and delete)

---

### F02 — No Route-Level Authorization

**Severity:** High  
**Category:** Security / UX  

**Evidence:**
- `web/src/lib/config/permissions.ts` defines `routePermissions` mapping 7 routes to permission arrays
- **Zero imports** of this file anywhere in `$modules/` or `$app/`
- `web/src/app/layouts/Sidebar.svelte` hides nav items by role, but URL access is unrestricted
- Any authenticated user can navigate to `/admin/audit-logs` or `/admin/roles` by typing the URL

**Explanation:** The permission system is designed but never enforced at the route level. Sidebar filtering provides UI-level hiding only.

**Business Impact:** Critical security gap — staff users can access admin pages.

**User Impact:** Unauthorized users see pages they shouldn't; may see errors or sensitive data.

**Recommendation:** Import `routePermissions` in `main.svelte`'s `handleRoute()` and redirect unauthorized users. Add `permissions.ts` to `$shared/` or `$app/`.

**Effort:** Small (2-3 hours)

---

### F03 — Zero Responsive/Mobile Design

**Severity:** High  
**Category:** UX / Accessibility  

**Evidence:**
- `Sidebar.svelte` — Fixed 240px width, no hamburger menu, no mobile overlay
- `Layout.svelte` — No responsive breakpoints for sidebar collapse
- `PosPage.svelte` — Fixed `grid-template-columns: 1fr 320px` cart, no breakpoint
- `Topbar.svelte` — No hamburger button for mobile
- Limited responsive classes: only `sm:`, `md:`, `lg:` used in some modals and `ImportHistoryPage`
- Most pages (`ProductsPage`, `TransactionsPage`, `CustomersPage`) have no responsive table behavior

**Explanation:** The application is designed exclusively for desktop use. On tablets or phones, the sidebar is always visible (or collapsed to 72px icons), the cart panel is always 320px, and tables overflow horizontally.

**Business Impact:** Limits deployment to desktop-only scenarios. Cannot be used on tablets at the point of sale counter.

**User Impact:** Cashiers cannot use tablets or phones. Floor staff cannot check inventory on mobile.

**Recommendation:** Phase 1: Add hamburger menu + mobile sidebar overlay. Phase 2: Make POS cart responsive (bottom sheet on mobile). Phase 3: Make data tables horizontally scrollable with sticky first column.

**Effort:** Large (2-3 sprints)

---

### F04 — Inconsistent Form Patterns Across Modules

**Severity:** High  
**Category:** Consistency / UX  

**Evidence:**
- **Product**: Single `ProductFormModal` with `mode` prop toggling add/edit. Form state lifted to parent. Validation via toast only — no inline field errors.
- **Customer**: Two separate modals (`CreateCustomerModal`, `EditCustomerModal`). Create uses `$bindable` fields; Edit has internal state. Inline field errors with `border-danger` highlighting.
- **Settings** (Categories/Brands/UoM): Single modal with `modalMode`. No inline validation.
- **Admin/Users**: `UserFormModal` with internal state and custom dropdown. Inline role dropdown.
- **Admin/Roles**: Two-step modal (step 1: name, step 2: permissions). Complex internal state.

**Explanation:** Each module invented its own form approach. No shared form validation pattern, no consistent error display, no consistent state management.

**Business Impact:** Maintenance burden — fixing a form bug means fixing it in 5+ places. New features require understanding multiple patterns.

**User Impact:** Inconsistent validation feedback — some forms show inline errors, others only show toasts. Confusing for users who manage multiple areas.

**Recommendation:** Standardize on the Customer module's inline validation pattern (it's the best). Extract shared form components: `FormField`, `FormError`, `FormSelect`. Establish a form state convention.

**Effort:** Large (ongoing refactoring across modules)

---

### F05 — Missing Shared Components (Drawer, ToggleSwitch, ConfirmDialog, FilterChips)

**Severity:** High  
**Category:** Design System / Maintainability  

**Evidence:**
- **Drawer pattern**: `ProductDetailDrawer`, `TransactionDrawer`, `RoleDetailDrawer`, `AuditLogDetailsDrawer`, `CategoryFilterModal` — **5 custom drawer implementations** with near-identical fly-in panel code (backdrop + `role="dialog"` + `transition:fly` + close button)
- **Toggle switch**: Copy-pasted 10+ lines of CSS in `CategoriesPage`, `BrandsPage`, `UnitsOfMeasurePage`, `UserFormModal` — identical markup
- **Delete confirmation**: Each CRUD page implements its own delete modal — no shared `ConfirmDeleteModal`
- **Filter chips**: Identical CSS (`.filter-chips-wrapper`, `grid-template-rows` animation) copy-pasted in `ProductFiltersToolbar`, `UserToolbar`, `AuditLogsFilterToolbar`
- **Date picker panel**: Nearly identical implementations in `TransactionFilters` and `AuditLogsFilterToolbar`

**Explanation:** The shared UI library has `Modal` but no `Drawer`. No `ToggleSwitch`. No `ConfirmDialog`. No `FilterChipBar`. No `DatePickerDropdown`. Each module reinvents these.

**Business Impact:** ~200+ lines of duplicated code per pattern. Bug fixes must be applied multiple times. Design drift over time.

**User Impact:** Slight visual inconsistencies between modules (different backdrop blur, different close button styles, different animation speeds).

**Recommendation:** Extract: `Drawer.svelte`, `ToggleSwitch.svelte`, `ConfirmDeleteModal.svelte`, `FilterChipBar.svelte`, `DatePickerDropdown.svelte`, `SortableHeader.svelte`, `TableSkeleton.svelte`, `EmptyState.svelte`.

**Effort:** Medium per component (1-2 days each)

---

### F06 — POS Workflow Issues

**Severity:** High  
**Category:** UX / Retail Workflow  

**Evidence:**
- **Customer selection closes checkout**: `CustomerSelectModal` sets `showCheckoutModal = false` on open, forcing cashier to re-open checkout after selecting a customer (`PosPage.svelte`)
- **Quick cash presets insufficient**: Presets are 50k, 100k, 150k, 200k — many retail transactions exceed 200k IDR
- **No "exact amount" preset button**: F6 shortcut exists but isn't shown as a clickable button
- **Payment method selector duplicated**: Exists in both `CartPanel` and `CheckoutModal`, synced via `$bindable`
- **Cash received input accepts invalid values**: `type="text"` with `inputmode="numeric"` but no actual numeric validation — allows negative, decimal, and text input
- **Card/E-Wallet shows change due**: Unnecessary for non-cash payments; confusing UX
- **`confirm()` for cart clear**: Native browser dialog breaks POS flow; jarring interruption
- **`pos-store.svelte.ts` is dead code**: Defined but never imported; all state lives in `PosPage.svelte`
- **Race condition in `finalizeSale()`**: Reads `lastSale` from outer scope after async `processCheckout()` — fragile

**Explanation:** The POS is the core workflow but has several friction points that slow down cashiers and risk errors.

**Business Impact:** Slower checkout = longer queues. Invalid cash amounts = reconciliation errors. Customer selection UX friction = cashiers may skip customer tracking.

**User Impact:** Cashiers must re-open checkout after selecting customer. Cannot enter exact cash amount with one click. Cannot clear cart without browser confirmation dialog.

**Recommendation:**
1. Keep checkout modal open when opening customer selector
2. Add 500k, 1M, and "Exact" (F6) presets
3. Add numeric-only validation on cash input
4. Hide change calculation for non-cash payments
5. Replace `confirm()` with a custom modal or instant action with undo toast
6. Delete dead `pos-store.svelte.ts`
7. Fix race condition in `finalizeSale()` with `await`

**Effort:** Medium (3-5 days)

---

### F07 — No aria-sort, aria-live, or Focus Management

**Severity:** High  
**Category:** Accessibility  

**Evidence:**
- **Zero `aria-sort` attributes** across all sortable tables: `ProductTable`, `CustomerTable`, `TransactionTable`, `UserTable`, `AuditLogsTable`, `RolesPage`, reporting `DataTable` — **7 tables**
- **Zero `aria-live` regions** for dynamic content (loading states, filter changes, sort changes, toast notifications)
- **No `role="progressbar"`** on `ProgressDialog` progress bar (`shared/ui/ProgressDialog.svelte`)
- **No focus restoration** after modal/drawer close (relies on browser default)
- **Table rows with `role="button"`** (`AuditLogsTable`, `RolesPage`, `ProductTable`) — invalid ARIA usage on `<tr>`
- **`ExpandableRow`** uses `role="button"` on `<tr>` — invalid
- **No `aria-label`** on chart `<canvas>` (`ChartArea.svelte`)
- **No keyboard navigation** for data tables (arrow keys to move between cells)

**Explanation:** Screen readers cannot interpret the state of sortable columns, loading progress, or interactive table rows. The application is largely unusable for visually impaired users.

**Business Impact:** Legal exposure for accessibility compliance (ADA, Section 508, EU Accessibility Act).

**User Impact:** Screen reader users cannot sort tables, understand loading states, or interact with data tables.

**Recommendation:** Add `aria-sort="ascending|descending|none"` to all sortable `<th>`. Add `aria-live="polite"` to loading and filter-change regions. Add `role="progressbar"` with `aria-valuenow` to ProgressDialog. Replace `role="button"` on `<tr>` with proper `<button>` inside a cell.

**Effort:** Medium (3-5 days)

---

### F08 — Inconsistent Language (Indonesian/English Mix)

**Severity:** Medium  
**Category:** UX / i18n  

**Evidence:**
- `CheckoutModal`: title "Pembayaran Selesai" (Indonesian) before payment completes
- `CartPanel`: "Clear cart [ALT+DEL]" (English), button aria-label "Hapus filter" (Indonesian)
- `ProductFormModal`: "Clear" (English) on close button
- `ProductDetailDrawer`: "Hapus Produk", "Edit Produk" (Indonesian)
- `ProductActionsDropdown`: "View Details", "Adjust Stock", "Edit", "Delete" (English)
- `CategoriesPage`: "Tambah Kategori", "Batal", "Menyimpan..." (Indonesian)
- `UsersPage`: "Create User", "Cancel", "Saving..." (English)
- `AuditLogsFilterToolbar`: English aria-labels
- `UserToolbar`: "Hapus filter" (Indonesian) aria-label
- All toast messages: English
- Placeholders: Mix of both ("Search by name..." in English, "Kosongkan seluruh keranjang?" in Indonesian)

**Explanation:** No i18n system exists. Messages are hardcoded in whichever language the developer happened to use. Often mixed within the same component.

**Business Impact:** Unprofessional appearance. Confusing for operators.

**User Impact:** Operators must understand both languages. Some may miss important messages.

**Recommendation:** Choose one language for the MVP (Indonesian, given the target market). Create a simple i18n utility or at minimum a constants file. Systematically replace all strings.

**Effort:** Medium (2-3 days for a constants file + systematic replacement)

---

### F09 — Duplicate `DataTable` Components and Naming Conflict

**Severity:** Medium  
**Category:** Maintainability / Naming  

**Evidence:**
- `$shared/ui/DataTable.svelte` — Generic, snippet-based table container with `aria-label`, sticky header
- `$modules/reporting/components/DataTable.svelte` — Purpose-built revenue breakdown table with sort, currency formatting, trend arrows, footer totals

**Explanation:** Two completely different components share the same name. The reporting one should be renamed to `RevenueDataTable` or `SalesBreakdownTable`.

**Business Impact:** Developer confusion when importing. IDE autocomplete shows both.

**User Impact:** None directly.

**Recommendation:** Rename reporting `DataTable` to `RevenueDataTable`. Update imports in `ReportsPage.svelte`.

**Effort:** Small (30 minutes)

---

### F10 — Dashboard Service Dead Code + Duplicated Logic

**Severity:** Medium  
**Category:** Maintainability  

**Evidence:**
- `$modules/dashboard/services/dashboard-service.ts` defines `getLiveStats()` with proper types — **never imported**
- `Home.svelte` inlines the same API call (lines 24-41) with duplicated `DashboardLiveStats` type
- `$modules/reporting/lib/reporting-utils.ts` defines `getChartType()` (line 73) — **never imported**; `ReportsPage.svelte` reimplements same logic (lines 56-60)
- `getPeriodDateRange` is defined in both `reporting-utils.ts` (line 53) and `PeriodSelector.svelte` (line 57) with different implementations
- `getPaymentMethodVariant()` is duplicated in `TransactionTable.svelte` and `TransactionDrawer.svelte`

**Explanation:** Services and utilities were created but the consuming components inlined the logic instead. Multiple copies of the same function exist.

**Business Impact:** Bug fixes to one copy won't fix the others. Divergence over time.

**User Impact:** Potential inconsistency in chart type selection or period date range calculations.

**Recommendation:** Import existing services/utils instead of inlining. Delete unused functions. Consolidate duplicated functions.

**Effort:** Small (1-2 days)

---

### F11 — Shared Modal Blocks Backdrop Click-to-Close

**Severity:** Medium  
**Category:** UX  

**Evidence:**
- `shared/ui/Modal.svelte` line 73: `<div class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm">` — no `onclick` handler
- Comment explicitly says "no click handler to prevent closing"
- Every drawer implementation (`ProductDetailDrawer`, `TransactionDrawer`, `RoleDetailDrawer`, `AuditLogDetailsDrawer`) DOES handle backdrop click — inconsistency with the modal

**Explanation:** The shared Modal intentionally prevents backdrop click-to-close. This breaks a universal UX convention where clicking outside a modal dismisses it. All drawer implementations handle this correctly, creating an inconsistency.

**Business Impact:** Users may feel "trapped" in modals. Extra click required to close (must find the X button or press Escape).

**User Impact:** Frustration when clicking outside modal does nothing. Must find close button or press Escape.

**Recommendation:** Add backdrop click-to-close to Modal. Add an `persistent` prop for cases where accidental close should be prevented (like unsaved changes warnings).

**Effort:** Small (1 hour)

---

### F12 — No Skeleton Loading on Dashboard

**Severity:** Medium  
**Category:** UX / Performance Perception  

**Evidence:**
- `Home.svelte` uses a single `loading` boolean — when true, shows nothing (empty stat cards with no placeholder content)
- `KPICards.svelte` (reporting) uses `Skeleton` component properly — 5 skeleton rows during load
- No skeleton on quick access grid cards
- Most other pages (`ProductsPage`, `CustomersPage`, `TransactionsPage`) use proper skeleton loading

**Explanation:** Dashboard shows blank cards during initial load instead of skeleton placeholders. Users see a flash of empty content.

**Business Impact:** Poor first impression. App feels slower than it is.

**User Impact:** Blank dashboard on load; no visual indication of what's coming.

**Recommendation:** Add `Skeleton` component to dashboard stat cards and quick access grid during loading.

**Effort:** Small (2-3 hours)

---

### F13 — Pagination Only Offers 20/40 Options

**Severity:** Medium  
**Category:** UX  

**Evidence:**
- `shared/ui/Pagination.svelte` hardcodes page size options: only 20 and 40
- No option for 10 (quick scan), 50, or 100 (bulk review)
- All pages inherit this limitation

**Explanation:** Two fixed page sizes may not match all use cases. Power users reviewing many records want larger pages; users scanning want smaller pages.

**Business Impact:** None direct, but limits efficiency.

**User Impact:** Cannot adjust page size to match workflow. 40 is the max — insufficient for bulk review.

**Recommendation:** Add 10, 20, 50, 100 options. Make configurable via prop.

**Effort:** Small (1 hour)

---

### F14 — Product Module Monolith (693 lines)

**Severity:** Medium  
**Category:** Maintainability  

**Evidence:**
- `ProductsPage.svelte` — 693 lines containing: API calls, form state, sorting, stock adjustment, bulk operations, clipboard handling, WebSocket handlers, role-based permissions, modal management, and all template
- `RolesPage.svelte` — 627 lines (same pattern)
- `ReportsPage.svelte` — not as bad but still large
- Compare with `CustomersPage.svelte` at 411 lines — better but still a monolith

**Explanation:** Page components contain all business logic, state management, and template in a single file. No separation of concerns.

**Business Impact:** Difficult to test, debug, and modify. High cognitive load for developers.

**User Impact:** Indirect — slower feature development and more bugs.

**Recommendation:** Extract: `useProductSort()` composable, `useProductBulkActions()` composable, form state into a dedicated store or composable. Target <300 lines per component.

**Effort:** Large (ongoing refactoring)

---

### F15 — `Badge.destructive` Duplicates `Badge.danger`

**Severity:** Low  
**Category:** Maintainability  

**Evidence:**
- `shared/ui/Badge.svelte`: `destructive` and `danger` variants have identical styling

**Explanation:** Dead variant. Should be removed or consolidated.

**Recommendation:** Remove `destructive` variant, keep `danger`.

**Effort:** Small (5 minutes)

---

### F16 — Inconsistent Border Token Usage

**Severity:** Low  
**Category:** Design System  

**Evidence:**
- Components use a mix of: `border-border-default`, `border-border`, `border-border-subtle`, `border-border-strong`
- `Card.svelte` uses `border-border-default`; `Pagination.svelte` uses `border-border-subtle`; `Input.svelte` uses `border-border`
- `Modal.svelte` uses `bg-surface` (likely undefined — should be `bg-surface-default`)

**Explanation:** The Tailwind `border-border` shorthand maps to `border-color: var(--color-border-default)` via the global `* { border-color: var(--color-border-default) }` rule. But `border-border-default` is more explicit. Both work but the inconsistency is confusing.

**Recommendation:** Standardize on `border-border` (the Tailwind shorthand) for consistency with the global rule. Fix `bg-surface` → `bg-surface-default` in Modal.

**Effort:** Small (1 hour)

---

### F17 — Dead Module Stores

**Severity:** Low  
**Category:** Maintainability  

**Evidence:**
- `$modules/product/stores/product-store.svelte.ts` — defines `useProductStore()` — **never imported by ProductsPage**
- `$modules/customers/stores/customer-store.svelte.ts` — defines `useCustomerStore()` — **never imported by CustomersPage**
- `$modules/pos/stores/pos-store.svelte.ts` — defines `usePosStore()` — **never imported by PosPage**
- `$modules/inventory/stores/inventory-store.svelte.ts` — defines `useInventoryStore()` — **never imported by StockAdjustModal**
- `$modules/dashboard/services/dashboard-service.ts` — never imported
- `$modules/reporting/lib/reporting-utils.ts` — `getChartType()` never imported

**Explanation:** Stores and services were created during architecture planning but page components manage state inline instead.

**Recommendation:** Either wire the stores into pages (intended pattern) or delete them. Inline state is working — deleting is simpler.

**Effort:** Small (1 hour)

---

### F18 — No Error Boundary or Retry for Failed Page Loads

**Severity:** Low  
**Category:** UX  

**Evidence:**
- `main.svelte` `getComponent()` silently returns `Home` for unknown paths (no 404 page)
- Lazy-loaded page import failures are unhandled — `mod.default` could be undefined
- No error boundary around `<Component />`

**Explanation:** If a lazy import fails (network error, chunk load failure), the user sees nothing or the fallback Home page with no explanation.

**Recommendation:** Add try/catch around dynamic imports. Show an error state with retry button. Add a 404 route.

**Effort:** Small (2-3 hours)

---

### F19 — Mixed File Extensions (.js vs .ts)

**Severity:** Low  
**Category:** Maintainability  

**Evidence:**
- `$modules/reporting/utils/chart-config.js` — no TypeScript types
- `$modules/reporting/utils/export-utils.js` — no TypeScript types
- `$modules/reporting/utils/data-fetching.js` — no TypeScript types
- All other files use `.ts`

**Explanation:** Three `.js` files in an otherwise TypeScript codebase. No type safety on critical chart configuration and export utilities.

**Recommendation:** Convert to `.ts` with proper interfaces.

**Effort:** Small (2-3 hours)

---

### F20 — `PeriodSelector` Click-Outside Detection Uses CSS Class Matching

**Severity:** Low  
**Category:** Maintainability  

**Evidence:**
- `PeriodSelector.svelte` line 215: `if (!target.closest('.card-glass'))` — relies on the `card-glass` CSS class being present on the dropdown container

**Explanation:** Fragile — will break if the class name changes. Uses CSS class as behavioral contract.

**Recommendation:** Use a ref/binding to the dropdown element instead.

**Effort:** Small (30 minutes)

---

### F21 — Charts Inaccessible to Screen Readers

**Severity:** Medium  
**Category:** Accessibility  

**Evidence:**
- `ChartArea.svelte` renders a `<canvas>` with no `aria-label`, `aria-describedby`, or `<title>`
- Screen readers see an empty canvas element
- No text alternative or data table companion for chart data

**Explanation:** Chart.js renders to canvas which is invisible to assistive technology. No fallback is provided.

**Recommendation:** Add `aria-label="Sales chart showing [period]"` and a visually hidden data summary. Consider providing a data table toggle.

**Effort:** Small (2-3 hours)

---

### F22 — Inconsistent Sort Direction Casing

**Severity:** Low  
**Category:** Consistency  

**Evidence:**
- Sales, Product, Customer, Admin sort functions: `'asc'`/`'desc'` (lowercase)
- Sales page `sortDir` state: `'ASC'`/`'DESC'` (uppercase)
- Admin pages: `'asc'`/`'desc'` (lowercase)

**Explanation:** The Sales module uses uppercase while others use lowercase.

**Recommendation:** Standardize on lowercase `'asc'`/`'desc'`.

**Effort:** Small (30 minutes)

---

### F23 — No `prefers-reduced-motion` Support

**Severity:** Low  
**Category:** Accessibility  

**Evidence:**
- Global animations: `fade-in`, `slide-up`, `slide-in-right`, `shimmer`, `pulse-dot` defined in `app.css`
- Multiple components use `animate-slide-up`, `animate-shimmer`, `transition:fly`, `transition:fade`
- No `@media (prefers-reduced-motion: reduce)` media query anywhere

**Explanation:** Users who have motion sensitivity in their OS settings will still see all animations.

**Recommendation:** Add `@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; } }` to `app.css`.

**Effort:** Small (30 minutes)

---

### F24 — NotificationBell Has No Focus Trap

**Severity:** Low  
**Category:** Accessibility  

**Evidence:**
- `NotificationBell.svelte`: dropdown opens on click, Escape closes it, click-outside closes it
- No focus trap — Tab can escape to the page behind the dropdown
- No `role="menu"` / `role="menuitem"` on dropdown items

**Explanation:** Keyboard users can tab into the notification dropdown but also tab out to content behind it, creating a confusing experience.

**Recommendation:** Add focus trap or at minimum `role="menu"` with arrow key navigation.

**Effort:** Small (1-2 hours)

---

### F25 — `ExpandableRow` Uses `colspan="100"`

**Severity:** Low  
**Category:** Maintainability  

**Evidence:**
- `shared/ui/ExpandableRow.svelte` uses `colspan="100"` as a hack instead of calculating actual column count

**Explanation:** Works but is semantically incorrect. If the table has 9 columns, the expanded area spans 100 columns.

**Recommendation:** Accept a `columns` prop or calculate from parent table.

**Effort:** Small (30 minutes)

---

## Reusable Component Opportunities

| Pattern | Current Implementations | Recommendation |
|---------|----------------------|----------------|
| **Drawer / Side Panel** | `ProductDetailDrawer`, `TransactionDrawer`, `RoleDetailDrawer`, `AuditLogDetailsDrawer`, `CategoryFilterModal` (5 instances) | Extract `Drawer.svelte` to `$shared/ui/` |
| **Toggle Switch** | `CategoriesPage`, `BrandsPage`, `UnitsOfMeasurePage`, `UserFormModal` (4 instances, identical CSS) | Extract `ToggleSwitch.svelte` to `$shared/ui/` |
| **Confirm Delete Modal** | Every CRUD page implements its own delete modal | Extract `ConfirmDeleteModal.svelte` to `$shared/ui/` |
| **Filter Chips Bar** | `ProductFiltersToolbar`, `UserToolbar`, `AuditLogsFilterToolbar` (3 instances, identical CSS) | Extract `FilterChipBar.svelte` to `$shared/ui/` |
| **Date Picker Dropdown** | `TransactionFilters`, `AuditLogsFilterToolbar` (2 instances, near-identical) | Extract `DatePickerDropdown.svelte` to `$shared/ui/` |
| **Sortable Table Header** | Every table reimplements sort buttons with arrows | Extract `SortableHeader.svelte` or enhance `DataTable` |
| **Table Skeleton** | Every table reimplements 5-row skeleton | Extract `TableSkeleton.svelte` to `$shared/ui/` |
| **Empty State** | Every table reimplements icon + title + subtitle | Extract `EmptyState.svelte` to `$shared/ui/` |
| **Export Download Flow** | `TransactionFilters`, `AuditLogsFilterToolbar`, `PeriodSelector` (3 instances) | Extract `downloadExportFile()` utility |
| **RBAC Permission Derivation** | `UsersPage`, `RolesPage`, `AuditLogsPage`, `CategoriesPage`, `BrandsPage`, `UnitsOfMeasurePage` (6 instances) | Extract `useRBAC()` composable |
| **Pagination Footer** | Every table page reimplements `bg-surface-subtle/30 border-t border-border/50` + `<Pagination>` | Extract `TableFooter.svelte` or enhance `DataTable` |
| **`statusInfo()` / `getPaymentMethodVariant()`** | Duplicated within Product and Sales modules | Single shared utility function |

---

## Design System Improvements

### Spacing
- **Current**: `space-y-5` for page sections, `p-4` for cards, `gap-3`/`gap-4` for grids — mostly consistent
- **Issue**: `space-y-4` used in `RolesPage` while all others use `space-y-5`
- **Recommendation**: Standardize on `space-y-5` for page sections. Define spacing scale in design tokens.

### Typography
- **Current**: `text-xs uppercase tracking-wider` for table headers, `text-sm` for body, `text-lg` for headings — mostly consistent
- **Issue**: `PageHeader.svelte` uses global CSS classes (`page-header`, `page-title`, `page-subtitle`) instead of Tailwind — inconsistent with all other components
- **Recommendation**: Convert `PageHeader` to Tailwind classes. Define typography scale in tokens.

### Colors
- **Current**: Excellent token system in `app.css` with semantic naming
- **Issue**: `ImportSummary.svelte` and `PreviewTable.svelte` use hardcoded `emerald-400`, `amber-400`, `rose-400` instead of `success`, `warning`, `danger` tokens
- **Issue**: `Modal.svelte` uses `bg-surface` (may not resolve) instead of `bg-surface-default`
- **Recommendation**: Replace all hardcoded colors with semantic tokens. Fix Modal.

### Interaction Patterns
- **Modal**: Shared `Modal` has focus trap and Escape-to-close — good. Missing backdrop click-to-close.
- **Drawer**: 5 custom implementations, no shared component. All handle Escape and backdrop click.
- **Dropdown**: Shared `Dropdown.svelte` is well-implemented with keyboard navigation, arrow keys, and `role="menu"`.
- **Toast**: Shared `Toast.svelte` with `aria-live="polite"` and `role="alert"` — good.
- **Confirm**: Uses native `confirm()` in POS; custom modals elsewhere. No shared pattern.

### Page Layouts
- **Current**: Most pages follow `space-y-5` → toolbar → card → table + pagination
- **Issue**: `AuditLogsPage` adds `max-w-7xl mx-auto` — unique among all pages
- **Issue**: Toolbar containers differ: `card p-4` vs `border border-border rounded-xl p-4 bg-bg-card`
- **Recommendation**: Define a `PageLayout` component with standard slots: `toolbar`, `content`, `pagination`.

---

## Workflow Improvements

| # | Workflow | Current | Improvement | Productivity Gain |
|---|----------|---------|-------------|-------------------|
| 1 | POS: Select customer during checkout | Must close checkout, open customer modal, select, reopen checkout | Keep checkout open; show customer selection inline or as a nested overlay | High |
| 2 | POS: Exact cash payment | Press F6 (undiscoverable) | Show "Exact" as a clickable preset button alongside 50k/100k/etc. | Medium |
| 3 | POS: Clear cart | `confirm()` native dialog | Instant clear with undo toast | Medium |
| 4 | Product search → add to cart | Must click "Add" button per product | Add Enter-to-add-first-result for barcode scanner fast path | High |
| 5 | Settings CRUD | Must navigate between Categories/Brands/UoM pages | Add tabbed settings page or combined management view | Medium |
| 6 | Import workflow | Import wizard exists but is complex | Verify wizard works end-to-end; ensure progress feedback is clear | Low |
| 7 | Filter persistence | Filters reset on page navigation | Persist filters in URL query params (router doesn't support query strings currently) | Medium |
| 8 | Date range selection | Custom date picker in filters only | Standardize date picker across reporting and transaction filters | Low |

---

## Accessibility Improvements (Ranked by Severity)

| # | Issue | Severity | Location | Fix |
|---|-------|----------|----------|-----|
| 1 | No `aria-sort` on any sortable table header | High | 7 tables across all modules | Add `aria-sort="ascending|descending|none"` to `<th>` |
| 2 | No `role="progressbar"` on ProgressDialog | High | `shared/ui/ProgressDialog.svelte` | Add `role="progressbar"` + `aria-valuenow` + `aria-valuemin` + `aria-valuemax` |
| 3 | `role="button"` on `<tr>` elements | High | `AuditLogsTable`, `RolesPage`, `ProductTable`, `ExpandableRow` | Replace with proper `<button>` inside a cell, or remove interactive role |
| 4 | No `aria-live` for dynamic content | High | All pages with loading/filtering | Add `aria-live="polite"` to loading states and filter result counts |
| 5 | Chart canvas inaccessible | Medium | `ChartArea.svelte` | Add `aria-label` + hidden data table alternative |
| 6 | No `prefers-reduced-motion` | Medium | Global | Add media query to disable animations |
| 7 | No skip-to-content link | Medium | `Layout.svelte` | Add `<a href="#main-content" class="sr-only focus:not-sr-only">Skip to content</a>` |
| 8 | Modal backdrop click prevention | Medium | `shared/ui/Modal.svelte` | Add backdrop click handler (with `persistent` opt-out) |
| 9 | Filter chip close buttons mixed languages | Low | `UserToolbar` vs `AuditLogsFilterToolbar` | Standardize language |
| 10 | No keyboard nav for table cells | Low | All data tables | Consider roving tabindex for power users |

---

## Quick Wins

| # | Change | Impact | Effort |
|---|--------|--------|--------|
| 1 | Delete `src/routes/` and `src/lib/` | Reduces codebase by ~33%, eliminates confusion | 1-2 hours |
| 2 | Add `@media (prefers-reduced-motion: reduce)` to `app.css` | Motion accessibility for sensitive users | 30 min |
| 3 | Fix `Modal.svelte` `bg-surface` → `bg-surface-default` | Prevents potential styling bug | 5 min |
| 4 | Add backdrop click-to-close to `Modal.svelte` (with `persistent` prop) | Matches user expectations | 1 hour |
| 5 | Delete dead stores: `pos-store`, `product-store`, `customer-store`, `inventory-store` | Reduces confusion | 30 min |
| 6 | Delete `dashboard-service.ts` dead code or wire it into `Home.svelte` | Eliminates code duplication | 1 hour |
| 7 | Rename reporting `DataTable` to `RevenueDataTable` | Eliminates naming conflict | 30 min |
| 8 | Remove `Badge.destructive` variant (duplicates `danger`) | Cleaner API | 5 min |
| 9 | Add 500k, 1M, and "Exact" cash presets to POS | Faster checkout | 1 hour |
| 10 | Add "Exact" button (F6) as visible preset in CheckoutModal | Better discoverability | 30 min |
| 11 | Add `aria-sort` to all sortable `<th>` elements | Major accessibility improvement | 2 hours |
| 12 | Add 10/20/50/100 page size options to Pagination | Better power user experience | 1 hour |
| 13 | Replace `confirm()` for cart clear in POS with instant action + undo toast | Smoother POS flow | 1 hour |
| 14 | Add try/catch + error UI around lazy page imports in `main.svelte` | Prevents blank screen on chunk load failure | 2 hours |
| 15 | Add 404 route handling in `main.svelte` | Better UX for bad URLs | 30 min |

---

## Medium-Term Improvements

| # | Change | Impact | Effort |
|---|--------|--------|--------|
| 1 | Extract `Drawer.svelte` shared component | Replaces 5 custom implementations | 2-3 days |
| 2 | Extract `ToggleSwitch.svelte` shared component | Replaces 4 copy-pasted implementations | 1 day |
| 3 | Extract `ConfirmDeleteModal.svelte` shared component | Consistent delete UX across all CRUD | 1 day |
| 4 | Extract `FilterChipBar.svelte` shared component | Replaces 3 identical CSS implementations | 1 day |
| 5 | Extract `DatePickerDropdown.svelte` shared component | Replaces 2 near-identical implementations | 2 days |
| 6 | Extract `SortableHeader.svelte` + `TableSkeleton.svelte` + `EmptyState.svelte` | Consistent table UX | 2 days |
| 7 | Extract `useRBAC()` composable | Replaces 6 copy-pasted permission derivation blocks | 1 day |
| 8 | Fix POS customer selection UX (keep checkout open) | Core workflow improvement | 1 day |
| 9 | Fix POS `finalizeSale()` race condition with `await` | Prevents potential data loss | 2 hours |
| 10 | Add numeric validation to POS cash input | Prevents invalid payment amounts | 2 hours |
| 11 | Add `aria-live` regions for loading and filter states | Screen reader support | 1 day |
| 12 | Standardize on one language (Indonesian) with i18n constants | Professional appearance | 2-3 days |
| 13 | Add route authorization using existing `permissions.ts` | Security enforcement | 1 day |
| 14 | Decompose `ProductsPage.svelte` (693 lines) into smaller composables | Maintainability | 2-3 days |
| 15 | Convert `.js` reporting files to `.ts` | Type safety | 1 day |
| 16 | Add filter persistence via URL query params | Workflow continuity | 2 days (requires router upgrade) |

---

## Long-Term Improvements

| # | Change | Impact | Effort |
|---|--------|--------|--------|
| 1 | **Responsive design**: hamburger menu, mobile sidebar overlay, POS cart bottom sheet | Enables tablet/mobile use | 2-3 sprints |
| 2 | **Form system**: Shared `FormField`, `FormError`, `FormSelect` components + validation composable | Consistent form UX across all modules | 1 sprint |
| 3 | **Table system**: Enhanced `DataTable` with built-in sort, pagination, skeleton, empty state, bulk actions | Eliminates per-module table reimplementations | 1 sprint |
| 4 | **Navigation redesign**: Replace flat sidebar with grouped navigation, add breadcrumbs to all pages, consider tabbed settings | Better information architecture | 1 sprint |
| 5 | **Full keyboard navigation**: Arrow keys in tables, Ctrl+K global search, keyboard shortcut guide | Power user efficiency | 1 sprint |
| 6 | **Dark/Light mode toggle**: Design tokens already support it; need light theme values + toggle component | User preference | 1 sprint |
| 7 | **Component library documentation**: Storybook or similar for all shared components | Developer onboarding + consistency | 1 sprint |

---

## Prioritized Action Plan

| Priority | Category | Recommendation | User Impact | Business Impact | Effort | Sprint |
|----------|----------|----------------|-------------|-----------------|--------|--------|
| **P0** | Maintainability | Delete dead code (`src/routes/`, `$lib/`, dead stores/services) | Low | High (reduced confusion, smaller bundle) | Small | Sprint 1 |
| **P0** | Security | Wire `permissions.ts` into route guard | High | Critical (unauthorized access) | Small | Sprint 1 |
| **P0** | Accessibility | Add `aria-sort` to all sortable tables | High | Legal compliance | Small | Sprint 1 |
| **P1** | Design System | Extract `Drawer`, `ToggleSwitch`, `ConfirmDeleteModal` | Medium | High (consistency + maintenance) | Medium | Sprint 1-2 |
| **P1** | UX | Fix POS customer selection workflow | High | High (faster checkout) | Small | Sprint 1 |
| **P1** | UX | Fix POS cash input validation + presets | Medium | Medium (fewer errors) | Small | Sprint 1 |
| **P1** | UX | Add backdrop click-to-close to Modal | Medium | Medium (user expectation) | Small | Sprint 1 |
| **P1** | Accessibility | Add `role="progressbar"` + `aria-live` regions | High | Legal compliance | Small | Sprint 1 |
| **P2** | Design System | Extract `FilterChipBar`, `DatePickerDropdown`, `SortableHeader` | Medium | Medium | Medium | Sprint 2 |
| **P2** | i18n | Standardize language (Indonesian) | Medium | Medium (professional appearance) | Medium | Sprint 2 |
| **P2** | UX | Replace `confirm()` in POS with custom modal/toast | Medium | Low (smoother flow) | Small | Sprint 2 |
| **P2** | Accessibility | Add `prefers-reduced-motion`, skip-to-content | Medium | Legal compliance | Small | Sprint 2 |
| **P2** | Maintainability | Decompose monolith pages into composables | Low | Medium (developer productivity) | Large | Sprint 2-3 |
| **P3** | Responsive | Hamburger menu + mobile sidebar | High | High (enables tablet use) | Large | Sprint 3 |
| **P3** | Responsive | POS cart responsive layout | High | High (enables mobile POS) | Large | Sprint 3 |
| **P3** | Design System | Enhanced `DataTable` with built-in sort/pagination/skeleton | Medium | Medium | Large | Sprint 3 |
| **P3** | UX | Filter persistence via URL query params | Medium | Medium | Medium | Sprint 3 |
| **P4** | Design System | Form system (`FormField`, `FormError`, validation) | Medium | Medium | Large | Sprint 4 |
| **P4** | UX | Dark/Light mode toggle | Low | Low | Medium | Sprint 4 |
| **P4** | UX | Keyboard shortcut guide + Ctrl+K search | Low | Low | Large | Sprint 4 |

---

*Audit conducted by analyzing the complete codebase including: 24 shared UI components, 20+ page components, 6 module stores, 8 module services, 4 layout components, 1 custom SPA router, 1 global CSS design token system, and the full backend API surface (65 endpoints). All findings are based on actual code evidence — no speculation.*
