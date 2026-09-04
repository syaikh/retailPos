# Plan: Add Consignment Filter to Suppliers Page

> **Status:** Implemented

## Problem

No dedicated place to see all suppliers flagged as `is_consignment`. They only appear in the "New Arrangement" dropdown (minimal info: id, name) and the Arrangements list (only if they have an existing arrangement). Users must navigate to Suppliers → Edit to check or toggle the consignment flag.

## Solution

Add a "Consignment" filter toggle to the existing Suppliers master data page (`/suppliers`). Support URL deep-linking (`?is_consignment=true`) so the Consignment module can link directly to a filtered view.

### Current state

- `is_consignment` exists in DB, domain model, and Supplier type — but is **invisible** in the list view
- Backend `GET /api/suppliers` supports `is_active` filter but NOT `is_consignment`
- Frontend `SupplierListParams` has no `is_consignment` field
- Suppliers table shows no consignment indicator

### Target state

```
[← Back] [Search bar]  [All] [Active] [Inactive]  [Konsinyasi]  [+ Add Supplier]
```

- "Konsinyasi" toggle filters to consignment-only suppliers
- Stacks with status filter (e.g., Active + Konsinyasi = active consignment suppliers)
- URL `/suppliers?is_consignment=true` opens with filter pre-applied
- URL `/suppliers?is_consignment=true&referrer=consignment` shows a back button linking to `/consignment/arrangements`
- Consignment module links to this filtered view

---

## Changes

### Backend

#### 1. `internal/supplier/repository.go` (line 192)

Add `isConsignment *bool` parameter to `GetAll`. Same pattern as `isActive`:

```go
func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, isConsignment *bool) ([]Supplier, int, error) {
    // ... existing isActive filter (lines 208-214) ...
    if isConsignment != nil {
        filter := fmt.Sprintf(" AND is_consignment = $%d", argIdx)
        countQuery += filter
        dataQuery += filter
        args = append(args, *isConsignment)
        argIdx++
    }
}
```

#### 2. `internal/supplier/service.go` (line 50)

Add `isConsignment *bool` to interface and implementation:

```go
// Interface (service interface definition)
GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, isConsignment *bool) ([]Supplier, int, error)

// Implementation
func (s *service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, isConsignment *bool) ([]Supplier, int, error) {
    return s.repo.GetAll(ctx, limit, offset, search, isActive, isConsignment)
}
```

#### 3. `internal/supplier/handler.go` (line 72)

Parse `is_consignment` query param, same pattern as `is_active` (lines 80-84):

```go
var isConsignment *bool
if v := c.Query("is_consignment"); v != "" {
    b := strings.EqualFold(v, "true") || v == "1"
    isConsignment = &b
}

suppliers, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive, isConsignment)
```

#### 4. `internal/supplier/repository_test.go`

Update all `GetAll` calls to include the new `nil` (or specific) `isConsignment` parameter.

#### 5. `internal/supplier/service_test.go`

Update all `GetAll` calls to include the new parameter.

---

### Frontend

#### 6. `web/src/modules/supplier/services/supplier-service.ts` (line 4)

Add `is_consignment` to params interface and URL building:

```ts
export interface SupplierListParams {
  limit: number;
  offset: number;
  search?: string;
  is_active?: boolean;
  is_consignment?: boolean;  // ← new
  sort_by?: string;
  sort_dir?: string;
}
```

In `getSuppliers()`, append to `urlParams`:

```ts
if (params.is_consignment !== undefined) {
  urlParams.set('is_consignment', String(params.is_consignment));
}
```

#### 7. `web/src/modules/supplier/components/SuppliersPage.svelte`

Add state and URL param reading:

```ts
let consignmentFilter = $state(false);
let referrer = $state<string | null>(null);

onMount(() => {
  const urlParams = new URLSearchParams(window.location.search);
  if (urlParams.get('is_consignment') === 'true') {
    consignmentFilter = true;
  }
  referrer = urlParams.get('referrer');
  load();
});
```

Update `load()` to pass `is_consignment`:

```ts
if (consignmentFilter) params.is_consignment = true;
```

Add handler:

```ts
function handleConsignmentChange() {
  offset = 0;
  load();
}
```

Add back button (shown when `referrer === 'consignment'`):

```svelte
{#if referrer === 'consignment'}
  <button
    class="inline-flex items-center gap-1.5 text-sm text-text-muted hover:text-text-secondary transition-colors"
    onclick={() => goto('/consignment/arrangements')}
  >
    <ArrowLeft size={16} /> {labels.back}
  </button>
{/if}
```

Wire to toolbar:

```svelte
<SuppliersToolbar
  bind:consignmentFilter
  onconsignmentchange={handleConsignmentChange}
  ...
/>
```

#### 8. `web/src/modules/supplier/components/SuppliersToolbar.svelte`

Add `consignmentFilter` bindable prop and toggle button:

```ts
let {
  consignmentFilter = $bindable(false),
  onconsignmentchange,
  ...
}: { ... } = $props();
```

Add toggle button after the status filter group:

```svelte
<button
  class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200
    {consignmentFilter
      ? 'bg-warning-subtle text-warning-light border border-warning-default/20'
      : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
  onclick={() => { consignmentFilter = !consignmentFilter; onconsignmentchange?.(); }}
  aria-pressed={consignmentFilter}
>{labels.consignmentFilter}</button>
```

#### 9. `web/src/modules/supplier/components/SuppliersTable.svelte`

Add consignment badge column after the Status column:

```svelte
<th class="px-4 py-3">{labels.type}</th>
...
<td class="px-4 py-3">
  {#if supplier.is_consignment}
    <Badge variant="warning">{labels.consignmentFilter}</Badge>
  {/if}
</td>
```

#### 10. `web/src/modules/consignment/components/ArrangementsPage.svelte`

Add "View Suppliers" link in the arrangement detail header area or as a secondary action. Navigates to filtered suppliers page with referrer for back button:

```svelte
<Button variant="secondary" size="sm" onclick={() => goto('/suppliers?is_consignment=true&referrer=consignment')}>
  <ExternalLink size={16} /> {labels.viewSuppliers}
</Button>
```

---

### i18n

#### 11. `web/src/shared/i18n/en.ts`

```ts
consignmentFilter: 'Consignment',
viewSuppliers: 'View Suppliers',
```

#### 12. `web/src/shared/i18n/id.ts`

```ts
consignmentFilter: 'Konsinyasi',
viewSuppliers: 'Lihat Pemasok',
```

---

## Files Summary

| # | File | Change |
|---|------|--------|
| 1 | `internal/supplier/repository.go` | Add `isConsignment` filter to SQL |
| 2 | `internal/supplier/service.go` | Add param to interface + impl |
| 3 | `internal/supplier/handler.go` | Parse `is_consignment` query param |
| 4 | `internal/supplier/repository_test.go` | Update `GetAll` test calls |
| 5 | `internal/supplier/service_test.go` | Update `GetAll` test calls |
| 6 | `web/src/modules/supplier/services/supplier-service.ts` | Add `is_consignment` param |
| 7 | `web/src/modules/supplier/components/SuppliersPage.svelte` | Add state + URL param reading |
| 8 | `web/src/modules/supplier/components/SuppliersToolbar.svelte` | Add consignment toggle button |
| 9 | `web/src/modules/supplier/components/SuppliersTable.svelte` | Add consignment badge column |
| 10 | `web/src/modules/consignment/components/ArrangementsPage.svelte` | Add "View Suppliers" link |
| 11 | `web/src/shared/i18n/en.ts` | 2 new labels |
| 12 | `web/src/shared/i18n/id.ts` | 2 new labels |

---

## Verification

1. `go build ./...` — backend compiles
2. `go test ./internal/supplier/...` — supplier tests pass
3. `cd web && npm run build` — frontend compiles
4. Manual: `/suppliers` shows all suppliers (no regression)
5. Manual: `/suppliers?is_consignment=true` shows only consignment suppliers
6. Manual: Toggle "Konsinyasi" button filters/unfilters
7. Manual: Consignment module "View Suppliers" link works
8. Manual: Back button appears when navigating from consignment, disappears otherwise
