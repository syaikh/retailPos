# ADR: Brand Creation Permission Gate (Inline Brand Creation)

| Field | Value |
|-------|-------|
| Status | **Accepted** |
| Date | 2026-09-04 |
| Deciders | Tim Produk, Tim Engineering |
| Scope | Product form modal, brand creation, RBAC |
| Out of scope | Brand CRUD management page, brand import/export |

---

## 1. Context

When implementing inline brand creation in the product form modal, we need to decide which permission gates the "Create Brand" button. Two options exist:

- **Option A:** Reuse existing `product.create` permission (no backend change needed)
- **Option B:** Add a new `brand.create` permission (requires migration + permission seed + backend route change)

The backend's `POST /api/brands` endpoint already uses `permissions.ProductCreate` (line 37 of `internal/brand/handler.go`).

---

## 2. Decision

**Reuse `product.create` permission for inline brand creation.**

No new permission, migration, or backend route changes are needed.

---

## 3. Rationale

### 3.1 Workflow Alignment

In retail POS operations, brand creation is inherently tied to product management. When a staff member receives a new product from an unknown brand, they need to:

1. Create the brand (if it doesn't exist)
2. Create the product referencing that brand

These are the same task — "add new product to catalog." Forcing the user to leave the product form, navigate to Brand Management, create the brand, then return to the product form breaks the natural workflow.

### 3.2 Role Responsibilities

| Role | Can create products? | Should create brands? | Permission needed |
|------|---------------------|----------------------|-------------------|
| Cashier | No | No | — |
| Manager | Yes | Yes (same workflow) | `product.create` |
| Admin | Yes | Yes (same workflow) | `product.create` |
| Superadmin | Yes | Yes (same workflow) | `product.create` |

There is no real-world scenario where someone should create a brand but NOT a product, or vice versa. The responsibilities are co-located.

### 3.3 Avoids Permission Fragmentation

Adding `brand.create` creates a "permission explosion" problem. If brand creation gets its own permission, should there also be:

- `supplier.create`?
- `unit_of_measure.create`?
- `tax_class.create`?

Each new permission adds:
- A database migration
- Permission seed data
- Backend route changes
- Cognitive load for administrators managing roles

Tying brand creation to `product.create` keeps the permission model clean and maintainable.

### 3.4 Matches Existing API Contract

The backend already enforces `product.create` on `POST /api/brands`. Changing this would require:

1. New migration file
2. Permission seed in `000_squash.sql`
3. Route change in `brand/handler.go`
4. Frontend permission constant update

All for a permission split that doesn't reflect real business needs.

---

## 4. Consequences

### Positive

- **No backend changes needed** — frontend can implement inline creation immediately
- **Natural UX** — users create brands as part of the product creation flow
- **Clean permission model** — no fragmentation, easy for admins to understand
- **Consistent with category pattern** — both use the same inline creation UX

### Negative

- **Less granular control** — cannot separately control brand creation vs. product creation
- **Future flexibility** — if a business requirement emerges where brand creation should be restricted independently, a migration will be needed

### Mitigation

The negative consequences are unlikely in a retail POS context. If granularity is needed later, it can be added as a backward-compatible migration without breaking existing functionality.

---

## 5. Alternatives Considered

### Alternative A: Separate `brand.create` permission

**Rejected** because:
- Requires migration + seed + route changes
- No real-world use case for separating brand creation from product creation
- Adds maintenance burden for minimal benefit

### Alternative B: Use `category.create` permission

**Rejected** because:
- Semantically incorrect — brand ≠ category
- Backend already uses `product.create` for `POST /api/brands`
- Would create inconsistency between frontend gate and backend enforcement

---

## 6. References

- `internal/brand/handler.go:37` — `POST /api/brands` route uses `permissions.ProductCreate`
- `web/src/shared/constants/permissions.ts:33` — `Permissions.product.create`
- `docs/design/Inline-Category-Creation.md` — Reference pattern for inline creation
