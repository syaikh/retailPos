# Role-Based Dashboard + Finance Settlement Shortcut

**Date:** 2026-09-22
**Status:** Implemented (all six parts shipped; tables verified against
`Home.svelte` and the live permission set on 2026-09-28)

## Problem

1. Dashboard quick access cards are hardcoded — same for all roles, including modules some roles can't access
2. Finance users need 7 steps to pay a settlement (sidebar → consignment → open → tab → find → pay → fill)
3. Settlement page has no status filter or product search

## Solution

1. Role-based quick access cards filtered by permissions
2. "Pending Settlements" quick access card for finance → modal with direct [Pay] buttons
3. Status tabs + searchbars on settlement page
4. Role-based stat cards (inventory staff gets stock-focused stats)

## Architecture

```
Dashboard (Home.svelte)
  ├── Quick Access Cards (role-filtered by canAny)
  │     └── "Pending Settlements" → PendingSettlementsModal
  │           └── [Pay] → PayoutModal → on success → reopens PendingSettlementsModal
  └── Stat Cards (role-filtered)

Consignment → Arrangement → Settlement Tab
  ├── Unsettled Sales + SearchBar (product_name)
  ├── Status Tabs: [ All ] [ Pending ] [ Paid ]
  ├── Settlement History + SearchBar (product_name)
  │     └── [Pay] → PayoutModal (shared component)
  └── PayoutModal (extracted, reusable)
```

## Part 1: Role-Based Quick Access Cards

All modules in one array with `required` permissions, filtered by `rbac.canAny()`
(Administration shows if **any** of its three permissions is held).

| Role | Cards |
|------|-------|
| Superadmin | **Pending Settlements**, POS, Inventory, Reports, Administration |
| Manager | **Pending Settlements**, POS, Inventory, Reports, Administration |
| Supervisor | POS, Inventory, Reports |
| Cashier | POS |
| Inventory Staff | *(none — the role has no `dashboard.view`, so the dashboard itself is unreachable)* |
| Finance | **Pending Settlements**, Reports |

| Card | Permission gate |
|------|-----------------|
| **Pending Settlements** | `consignment.pay` (superadmin, manager, finance) |
| POS | `sale.create` |
| Inventory | `product.view` → links to `/inventory/products` |
| Reports | `report.view` |
| Administration | `user.view` **or** `role.view` **or** `store.view` |

> The original draft of the table above also listed Consignment, Shifts and
> Transactions cards; only POS, Inventory, Reports and Administration were
> built as navigation cards, plus the Pending Settlements shortcut.

"Pending Settlements" card:
- Permission: `consignment.pay`
- Badge: pending count fetched on mount
- Click: opens `PendingSettlementsModal`
- href: `#pending-settlements` (no navigation)

## Part 2: Pending Settlements Modal

**New file:** `PendingSettlementsModal.svelte`

Props: `{ show: boolean, onclose: () => void }`

Behavior:
1. Opens → fetches `listSettlements({ status: "pending_payment" })`
2. Table: Supplier, Settlement #, Total Payable, [Pay]
3. Click [Pay] → closes this modal → opens `PayoutModal`
4. Payout success → PayoutModal closes → reopens this modal → re-fetches data

## Part 3: Extract PayoutModal

**New file:** `PayoutModal.svelte`

Props: `{ settlement: Settlement, show: boolean, onclose: () => void, onpaid: () => void }`

Contains: outstanding amount, payment method selector, amount input, reference, notes, submit.

Used by:
- `PendingSettlementsModal` (dashboard shortcut)
- `SettlementPage` (settlement history [Pay] button)

## Part 4: Role-Based Stat Cards

| Stat card | Shown for |
|-----------|-----------|
| Today's Revenue, Transactions | everyone who can open the dashboard |
| Categories | superadmin, manager |
| Out of Stock | superadmin, manager, supervisor |

Backend `/api/dashboard/live` returns `out_of_stock_count` and
`categories_count`.

> The original design also planned "Products"/"Low Stock" stat cards and an
> inventory-staff stat set — neither was built. `inventory_staff` has no
> `dashboard.view`, so it never reaches this screen anyway.

## Part 5: Settlement Page — Status Tabs + Search

Props: add `initialTab?: "all" | "pending" | "paid"` (default: `"all"`)

Status tabs on settlement history: [ All ] [ Pending ] [ Paid ]

Two SearchBars:
1. Settlement History — filter by product_name across settlement items
2. Unsettled Sales — filter by product_name in preview items

## Part 6: Add product_sku to SettlementItem

For display only (not used in search). Update Go struct, TS interface, and hydration callback.

## Files

| File | Action |
|------|--------|
| `Home.svelte` | Modify — role-based modules + stat cards |
| **NEW** `PendingSettlementsModal.svelte` | Create |
| **NEW** `PayoutModal.svelte` | Create — extracted from SettlementPage |
| `SettlementPage.svelte` | Modify — tabs, searchbars, use PayoutModal, initialTab |
| `consignment-service.ts` | Modify — `listSettlements(supplierId?, status?)` |
| `types/index.ts` | Modify — `product_sku` on SettlementItem |
| `domain.go` | Modify — `ProductSKU` on SettlementItem |
| `handler.go` (consignment) | Modify — optional supplier_id, status filter |
| `service.go` (consignment) | Modify — hydrate SKU, optional params |
| `repository.go` (consignment) | Modify — dynamic WHERE |
| `handler.go` (report) | Modify — out_of_stock_count, categories_count |
| `repository.go` (report) | Modify — new queries |
| `i18n/en.ts` + `id.ts` | Modify — new labels |
