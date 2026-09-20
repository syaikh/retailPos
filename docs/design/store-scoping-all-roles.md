# Store Scoping: All Non-Superadmin Roles

> **Status:** Planned
> **Created:** 2026-09-19
> **Related:** [Store-First Enforcement](./store-first-and-finance-role.md)

## Problem

Manager bypasses `RequireStoreID` middleware and can see cross-store data in audit logs, users, shifts, and storage locations. The design doc says manager is "Single store" scoped, but the code doesn't enforce it.

## Business Rule

**ALL roles except superadmin must be scoped to their assigned store.**

- superadmin → all stores (HQ IT admin)
- manager → single store (Manager Toko)
- supervisor → single store (shift lead)
- finance → single store (payments)
- cashier → single store (sales)
- inventory_staff → single store (stock)

## Changes Required

### 1. Middleware: Enforce store_id for manager
**File**: `internal/middleware/auth.go:139-164`

Remove manager from the bypass list. Only superadmin should bypass.

```go
// Current: superadmin AND manager bypass
if roleStr == permissions.RoleSuperadmin || roleStr == permissions.RoleManager {

// New: only superadmin bypasses
if roleStr == permissions.RoleSuperadmin {
```

Update `OperationalRoles` or add a store-scoped check. Manager is not "operational" (they're the store boss), so a cleaner approach is to check `role != superadmin` directly.

### 2. Audit Logs: Filter by store_id
**Files**: `internal/audit/handler.go:37-72`, `internal/audit/repository.go:161-253`

- Handler reads `shared.GetStoreID(c)` and passes to service/repository
- Repository adds `AND (al.store_id IS NULL OR al.store_id = $N)` filter
- Superadmin passes `nil` → no filter (sees all stores)
- Everyone else passes their store_id

### 3. Users: Filter by store_id
**Files**: `internal/user/handler.go:134-160`, `internal/user/repository.go:140-219`

- Handler reads `shared.GetStoreID(c)` and passes to service/repository
- Repository adds `AND (u.store_id IS NULL OR u.store_id = $N)` filter
- Superadmin passes `nil` → no filter
- Manager sees only users in their store

### 4. Shifts: Filter by store_id
**Files**: `internal/shift/handler.go:207-237`, `internal/shift/repository.go:348-414`

- Handler reads `shared.GetStoreID(c)` and passes to service/repository
- Repository adds `AND (s.store_id IS NULL OR s.store_id = $N)` filter
- Superadmin passes `nil` → no filter
- Manager sees only shifts in their store

### 5. Storage Locations: Filter by store_id
**Files**: `internal/storagelocation/handler.go:49`, `internal/storagelocation/repository.go:48-95`

- Handler reads `shared.GetStoreID(c)` and passes to service/repository
- Repository adds `AND (sl.store_id IS NULL OR sl.store_id = $N)` filter
- Superadmin passes `nil` → no filter

### 6. Docs Update
- `docs/design/store-first-and-finance-role.md` — update business rules, middleware description, manager scope
- `docs/design/role-permission-audit.md` — update permission matrix

## Migration

None required. All changes are in Go handlers/repositories.

## Testing

- Unit tests: verify store_id filtering in each handler/repository
- E2E tests: verify manager sees only their store's data
- Regression: verify superadmin still sees all stores

## Risk

Low. This is a narrowing of data visibility, not a widening. Existing operational roles (cashier/supervisor/finance/inventory_staff) are already scoped — this change adds manager to the same pattern.
