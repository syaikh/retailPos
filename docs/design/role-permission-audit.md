# Role Permission Audit

> **Date:** 2026-09-11
> **Status:** Analysis complete — finance role + role restructuring recommended
> **Related:** [Store-First and Finance Role](./store-first-and-finance-role.md)

## Executive Summary

The current role hierarchy has two problems:

1. **Separation of duties gap** — the admin role (80 permissions) bundles operational management with financial operations. The person who manages staff shouldn't also pay suppliers.

2. **Role names don't match business reality** — "admin" is actually the store manager, "manager" is actually a shift supervisor, "staff" permissions are redundant.

**Recommendation:**
1. Introduce a **finance** role to separate financial operations from store operations
2. Rename roles to match real retail job titles

---

## New Role Structure

| Old Name | New Name | Permissions | Scope | Business Context |
|----------|----------|-------------|-------|------------------|
| superadmin | **superadmin** | 85 | All stores | IT admin at HQ |
| admin | **manager** | 80 | Single store | Store manager (the boss) |
| manager | **supervisor** | 57+2 | Single store | Shift supervisor |
| finance | **finance** | 5+1 | Single store | Payment processing |
| cashier | **cashier** | 19 | Single store | Sales |
| staff | **inventory_staff** | 6 | Single store | Stock management |

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

### Before (Current)

| Role | Permissions | Scope |
|------|-------------|-------|
| superadmin | 85 | System-wide, all stores |
| admin | 80 | Store-level (mostly) |
| manager | 57 | Store-level (operational) |
| cashier | 19 | Store-level (sales) |
| staff | 6 | Store-level (minimal) |

### After (Proposed)

| Role | Permissions | Scope | Changes |
|------|-------------|-------|---------|
| superadmin | 85 | All stores | unchanged |
| manager | 80 | Single store | renamed from "admin" |
| supervisor | 59 | Single store | renamed from "manager", +sale.create, +store.view |
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

### After (Fixed)

```
Supervisor creates settlement (consignment.settle)
    "We owe CV Lestari Rp 5,000,000"
         │
         ▼
Finance records payment (consignment.pay)
    "Paid Rp 5,000,000 via bank transfer"
```

**No single person can both create AND pay a settlement.**

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
| `store.create` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `store.view` | ✅ | ✅ | ➕ | ✅ | ❌ | ❌ |
| `store.update` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `store.delete` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |

### Product Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `product.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `product.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.export` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.import` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.history.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `product.cost.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### Category Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `category.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `category.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.export` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `category.import` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### Customer Management

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `customer.view` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `customer.create` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
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
| `consignment.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
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
| `receipt.print` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |

### Inventory

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `inventory.adjust` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |

### Stock Opname

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `stock_opname.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.assign` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.count` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.recount` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `stock_opname.submit` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
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
| `storage_location.view` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `storage_location.create` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `storage_location.update` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |
| `storage_location.delete` | ✅ | ✅ | ✅ | ❌ | ❌ | ➕ |

### Reports & Dashboard

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `dashboard.view` | ✅ | ✅ | ✅ | ➕ | ❌ | ❌ |
| `report.view` | ✅ | ✅ | ✅ | ➕ | ❌ | ❌ |

### Audit

| Permission | superadmin | manager | supervisor | finance | cashier | inventory_staff |
|------------|------------|---------|------------|---------|---------|-----------------|
| `audit.view` | ✅ | ✅ | ✅ | ➕ | ❌ | ❌ |
| `audit.export` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

---

## Summary of Changes

### Permissions Added

| Role | Permission | Reason |
|------|------------|--------|
| supervisor | `sale.create` | Supervisors can ring up sales at POS |
| supervisor | `store.view` | Supervisors need to see store dropdown |
| finance | `consignment.pay` | Finance records payments to suppliers |
| finance | `report.view` | Finance views financial reports |
| finance | `audit.view` | Finance views audit trail |
| finance | `sale.view` | Finance views sales for reconciliation |
| finance | `store.view` | Finance sees which store they're paying for |
| finance | `dashboard.view` | Finance sees dashboard |
| inventory_staff | `inventory.adjust` | Inventory staff adjusts stock levels |
| inventory_staff | `stock_opname.*` (12) | Inventory staff manages stock opname |
| inventory_staff | `storage_location.*` (4) | Inventory staff manages warehouse locations |

### Permissions Removed from Manager (Moved to Finance)

| Permission | Now Has |
|------------|---------|
| `consignment.pay` | finance |

### Roles Renamed

| Old Name | New Name |
|----------|----------|
| admin | manager |
| manager | supervisor |
| staff | inventory_staff |

---

## Recommendation

Implement the role restructuring as part of the store-first enforcement migration.
This ensures:
1. Clean role names that match business reality
2. Proper separation of duties (finance vs operations)
3. Store-first enforcement for all operational roles
