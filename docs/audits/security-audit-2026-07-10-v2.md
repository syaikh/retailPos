# Security Audit Report: Retail POS System — Round 2

**Date:** July 10, 2026
**Scope:** Full-stack (Go backend, Svelte 5 frontend, PostgreSQL)
**Previous Audits:** `docs/security-audit.md` (June 29), `docs/security-audit-2026-07-10.md` (July 10)

---

## 🔴 CRITICAL (8 Findings)

### C-2026-01: Double Stock Deduction on Every Sale

**Files:** `internal/sale/service.go:57-65`, `internal/inventory/listener.go:24`

Stock is deducted **twice** for every sale:
1. **Inline** in `sale/service.go` `CreateSale` — inside the transaction, after the `FOR UPDATE` check.
2. **Async** in `inventory/listener.go` via `StockDeductListener` — triggered by the `sale.created` event after commit.

`AdjustStock` (`inventory/repository.go`) begins its own transaction, reads the current stock, subtracts quantity, and writes it back. Since the inline deduction already ran, every sale deducts `quantity * 2` from stock.

**Impact:** Inventory depletes at 2× rate. Products reach zero stock prematurely. False "insufficient stock" errors for legitimate purchases.

**Fix:** Remove the `AdjustStock` call from the listener. Stock deduction is already handled safely inside the sale transaction.

---

### C-2026-02: No Store Isolation on Customer Table

**Files:** `internal/customer/repository.go` (all queries), `internal/customer/handler.go` (all handlers)

The `customers` table has **no `store_id` column**. Every repository query operates globally. Every handler endpoint accepts an arbitrary customer ID with zero store-scoping.

**Impact:** A cashier at Store A with `customer:read` can view, edit, or delete any customer in the system (including PII: phone, email, address, tax ID). This is a **data isolation failure**.

**Fix:** Either add `store_id` to the customers table with scoped queries, or document that customers are intentionally global (read: design bug).

---

### C-2026-03: Report Endpoints Leak Cross-Store Data

**Files:** `internal/report/handler.go:483-525`, `internal/report/service.go:39-44`, `internal/report/repository.go:35-208`

`GetPeriodComparison` and `GetDualChartData` **do not receive or pass `storeID`**. Their SQL queries have no `WHERE store_id = ...` clause. Every other report method (`GetDashboardStats`, `GetHourlySales`, `GetDailySales`, etc.) properly scopes by store.

**Impact:** Any user with `report:read` can see revenue, orders, averages, and chart data for **all stores**. This leaks financial data across store boundaries.

**Fix:** Add `storeID` parameter to both methods and their SQL queries.

---

### C-2026-04: Refresh Token TOCTOU Race Condition

**Files:** `internal/user/auth_service.go:118-166`

`RefreshToken` checks token existence OUTSIDE the transaction, then deletes and inserts INSIDE it. Between the check and the transaction, a concurrent request with the same token can also pass the existence check. Both transactions succeed, creating two valid refresh tokens.

**Impact:** An attacker who intercepts one refresh token can race to create multiple valid tokens. Breaks the "one refresh token per user" invariant.

**Fix:** Use `DELETE ... RETURNING id` as an atomic check-and-delete, or move the existence check inside the transaction with `SELECT ... FOR UPDATE`.

---

### C-2026-05: Product CRUD Missing Store Isolation (IDOR)

**Files:** `internal/product/service.go:87,96`, `internal/product/handler.go:197,218`

`DeleteProduct` and `BulkUpdateProductStatus` always pass `nil` as `storeID` to the repository. `UpdateProduct` uses the **client-supplied** `product.StoreID` from the request body instead of the authenticated user's storeID.

**Impact:** User with `product:delete` can delete any product in any store. User with `product:update` can change which store a product belongs to by manipulating the JSON body.

**Fix:** Extract storeID from auth context and pass it to all service/repo methods. Never trust client-supplied `store_id`.

---

### C-2026-06: Product Price/Cost/Brand Filters Silently Dropped

**Files:** `internal/product/service.go:46,65`

The `GetAllProducts` service method accepts `minPrice`, `maxPrice`, `brand` parameters but **never passes them to the repository**. The repository signature doesn't even accept them.

**Impact:** All product queries asking for `minPrice`, `maxPrice`, or `brand` return incorrect results (no filtering applied). Users see all products regardless of their filter criteria.

**Fix:** Either implement the filters at the repository level or reject unsupported query parameters with a 400 error.

---

### C-2026-07: Access Token & Refresh Token in Login Response Body

**Files:** `internal/user/auth_handler.go:55-60`

The `Login` endpoint returns `refresh_token` in the JSON response body AND sets it as an `httpOnly` cookie. Any XSS vulnerability gives the attacker the refresh token from the response.

**Impact:** The `httpOnly` cookie is rendered useless for protecting the refresh token because it's also available via JavaScript from the response body.

**Fix:** Remove `refresh_token` from the JSON response body. Clients should only access it via the `httpOnly` cookie. The access token needs to remain accessible to JS (it's sent as `Authorization: Bearer` header by the frontend), so store it in memory only, not `localStorage`/`sessionStorage`.

---

### C-2026-08: Product List Endpoint Has No Auth

**Files:** `internal/product/handler.go:23-28`, `cmd/server/main.go`

`GET /api/products` and `GET /api/products/next-sku` are registered WITHOUT the `auth` middleware. The list endpoint returns product names, prices, SKUs, barcodes, cost, and stock.

**Impact:** Unauthenticated attackers can enumerate the entire product catalog, including cost and stock data (commercially sensitive).

**Fix:** Add `auth` middleware to these routes. If needed, create a separate public route group with limited fields.

---

## 🔴 HIGH (9 Findings)

### H-2026-01: No DB-Level Session Validation & Logout Gap

**Files:** `internal/middleware/auth.go:33-56`, `internal/user/auth_service.go`

The auth middleware validates JWTs cryptographically but never checks the database to see if the user is still active or if the session was revoked. `Logout` only deletes the refresh token — the access token (JWT) remains valid for its full 15-minute TTL.

**Impact:** A deactivated user continues accessing the system for up to 15 minutes. A stolen access token cannot be revoked until it expires.

**Fix:** Add a lightweight DB check (e.g., Redis or a `user_sessions` table) on each authenticated request, or reduce the access token TTL to ~1 minute and rely on refresh tokens.

---

### H-2026-02: UpdateProduct Uses Client-Supplied StoreID (IDOR)

**Files:** `internal/product/service.go:80`

`s.repo.UpdateProduct(ctx, product, product.StoreID)` — `product.StoreID` comes from the unvalidated JSON request body. An attacker can set `store_id` in the request to any value, effectively moving a product to another store or bypassing store-scoped access.

**Impact:** Cross-store product modification. A user at Store A can modify products belonging to Store B.

**Fix:** Override `product.StoreID` with the authenticated user's storeID from context before passing to the repo.

---

### H-2026-03: ExportSales Has No Store Filtering (IDOR)

**Files:** `internal/sale/handler.go:256`, `internal/sale/service.go:113-115`, `internal/sale/repository.go:339-403`

The sales export endpoint does not pass or use `storeID`. The SQL query exports all sales across all stores.

**Impact:** Any user with `report:read` permission can export every sale in the system.

**Fix:** Add `storeID` parameter to the export pipeline.

---

### H-2026-04: Password Change Without Current Password

**Files:** `internal/user/handler.go:188-195`

The `UpdateUser` handler (gated by `user:update` permission) allows changing any user's password by simply sending a new `password` field. No current password confirmation is required.

**Impact:** An attacker with `user:update` privileges (or compromised admin account) can change any user's password and lock them out.

**Fix:** Require the current password for self-service password changes. For admin-level resets, require explicit confirmation or secondary auth.

---

### H-2026-05: CreateProduct & UpdateProduct Not Wrapped in Transaction

**Files:** `internal/product/repository.go:434-460, 509-550`

`CreateProduct` executes three separate SQL statements (`INSERT INTO products`, `INSERT INTO product_stock`, `UPDATE products SET stock`) outside any database transaction. If the process crashes between steps, the product table and stock table become inconsistent. `UpdateProduct` has the same issue with its stock sync INSERT.

**Impact:** Orphaned products without stock records, or phantom stock rows. Data inconsistency between `products.stock` and `product_stock.quantity`.

**Fix:** Wrap multi-statement operations in a `pgx.Tx` transaction.

---

### H-2026-06: GetProductBySKU Crashes on NULL Barcode

**Files:** `internal/product/repository.go:163`

```go
&p.Barcode  // scans directly into *string
```

If the database `barcode` column is `NULL`, `pgx` returns a scan error instead of setting the `*string` to `nil`. Compare with `GetProductByID` (line 80) which correctly uses `sql.NullString` first.

**Impact:** Any product with a NULL barcode returns "product not found" when fetched by SKU.

**Fix:** Use `sql.NullString` intermediate variable, same as `GetProductByID`.

---

### H-2026-07: Unbounded IP Rate Limiter Memory Growth (DoS)

**Files:** `internal/middleware/rate_limit.go:12-55`

The `IPRateLimiter` map has no cleanup mechanism. Every unique IP address that hits the server creates a permanent entry. No TTL, no eviction, no size cap.

**Impact:** After enough unique IPs, the server runs out of memory. This is a DoS vector — an attacker can iterate through millions of IPs to exhaust RAM.

**Fix:** Add periodic cleanup (LRU eviction, TTL-based expiry, or max-size cap with oldest-entry removal). Use `go-cache` or `hashmap` with auto-eviction, or switch to Redis for production.

---

### H-2026-08: Discount Has No Validation

**Files:** `internal/sale/handler.go:108`

`Discount: req.Discount` — no check that discount is `>= 0` and no check that `discount <= subtotal`. When `priceStore` is nil (misconfigured), a negative discount results in a negative `TotalAmount` stored in the database. A discount larger than subtotal produces a negative total (only clamped when `priceStore` is set, service.go:87-89).

**Impact:** Financial records with negative totals. Potential to bypass payment logic.

**Fix:** Validate discount >= 0 and discount <= subtotal at the handler layer.

---

### H-2026-09: Products.stock vs product_stock.quantity Desync

**Files:** `internal/product/repository.go:434-460, 490-550`, `internal/inventory/repository.go:111-116`

Product creation and update maintain both `products.stock` and `product_stock.quantity`. But `AdjustStock` (`inventory/repository.go`) only updates `product_stock`, never `products.stock`. After an inventory adjustment, the two values diverge.

**Impact:** Queries reading from `products.stock` (including views) show stale values. Reports relying on product stock are inaccurate.

**Fix:** Add `UPDATE products SET stock = $2 WHERE id = $1` to `AdjustStock`.

---

## 🟡 MEDIUM (11 Findings)

### M-2026-01: PaymentMethod Not Validated Against Allowed List

**Files:** `internal/sale/handler.go:111`

The `PaymentMethod` field accepts any arbitrary string. No check against valid payment methods in the database.

### M-2026-02: No Date Range Bounds Checking on Report Endpoints

**Files:** `internal/report/handler.go:319-333, 402-415, 436-451`

`startDate` and `endDate` are parsed but never validated for ordering (start > end) or range span (max window). Large date ranges cause full-table scans.

### M-2026-03: File Upload Only Validates Extension, Not Content

**Files:** `internal/platform/importexport/handler/handler.go:172-176`

Only `strings.HasSuffix(filename, ".csv")` / `.xlsx` is checked. MIME type and file magic bytes are not validated.

### M-2026-04: Preview Token Is Predictable

**Files:** `internal/platform/importexport/import/engine.go:108`

`fmt.Sprintf("pv_%s_%d_%d", module, len(rows), time.Now().UnixNano())` — predictable token allows import replay/race attacks.

### M-2026-05: No Server-Side Payment Method Validation

**Files:** `internal/sale/handler.go:111`

Client can submit any string as `PaymentMethod`. Should be validated against `GetAllPaymentMethods()` on the server.

### M-2026-06: Bulk Operation Limits Missing

**Files:** `internal/customer/handler.go:202-232`, `internal/product/handler.go:175-218`

`BulkUpdateCustomerStatus`, `BulkDeleteCustomers`, `BulkUpdateProductStatus` accept unbounded `[]int` arrays. No max-size limit.

### M-2026-07: Silent Error Swallowing in Sale Item Scanning

**Files:** `internal/sale/repository.go:277, 321-329`

Scan errors in `GetAllSales` are silently discarded (`if err != nil { continue }`). Sale items with scan errors are silently omitted from results.

### M-2026-08: No Rate Limit on Refresh Endpoint

**Files:** `internal/user/auth_handler.go:32-35`

The `/api/refresh` endpoint has no CSRF bypass risk (it's behind CSRF middleware now) but has no rate limiting. An attacker can brute-force refresh tokens at high velocity.

### M-2026-09: IPRateLimiter TOCTOU

**Files:** `internal/middleware/rate_limit.go:38-52`

`GetLimiter` uses double-checked locking. Between `RUnlock` and `Lock`, two goroutines could both check `exists=false`. The second `Lock+check` pattern prevents duplicates, but the code is complex and could be simplified with `sync.Map` or a pre-sized map.

### M-2026-10: Refresh Token Hardcodes 7-Day DB Expiry

**Files:** `internal/user/auth_service.go:278, 322`

`NOW() + INTERVAL '7 days'` is hardcoded in SQL. If `s.refreshTTL` is changed in code, the DB expiry won't match.

### M-2026-11: Leaked X-User-ID / X-User-Role Response Headers

**Files:** `internal/middleware/auth.go:49-50`

The previous audit listed these headers as leaking internal info. Verify they are still present in responses.

---

## 🟢 LOW (5 Findings)

| Finding | File |
|---|---|
| CSP `'unsafe-inline'` for styles (framework requirement) | `middleware/security_headers.go:39` |
| No `return` enforcement after `shared.InternalError` | `shared/response.go:22-27` |
| Content-Disposition header injection via module name | `report/handler.go:595`, `importexport/handler.go:148,294,297` |
| Integer division truncation in report revenue/day calc | `report/repository.go:139` |
| Hardcoded `$1` in report customer query (fragile) | `report/repository.go:531` |

---

## Summary

| Severity | Count |
|---|---|
| 🔴 CRITICAL | 8 |
| 🔴 HIGH | 9 |
| 🟡 MEDIUM | 11 |
| 🟢 LOW | 5 |
| **Total** | **33** |

**Security Score:** ~40/100 (down from ~65/100 due to newly identified issues)

### Most Critical Fixes Needed Immediately:
1. **Double stock deduction** — every sale deducts stock × 2
2. **Customer/store isolation** — global customers table leaks PII across stores
3. **Cross-store report leakage** — `GetPeriodComparison` and `GetDualChartData` leak all stores' financial data
4. **Product CRUD IDOR** — delete/update ignores store boundaries
5. **Login response leaks refresh token** — httpOnly cookie protection defeated
6. **Refresh token TOCTOU** — race creates multiple valid tokens
