# ADR: Frontend Module Import Boundaries — No ESLint Enforcement (For Now)

| Field         | Nilai |
|---------------|-------|
| Status        | **On Hold** — pending review |
| Date          | 2026-08-20 |
| Deciders      | Tim Engineering |
| Scope         | `web/src/modules/` cross-module import policy, enforcement mechanism |
| Related       | `docs/adr/ADR_Modular_Monolith_Module_Boundaries.md` (backend equivalent) |

---

## 1. Context

The frontend follows a modular monolith structure under `web/src/`:

```
src/
  app/       — App shell (router, layouts, providers)
  modules/   — Domain modules (auth, product, pos, sales, stock-opname, etc.)
  shared/    — Cross-cutting code (composables, UI, i18n, utils)
```

The documented convention (from `MIGRATION-COMPLETION-SUMMARY.md`) states:

> *"Each module exposes only through `modules/<name>/index.ts`. No cross-module internal imports. Pages import only from `$modules/<name>` or `$shared/`."*

An audit found **~20+ deep cross-module imports** across the codebase — modules importing directly from other modules' internal `services/` directories instead of through the barrel `index.ts`. Examples:

- `stock-opname` → `$modules/product/services/product-service` (deep)
- `consignment` → `$modules/product/services/product-service` (deep)
- `pos` → `$modules/sales/types` (deep)
- `pricing` → `$modules/product/services/product-service` (deep)
- `purchase-orders` → `$modules/supplier/services/supplier-service` (deep)

These work correctly at runtime. The violations are mechanical (wrong import path), not logical (wrong module boundary). There is **no ESLint** in the web project — no `.eslintrc*` exists.

---

## 2. Decision

**Do not introduce ESLint import-boundary enforcement at this time.** Fix violations incrementally as files are touched.

### Rationale

| Factor | Assessment |
|--------|------------|
| Scale of violations | ~20+ across the entire codebase — manageable without tooling |
| Runtime impact | None — deep imports work correctly; this is purely organizational debt |
| Team size | Small — code review is sufficient catch mechanism |
| ESLint setup cost | `eslint-plugin-boundaries` + config for 18+ modules + rule definitions |
| ESLint maintenance cost | Every new module needs barrel updates; false positives with Svelte 5 runes |
| Existing precedent | Backend uses `depguard` + `archtest` but has a larger team and stricter requirements |

### Enforcement Strategy

1. **Code review convention** — reviewers check that new/modified imports use barrel paths (`$modules/<name>`), not internal paths.
2. **Incremental cleanup** — when touching a file with deep imports, refactor to barrel imports in the same change.
3. **No separate cleanup PR** — the debt is small enough to absorb incrementally.
4. **Revisit if** the team grows beyond 3-4 developers, or violations begin accumulating faster than they are caught.

---

## 3. Consequences

- **Positive:** No infrastructure overhead; no ESLint config to maintain; no false-positive friction during development.
- **Negative:** Deep imports can still be introduced if a reviewer misses them. The convention is trust-based.
- **Mitigation:** The barrel pattern is simple to spot in review. If violations become frequent, ESLint can be added retroactively (the convention documentation already exists).

---

## 4. Barrel Re-exports Required (Current Debt)

When cleaning up existing deep imports, the following modules need additional re-exports in their `index.ts`:

| Module | Functions to Re-export | Imported By |
|--------|----------------------|-------------|
| `product` | `getBrands`, `getCategories`, `getProductOptions`, `getWarehouses` | `stock-opname`, `consignment`, `pricing` |
| `supplier` | `getSuppliers` | `stock-opname`, `purchase-orders` |
| `storage-location` | `getStorageLocations` | `stock-opname`, `inventory` |
| `stores` | `getActiveStores` | `stock-opname`, `purchase-orders` (already via barrel in some places) |
| `customers` | `getCustomerGroups` | `customer-groups` |
| `sales` | `Sale`, `SaleItem` types | `pos` |
| `customers` | `Customer` type | `pos` |

These should be added to each module's `index.ts` as files are touched, not in a dedicated cleanup PR.
