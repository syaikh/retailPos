# Role Permission Audit

> **Date:** 2026-09-11
> **Status:** Implemented — migration 044 applied, backend guards, frontend, E2E tests
> **Applied:** 2026-09-11
> **Reconciled:** 2026-09-28 — counts and matrix verified against the live database with migrations 000–052 applied; see [Post-Implementation Changes](#post-implementation-changes)
> **Related:** [Store-First and Finance Role](./store-first-and-finance-role.md)

## Executive Summary

The role hierarchy at the time of this audit had two problems:

1. **Separation of duties gap** — the admin role (80 permissions) bundles operational management with financial operations. The person who manages staff shouldn't also pay suppliers.

2. **Role names don't match business reality** — "admin" is actually the store manager, "manager" is actually a shift supervisor, "staff" permissions are redundant.

**Recommendation:**
1. Introduce a **finance** role to separate financial operations from store operations
2. Rename roles to match real retail job titles

---

## New Role Structure

| Old Name | New Name | Permissions (current) | Scope | Business Context |
|----------|----------|-------------|-------|------------------|
| superadmin | **superadmin** | 85 | All stores | IT admin at HQ |
| admin | **manager** | 79 | Single store | Store manager (the boss) |
| manager | **supervisor** | 58 | Single store | Shift supervisor |
| finance | **finance** | 6 | Single store | Payment processing |
| cashier | **cashier** | 19 | Single store | Sales |
| staff | **inventory_staff** | 17 | Single store | Stock management |

### Visual Hierarchy

```
superadmin (HQ)
    │
    ▼
manager (store boss)
    │
    ├── supervisor (shift lead)
    ├── finance (payments)
    ├── inventory_staff (stock)
    └── cashier (sales)
```

---

## Permission Count by Role

> **Historical snapshots** — the two tables below are the planning-phase numbers
> from the 044 work. Current counts (86 codes total; superadmin lacks
> `sale.lookup`): superadmin 85, manager 79, supervisor 58, cashier 19,
> inventory_staff 17, finance 6.

### Before (at audit date)

| Role | Permissions | Scope |
|------|-------------|-------|
| superadmin | 85 | System-wide, all stores |
| admin | 80 | Store-level (mostly) |
| manager | 57 | Store-level (operational) |
| cashier | 19 | Store-level (sales) |
| staff | 6 | Store-level (minimal) |

### After (Proposed for 044)

| Role | Permissions | Scope | Changes |
|------|-------------|-------|---------|
| superadmin | 85 | All stores | unchanged |
| manager | 80 | Single store | renamed from "admin" |
| supervisor | 58 | Single store | renamed from "manager", +sale.create |
| finance | 6 | Single store | new role |
| cashier | 19 | Single store | unchanged |
| inventory_staff | 6 | Single store | renamed from "staff", permissions replaced |

---

## Separation of Duties

### Before (Current — Broken)

```
Manager (admin) can:
├── Create settlement (consignment.settle)
└── Pay supplier (consignment.pay)  ← SAME PERSON!
```

### After (As Implemented)

```
Supervisor creates settlement (consignment.settle)
    "We owe CV Lestari Rp 5,000,000"
         │
         ▼
Finance records payment (consignment.pay)
    "Paid Rp 5,000,000 via bank transfer"
```

**The supervisor↔finance split holds: neither role alone can both settle and
pay.** Two deliberate exemptions exist:

- **superadmin** holds all five `consignment.*` permissions (full access).
- **manager** was re-granted `consignment.pay` in **046** (settle-without-pay
  bug fix — the settlement screen hid the Pay button from managers who had just
  created the settlement). A manager can both settle and pay.

---

## Detailed Permission Matrix

### Legend
- ✅ = has permission
- ❌ = does not have permission
- 🔒 = superadmin-only (exclusive)
- ➕ = new permission added

### System & Configuration

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `app_settings.update` | ✅ 🔒 | ❌ | ❌ | ❌ | ❌ | ❌ |
| `app_settings.view` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `role.create` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `role.view` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `role.update` | ✅ 🔒 | ❌ | ❌ | ❌ | ❌ | ❌ |
| `role.delete` | ✅ 🔒 | ❌ | ❌ | ❌ | ❌ | ❌ |

### User Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `user.create` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `user.view` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `user.update` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `user.delete` | ✅ 🔒 | ❌ | ❌ | ❌ | ❌ | ❌ |

### Store Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `store.create` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `store.view` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `store.update` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `store.delete` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |

### Product Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `product.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `product.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.delete` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `product.export` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `product.import` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `product.history.view` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `product.cost.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### Category Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `category.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `category.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.export` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `category.import` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |

### Customer Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `customer.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `customer.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `customer.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `customer.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `customer.export` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `customer.import` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### Customer Group

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `customer_group.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `customer_group.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `customer_group.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `customer_group.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### Pricing

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `pricing.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `pricing.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `pricing.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `pricing.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### Consignment

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `consignment.view` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `consignment.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `consignment.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `consignment.settle` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `consignment.pay` | ✅ | ✅ | ❌ | ➕ | ❌ | ❌ |

### Sales/POS

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `sale.view` | ✅ | ✅ | ✅ | ➕ | ✅ | ❌ |
| `sale.create` | ✅ | ✅ | ➕ | ❌ | ✅ | ❌ |
| `sale.detail` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `sale.park` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `sale.lookup` | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| `receipt.print` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |

### Inventory

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `inventory.adjust` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |

### Stock Opname

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `stock_opname.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ➕ |
| `stock_opname.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.assign` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.count` | ✅ | ✅ | ✅ | ❌ | ✅ | ➕ |
| `stock_opname.recount` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.submit` | ✅ | ✅ | ✅ | ❌ | ✅ | ➕ |
| `stock_opname.verify` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.close` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.cancel` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.post` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.report` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.export` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |

### Purchase Orders

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `purchase_order.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `purchase_order.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `purchase_order.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `purchase_order.confirm` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `purchase_order.receive` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `purchase_order.cancel` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `purchase_order.delete` | ✅ 🔒 | ❌ | ❌ | ❌ | ❌ | ❌ |

### Shift Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `shift.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `shift.create` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `shift.review` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `shift.audit` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `shift.cash_movement` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |

### Storage Locations

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `storage_location.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ➕ |
| `storage_location.create` | ✅ | ✅ | ❌ | ❌ | ❌ | ➕ |
| `storage_location.update` | ✅ | ✅ | ❌ | ❌ | ❌ | ➕ |
| `storage_location.delete` | ✅ | ✅ | ❌ | ❌ | ❌ | ➕ |

### Reports & Dashboard

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `dashboard.view` | ✅ | ✅ | ✅ | ➕ | ✅ | ❌ |
| `report.view` | ✅ | ✅ | ✅ | ➕ | ❌ | ❌ |

### Audit

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `audit.view` | ✅ | ✅ | ❌ | ➕ | ❌ | ❌ |
| `audit.export` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |

---

## Summary of Changes

### Permissions Added

| Role | Permission | Reason |
|------|------------|--------|
| supervisor | `sale.create` | Supervisors can ring up sales at POS |
| finance | `consignment.pay` | Finance records payments to suppliers |
| finance | `report.view` | Finance views financial reports |
| finance | `audit.view` | Finance views audit trail |
| finance | `sale.view` | Finance views sales for reconciliation |
| finance | `dashboard.view` | Finance sees dashboard |
| inventory_staff | `inventory.adjust` | Inventory staff adjusts stock levels |
| inventory_staff | `stock_opname.*` (12) | Inventory staff manages stock opname |
| inventory_staff | `storage_location.*` (4) | Inventory staff manages warehouse locations |

### Permissions Removed from Manager (Moved to Finance)

| Permission | Now Has |
|------------|---------|
| `consignment.pay` | finance |

> **Update:** migration **046** later re-granted `consignment.pay` to manager
> (settle-without-pay bug fix), so the store boss can both settle and pay —
> see [Separation of Duties](#separation-of-duties).

### Roles Renamed

| Old Name | New Name |
|----------|----------|
| admin | manager |
| manager | supervisor |
| staff | inventory_staff |

---

## Post-Implementation Changes

Migrations applied after 044 changed role grants; the Detailed Permission
Matrix above already reflects the final state:

| Migration | Change |
|-----------|--------|
| 045 | Default usernames renamed to match role names (admin→manager, manager→supervisor, staff→inventory_staff) |
| 046 | manager regained `consignment.pay` (settle-without-pay bug fix) |
| 047 | finance gained `consignment.view` (the module and pay-flow gate) — still lacks `consignment.create/update/settle`, so it can view and pay but never originate or settle |
| 049 | `store.view` revoked from finance and supervisor (redundant — both are store-scoped via JWT) |
| 050 | `store.create` revoked from manager (HQ-only store provisioning); added `users.must_change_password` |
| 052 | The six seed accounts flagged `must_change_password` (forced first-login rotation) |

Some cells in the matrix as originally written for 044 also never matched the
grants actually applied by `000_squash.sql` (for example supervisor never
received `product.delete/export/import/history.view`, and cashier holds
`dashboard.view`, `stock_opname.view/count/submit` and
`storage_location.view`). All 86 permission codes were reconciled against the
live database on 2026-09-28.

The phantom `inventory_staff` / `product.view` reading had a concrete source: the
pre-044 grant block at `000_squash.sql:1612` still granted `product.view` to the
role on every replay, relying on 044's wholesale `DELETE` (line 78) to strip it
again. That dead block was removed 2026-09-28, so 044 is now the sole owner of
`inventory_staff` grants — see `.opencode/plans/migration-replay-idempotency-reconciliation.md`.

## Implementation

The role restructuring was implemented as part of migration 044. All items above
have been applied to the dev database. Go role constants, middleware, frontend
role labels, seeder files, and E2E tests all reflect the new naming.

## Store Scoping Enforcement

**All roles except superadmin are scoped to their assigned store.**

- `RequireStoreID` middleware rejects any non-superadmin role without a `store_id` in JWT
- Query-layer filtering ensures manager, supervisor, finance, cashier, and inventory_staff only see data from their assigned store
- Superadmin sees all stores (no filter applied)
- Affected modules: audit logs, users, shifts, storage locations
- See [Store Scoping All Roles](./store-scoping-all-roles.md) for details

### Writes are scoped too, and `users` needed it

The bullet above is about *reads*, and it was over-claimed for `users`: a store-scoped caller with
`user.create`/`user.update` could name any `store_id` in the body and create or **move** a user
into or out of a store they do not belong to. `GetSubordinates`, `GetManager` and `GetOrgChart` also
returned foreign-store rows, so the "only see their own store" claim did not hold on those three
routes either. Enforced in Wave 3 of
[Master-Data Store Boundary Audit](./master-data-store-boundary-audit.md) — `bindStoreScopedUser`
for create/update/delete, `claimsStore *int` through the three read paths. See that doc for the
full behaviour table, including the deliberate exception: a **global** user (`store_id IS NULL`) is
an HQ identity that stays visible and editable by a store-scoped caller, but naming a store on one
is refused, because that would capture it rather than edit it.

**Practical rule for a non-superadmin:** the only store id you may create under, move a user into,
or see in a hierarchical read is your own.
