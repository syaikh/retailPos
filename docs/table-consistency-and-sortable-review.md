# Table Consistency & Sortable Headers — Review & Decisions

Status: implemented (2026-08-12) — parked items marked **PARKED**
Scope: web frontend (`web/src`)

## Context

A review of the data tables found three classes of issues:

1. Table title (header) columns were not consistently uppercase across pages.
2. The brands table lacked sortable headers while every other list table had them.
3. A shared `DataTable.svelte` component is exported but never used — all tables
   hand-roll their own `<table>` markup (28 files contain `<table>`; ~18 are list
   tables, the rest are detail pages/modals/forms).

## Root cause of the uppercase bug

Header casing is applied by **two redundant mechanisms**:

- A global rule in `web/src/app.css` (`th { @apply ... uppercase ... }`) that
  uppercases every `<th>`, and
- An explicit `uppercase` class on most `<thead>` elements.

The actual visual bug was isolated to sortable headers: `SortableHeader` renders
its label inside a `<button>`, which is the one spot not guaranteed to inherit the
`text-transform` from `th`/`thead`. Several tables worked around this with
`labels.X.toUpperCase()` hacks.

## Decisions & work executed

### 1. Uppercase is now a single mechanism ✓

- ✓ Kept the global `th` rule in `app.css` as the source of truth.
- ✓ Added `uppercase` to the `SortableHeader` button class
  (`web/src/shared/ui/SortableHeader.svelte`) so sortable labels are uppercased
  at the component level, independent of CSS inheritance.
- ✓ Removed the now-redundant `uppercase` class from all `<thead>` elements.
- ✓ Removed the `labels.X.toUpperCase()` label hacks in
  `CustomerGroupsTable.svelte` and `PricingRulesTable.svelte`.

### 2. Brands table is now sortable ✓

`BrandsPage.svelte` uses `SortableHeader` on `name` / `description` / `created_at`
with client-side sorting, mirroring `CategoriesPage`.

### 3. Sort logic consolidated into a `useSortable` composable ✓

New `web/src/shared/composables/useSortable.svelte.ts` centralizes the identical
`sortBy`/`sortDir` state and `handleSort` toggle logic. The reactive state lives in
a single `$state` object returned as `sortState`, so consumers read/write via member
access (`sortState.sortBy` / `sortState.sortDir`). The name deliberately avoids
`state` (which would collide with the `$state` rune and trigger Svelte's
`store_rune_conflict` diagnostic):

```ts
const { sortState, handleSort } = useSortable('name', 'asc'[, onChange]);
// templates: <SortableHeader sortColumn={sortState.sortBy} sortDirection={sortState.sortDir} onsort={handleSort} />
```

- ✓ `onChange` is invoked after each toggle (used by pages that re-sort or refetch).
- ✓ Migrated 9 tables: Brands, Categories, Stores, Customers, Suppliers,
  PricingRules, Products, CustomerGroups, StorageLocations.
- ✓ Left as-is (documented, owned elsewhere): `PurchaseOrdersPage` (sort state lives
  in a store) and `TransactionTable` (`sortBy`/`sortDir` are `$bindable` props owned
  by the parent).

### 4. Sort scope (PARKED / intentional)

Sorting happens in two places today:

- **Client-side** over the current page: Brands, Categories, Stores,
  StorageLocations, CustomerGroups (small tables).
- **Server-side** via `sort_by`/`sort_dir`: Suppliers, PricingRules (larger tables).

No API changes were made. Client-side sorting for the small settings tables is
treated as intentional. If a table grows beyond a few hundred rows, migrate it to
server-side sorting for consistency.

## Parked: `DataTable.svelte` adoption

`web/src/shared/ui/DataTable.svelte` is dead code — exported from `$shared/ui` but
not imported anywhere. Two options were considered:

- **Delete it** (lowest effort, removes a misleading export).
- **Repurpose it** as a slot-based wrapper (`columns` + `row`/`header` snippets,
  built-in skeleton/empty/pagination) to absorb the ~40–60 lines of scaffolding
  every list table re-implements.

**Verdict: parked.** It is a pure refactor (no user-visible value) touching ~18
pages with medium regression risk, and the abstraction is thin (row cells remain
custom). Recommended path if revisited:

1. Adopt `DataTable` as the default template for **new** tables only (zero risk,
   captures future consistency).
2. Pilot a migration of the 3 small settings tables (Brands, Categories,
   UnitsOfMeasure) as a proof before any wider rollout.

The underlying benefit of a shared wrapper is single-point fixes (e.g., the
uppercase bug), uniform loading/empty/pagination UX, and less boilerplate per page.
