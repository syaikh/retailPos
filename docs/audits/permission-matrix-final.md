# PERMISSION MATRIX FINAL — Current State

**Status:** UPDATED (business-perspective audit)
**Date:** 2026-09-02
**Source of truth:** Live database `retail_pos` (`permissions`, `role_permissions`) — query result 2026-09-02. Seed definition: `database/migrations/000_squash.sql` + migrations `001`, `005`, `007`, `031`, `032`, `036`, `038`, `039`.

---

## 1. Background

- Total permissions in DB: **85** (0 ungranted).
- Total grants after migration 039: **243** (superadmin=84, admin=79, manager=56, cashier=18, staff=6).
- Matrix covers 72 Sprint 0 permissions + 13 additional permissions (consignment.*, app_settings.*, sale.detail, receipt.print, sale.lookup, audit.export, product.history.view, product.cost.view).

## 2. COMPLETE MATRIX (85 × 5 roles)

Legend: ✅ = granted, — = not granted. ◀ marks changes from migration 039.

### 2.1 System & Account

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 1 | `dashboard.view` | ✅ | ✅ | ✅ | ✅ ◀ | — |
| 2 | `user.view` | ✅ | ✅ | — | — | — |
| 3 | `user.create` | ✅ | ✅ | — | — | — |
| 4 | `user.update` | ✅ | ✅ | — | — | — |
| 5 | `user.delete` | ✅ | — | — | — | — |
| 6 | `role.view` | ✅ | ✅ | — | — | — |
| 7 | `role.create` | ✅ | ✅ | — | — | — |
| 8 | `role.update` | ✅ | — | — | — | — |
| 9 | `role.delete` | ✅ | — | — | — | — |
| 10 | `audit.view` | ✅ | ✅ ◀ | — | — | — |
| 11 | `audit.export` | ✅ | ✅ | — | — | — |
| 12 | `report.view` | ✅ | ✅ | ✅ | — | — |
| 13 | `app_settings.view` | ✅ | ✅ | — | — | — |
| 14 | `app_settings.update` | ✅ | — | — | — | — |

### 2.2 Product & Category

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 15 | `product.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| 16 | `product.create` | ✅ | ✅ | ✅ ◀ | — | — |
| 17 | `product.update` | ✅ | ✅ | ✅ | — | — |
| 18 | `product.delete` | ✅ | ✅ | — | — | — |
| 19 | `product.export` | ✅ | ✅ | — | — | — |
| 20 | `product.import` | ✅ | ✅ | — | — | — |
| 21 | `product.history.view` | ✅ | ✅ | — | — | — |
| 22 | `product.cost.view` | ✅ | ✅ | ✅ | — | — |
| 23 | `category.view` | ✅ | ✅ | ✅ | ✅ ◀ | ✅ ◀ |
| 24 | `category.create` | ✅ | ✅ | ✅ | — | — |
| 25 | `category.update` | ✅ | ✅ | ✅ ◀ | — | — |
| 26 | `category.delete` | ✅ | ✅ | ✅ ◀ | — | — |
| 27 | `category.export` | ✅ | ✅ | — | — | — |
| 28 | `category.import` | ✅ | ✅ | — | — | — |

### 2.3 Sales & Shift

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 29 | `sale.view` | ✅ | ✅ | ✅ | ✅ | — |
| 30 | `sale.create` | ✅ | ✅ | — | ✅ | — |
| 31 | `sale.park` | ✅ | ✅ | ✅ | ✅ | — |
| 32 | `sale.detail` | ✅ | ✅ | ✅ | ✅ | — |
| 33 | `sale.lookup` | — | — | — | ✅ | — |
| 34 | `receipt.print` | ✅ | ✅ | ✅ | ✅ | — |
| 35 | `shift.view` | ✅ | ✅ | ✅ | ✅ | — |
| 36 | `shift.create` | ✅ | ✅ | ✅ | ✅ | — |
| 37 | `shift.review` | ✅ | ✅ | ✅ | — | — |
| 38 | `shift.audit` | ✅ | ✅ | ✅ | — | — |

### 2.4 Customer & Pricing

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 39 | `customer.view` | ✅ | ✅ | ✅ | ✅ | — |
| 40 | `customer.create` | ✅ | ✅ | ✅ | — | — |
| 41 | `customer.update` | ✅ | ✅ | ✅ | — | — |
| 42 | `customer.delete` | ✅ | ✅ | ✅ ◀ | — | — |
| 43 | `customer.export` | ✅ | ✅ | ✅ ◀ | — | — |
| 44 | `customer.import` | ✅ | ✅ | ✅ ◀ | — | — |
| 45 | `pricing.view` | ✅ | ✅ | ✅ | ✅ ◀ | — |
| 46 | `pricing.create` | ✅ | ✅ | ✅ | — | — |
| 47 | `pricing.update` | ✅ | ✅ | ✅ | — | — |
| 48 | `pricing.delete` | ✅ | ✅ | ✅ ◀ | — | — |
| 49 | `customer_group.view` | ✅ | ✅ | ✅ | ✅ ◀ | — |
| 50 | `customer_group.create` | ✅ | ✅ | ✅ ◀ | — | — |
| 51 | `customer_group.update` | ✅ | ✅ | ✅ ◀ | — | — |
| 52 | `customer_group.delete` | ✅ | ✅ | ✅ ◀ | — | — |

### 2.5 Inventory

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 53 | `inventory.adjust` | ✅ | ✅ | ✅ | — | — |

### 2.6 Store & Storage

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 54 | `store.view` | ✅ | ✅ | — | — | — |
| 55 | `store.create` | ✅ | ✅ | — | — | — |
| 56 | `store.update` | ✅ | ✅ | — | — | — |
| 57 | `store.delete` | ✅ | ✅ | — | — | — |
| 58 | `storage_location.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| 59 | `storage_location.create` | ✅ | ✅ | — | — | — |
| 60 | `storage_location.update` | ✅ | ✅ | — | — | — |
| 61 | `storage_location.delete` | ✅ | ✅ | — | — | — |

### 2.7 Purchase Order

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 62 | `purchase_order.view` | ✅ | ✅ | ✅ | — | — |
| 63 | `purchase_order.create` | ✅ | ✅ | ✅ | — | — |
| 64 | `purchase_order.update` | ✅ | ✅ | ✅ | — | — |
| 65 | `purchase_order.delete` | ✅ | — | — | — | — |
| 66 | `purchase_order.confirm` | ✅ | ✅ | ✅ | — | — |
| 67 | `purchase_order.receive` | ✅ | ✅ | ✅ | — | — |
| 68 | `purchase_order.cancel` | ✅ | ✅ | ✅ | — | — |

### 2.8 Stock Opname

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 69 | `stock_opname.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| 70 | `stock_opname.create` | ✅ | ✅ | ✅ | — | — |
| 71 | `stock_opname.assign` | ✅ | ✅ | ✅ | — | — |
| 72 | `stock_opname.count` | ✅ | ✅ | ✅ ◀ | ✅ | ✅ |
| 73 | `stock_opname.submit` | ✅ | ✅ | ✅ ◀ | ✅ | ✅ |
| 74 | `stock_opname.recount` | ✅ | ✅ | ✅ | — | — |
| 75 | `stock_opname.cancel` | ✅ | ✅ | ✅ | — | — |
| 76 | `stock_opname.export` | ✅ | ✅ | ✅ | — | — |
| 77 | `stock_opname.verify` | ✅ | ✅ | ✅ | — | — |
| 78 | `stock_opname.post` | ✅ | ✅ | ✅ | — | — |
| 79 | `stock_opname.close` | ✅ | ✅ | ✅ | — | — |
| 80 | `stock_opname.report` | ✅ | ✅ | ✅ | — | — |

### 2.9 Consignment

| # | Permission | SA | Admin | Manager | Cashier | Staff |
|---|-----------|----|-------|---------|---------|-------|
| 81 | `consignment.view` | ✅ | ✅ | ✅ | — | — |
| 82 | `consignment.create` | ✅ | ✅ | ✅ | — | — |
| 83 | `consignment.update` | ✅ | ✅ | ✅ | — | — |
| 84 | `consignment.settle` | ✅ | ✅ | ✅ | — | — |
| 85 | `consignment.pay` | ✅ | ✅ | — | — | — |

## 3. ROLE SUMMARY

| Role | Permissions | Notes |
|------|------------|-------|
| superadmin | **84** | Full access (owner/god mode) |
| admin | **79** | Operational — missing: user.delete, role.update, role.delete, app_settings.update, purchase_order.delete |
| manager | **56** | Store operator — full product/category/customer/pricing/PO/stock opname management |
| cashier | **18** | POS & basic tasks — sales, shifts, stock count, view-only master data |
| staff | **6** | View-only + stock count |
| **TOTAL** | **243** | |

## 4. CHANGE REGISTER

| ID | Migration | Change |
|----|-----------|--------|
| R1 | `023_sprint0_finalize_permissions.sql` | REVOKE `staff.product.update`, `staff.inventory.adjust` |
| R2 | `038_grant_audit_view_to_admin.sql` | GRANT `audit.view` to admin |
| R3 | `039_business_permission_audit.sql` | GRANT 12 to manager, 4 to cashier, 1 to staff (see §5) |

## 5. CHANGE DETAILS (Migration 039)

### Manager (+12)
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

### Cashier (+4)
| Permission | Reason |
|-----------|--------|
| `category.view` | Product filter in POS broken without categories |
| `pricing.view` | Cannot see active promotions/pricing at checkout |
| `customer_group.view` | Cannot see loyalty tier pricing |
| `dashboard.view` | Cannot see daily sales summary at shift start |

### Staff (+1)
| Permission | Reason |
|-----------|--------|
| `category.view` | Same UX issue as cashier |

## 6. BEHAVIOR DELTA REGISTER

| ID | Delta | Detail | Classification |
|----|-------|--------|----------------|
| D1 | Staff loses "Adjust Stock" button | After 023 staff no longer has `inventory.adjust` | **Intentional** |
| D2 | Bug 403 "Add Product" for manager fixed | Add Product button now only appears for roles with `product.create` | **Bug fix** |
| D3 | Staff still cannot edit products | Consistent with least privilege | **Consistent** |
| D4 | Admin can view audit logs | Previously admin had export but not view | **Bug fix** |
| D5 | Manager can add products | Add Product button now appears for manager | **Enhancement** |
| D6 | Manager can edit/delete categories | Consistent — if can create, must be able to edit/delete | **Enhancement** |
| D7 | Cashier can view categories, pricing, customer groups | Product filter and pricing info visible in POS | **Enhancement** |
| D8 | Manager can participate in stock opname | Manager can now count and submit stock opname | **Enhancement** |

## 7. VERIFICATION

```sql
-- Total grants per role (after migration 039)
SELECT r.name, COUNT(*) AS grants
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
GROUP BY r.name
ORDER BY r.name;
```

**Expected:** superadmin=84, admin=79, manager=56, cashier=18, staff=6.

```sql
-- Total permission codes in DB
SELECT COUNT(*) FROM permissions;
```

**Expected:** 85.

---

*Document updated on 2026-09-02 based on business-perspective audit.*
