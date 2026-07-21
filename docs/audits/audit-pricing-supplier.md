# Audit Report: Pricing Engine & Supplier Management

**Date:** 2026-07-17
**Branch:** `feature/pricing-engine-supplier-mgmt`
**Scope:** Frontend (Svelte 5), Backend (Go/Gin), E2E (Playwright), Security, Performance, Architecture, Code Quality

---

## Executive Summary

| Dimension | Pricing | Supplier | Overall |
|-----------|---------|----------|---------|
| **UI/UX** | 7.5/10 | 7.0/10 | **7.3/10** |
| **Security** | 8.0/10 | 8.0/10 | **8.0/10** |
| **Performance** | 7.5/10 | 7.0/10 | **7.3/10** |
| **Architecture** | 8.5/10 | 8.0/10 | **8.3/10** |
| **Code Quality** | 7.5/10 | 6.5/10 | **7.0/10** |
| **Test Coverage** | 7.0/10 | 5.0/10 | **6.0/10** |
| **OVERALL** | **7.7/10** | **6.9/10** | **7.3/10** |

---

## 1. UI/UX

### Pricing (7.5/10)

**Strengths:**
- Full CRUD with modal-based form, inline validation, live price preview (Section 5)
- Debounced search (300ms), client-side sort via `$derived`
- Permission-gated actions (`pricing:create/update/delete`)
- Day-of-week quick-select (all/workdays/weekend)
- Category/brand inline filtering from pre-fetched arrays

**Weaknesses:**
- 930-line monolith component — hard to maintain
- No loading skeleton for individual rule editing
- Hardcoded fallback labels (`Product #${id}`) when name lookup fails
- No bulk actions (select multiple, bulk delete/activate)
- No keyboard shortcuts for common actions
- No confirmation before destructive bulk operations
- Table doesn't show pricing method/value summary at a glance

**Accessibility:**
- `<label for="">` present on all form fields
- No ARIA attributes on modals or interactive elements
- No `role` or `aria-label` on table actions
- Keyboard navigation not explicitly handled

### Supplier (7.0/10)

**Strengths:**
- Clean CRUD with search, status filter, pagination
- Permission-gated actions using `pricing:*` permissions (shared permission namespace)
- Soft delete with confirmation modal

**Weaknesses:**
- Mixed language: form labels Indonesian ("Nama Supplier"), table headers English
- No product-supplier linking UI in the module (only API exists)
- No bulk import/export from frontend
- No supplier detail view (only list + edit modal)
- Client-side sorting only sorts current page, not full dataset
- No loading states for individual actions (only page-level skeleton)

**Accessibility:**
- Same baseline as pricing — labels present, no ARIA

---

## 2. Security

### Score: 8.0/10

**Strengths:**
- RBAC middleware chain: `authMiddleware → CSRFMiddleware → permMiddleware`
- Permission-gated endpoints (`pricing:read/create/update/delete`, `supplier_cost:view/update`)
- Rate limiting: global 50 RPS, login 5 RPM, refresh 10 RPM
- Security headers: CSP, HSTS, X-Frame-Options: DENY, X-Content-Type-Options: nosniff
- Body size limit: 1MB
- CORS: origin-validated, production guard against `*`
- Audit logging on all pricing/supplier mutations
- `shared.InternalError` — no stack traces exposed to client
- Input validation in service layer (not just handler)

**Weaknesses:**
- **CSRF bypass**: Presence of `Authorization` header or `X-Refresh-Token` header skips CSRF check — no actual token verification
- **No struct validation tags**: All validation is manual in service layer; new code paths could bypass it
- **Supplier email/phone**: No format validation — accepts any string
- **Rate limiter**: Uses `RemoteAddr` directly, no trusted proxy validation (documented limitation)
- **No WebSocket rate limiting**: `GET /ws` has no rate limit
- **No Go-level E2E test file**: `cmd/server/e2e_test.go` referenced in AGENTS.md but doesn't exist
- **`noopAuth` pattern**: Auth is applied at group level, individual routes use `noopAuth` — relies entirely on group-level middleware

---

## 3. Performance

### Pricing (7.5/10)

**Strengths:**
- Batch resolution API (`/pricing/resolve`) with `GetBasePricesBatch` and `GetActiveRulesBatch`
- Debounced search (300ms) reduces API calls
- Client-side sort avoids round-trips for common operations
- PostgreSQL indexes on `product_id`, `category_id`, `brand_id`, `pricing_type`, `is_active`

**Weaknesses:**
- **Client-side sort**: Only sorts current page (20 items), not full dataset — misleading for users
- **No pagination optimization**: `COUNT(*)` on every list query
- **No caching**: Rules are re-fetched on every page load, no stale-while-revalidate
- **930-line component**: Re-renders entire tree on any state change
- **Resolver N+1 potential**: `GetActiveRulesBatch` fetches rules per product, but category/brand rules are fetched separately
- **CSV import**: `BulkInsertPricingRules` is unbounded — large imports could cause OOM

### Supplier (7.0/10)

**Strengths:**
- Same debounce and client-side sort patterns as pricing
- PostgreSQL indexes on `name`, `code`, `is_active`, `store_id`

**Weaknesses:**
- **Same client-side sort limitation**: Only current page
- **No search debouncing on supplier module** (unlike pricing)
- **`getSuppliersByProduct`**: Returns `any[]` — no type safety, potential N+1 if called in loops
- **Bulk import**: `BulkInsertSuppliers` unbounded, no streaming/chunking

---

## 4. Architecture

### Pricing (8.5/10)

**Strengths:**
- Clean layered architecture: `domain → repository → service → handler`
- Resolver as separate concern with interface (`PriceResolver`)
- Adapter pattern for import/export (`adapter.go`, `schema.go`)
- Domain-first: enums, entities, and errors defined in `domain.go`
- Handler registers routes in dedicated function (`RegisterRoutes`)
- Jakarta timezone handling consistent across all queries

**Weaknesses:**
- **Stacking not implemented**: `allow_combine` field exists but resolver picks best single rule only
- **Supplier types in pricing types file**: `Supplier`, `CreateSupplierPayload`, etc. in `pricing/types/index.ts` — cross-module dependency
- **Service types in pricing-service.ts**: `PricingRuleListParams`, `PricingRuleListResponse` defined in service, not types
- **No domain events**: Pricing changes don't propagate (e.g., to cache invalidation)
- **Repository size**: 635 lines — could be split by concern (CRUD vs batch vs export)

### Supplier (8.0/10)

**Strengths:**
- Same clean layered architecture as pricing
- Separate domain entities for `Supplier` and `ProductSupplier`
- Audit logging integrated at handler level
- Import/export adapter pattern

**Weaknesses:**
- **No service_test.go**: Validation logic untested
- **Handler tests are integration tests**: All require real DB, no mocked service
- **Permission namespace**: Uses `pricing:*` for supplier CRUD — may confuse developers expecting `supplier:*`
- **No domain events**: Supplier changes don't propagate

---

## 5. Code Quality

### Pricing (7.5/10)

**Strengths:**
- Consistent Go conventions: exported types, receiver methods, error wrapping
- Consistent Svelte 5 patterns: runes, `$state`, `$derived`, `$props`, `$bindable`
- Consistent error handling in service layer
- Proper linting (`golangci-lint` passes clean, `svelte-check` 0 errors)

**Weaknesses:**
- **930-line PricingRulesPage.svelte**: Should be decomposed into smaller components
- **`any` types**: Some `any` usage in callbacks and service responses
- **Silent error swallowing**: All service functions return `boolean`/`null`/`[]` on failure — callers can't distinguish "empty" from "error"
- **Mixed type locations**: Some types in `types/index.ts`, others in `pricing-service.ts`
- **Hardcoded fallback labels**: `Product #${id}` in edit handler
- **No error boundaries**: Frontend silently swallows errors, no retry mechanisms

### Supplier (6.5/10)

**Strengths:**
- Clean Go code, consistent patterns
- Proper domain entity design

**Weaknesses:**
- **No frontend component tests**: Zero unit tests for `SuppliersPage.svelte`
- **`any` in service**: `getSuppliersByProduct` returns `any[]`
- **Mixed language**: Indonesian labels, English headers — inconsistent UX
- **Permission mismatch**: `pricing:*` permissions for supplier CRUD
- **No `service_test.go`**: Backend validation untested
- **Client-side sort bug**: Only sorts current page, not full dataset

---

## 6. Test Coverage

### Pricing (7.0/10)

| Layer | Tests | Coverage |
|-------|-------|----------|
| **Resolver** | 28 | Excellent — all 4 pricing methods, priority, scope, filters, batch, edge cases |
| **Service** | 10 | Good — `validateRule()` covers all validation branches |
| **Repository** | 14 | Good — CRUD, batch, search (requires DB) |
| **Handler** | 7 | Fair — basic CRUD + resolve (requires DB) |
| **Frontend Unit** | 11 | Fair — service functions mocked |
| **E2E (Playwright)** | 12 | Good — full CRUD lifecycle, auth, stacking, scope, max qty |

**Total: 82 tests**

**Gaps:**
- No tests for `adapter.go` (import/export mapping)
- No tests for `schema.go`
- No tests for `BulkInsertPricingRules` / `BulkUpdatePricingRules`
- No tests for `GetAllForExport`
- No tests for `GetAll` with all filter combinations
- No tests for stacking behavior (feature not implemented)
- `getPricingRule`, `updatePricingRule`, `deletePricingRule` frontend functions untested

### Supplier (5.0/10)

| Layer | Tests | Coverage |
|-------|-------|----------|
| **Domain** | 3 | Basic struct/error tests |
| **Repository** | 8 | CRUD, linking, preferred, soft delete (requires DB) |
| **Handler** | 7 | Basic CRUD only (requires DB) |
| **Service** | 0 | **NONE** |
| **Frontend Unit** | 0 | **NONE** |
| **E2E (Playwright)** | 2 files | CRUD lifecycle, product linking, auth check |

**Total: ~20 tests**

**Gaps:**
- **No `service_test.go`**: `validateSupplier()` and `validateProductSupplier()` untested
- **No frontend component tests**: `SuppliersPage.svelte` has zero test files
- **No handler tests for product-supplier sub-routes**: 6 handler methods untested
- **No bulk import/export tests**
- **No adapter tests**
- **Audit logging path never exercised** in tests (handler tests pass `nil` for `auditSvc`)

---

## 7. Critical Issues (Priority Order)

| # | Issue | Severity | Module | Fix Effort |
|---|-------|----------|--------|------------|
| 1 | **930-line monolith component** (PricingRulesPage.svelte) | High | Pricing | Medium |
| 2 | **Silent error swallowing** in all service functions | High | Both | Low |
| 3 | **No supplier service tests** | High | Supplier | Low |
| 4 | **No supplier frontend tests** | High | Supplier | Low |
| 5 | **Client-side sort only sorts current page** | Medium | Both | Low |
| 6 | **Supplier types in pricing types file** | Medium | Pricing | Low |
| 7 | **Mixed language** (Indonesian/English) in supplier | Medium | Supplier | Low |
| 8 | **CSRF bypass** via header presence | Low | Security | Medium |
| 9 | **No struct validation tags** | Low | Both | Low |
| 10 | **Stacking not implemented** | Low | Pricing | High |

---

## 8. Recommendations

### Quick Wins (1-2 hours each)
1. Extract `PricingRulesPage.svelte` into 5-6 sub-components (Form, Filters, Table, etc.)
2. Add error return types to service functions (or at least `toast.error` in callers)
3. Create `supplier/service_test.go` with validation tests
4. Create `SuppliersPage.test.ts` with mocked service tests
5. Move supplier types from `pricing/types/index.ts` to `supplier/types/index.ts`
6. Standardize language (pick Indonesian or English consistently)

### Medium-term (1-2 days each)
7. Implement server-side sorting for pricing and suppliers
8. Add ARIA attributes to modals and tables
9. Add CSRF token-based protection (replace presence check)
10. Decompose `SuppliersPage.svelte` into sub-components

### Long-term (1+ week each)
11. Implement stacking/promotion combination in resolver
12. Add domain events for cache invalidation
13. Add E2E tests for supplier product-supplier linking UI
14. Add comprehensive Playwright tests for pricing form edge cases
