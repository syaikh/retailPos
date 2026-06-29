# Security Audit Report: Retail POS System

**Date:** June 29, 2026  
**Scope:** Full-stack (Go backend, Svelte 5 frontend, PostgreSQL)  
**Auditor:** Automated Security Analysis

---

## Executive Summary

The application has a **moderate security posture** with several **critical and high-severity** vulnerabilities that require immediate remediation. The most significant issues are **lack of CSRF protection**, **JWT in WebSocket URL query parameters**, **no brute-force protection on login**, **missing security headers**, **hardcoded secrets**, **sessionStorage token storage (XSS-exposed)**, and **potential SQL injection via dynamic sort columns**. The RBAC implementation is solid but has gaps in ownership validation and store isolation.

**Security Score: 52/100**

---

## Risk Heatmap

```
                     Impact
              Low    Medium   High   Critical
   Critical    -       -       -    C-01, C-02,
                                   C-03, C-04
       High    -     H-07    H-01,  H-02, H-03,
  L                         H-04,  H-05, H-06,
  i                          H-08
  k  Medium  L-03,   M-01,  M-03,
  e          L-04,   M-02,  M-04,
  d          L-07    M-05,  M-06,
                     M-07
        Low  L-01,   L-06
             L-02,
             L-05,
             L-08
```

---

## Critical Findings

### C-01: WebSocket JWT Token Exposed in URL Query Parameter

**Severity:** Critical

**Risk:** The JWT access token is transmitted as a query parameter in the WebSocket URL (`/ws?token=<jwt>`). URLs are logged by proxy servers, reverse proxies, load balancers, and browser history. The token can be leaked via `Referer` headers, server access logs, and browser history.

**Evidence:**
- `pkg/websocket/hub.go:336`: `tokenString := c.Query("token")`
- `web/src/shared/api/websocket.ts:33`: `` const wsUrl = `${protocol}//${backendHost}/ws?token=${encodeURIComponent(token || '')}` ``

**Attack Scenario:** An attacker with access to server access logs, proxy logs, or browser history can extract valid JWT tokens and impersonate any user.

**Recommendation:** Authenticate WebSocket connections via a subprotocol header or an initial authentication message after the WebSocket connection is established, rather than embedding the token in the URL.

**OWASP:** A3 (Sensitive Data Exposure), CWE-200, ASVS V3 (Session Management)

---

### C-02: No CSRF Protection on Any State-Changing Endpoint

**Severity:** Critical

**Risk:** The application uses cookie-based refresh tokens (`refresh_token` cookie with `httpOnly` and `Secure` flags) but **no CSRF protection exists**. The refresh endpoint (`POST /refresh`) reads the cookie automatically without any CSRF token or custom header requirement. Any authenticated user visiting a malicious site can have their session hijacked.

**Evidence:**
- No `X-CSRF-Token` header anywhere in the codebase
- `internal/user/auth_handler.go:42-48`: Sets `refresh_token` cookie
- `internal/user/auth_handler.go:52-55`: Reads refresh token from cookie or header
- `web/src/modules/auth/services/auth-service.ts:12-13`: Calls `/refresh` without CSRF token
- `cmd/server/main.go:131-139`: CORS config with `AllowCredentials: true` + specific origin

**Attack Scenario:** Attacker crafts `<form action="https://target.com/api/refresh" method="POST">` on malicious site. The browser auto-sends the `refresh_token` cookie. The response contains a new `access_token`. The attacker now has the user's access token.

**Recommendation:** Implement CSRF protection using custom headers (e.g., `X-CSRF-Token`) for all state-changing requests. Since the frontend uses Svelte with Axios, a double-submit cookie pattern or a custom header check on the backend would work.

**OWASP:** A1 (Broken Access Control), CWE-352, ASVS V4 (Access Control)

---

### C-03: Hardcoded JWT Secret with Weak Fallback

**Severity:** Critical

**Risk:** The JWT secret is loaded from environment variable `JWT_SECRET`, but **if the env var is not set, it falls back to `"your-secret-key-change-in-production"`**. This is a well-known weak secret. Any attacker who knows this default can forge arbitrary JWT tokens with any role/permissions.

**Evidence:**
- `internal/user/auth_service.go:47-49`: `secret := os.Getenv("JWT_SECRET"); if secret == "" { secret = "your-secret-key-change-in-production" }`
- `internal/config/config.go:55-58`: Same fallback

**Attack Scenario:** Attacker forges JWT with `superadmin` role and arbitrary permissions, gains full system access.

**Recommendation:** Remove the fallback entirely. Crash on startup if `JWT_SECRET` is not set. Use `os.Getenv` and validate the value is non-empty.

```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    panic("JWT_SECRET environment variable is required")
}
```

**OWASP:** A2 (Cryptographic Failures), CWE-798, ASVS V2 (Authentication)

---

### C-04: No Brute-Force Protection on Login

**Severity:** Critical

**Risk:** The login endpoint `POST /api/login` has **no rate limiting, account lockout, or progressive delay**. An attacker can brute-force credentials at high speed. The `RateLimitMiddleware` exists but is **never registered on any route**.

**Evidence:**
- `internal/middleware/rate_limit.go`: `RateLimitMiddleware()` function exists but is never used
- `cmd/server/main.go`: No rate limiter middleware is applied to any route
- `internal/user/auth_handler.go:27-49`: Login handler has no protection

**Attack Scenario:** Automated password spraying or brute-force attack against known usernames with thousands of attempts per second.

**Recommendation:** Register the rate limiter on the login endpoint specifically, and add account lockout after N failed attempts. For production, use Redis-backed rate limiting.

```go
api.POST("/login", middleware.RateLimitMiddleware(), h.Login)
```

**OWASP:** A7 (Identification and Authentication Failures), CWE-307, ASVS V2 (Authentication)

---

## High Findings

### H-01: No Input Validation / No Server-Side Validation for Product Creation

**Severity:** High

**Risk:** Product creation (`POST /products`) accepts raw JSON and binds directly to the `Product` struct without validation of field types, lengths, or allowed values. An attacker can set arbitrary `Price`, `Cost`, `Stock`, etc. including negative values.

**Evidence:**
- `internal/product/handler.go:120-132`: `CreateProduct` calls `c.ShouldBindJSON(&product)` then immediately calls `h.svc.CreateProduct()`. No field validation.

**Recommendation:** Add validation for all fields — price and cost must be >= 0, stock >= 0, name length limits, etc.

**OWASP:** A4 (Insecure Design), CWE-20, ASVS V5 (Validation)

---

### H-02: Sort-by Parameters Allow SQL Injection in ORDER BY Clause

**Severity:** High

**Risk:** Some repository methods accept raw `sortBy` and `sortDir` parameters from user input. While many have allowlists, the **product repository** (`GetAllProducts`) directly interpolates `sortBy` into ORDER BY without validation when it's not empty. If `sortBy` is passed from the API without sanitization, it could be exploited.

**Evidence:**
- `internal/product/repository.go:296-299`: `query2 += fmt.Sprintf(" ORDER BY %s", sortBy)` — no allowlist check for `sortBy`
- Compare with `internal/sale/repository.go:272-275`: Properly validated `allowedSortBy`
- Compare with `internal/user/repository.go:133-135`: Properly validated `validSortColumns`

**Attack Scenario:** `GET /api/products?sortBy=1;--` or time-based blind injection via ORDER BY.

**Recommendation:** Add an allowlist for valid sort columns, consistent with other repositories.

```go
allowedSortBy := map[string]bool{"id": true, "name": true, "price": true, "stock": true, "created_at": true}
if !allowedSortBy[sortBy] {
    sortBy = "id"
}
```

**OWASP:** A3 (Injection), CWE-89, ASVS V5 (Validation)

---

### H-03: Refresh Token Reuse / No Token Rotation on Refresh

**Severity:** High

**Risk:** The refresh token endpoint **does not rotate the refresh token**. It validates the old refresh token exists, then issues a new access token but keeps the old refresh token active. This means a stolen refresh token can be used indefinitely (for up to 7 days) to obtain new access tokens. There is also **no refresh token revocation list** — `deleteRefreshToken` is only called on explicit logout.

**Evidence:**
- `internal/user/auth_service.go:113-154`: `RefreshToken` validates old refresh token but does not rotate it (never calls `deleteRefreshToken` + `storeRefreshToken` with a new one)
- `internal/user/auth_service.go:156-166`: `Logout` deletes the single specific refresh token

**Attack Scenario:** Attacker steals refresh token → can refresh indefinitely without the victim knowing until the 7-day TTL expires.

**Recommendation:** Rotate refresh tokens on each use: delete the old one and issue a new one. This limits the window of a stolen token.

**OWASP:** A7 (Identification and Authentication Failures), CWE-613, ASVS V3 (Session Management)

---

### H-04: Refresh Tokens Stored in Plaintext (Not Hashed)

**Severity:** High

**Risk:** Refresh tokens are stored in the database **in plaintext** via `storeRefreshToken`. If the database is compromised, all active refresh tokens are immediately usable.

**Evidence:**
- `internal/user/auth_service.go:253-258`: `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)` — despite the column name `token_hash`, the raw JWT string is stored
- `internal/user/auth_service.go:261-266`: `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2 ...)` — compares the raw token

**Recommendation:** Hash refresh tokens with SHA-256 before storing. Store the hash, not the raw token.

```go
func hashToken(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}
```

**OWASP:** A2 (Cryptographic Failures), CWE-257, ASVS V2 (Authentication)

---

### H-05: No Ownership Validation — IDOR on Sales/Customers/Products

**Severity:** High

**Risk:** While endpoints check for required permissions, there is **no ownership validation**. A cashier at Store A can view sales from Store B by manipulating the sale ID in the URL. The `storeID` context is extracted from the JWT but not consistently enforced as a filter on queries.

**Evidence:**
- `internal/sale/handler.go:192-211`: `GetSaleByID` loads any sale by ID without filtering by the user's store
- `internal/sale/handler.go:119-189`: `GetSalesHistory` uses `storeIDPtr` from JWT but `GET /sales/:id` does not
- `internal/product/handler.go:99-117`: `GetProductByID` uses storeID from JWT but many other handlers don't

**Recommendation:** Add store ID filtering to all single-resource GET endpoints. The repository should always filter by store ID when the user has a store-scoped role.

**OWASP:** A1 (Broken Access Control), CWE-639, ASVS V4 (Access Control)

---

### H-06: Massive Business Logic Flaw — Client Controls Price via Subtotal

**Severity:** High

**Risk:** The sale creation endpoint (`POST /api/sales`) accepts `subtotal` and `discount` from the **client**. The server computes `unitPrice = subtotal / quantity` but never verifies that the unit price matches the actual product price from the database. A cashier can set a subtotal of 0 for products, manipulating the transaction total.

**Evidence:**
- `internal/sale/handler.go:39-117`: `CreateSale` — `items[i].Subtotal` comes from the request; server computes `unitPrice` from `subtotal / quantity`
- No server-side lookup of actual product price
- `req.Discount` also comes directly from the client without validation

**Attack Scenario:** Cashier creates a sale for product X (price 100,000) but sends `subtotal: 0` for 1 quantity. The total is 0. The product is deducted from stock. The store loses revenue.

**Recommendation:** Look up product prices server-side. Reject any client-supplied price/subtotal. Calculate the total from the actual product prices.

```go
product, err := productRepo.GetByID(ctx, item.ProductID)
if err != nil || product.Price != item.UnitPrice {
    return error("price mismatch")
}
```

**OWASP:** A8 (Software and Data Integrity Failures), CWE-841, ASVS V7 (Business Logic)

---

### H-07: No File Upload Validation (CSV Import)

**Severity:** High

**Risk:** The CSV import endpoints (`/api/products/import`, `/api/brands/import`, `/api/customers/import`) accept any file and parse it as CSV. There is **no MIME type validation, no file size limit, no extension validation, no antivirus scanning**. An attacker could upload a malicious file masquerading as CSV.

**Evidence:**
- `internal/product/handler.go:760-768`: `c.Request.FormFile("file")` — no size limit, no MIME check, no extension check
- Same pattern in `ImportBrands` (line 364), `ImportUnitsOfMeasure` (line 568), `ImportCustomers` (157)

**Recommendation:** Limit file size via `c.Request.ParseMultipartForm(10 << 20)`, validate MIME type, validate file extension (.csv, .xlsx only).

**OWASP:** A4 (Insecure Design), CWE-434, ASVS V12 (File Uploads)

---

### H-08: Inconsistent bcrypt Cost

**Severity:** High

**Risk:** Passwords are hashed with `bcrypt cost 14` in `auth_service.go` but with `bcrypt.DefaultCost` (cost **10**) in `handler.go`. This means admin-created users have weaker password hashes.

**Evidence:**
- `internal/user/handler.go:117`: `bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)` = cost 10
- `internal/user/handler.go:183`: Same
- `internal/user/auth_service.go:173`: `bcrypt.GenerateFromPassword([]byte(password), 14)` = cost 14

**Recommendation:** Use a consistent cost factor (preferably 14). Define a package-level constant.

```go
const bcryptCost = 14
```

**OWASP:** A2 (Cryptographic Failures), CWE-916, ASVS V2 (Authentication)

---

## Medium Findings

### M-01: No Rate Limiter Registered on Any Route

**Severity:** Medium

**Risk:** The `RateLimitMiddleware` exists in `internal/middleware/rate_limit.go` but is **never imported or registered** in `cmd/server/main.go`. The in-memory IP-based rate limiter is also marked as "development only" and would not work in multi-instance production.

**Evidence:**
- `cmd/server/main.go`: No `middleware.RateLimitMiddleware()` is applied to any route group
- `internal/middleware/rate_limit.go:11-12`: Comment says "Not suitable for multi-instance production (use Redis then)"

**Recommendation:** Register the rate limiter on all API routes. For production, replace with Redis-backed rate limiting.

---

### M-02: User Context Lost in Event Bus Listeners

**Severity:** Medium

**Risk:** The event bus passes context to listeners, but the audit listener **replaces context with `context.Background()`** (`internal/audit/listener.go:42`), losing the user ID/username/role context. This means audit logs for user actions may have empty or incorrect `user_id` fields.

**Evidence:**
- `internal/audit/listener.go:37-42`: Extracts `userID`, `username`, `role` from context, then immediately overrides with `ctx = context.Background()`

**Recommendation:** Remove the `ctx = context.Background()` line. Use the passed context directly to preserve user identity in audit logs.

---

### M-03: CSV Export — Formula Injection

**Severity:** Medium

**Risk:** CSV exports write data without sanitization. If any exported field starts with `=`, `+`, `-`, or `@`, Excel/Google Sheets will interpret it as a formula when opened.

**Evidence:**
- `internal/sale/handler.go:259-275`: `exportCSV` writes raw data without sanitization
- `internal/product/handler.go:714-756`: Product CSV export without formula sanitization
- `internal/customer/handler.go:120-153`: Customer CSV export without formula sanitization

**Recommendation:** Sanitize CSV fields that start with `=`, `+`, `-`, or `@` by prefixing with a single quote.

```go
func sanitizeCSVField(s string) string {
    if len(s) > 0 && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
        return "'" + s
    }
    return s
}
```

**OWASP:** A3 (Injection), CWE-1236, ASVS V5 (Validation)

---

### M-04: X-User-ID and X-User-Role Response Headers Leak Internal Info

**Severity:** Medium

**Risk:** The auth middleware sets `X-User-ID` and `X-User-Role` response headers on every authenticated request. These leak internal identifiers and roles.

**Evidence:**
- `internal/middleware/auth.go:49-50`: `c.Header("X-User-ID", ...)` and `c.Header("X-User-Role", ...)`

**Recommendation:** Remove these response headers. They provide no benefit to the client (the data is already in the JWT) and expose internal information.

---

### M-05: No Input Validation on Customer Email/Phone

**Severity:** Medium

**Evidence:**
- `internal/customer/handler.go:271-297`: `CreateCustomer` — `req.Email` and `req.Phone` have no format validation

**Recommendation:** Add email format validation and phone number format validation.

---

### M-06: Audit Log May Store Sensitive Data

**Severity:** Medium

**Risk:** The audit listener serializes event payloads to JSON maps via `toJSONMap()`. For user creation/update events, passwords could be included.

**Recommendation:** Explicitly strip sensitive fields (`Password`, `Token`, etc.) before serializing audit payloads.

---

### M-07: No CSRF on Sale Creation (Bearer token mitigates partially)

**Severity:** Medium

**Recommendation:** Even with Bearer auth, add CSRF protection as defense-in-depth, especially since the refresh_token cookie is sent automatically.

---

## Low / Informational Findings

### L-01: Default Database Credentials in .env.example

`.env.example` contains `DB_PASSWORD=admin123`. Should be changed per deployment.

### L-02: Database Connection String Contains Password in Plaintext

`cmd/server/main.go:74-75`: DSN fallback contains `pos:admin123@localhost:5433/retail_pos`.

### L-03: CORS Allows Credentials with Single Origin

`cmd/server/main.go:133-138`: If `CORS_ORIGIN` is set to `*` (possible via env), credentials would be sent to any origin.

### L-04: Panic in main() on DB Connection Failure

`cmd/server/main.go:78`: `panic(...)` — production services should handle startup failures gracefully.

### L-05: JWT Issuer and Subject Not Validated

`internal/user/auth_service.go:215-231`: `parseToken` does not validate `issuer` or `subject` claims.

### L-06: No MFA / Password Reset / Email Verification

The application has no MFA, password reset flow, or email verification for users.

### L-07: Token Prefix Logged on Auth Failure

`pkg/websocket/hub.go:343-348`: Logs token prefix on failed WebSocket auth, leaking part of the token to logs.

### L-08: Dependencies Not Audited

`go.sum` and `package.json` contain many dependencies. Run `govulncheck` and `npm audit` to identify known CVEs.

---

## Security Architecture Review

### Strengths
1. **RBAC via JWT claims** — Permissions embedded in JWT, enforced at route level
2. **Parameterized queries** — All PostgreSQL queries use `$1`, `$2` style parameters (except sort columns)
3. **bcrypt with cost 14** for login password hashing
4. **Transaction-based stock updates** with `FOR UPDATE` row locking
5. **HttpOnly + Secure + SameSite=Strict** cookie flags on refresh token
6. **Graceful shutdown** with WebSocket/event bus drain pattern
7. **Audit logging** for all entity CRUD operations
8. **Event-driven architecture** with decoupled modules

### Weaknesses
1. **No CSRF protection** — Single biggest architectural gap
2. **Client-controlled pricing** — Core business logic trusted to client
3. **No input validation layer** — Endpoints directly bind JSON to domain structs
4. **Token in WebSocket URL** — Violates principle of not putting secrets in URLs
5. **No security headers** — No defense-in-depth at HTTP layer
6. **Inconsistent store isolation** — Some endpoints filter by store, others don't
7. **Session tokens in sessionStorage** — No protection against XSS token theft
8. **No rate limiting applied** — Login and API endpoints completely unprotected

---

## Prioritized Remediation Plan

### Quick Wins (1-2 days)

| Priority | Issue | Effort |
|----------|-------|--------|
| Critical | **Add CSRF protection** — Use `X-CSRF-Token` custom header pattern | 4h |
| Critical | **Apply rate limiter to login** — Register `RateLimitMiddleware` on `/api/login` | 1h |
| Critical | **Remove JWT from WebSocket URL** — Authenticate via subprotocol header or initial message | 4h |
| High | **Harden JWT secret** — Remove fallback, crash on missing `JWT_SECRET` env var | 0.5h |
| High | **Add security headers middleware** — CSP, HSTS, XFO, X-Content-Type-Options, Referrer-Policy | 2h |
| High | **Validate sale prices server-side** — Lookup actual product price, reject client-supplied subtotals | 3h |
| High | **Add sortBy allowlist** to `GetAllProducts` in product repository | 0.5h |

### Short-Term (1 week)

| Priority | Issue | Effort |
|----------|-------|--------|
| High | **Implement refresh token rotation** — Issue new refresh token on each refresh, revoke old one | 4h |
| High | **Hash refresh tokens** — Store SHA-256 hash instead of plaintext | 2h |
| High | **Add ownership/store validation** on all `GET /:id` endpoints | 6h |
| High | **Add file upload validation** — MIME, size, extension checks on all imports | 3h |
| High | **Fix bcrypt cost inconsistency** — Use cost 14 everywhere | 0.5h |
| Medium | **Fix audit context loss** — Don't replace context in audit listener | 1h |
| Medium | **Sanitize CSV exports** — Strip `=`, `+`, `-`, `@` from leading characters | 1h |

### Long-Term (1-2 sprints)

| Priority | Issue | Effort |
|----------|-------|--------|
| Critical | **Use HttpOnly cookies for access tokens** — Store tokens in httpOnly, secure, SameSite cookies | 8h |
| High | **Add server-side input validation** — Implement DTO validation for all endpoints | 16h |
| Medium | **Redis-based rate limiter** — Replace in-memory with Redis for production | 4h |
| Medium | **Remove X-User-ID / X-User-Role response headers** | 0.5h |
| Low | **Graceful startup** — Replace `panic` with `log.Fatal` or graceful retry | 1h |
| Low | **Add govulncheck / npm audit to CI pipeline** | 2h |

---

## List of Critical Vulnerabilities

1. **C-01:** WebSocket JWT in URL query parameter
2. **C-02:** No CSRF protection on any endpoint
3. **C-03:** Hardcoded weak JWT secret fallback
4. **C-04:** No brute-force protection on login

---

## Secure Coding Recommendations

1. **Defense in depth** — Never trust client input. Validate everything server-side.
2. **Principle of least privilege** — Every repository query should be scoped to the user's store/role.
3. **Fail securely** — Validate all env vars on startup; crash on missing required secrets.
4. **Token security** — Never put tokens in URLs. Use headers or cookies with proper flags.
5. **Input validation** — Define explicit DTOs with validation rules for every endpoint.
6. **Consistent crypto** — Use the same bcrypt cost everywhere. Hash tokens before storage.
7. **Audit integrity** — Preserve user context through event bus pipelines.
8. **Rate limit everything** — Especially auth endpoints and public APIs.

---

## Hardening Checklist

- [ ] Add CSRF protection middleware
- [ ] Move WebSocket auth to subprotocol/tunnel
- [ ] Enforce JWT_SECRET env var (fail on missing)
- [ ] Register rate limiter on login route
- [ ] Add security headers middleware (CSP, HSTS, XFO, X-CT-O, RP, PP)
- [ ] Validate sale prices server-side
- [ ] Add sort allowlist to product repository
- [ ] Rotate refresh tokens on each refresh
- [ ] Hash refresh tokens before storing
- [ ] Add store isolation to all GET endpoints
- [ ] Add file upload validation (MIME, size, extension)
- [ ] Fix bcrypt cost consistency (cost 14 everywhere)
- [ ] Fix audit listener context loss
- [ ] Sanitize CSV exports (formula injection)
- [ ] Remove X-User-ID and X-User-Role response headers
- [ ] Migrate access tokens to httpOnly cookies
- [ ] Add input validation layer for all DTOs
- [ ] Add Redis-based rate limiter for production
- [ ] Run govulncheck and npm audit
- [ ] Add linter rules for security issues (gosec, eslint-plugin-security)

---

## Deployment Security Checklist

- [ ] JWT_SECRET set to strong random value (not default)
- [ ] Database credentials changed (not `admin123`)
- [ ] SSL/TLS enabled with valid certificate
- [ ] CORS_ORIGIN set to specific production domain (not `*`)
- [ ] ENV set to `production` (enables Gin release mode)
- [ ] Security headers configured in reverse proxy or application
- [ ] Rate limiting configured with Redis backend
- [ ] Database connection uses `sslmode=require`
- [ ] COOKIE_SECURE set to `true` (HTTPS only)
- [ ] COOKIE_DOMAIN set to production domain
- [ ] File size limits configured at reverse proxy level
- [ ] Audit logging enabled and monitored
- [ ] Backup strategy in place for database
- [ ] Monitoring and alerting configured

---

## Security Regression Test Checklist

Test these scenarios after every change:

1. **Auth bypass** — Access protected endpoint without token
2. **Privilege escalation** — Cashier accessing admin endpoints
3. **Token forgery** — Modified JWT signature / claims
4. **IDOR** — Access another user's data by changing ID
5. **SQL injection** — SortBy, search parameters with injection payloads
6. **CSRF** — State-changing request without token/custom header
7. **XSS** — Store `<script>` in product/customer name
8. **Price manipulation** — Create sale with zero/discounted prices
9. **Rate limiting** — Brute-force attempt on login
10. **Refresh token reuse** — Replay old refresh token after refresh
11. **Hardcoded secrets** — Ensure JWT_SECRET is set in environment
12. **Security headers** — Verify CSP, HSTS, XFO headers on responses
