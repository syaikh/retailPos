# Implementation Plan: Customer Groups Enterprise Overhaul

> **Date**: 2026-07-18
> **Branch**: `feature/pricing-engine-supplier-mgmt`
> **Related Audit**: `docs/audit-customer-groups-layout.md`

---

## Phase 1 — Quick Alignment (Frontend Only, ~1.5 hours)

### 1.1 CustomerGroupsToolbar.svelte
- Match PricingRulesToolbar's 2-row layout pattern
- Row 1: `SearchBar` + `BulkActionDropdown` + "+ Add Group" button
- Row 2: Segmented status filter with `aria-pressed`, `FilterChipBar` for active filters
- Add `FilterChipBar` import and active filter chips (search query)
- Add `aria-pressed` to all 3 status filter buttons
- Change wrapper from `card p-4 space-y-3` → `card px-4 py-3`
- Add `role="group"` and `aria-label` to status filter

### 1.2 CustomerGroupsTable.svelte
- Add `<colgroup>` with explicit widths (match PricingRulesTable's `table-layout: fixed`)
- Row height: `py-1.5 h-12` → `py-4`
- Replace inline icons with `Dropdown` kebab menu (`MoreVertical` trigger)
- Merge "Description" under name as secondary text (eliminate 35% column)
- Add relative time with `Tooltip` on "DIPERBARUI" column (use `timeAgo()` pattern from PricingRulesTable)
- Use `<col>` widths instead of `<th>` widths
- Add `aria-live="polite"` to empty state
- Import `Dropdown`, `Tooltip` from `$shared/ui`
- Import `MoreVertical` from `lucide-svelte`

### 1.3 CustomerGroupsPage.svelte
- Add page heading with subtitle (remove inline heading from children)
- Change `card overflow-hidden` → `card overflow-x-auto`
- Wire new kebab menu callbacks: `onviewmembers`, `onduplicate`, `onedit`, `ondelete`

### Files Modified
| File | Change |
|------|--------|
| `web/src/modules/customer-groups/components/CustomerGroupsToolbar.svelte` | 2-row layout, FilterChipBar, BulkActionDropdown, aria-pressed |
| `web/src/modules/customer-groups/components/CustomerGroupsTable.svelte` | Kebab menu, colgroup, py-4, merge description, relative time, Tooltip |
| `web/src/modules/customer-groups/components/CustomerGroupsPage.svelte` | Heading, overflow-x-auto, wire kebab callbacks |

---

## Phase 2 — Core Data (Backend + Frontend, ~2.5 hours)

### 2.1 Backend: Add `customer_count` to CustomerGroup
- **domain.go**: Add `CustomerCount int` field to `CustomerGroup` struct
- **repository.go**: Modify `GetAll()` — add `LEFT JOIN (SELECT customer_group_id, COUNT(*) AS cnt FROM customers GROUP BY customer_group_id) cc ON cc.customer_group_id = cg.id`
  - Add `COALESCE(cc.cnt, 0)` to SELECT and Scan
  - Also update `GetByID()` with same pattern
- **Service**: No changes needed (transparent to service layer)

### 2.2 Frontend: Statistics Cards
- **CustomerGroupsToolbar.svelte**: Add stats row with 4 cards (Total, Active, Inactive, Customers)
  - Stats props: `{ total: number; activeCount: number; inactiveCount: number; customerCount: number }`
  - Use same card style as existing page cards
- **CustomerGroupsPage.svelte**: Compute stats from `groups` and `total`, pass to toolbar

### 2.3 Frontend: Customer Count Column
- **CustomerGroupsTable.svelte**: Add sortable "CUSTOMERS" column after NAMA
  - Display `customer_count` with `toLocaleString('id-ID')`
  - Make sortable by adding `customer_count` to sort options
- **types/index.ts**: Add `customer_count?: number` to `CustomerGroup`

### Files Modified
| File | Change |
|------|--------|
| `internal/customergroup/domain.go` | Add `CustomerCount int` field |
| `internal/customergroup/repository.go` | LEFT JOIN for count in GetAll/GetByID |
| `web/src/modules/customer-groups/components/CustomerGroupsToolbar.svelte` | Statistics cards row |
| `web/src/modules/customer-groups/components/CustomerGroupsTable.svelte` | CUSTOMERS column |
| `web/src/modules/customer-groups/components/CustomerGroupsPage.svelte` | Compute and pass stats |
| `web/src/modules/customer-groups/types/index.ts` | Add `customer_count` field |

---

## Phase 3 — Enterprise Features (Backend + Frontend, ~4 hours)

### 3.1 Bulk Operations Backend
- **repository.go**: Add methods:
  - `BulkUpdate(ctx, ids []int, isActive bool) (int, error)` — `UPDATE customer_groups SET is_active = $1, updated_at = NOW() WHERE id = ANY($2)`
  - `BulkDelete(ctx, ids []int) (int, error)` — `DELETE FROM customer_groups WHERE id = ANY($1)`
- **service.go**: Add `BulkUpdate(ctx, ids, isActive)`, `BulkDelete(ctx, ids)` with validation
- **handler.go**: Add endpoints:
  - `PUT /customer-groups/bulk` — `{ ids: [1,2,3], is_active: true }`
  - `DELETE /customer-groups/bulk` — `{ ids: [1,2,3] }`
  - Both require `customer_group:update` / `customer_group:delete` permissions

### 3.2 Export/Import Backend
- **repository.go**: Add:
  - `BulkUpsertCustomerGroups(ctx, records []CustomerGroupImportRow) ImportResult` — batch insert/update with ON CONFLICT
  - `GetAllForExport(ctx) ([]CustomerGroup, error)` — fetch all for export (no pagination)
- **adapter.go**: Update `Insert` to use `BulkUpsertCustomerGroups` (like category pattern)
- **schema.go**: Already exists — no changes needed
- **main.go**: Already registered — no changes needed

### 3.3 Frontend: Bulk Actions
- **CustomerGroupsTable.svelte**: Add checkbox column, selection state, bulk action bar
  - Use `CheckSquare`/`Square` pattern from PricingRulesTable
  - Bulk bar: "X group dipilih" + Activate/Deactivate/Delete buttons
  - New props: `onbulkactivate`, `onbulkdeactivate`, `onbulkdelete`, `onduplicate`
- **CustomerGroupsPage.svelte**: Wire bulk operations with `bulkUpdate`, `bulkDelete` service calls
- **services/customer-group-service.ts**: Add `bulkUpdate(ids, isActive)`, `bulkDelete(ids)`

### 3.4 Frontend: Import Wizard
- **CustomerGroupsPage.svelte**: Add `showImportWizard` state + `ImportWizard` component
  - Copy pattern from PricingRulesPage's import wizard
  - Module: `customer_groups`

### 3.5 BulkActionDropdown Update
- **BulkActionDropdown.svelte**: Add `customer_groups` to `historyRoutes` map

### Files Modified
| File | Change |
|------|--------|
| `internal/customergroup/repository.go` | BulkUpdate, BulkDelete, BulkUpsertCustomerGroups, GetAllForExport |
| `internal/customergroup/service.go` | BulkUpdate, BulkDelete |
| `internal/customergroup/handler.go` | PUT /bulk, DELETE /bulk endpoints |
| `internal/customergroup/adapter.go` | Use BulkUpsertCustomerGroups in Insert |
| `web/src/modules/customer-groups/services/customer-group-service.ts` | bulkUpdate, bulkDelete |
| `web/src/modules/customer-groups/components/CustomerGroupsTable.svelte` | Checkbox column, bulk action bar, kebab menu items |
| `web/src/modules/customer-groups/components/CustomerGroupsPage.svelte` | Bulk ops wiring, ImportWizard |
| `web/src/shared/ui/BulkActionDropdown.svelte` | Add customer_groups to historyRoutes |

---

## Phase 4 — Polish (Frontend + Migration, ~1.5 hours)

### 4.1 Group Color
- **New migration**: `042_customer_groups_add_color.sql`
  ```sql
  ALTER TABLE customer_groups ADD COLUMN IF NOT EXISTS color VARCHAR(7);
  ```
- **domain.go**: Add `Color string` field (JSON: `color`)
- **repository.go**: Add `color` to SELECT/INSERT/UPDATE
- **Frontend**: Colored avatar using `color` field
  - Default colors for seed data: VIP → `#6C5CE7`, Member → `#00B894`, Walk-in → `#636E72`
  - Color picker in Create/Edit modal
  - Avatar uses inline `style="background-color: {color}"` instead of fixed class

### 4.2 Advanced Filters
- **Toolbar**: Add "Has Customers" toggle (show only groups with count > 0)
  - Backend: Add `has_customers bool` query param to `GetAll`
  - Repository: Add `AND cc.cnt > 0` when `has_customers` is true

### Files Modified
| File | Change |
|------|--------|
| `database/migrations/042_customer_groups_add_color.sql` | New migration |
| `internal/customergroup/domain.go` | Add `Color` field |
| `internal/customergroup/repository.go` | Color in queries, has_customers filter |
| `internal/customergroup/handler.go` | has_customers query param |
| `web/src/modules/customer-groups/components/CustomerGroupsToolbar.svelte` | Has Customers toggle |
| `web/src/modules/customer-groups/components/CustomerGroupsTable.svelte` | Colored avatar |
| `web/src/modules/customer-groups/components/CreateCustomerGroupModal.svelte` | Color picker |
| `web/src/modules/customer-groups/components/EditCustomerGroupModal.svelte` | Color picker |
| `web/src/modules/customer-groups/types/index.ts` | Add `color` field |

---

## Migration Summary

| Migration | Description | Breaking? |
|-----------|-------------|-----------|
| 042_customer_groups_add_color | Add `color VARCHAR(7)` to `customer_groups` | No — nullable column |

---

## Test Strategy

### Go Tests
- **Repository**: `BulkUpdate` success/failure, `BulkDelete` success/failure, `BulkUpsertCustomerGroups` insert/update/conflict, `GetAllForExport`, customer_count LEFT JOIN correctness
- **Handler**: `PUT /customer-groups/bulk` with valid/invalid payloads, `DELETE /customer-groups/bulk` with valid/invalid payloads, permission checks
- **Service**: `BulkUpdate` validation (empty IDs), `BulkDelete` validation

### Frontend Tests
- **Service**: `bulkUpdate()`, `bulkDelete()` API calls
- **Table**: Checkbox selection, bulk action bar visibility, kebab menu items, customer_count rendering
- **Toolbar**: Statistics cards rendering, FilterChipBar, BulkActionDropdown

### E2E
- Existing E2E tests should continue to pass (no breaking changes)
- No new E2E needed for CRUD operations

---

## Estimated Effort

| Phase | Time | Files Changed |
|-------|------|---------------|
| Phase 1: Quick Alignment | ~1.5 hours | 3 frontend |
| Phase 2: Core Data | ~2.5 hours | 2 backend, 4 frontend |
| Phase 3: Enterprise | ~4 hours | 4 backend, 4 frontend, 1 shared |
| Phase 4: Polish | ~1.5 hours | 1 migration, 3 backend, 5 frontend |
| **Total** | **~9.5 hours** | **~25 file modifications** |

---

## Risk Assessment

- **Low risk**: All changes are additive — no breaking changes to existing APIs
- **Backend**: LEFT JOIN pattern already used in `customer/repository.go` for the same FK
- **Frontend**: All shared components (Dropdown, BulkActionDropdown, FilterChipBar, Tooltip) are battle-tested in PricingRules
- **Import/Export**: Adapter already exists — just needs BulkUpsert optimization
- **Migration**: Adding nullable column — zero downtime, no data loss

---

## Completion Status

> **Completed**: 2026-07-18
> **All 4 phases: ✅ COMPLETE**
> **Total test count**: 59 frontend tests (4 files) + all Go tests passing

### Phase 1 ✅ — Quick Alignment
Commit: `31fa1c7` — 2-row toolbar, kebab menu, colgroup, py-4, merge description, relative time, Tooltip, aria-pressed, aria-live, page heading

### Phase 2 ✅ — Core Data
Commits: `31fa1c7` + `39f0ee6` — customer_count LEFT JOIN, statistics cards, color avatar, BulkUpdate/BulkDelete endpoints, ImportWizard, export endpoint, has_customers filter

### Phase 3 ✅ — Enterprise Features
Commits: `31fa1c7` + `39f0ee6` — Bulk activate/deactivate/delete, import/export, duplicate group, audit trail merged into preview panel

### Phase 4 ✅ — Polish
Commit: Uncommitted — Color picker in Create/Edit modals, has_customers filter (backend + frontend), preview panel with audit trail, row click → drawer, entity_id filter on audit-logs API

### Additional Items Implemented
| Item | Commit |
|------|--------|
| Color column (migration 042) | `31fa1c7` |
| Backend audit logging on customer group CRUD | `31fa1c7` |
| Frontend source-structure guard tests (59 tests) | `31fa1c7` |
| Preview panel (CustomerGroupDetailDrawer) | Uncommitted |
| Audit trail in preview panel | Uncommitted |
| entity_id filter on GET /audit-logs | Uncommitted |
