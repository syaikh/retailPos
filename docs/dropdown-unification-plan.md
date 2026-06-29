# Dropdown Unification Plan

## Goal

Replace all inline dropdown implementations with a single reusable `Dropdown.svelte` component to eliminate code duplication and standardize behavior.

## Architecture

**Single file component** at `web/src/shared/ui/Dropdown.svelte` — consistent with other shared UI components (Button, Modal, Input).

Uses **Svelte 5 snippets** for flexible trigger/content rendering, with a fallback `items` array prop for simple menus.

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `items` | `DropdownItem[]` | `[]` | Simple menu items (rendered when `content` snippet absent) |
| `itemsClass` | `string` | `''` | Tailwind classes for items container |
| `trigger` | `Snippet<{ open, toggle }>` | Required | Custom trigger content |
| `content` | `Snippet<{ close }>` | `undefined` | Custom content (overrides `items`) |
| `placement` | `Placement` | `'bottom-end'` | Dropdown position relative to trigger |
| `open` | `$bindable(false)` | `false` | Controlled open state |
| `menu` | `boolean` | `true` | Add ARIA menu roles |
| `menuClass` | `string` | `''` | Extra classes for the dropdown panel |

### Behavior

- **Open/close**: Click trigger toggles; outside click / Escape closes
- **Keyboard nav**: Arrow Up/Down (sirkular), Enter/Space select (items mode only)
- **Focus**: Auto-focus first item on open; return focus to trigger on close
- **ARIA**: `aria-expanded`, `aria-haspopup`, `role="menu"`/`role="menuitem"`
- **Transition**: `fly` animation on open/close
- **Lifecycle**: No memory leaks — `$effect` cleanup removes listeners

## Files to Refactor

### Phase 1 — Simple dropdowns → reusable `<Dropdown>`

| File | Dropdown(s) | Changes |
|------|-------------|---------|
| `shared/ui/ExportImportButtons.svelte` | Export (CSV/XLSX) | Replace inline `showExportDropdown` + `{#if}` block |
| `modules/product/ProductFiltersToolbar.svelte` | Status filter | Replace inline status dropdown |
| `modules/admin/UserToolbar.svelte` | Role filter + Status filter (×2) | Replace both inline dropdowns |
| `modules/admin/AuditLogsFilterToolbar.svelte` | Resource filter + Action filter (×2) | Replace both inline dropdowns |
| `modules/sales/TransactionFilters.svelte` | Export dropdown | Replace inline export dropdown |
| `modules/reporting/PeriodSelector.svelte` | Export dropdown | Replace inline export dropdown |
| `modules/admin/RolesPage.svelte` | Role action per row | Replace inline action dropdown |

### Phase 2 — Payment dropdown (TransactionFilters)

Use `content` snippet from `<Dropdown>` for the multi-select checkbox layout + apply/cancel footer. No separate component needed.

### Phase 3 — Parent page state cleanup

| File | Changes |
|------|---------|
| `modules/admin/AuditLogsPage.svelte` | Remove `handleClickOutside`/`handleEsc` for resource/action/export dropdowns (encapsulated in `<Dropdown>`). Keep date picker + drawer handling. |
| `modules/sales/TransactionsPage.svelte` | Remove `handleWindowClick`/`handleKeydown` for export dropdown. Keep date picker + payment + drawer handling. |
| `modules/product/ProductsPage.svelte` | Remove `close-all-dropdowns` dispatch. |

### Files NOT refactored

| File | Reason |
|------|--------|
| `PeriodSelector.svelte` (period type) | Dual-panel layout + embedded calendar components |
| `AuditLogsFilterToolbar.svelte` (date picker) | Calendar presets + custom date range inputs |
| `TransactionFilters.svelte` (date picker) | Same as above |
| `UserFormModal.svelte` (role selector) | `position:fixed`, grid layout inside modal |
| `ProductFormModal.svelte` (category search) | Searchable combobox — fundamentally different behavior |
| `NotificationBell.svelte` | Full notification panel with WebSocket integration |
| `ProductActionsDropdown.svelte` | Keep as-is (dedicated component, user preference) |

## Testing

- New: `shared/ui/__tests__/Dropdown.svelte.test.ts` — unit tests for component behavior
- Updated: Source-structure guard tests for each refactored file
