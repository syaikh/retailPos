# PERMISSION MATRIX FINAL — Current State

**Status:** UPDATED (business-perspective audit + role rename 044/045/046)
**Date:** 2026-09-14
**Source of truth:** Live database `retail_pos` (`permissions`, `role_permissions`) — query result 2026-09-14. Seed definition: `database/migrations/000_squash.sql` + migrations `039`, `044`, `045`, `046`.

> Roles are now named after retail job titles (migration 044): `superadmin`, `manager` (was `admin`), `supervisor` (was `manager`), `cashier`, `inventory_staff` (was `staff`), and `finance` (new). Default usernames were aligned to role names in migration 045.

---

## 1. Background

- Total permissions in DB: **86** (0 ungranted).
- Total grants after migrations 039/044/046/047: **267** (superadmin=85, manager=80, supervisor=59, cashier=19, inventory_staff=17, finance=7).
- Matrix covers 72 Sprint 0 permissions + 14 additional permissions (consignment.*, app_settings.*, sale.detail, receipt.print, sale.lookup, audit.export, product.history.view, product.cost.view, shift.cash_movement).

## 2. COMPLETE MATRIX (86 × 6 roles)

Legend: ✅ = granted, — = not granted.

### 2.1 System & Account

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 1 | `dashboard.view` | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| 2 | `user.view` | ✅ | ✅ | — | — | — | — |
| 3 | `user.create` | ✅ | ✅ | — | — | — | — |
| 4 | `user.update` | ✅ | ✅ | — | — | — | — |
| 5 | `user.delete` | ✅ | — | — | — | — | — |
| 6 | `role.view` | ✅ | ✅ | — | — | — | — |
| 7 | `role.create` | ✅ | ✅ | — | — | — | — |
| 8 | `role.update` | ✅ | — | — | — | — | — |
| 9 | `role.delete` | ✅ | — | — | — | — | — |
| 19 | `audit.view` | ✅ | ✅ | — | — | — | ✅ |
| 85 | `audit.export` | ✅ | ✅ | — | — | — | — |
| 18 | `report.view` | ✅ | ✅ | ✅ | — | — | ✅ |
| 80 | `app_settings.view` | ✅ | ✅ | — | — | — | — |
| 81 | `app_settings.update` | ✅ | — | — | — | — | — |

### 2.2 Product & Category

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 10 | `product.view` | ✅ | ✅ | ✅ | ✅ | — | — |
| 11 | `product.create` | ✅ | ✅ | ✅ | — | — | — |
| 12 | `product.update` | ✅ | ✅ | ✅ | — | — | — |
| 13 | `product.delete` | ✅ | ✅ | — | — | — | — |
| 31 | `product.export` | ✅ | ✅ | — | — | — | — |
| 32 | `product.import` | ✅ | ✅ | — | — | — | — |
| 73 | `product.history.view` | ✅ | ✅ | — | — | — | — |
| 74 | `product.cost.view` | ✅ | ✅ | ✅ | — | — | — |
| 14 | `category.view` | ✅ | ✅ | ✅ | ✅ | — | — |
| 15 | `category.create` | ✅ | ✅ | ✅ | — | — | — |
| 20 | `category.update` | ✅ | ✅ | ✅ | — | — | — |
| 21 | `category.delete` | ✅ | ✅ | ✅ | — | — | — |
| 33 | `category.export` | ✅ | ✅ | — | — | — | — |
| 34 | `category.import` | ✅ | ✅ | — | — | — | — |

### 2.3 Sales & Shift

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 16 | `sale.view` | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| 17 | `sale.create` | ✅ | ✅ | ✅ | ✅ | — | — |
| 49 | `sale.park` | ✅ | ✅ | ✅ | ✅ | — | — |
| 82 | `sale.lookup` | — | — | — | ✅ | — | — |
| 83 | `sale.detail` | ✅ | ✅ | ✅ | ✅ | — | — |
| 84 | `receipt.print` | ✅ | ✅ | ✅ | ✅ | — | — |
| 22 | `shift.view` | ✅ | ✅ | ✅ | ✅ | — | — |
| 23 | `shift.create` | ✅ | ✅ | ✅ | ✅ | — | — |
| 86 | `shift.cash_movement` | ✅ | ✅ | ✅ | ✅ | — | — |
| 24 | `shift.review` | ✅ | ✅ | ✅ | — | — | — |
| 25 | `shift.audit` | ✅ | ✅ | ✅ | — | — | — |

### 2.4 Customer & Pricing

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 26 | `customer.view` | ✅ | ✅ | ✅ | ✅ | — | — |
| 27 | `customer.create` | ✅ | ✅ | ✅ | — | — | — |
| 28 | `customer.update` | ✅ | ✅ | ✅ | — | — | — |
| 29 | `customer.delete` | ✅ | ✅ | ✅ | — | — | — |
| 35 | `customer.export` | ✅ | ✅ | ✅ | — | — | — |
| 36 | `customer.import` | ✅ | ✅ | ✅ | — | — | — |
| 37 | `pricing.view` | ✅ | ✅ | ✅ | ✅ | — | — |
| 38 | `pricing.create` | ✅ | ✅ | ✅ | — | — | — |
| 39 | `pricing.update` | ✅ | ✅ | ✅ | — | — | — |
| 40 | `pricing.delete` | ✅ | ✅ | ✅ | — | — | — |
| 45 | `customer_group.view` | ✅ | ✅ | ✅ | ✅ | — | — |
| 46 | `customer_group.create` | ✅ | ✅ | ✅ | — | — | — |
| 47 | `customer_group.update` | ✅ | ✅ | ✅ | — | — | — |
| 48 | `customer_group.delete` | ✅ | ✅ | ✅ | — | — | — |

### 2.5 Inventory

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 30 | `inventory.adjust` | ✅ | ✅ | ✅ | — | ✅ | — |

### 2.6 Store & Storage

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 41 | `store.view` | ✅ | ✅ | — | — | — | — |
| 42 | `store.create` | ✅ | ✅ | — | — | — | — |
| 43 | `store.update` | ✅ | ✅ | — | — | — | — |
| 44 | `store.delete` | ✅ | ✅ | — | — | — | — |
| 69 | `storage_location.view` | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| 70 | `storage_location.create` | ✅ | ✅ | — | — | ✅ | — |
| 71 | `storage_location.update` | ✅ | ✅ | — | — | ✅ | — |
| 72 | `storage_location.delete` | ✅ | ✅ | — | — | ✅ | — |

### 2.7 Purchase Order

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 51 | `purchase_order.view` | ✅ | ✅ | ✅ | — | — | — |
| 52 | `purchase_order.create` | ✅ | ✅ | ✅ | — | — | — |
| 53 | `purchase_order.update` | ✅ | ✅ | ✅ | — | — | — |
| 54 | `purchase_order.delete` | ✅ | — | — | — | — | — |
| 55 | `purchase_order.confirm` | ✅ | ✅ | ✅ | — | — | — |
| 56 | `purchase_order.receive` | ✅ | ✅ | ✅ | — | — | — |
| 50 | `purchase_order.cancel` | ✅ | ✅ | ✅ | — | — | — |

### 2.8 Stock Opname

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 57 | `stock_opname.view` | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| 58 | `stock_opname.create` | ✅ | ✅ | ✅ | — | ✅ | — |
| 59 | `stock_opname.assign` | ✅ | ✅ | ✅ | — | ✅ | — |
| 60 | `stock_opname.count` | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| 61 | `stock_opname.submit` | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| 62 | `stock_opname.recount` | ✅ | ✅ | ✅ | — | ✅ | — |
| 63 | `stock_opname.cancel` | ✅ | ✅ | ✅ | — | ✅ | — |
| 64 | `stock_opname.export` | ✅ | ✅ | ✅ | — | ✅ | — |
| 65 | `stock_opname.verify` | ✅ | ✅ | ✅ | — | ✅ | — |
| 66 | `stock_opname.post` | ✅ | ✅ | ✅ | — | ✅ | — |
| 67 | `stock_opname.close` | ✅ | ✅ | ✅ | — | ✅ | — |
| 68 | `stock_opname.report` | ✅ | ✅ | ✅ | — | ✅ | — |

### 2.9 Consignment

| # | Permission | SA | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|---|-----------|----|---------|------------|---------|-----------------|---------|
| 75 | `consignment.view` | ✅ | ✅ | ✅ | — | — | ✅ |
| 76 | `consignment.create` | ✅ | ✅ | ✅ | — | — | — |
| 77 | `consignment.update` | ✅ | ✅ | ✅ | — | — | — |
| 78 | `consignment.settle` | ✅ | ✅ | ✅ | — | — | — |
| 79 | `consignment.pay` | ✅ | ✅ | — | — | — | ✅ |

## 3. ROLE SUMMARY

| Role | Permissions | Notes |
|------|------------|-------|
| superadmin | **85** | Full access (owner/god mode; only `sale.lookup` not granted — cashier-only) |
| manager | **80** | Operational — missing: user.delete, role.update, role.delete, app_settings.update, purchase_order.delete, sale.lookup |
| supervisor | **59** | Store operator — full product/category/customer/pricing/PO/stock opname management; no user/role management, no product delete/import/export, no consignment.pay |
| cashier | **19** | POS & basic tasks — sales, shifts, stock count, view-only master data, sale.lookup (only role besides direct grant) |
| inventory_staff | **17** | Inventory-focused — stock opname full lifecycle, storage locations, inventory.adjust (no product/category view) |
| finance | **5** | Payments & reporting — consignment.pay, consignment.view, report.view, audit.view, sale.view, dashboard.view |
| **TOTAL** | **267** | |

## 4. CHANGE REGISTER

| ID | Migration | Change |
|----|-----------|--------|
| R1 | `023_sprint0_finalize_permissions.sql` | REVOKE `staff.product.update`, `staff.inventory.adjust` |
| R2 | `038_grant_audit_view_to_admin.sql` | GRANT `audit.view` to admin |
| R3 | `039_business_permission_audit.sql` | GRANT 12 to manager, 4 to cashier, 1 to staff (see §5) |
| R4 | `044_store_first_and_finance_role.sql` | Rename roles (admin→manager, manager→supervisor, staff→inventory_staff), create `finance` (6 perms), GRANT supervisor sale.create + store.view, replace inventory_staff permissions, backfill store_id |
| R5 | `045_rename_usernames.sql` | Align default usernames to role names (admin→manager, manager→supervisor, staff→inventory_staff) |
| R6 | `046_manager_consignment_pay.sql` | GRANT `consignment.pay` to manager (bug fix — settle without pay was illogical) |
| R7 | `047_finance_consignment_view.sql` | GRANT `consignment.view` to finance (bug fix — finance could pay settlements but could not view them) |

### R3 Detail (migration 039, historical — roles were renamed later in 044)

#### Old `manager` role (+12) → now `supervisor`
| Permission | Reason |
|-----------|--------|
| `product.create` | Can edit but not add — operational bottleneck |
| `category.update` | Can create but not edit — inconsistent |
| `category.delete` | Can create but not delete — inconsistent |
| `customer.delete` | Can create/edit but not delete |
| `customer.export` | Cannot export customer data |
| `customer.import` | Cannot bulk-import customers |
| `customer_group.create` | View-only — cannot manage loyalty/pricing groups |
| `customer_group.update` | View-only |
| `customer_group.delete` | View-only |
| `pricing.delete` | Can create/update but not delete pricing rules |
| `stock_opname.count` | Can create/assign/verify but not count — lifecycle gap |
| `stock_opname.submit` | Same as count |

#### Cashier (+4)
| Permission | Reason |
|-----------|--------|
| `category.view` | Product filter in POS broken without categories |
| `pricing.view` | Cannot see active promotions/pricing at checkout |
| `customer_group.view` | Cannot see loyalty tier pricing |
| `dashboard.view` | Cannot see daily sales summary at shift start |

#### Old `staff` role (+1) → now `inventory_staff`
| Permission | Reason |
|-----------|--------|
| `category.view` | Same UX issue as cashier |

### R4 Detail (migration 044 — role restructuring)

| Change | Detail |
|--------|--------|
| Roles renamed | `admin`→`manager`, `manager`→`supervisor`, `staff`→`inventory_staff` (order-aware, idempotent) |
| `finance` role created | 5 permissions: `consignment.pay`, `report.view`, `audit.view`, `sale.view`, `dashboard.view` |
| Supervisor +1 | GRANT `sale.create` — supervisors ring up POS sales |
| Inventory staff replaced | REMOVE all prior grants; GRANT 17 inventory permissions: `inventory.adjust`, stock_opname.* (12), storage_location.* (4) |
| Store backfill | Existing users with supervisor/manager/cashier/finance/inventory_staff roles get `store_id` = default store |

### R6 Detail (migration 046 — manager consignment.pay)

| Permission | Reason |
|-----------|--------|
| `consignment.pay` for manager | Migration 001 granted manager consignment.view/create/update/settle but omitted pay — settlement screen hid the "Pay" button for managers who can settle |

## 5. BEHAVIOR DELTA REGISTER

| ID | Delta | Detail | Classification |
|----|-------|--------|----------------|
| D1 | Inventory staff can adjust stock | 044 grants `inventory.adjust` to inventory_staff | **Change** |
| D2 | Bug 403 "Add Product" for manager fixed | Add Product button now only appears for roles with `product.create` | **Bug fix** |
| D3 | inventory_staff loses product/category view | 044 replaced staff grants with pure inventory set — least privilege for stock ops | **Intentional** |
| D4 | Manager can view audit logs | Manager (was admin) retains audit.view/export | **Retained** |
| D5 | Supervisor can add products & ring sales | 039 gave product.create; 044 gave sale.create | **Enhancement** |
| D6 | Cashier can view categories, pricing, customer groups | Product filter and pricing info visible in POS | **Enhancement** |
| D7 | Manager can pay consignments | 046 grants consignment.pay — settle + pay consistent | **Bug fix** |
| D8 | Finance role exists | Read-only payments/reporting scope — no master-data or inventory grants | **Enhancement** |

## 6. VERIFICATION

```sql
-- Total grants per role (after migrations 039/044/046/047)
SELECT r.name, COUNT(*) AS grants
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
GROUP BY r.name
ORDER BY r.name;
```

**Expected:** cashier=19, finance=7, inventory_staff=17, manager=80, superadmin=85, supervisor=59.

```sql
-- Total permission codes in DB
SELECT COUNT(*) FROM permissions;
```

**Expected:** 86.

---

*Document updated on 2026-09-14 based on business-perspective audit + role rename migrations 044/045/046.*