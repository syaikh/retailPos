# Store-First Enforcement + Finance Role + Role Restructuring

> **Status:** Planned
> **Related:** [Role Permission Audit](./role-permission-audit.md)

## Problem

### 1. No Store Enforcement
The system can fully operate without a store configured. A user with `store_id = NULL`
can log in, create carts, complete sales — all with `store_id = NULL` in the database.
This breaks:

- **Revenue reporting** — can't attribute sales to a store
- **Tax compliance** — Indonesian tax (PPN) is per-store
- **Consignment settlement** — requires `store_id` from JWT
- **Multi-store isolation** — no boundary between store operations

### 2. Role Names Don't Match Business Reality
- "admin" is actually the store manager (the boss of a single store)
- "manager" is actually a shift supervisor (day-to-day operations)
- "staff" permissions are redundant (cashier can already do everything staff can)

### 3. No Separation of Duties
The admin/manager role bundles operational management with financial operations.
The person who manages staff shouldn't also pay suppliers.

## Business Rules

1. **A store must exist before any user, transaction, or consignment operation.**
2. **The person who creates a settlement cannot be the same person who pays it.**

## Role Restructuring

### New Role Hierarchy

| Old Name | New Name | Permissions | Scope | Business Context |
|----------|----------|-------------|-------|------------------|
| superadmin | **superadmin** | 85 | All stores | IT admin at HQ |
| admin | **manager** | 80 | Single store | Store manager (the boss) |
| manager | **supervisor** | 57 | Single store | Shift supervisor |
| finance | **finance** | 5 | Single store | Payment processing |
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

### Why These Names?

**"manager" instead of "admin"**
- In retail, the person running a store is called "Manager Toko"
- "Admin" sounds like a technical role, not a business role
- Matches real job titles in Indonesian retail

**"supervisor" instead of "manager"**
- Shift supervisors are called "Supervisor" or "Kepala Shift"
- They report to the Manager
- They handle day-to-day operations

**"inventory_staff" instead of "staff"**
- Current "staff" permissions are redundant (cashier can do everything)
- Inventory is a real business need (stock opname, receiving goods)
- Clear separation: sales vs inventory vs finance

### Separation of Duties

```
Supervisor creates settlement (consignment.settle)
    "We owe CV Lestari Rp 5,000,000"
         │
         ▼
Finance records payment (consignment.pay)
    "Paid Rp 5,000,000 via bank transfer"
```

**No single person can both create AND pay a settlement.**

## Current Gaps

| Layer | Gap | Risk |
|-------|-----|------|
| Migration `000_squash.sql` | Creates 0 stores | Fresh deploy has no store |
| User creation | `store_id` is optional | Users created without store |
| Seeded users | All have `store_id = NULL` | Can't operate properly |
| Auth middleware | Never rejects nil store_id | Users without store can access everything |
| Sale creation | `store_id` is nullable | Sales saved with NULL store |
| Manager role | Has `consignment.pay` | No separation of duties |
| Supervisor role | Missing `sale.create`, `store.view` | Can't ring sales, can't see stores |

## Implementation Plan

### Phase 1: Migration (044_store_first_and_finance_role.sql)

**1.1. Seed default store**
```sql
INSERT INTO stores (name, address, phone, is_active, created_at)
SELECT 'Default Store', 'Alamat toko default', '0000000000', true, NOW()
WHERE NOT EXISTS (SELECT 1 FROM stores);
```
Note: `stores` table has no `deleted_at` column — use simple existence check.

**1.2. Rename roles**
```sql
-- admin → manager
UPDATE roles SET name = 'manager', description = 'Manajer Toko — pengelolaan toko secara penuh'
WHERE name = 'admin';

-- manager → supervisor
UPDATE roles SET name = 'supervisor', description = 'Supervisor — pengawasan operasional harian'
WHERE name = 'manager';

-- staff → inventory_staff
UPDATE roles SET name = 'inventory_staff', description = 'Staf Inventaris — pengelolaan stok dan opname'
WHERE name = 'staff';
```

**1.3. Create finance role**
```sql
INSERT INTO roles (name, description)
VALUES ('finance', 'Keuangan — mencatat pembayaran supplier dan melihat laporan')
ON CONFLICT (name) DO NOTHING;
```

**1.4. Assign finance permissions**
- `consignment.pay` — record payments to suppliers
- `report.view` — view financial reports
- `audit.view` — view audit trail
- `sale.view` — view sales for reconciliation
- `store.view` — see which store they're paying for
- `dashboard.view` — see dashboard

**1.5. Update supervisor role**
- Add `sale.create` — supervisors can ring up sales at POS
- Add `store.view` — supervisors can see store dropdown

**1.6. Update inventory_staff permissions**
Replace current permissions with inventory-related ones:
- `inventory.adjust` — adjust stock levels
- `stock_opname.*` — full stock opname workflow
- `storage_location.*` — manage warehouse locations

**1.7. Backfill store_id**
```sql
UPDATE users SET store_id = (
    SELECT id FROM stores ORDER BY id LIMIT 1
)
WHERE store_id IS NULL
  AND role_id IN (SELECT id FROM roles WHERE name IN ('supervisor', 'manager', 'cashier', 'finance', 'inventory_staff'));
```

**1.8. Update seeder roles**
- `cmd/dummy/main.go`: Update role names in seeder

### Phase 2: Backend Guards

**2.1. Middleware: Require store_id for operational roles**
- File: `internal/middleware/auth.go`
- After JWT validation, check: if role is `cashier`, `supervisor`, `finance`, or `inventory_staff` and `store_id` is nil → reject with 403
- superadmin/manager bypass (they can operate across stores)

**2.2. User creation: Require store_id for cashier/supervisor/finance/inventory_staff**
- File: `internal/user/handler.go`
- In `CreateUser` validation:
  - If role is `cashier`, `supervisor`, `finance`, or `inventory_staff` → `store_id` must be non-nil and reference an active store
  - If role is `superadmin` or `manager` → `store_id` is optional

**2.3. Sale creation: Reject if store_id is nil**
- File: `internal/sale/cart_service.go`
- In `CreateOrGetOpenCart`:
  - If `storeID` is nil → return error "Store not configured for this user"

**2.4. User update: Prevent removing store_id from operational users**
- File: `internal/user/handler.go`
- In `UpdateUser`:
  - If role is `cashier`, `supervisor`, `finance`, or `inventory_staff` → cannot set `store_id` to NULL

### Phase 3: Code Updates

**3.1. Rename role constants**
- File: `internal/permissions/permissions.go`
- Update role name constants:
  - `Admin = "admin"` → `Manager = "manager"`
  - `Manager = "manager"` → `Supervisor = "supervisor"`
  - `Staff = "staff"` → `InventoryStaff = "inventory_staff"`

**3.2. Update all role references**
- Search for `"admin"`, `"manager"`, `"staff"` in Go files
- Update to `"manager"`, `"supervisor"`, `"inventory_staff"`

**3.3. Seeder fixes** (already done in commit `9f4e1c7b`)
- Users get `store_id` assigned
- Products get `ownership_type = 'consignment'`

### Phase 4: Frontend

**4.1. User creation form**
- File: `web/src/modules/admin/components/UserFormModal.svelte`
- Add `store_id` to form state (currently missing)
- Show store dropdown as **required** for cashier/supervisor/finance/inventory_staff roles
- Show store dropdown as **optional** for manager/superadmin roles
- Fetch active stores via `GET /stores/active`

**4.2. Finance role in user creation**
- Add "finance" to role dropdown
- Description: "Keuangan — mencatat pembayaran supplier"

**4.3. Update role names in frontend**
- Update role labels in user management, login, etc.
- Update role descriptions

**4.4. Update i18n**
- Add new role name translations
- Update role descriptions

### Files to Change

| File | Change |
|------|--------|
| `database/migrations/044_store_first_and_finance_role.sql` | New migration: rename roles + finance role |
| `internal/permissions/permissions.go` | Update role name constants |
| `internal/middleware/auth.go` | Reject nil store_id for cashier/supervisor/finance/inventory_staff |
| `internal/user/handler.go` | Validate store_id in CreateUser/UpdateUser |
| `internal/sale/cart_service.go` | Reject sale if store_id is nil |
| `cmd/dummy/main.go` | Update role names in seeder |
| `web/src/modules/admin/components/UserFormModal.svelte` | Store dropdown + finance role |
| `web/src/shared/i18n/` | Update role name translations |

### Testing

- **Unit tests**: User creation without store_id → 400 for cashier/supervisor/finance/inventory_staff
- **Unit tests**: Sale creation without store_id → 400
- **Unit tests**: Middleware rejects nil store_id for cashier/supervisor/finance/inventory_staff → 403
- **E2E tests**: Full flow — create store → create user → assign store → login → create sale
- **E2E tests**: Consignment — supervisor settles → finance pays → verify separation

### Migration Ordering

Apply `044_store_first_and_finance_role.sql` **before** deploying the binary that
validates store_id. The migration renames roles and backfills store_id for existing
users to prevent breaking changes.

### Rollback

If issues arise:
1. Remove middleware store_id check
2. Remove user creation store_id validation
3. Remove sale store_id validation
4. Keep the default store, finance role, and user backfill (harmless)
5. Rename roles back if needed (admin, manager, staff)
