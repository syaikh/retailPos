# E2E Test Coverage Remediation Plan

**Date:** 2026-07-13
**Status:** Planning

## Current State

| Metric | Count | % |
|--------|-------|---|
| E2E test files | 28 | — |
| E2E test cases | ~196 | — |
| API endpoints | 74 | — |
| Endpoint with E2E coverage | 57 | 77% |
| Endpoint ZERO coverage | 17 | 23% |
| Frontend routes | 24 | — |
| Routes with E2E coverage | 17 | 71% |
| Routes ZERO coverage | 7 | 29% |

---

## Phase 1 — P0 Critical Security Gaps (7 endpoints)

### 1a. `tests/e2e/change-password.spec.ts` — NEW FILE

**Endpoint:** `POST /api/change-password`

Tests:
- `should change password with valid current password` — 200
- `should reject with incorrect current password` — 401
- `should reject with short new password` — 400
- `should reject without authentication` — 401

### 1b. `tests/e2e/roles.spec.ts` — ADD TO EXISTING

**Endpoints:** `PUT /api/admin/roles/:id`, `DELETE /api/admin/roles/:id`

Tests to add:
- `should update an existing role via API` — create role, update name+description, verify
- `should return 404 when updating non-existent role`
- `should delete an unassigned role via API` — create role, delete, verify 404 on GET
- `should reject deleting role with assigned users` — create role, assign user, delete → 400
- `should return 404 when deleting non-existent role`

### 1c. `tests/e2e/products-api.spec.ts` — ADD TO EXISTING

**Endpoint:** `DELETE /api/products/:id`

Tests to add:
- `should delete a product via API` — create product, delete, verify 404
- `should return 404 when deleting non-existent product`
- `should reject delete without auth` — 401

### 1d. `tests/e2e/categories.spec.ts` — ADD TO EXISTING

**Endpoint:** `DELETE /api/categories/:id`

Tests to add:
- `should delete a category via API` — create category, delete, verify
- `should return 404 when deleting non-existent category`

### 1e. `tests/e2e/customers.spec.ts` — ADD TO EXISTING

**Endpoints:** `POST /api/customers/bulk/status`, `POST /api/customers/bulk/delete`

Tests to add:
- `should bulk deactivate customers via API` — create 2 customers, bulk deactivate, verify inactive
- `should bulk activate customers via API` — bulk activate, verify active
- `should bulk delete customers via API` — create customer, bulk delete, verify 404
- `should reject bulk delete with empty array` — 400
- `should reject bulk ops without auth` — 401

---

## Phase 2 — P1 High-Value Functional Gaps (7 endpoints)

### 2a. `tests/e2e/audit-logs-search.spec.ts` — ADD TO EXISTING

**Endpoints:** `GET /api/audit-logs/:id`, `GET /api/audit-logs/entity-types`, `GET /api/audit-logs/export`

Tests to add:
- `should fetch audit log by ID` — list logs, pick first ID, fetch by ID, verify fields
- `should list entity types` — GET entity-types, verify array contains known types
- `should export audit logs as CSV` — GET export?format=csv, verify Content-Type and headers
- `should export audit logs as XLSX` — GET export?format=xlsx, verify Content-Type

### 2b. `tests/e2e/reports-api.spec.ts` — NEW FILE

**Endpoints:** `GET /api/dashboard/chart`, `GET /api/dashboard/chart/weekly`, `GET /api/dashboard/chart/monthly`, `GET /api/dashboard/comparison`

Tests:
- `should fetch chart data with default date range` — 200, data array
- `should fetch chart data with custom date range` — 200
- `should reject chart with endDate before startDate` — 400
- `should reject chart with range exceeding 366 days` — 400
- `should fetch weekly chart data` — 200
- `should fetch monthly chart data` — 200
- `should fetch period comparison with daily period` — 200, meta contains period_type
- `should fetch period comparison with completed mode` — 200
- `should fetch period comparison with 30days mode` — 200

### 2c. `tests/e2e/export-import.spec.ts` — ADD TO EXISTING

**Endpoints:** `GET /api/import-export/template/:module`, `POST /api/import-export/cancel/:jobId`

Tests to add:
- `should download import template for products` — GET template/products, verify CSV headers
- `should cancel an import job` — start import, cancel, verify status

### 2d. Import History frontend routes

**Routes:** `/{module}/import-history` (5 routes)

Tests to add in `export-import.spec.ts`:
- `should navigate to import history page` — navigate to /products/import-history, verify table renders
- `should show job detail in history` — (if history exists)

---

## Phase 3 — P2 RBAC and Edge Cases

### 3a. RBAC for categories/brands/UOM

Add to `categories.spec.ts`, `brands.spec.ts`, `units-of-measure.spec.ts`:
- Verify cashier role sees read-only view (no create/edit/delete buttons)
- Verify API returns 403 for cashier on POST/PUT/DELETE

### 3b. Sale creation error paths

Add to `pos-flow.spec.ts`:
- `should reject sale with insufficient stock` — 409
- `should reject sale with invalid payment method` — 400
- `should auto-generate invoice number when omitted`

### 3c. Product UI modal flow

Add to `products.spec.ts`:
- `should create product via UI modal` — open modal, fill form, submit, verify in table
- `should edit product via UI modal` — click edit, change name, save, verify

### 3d. Audit log export download

Add to `audit-logs-search.spec.ts`:
- `should export and download CSV` — click export, verify download
- `should export and download XLSX` — click export xlsx, verify download

---

## Verification

After each phase:
1. `npx playwright test` — all tests pass
2. No flaky tests introduced
3. Test execution time remains under 5 minutes
