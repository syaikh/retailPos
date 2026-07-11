# Security Audit Report: Retail POS System

**Date:** July 10, 2026
**Scope:** Full-stack (Go backend, Svelte 5 frontend, PostgreSQL)
**Previous Audit:** `docs/security-audit.md` (June 29, 2026)

---

## Remediation Status (from June 29 Audit)

| Finding | Status |
|---|---|
| C-01: JWT in WebSocket URL | ✅ **FIXED** — Auth via message protocol |
| C-02: No CSRF protection | ⚠️ **PARTIAL** — Middleware exists but bypassed via Authorization header + static "1" token |
| C-03: Weak JWT secret fallback | ✅ **FIXED** — Panics on missing `JWT_SECRET` |
| C-04: No rate limiter registered | ✅ **FIXED** — Registered globally in `main.go:190` |
| H-01: No input validation | ❌ **NOT FIXED** — Product/handler.go still binds directly without validation |
| H-02: SQL injection in sortBy | ✅ **FIXED** — Allowlist added in `product/repository.go:303` |
| H-03: Refresh token rotation | ✅ **FIXED** — `auth_service.go:148-158` deletes old + creates new |
| H-04: Refresh token plaintext | ✅ **FIXED** — SHA-256 hash via `hashToken()` |
| H-05: IDOR on single-resource | ❌ **NOT FIXED** — `GetSaleByID` still lacks store isolation |
| H-06: Price manipulation | ✅ **FIXED** — `SetPriceStore` with server-side lookup |
| H-07: No file upload validation | ❌ **NOT FIXED** |
| H-08: bcrypt cost inconsistency | ✅ **FIXED** — Consistently cost 14 |

---

## 🔴 HIGH (3 Findings)

### H-2026-01: CSRF Middleware Bypass via Authorization Header + Static Token

**Files:** `internal/middleware/security_headers.go:51-70`, `cmd/server/main.go:207`

The CSRF middleware skips all requests with an `Authorization` header (`line 58`). Since the frontend sends Bearer tokens on every authenticated request (via Axios interceptor), **no authenticated request is ever checked for CSRF**. The CSRF token value is the hardcoded string `"1"` (`line 64`), which offers zero protection — any attacker can send `X-CSRF-Token: 1`.

Additionally, the refresh endpoint (`POST /api/refresh`) reads the `refresh_token` from an `httpOnly` cookie but is registered **before** the CSRF middleware in `main.go:204`, so it has no CSRF protection at all. A malicious site can trigger a POST to `/api/refresh`, the browser auto-sends the cookie, and the attacker receives a new access token via the response.

**Risk:** Session hijacking via CSRF on the refresh endpoint. An attacker can forge authenticated requests on any Bearer-token endpoint.

**Recommendation:**
1. Replace the static "1" token with a real check: require either `Authorization` header (Bearer token — not auto-sent by browsers) or `X-Requested-With` custom header.
2. Move `/api/refresh` behind CSRF middleware.

---

### H-2026-02: Hardcoded Database DSN with Credentials + SSL Disabled

**Files:** `cmd/server/main.go:92-94`, `.env.example:12`

```go
if dsn == "" {
    dsn = "postgres://pos:admin123@localhost:5433/retail_pos?sslmode=disable&timezone=Asia/Jakarta"
}
```

The hardcoded DSN contains database credentials (`pos:admin123`) and uses `sslmode=disable` (unencrypted connection). If the `DATABASE_URL` env var is accidentally unset in production, the server connects with these weak credentials over an unencrypted channel. The `.env` and `.env.example` files also contain `JWT_SECRET=dev-jwt-secret-change-in-production` and `DB_PASSWORD=admin123`.

**Risk:** Database compromise in production via weak default credentials. MITM attack on database connection.

**Recommendation:** Remove hardcoded DSN fallback. Require `DATABASE_URL` env var. Use `sslmode=require` in production.

---

### H-2026-03: Internal Error Messages Leaked to API Consumers

**Files:** ~83 occurrences across all handler files:
- `internal/user/handler.go` (lines 91, 137, 200, 214, 223, 240, 269, 287, 292, 306, 314, 323)
- `internal/sale/handler.go` (lines 112, 185, 207, 247, 312)
- `internal/report/handler.go` (lines 254, 260, 287, 328, 339, 366, 375, 406, 452, 461, 495, 536, 581, 592)
- `internal/customer/handler.go` (lines 53, 71, 101, 138, 152, 169, 185)
- `internal/category/handler.go` (lines 29, 54, 69, 90, 104)
- `internal/platform/importexport/handler/handler.go` (lines 128, 197, 249, 273, 327, 361)
- And others

Pattern used everywhere:
```go
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// or
shared.JSONError(c, http.StatusInternalServerError, err.Error())
```

Internal Go error messages (SQL query details, file paths, database connection info, stack traces) are returned directly to the API consumer. This aids attacker reconnaissance.

**Risk:** Information disclosure — SQL syntax, table structures, file paths, and infrastructure details leaked.

**Recommendation:** Create a sanitized error helper that logs the real error server-side and returns a generic `"internal server error"` message to the client.

---

## 🟡 MEDIUM (6 Findings)

### M-2026-01: Access Token Stored in sessionStorage (XSS-Vulnerable)

**Files:** `web/src/modules/auth/lib/session.ts:8-9`, `web/src/lib/api/client.ts:28`

The JWT access token is stored in `sessionStorage`, which is accessible to any JavaScript running on the page. An XSS vulnerability (even via a dependency) would allow token exfiltration. The refresh token is also stored in `sessionStorage` in addition to the `httpOnly` cookie, creating an inconsistent dual-storage model.

**Recommendation:** Store access tokens in `httpOnly` cookies, or use a backend-for-frontend (BFF) pattern.

---

### M-2026-02: IDOR — GetSaleByID Lacks Store Isolation

**Files:** `internal/sale/handler.go:192-211`, `internal/sale/repository.go:117-134`

`GetSaleByID` queries `WHERE s.id = $1` without filtering by `storeID`. A cashier at Store A can enumerate sale IDs and read sales from Store B. The list endpoint (`GetSalesHistory`) properly filters by store, but the single-resource endpoint does not.

**Recommendation:** Add `store_id` filter to `GetSaleByID` repository query.

---

### M-2026-03: No Dedicated Brute-Force Protection on Login

**Files:** `internal/middleware/rate_limit.go:51-63`, `cmd/server/main.go:190`

The global rate limiter (5 req/s, burst 10) is applied uniformly to ALL routes. There is no stricter per-endpoint limit on `/api/login`, no account lockout, and no progressive delay after failed attempts. An attacker can sustain 18,000 login attempts per hour.

**Recommendation:** Add a dedicated low-rate limiter on the login endpoint (e.g., 5 attempts/minute/IP) and implement account lockout after N failures.

---

### M-2026-04: JWT Token Accepted from URL Query Parameter

**Files:** `internal/middleware/auth.go:137-139`

```go
token := c.Query("token")
if token != "" {
    return token
}
```

The `extractToken` function accepts JWTs from URL query parameters (`?token=...`). URLs are logged by proxies, web servers, load balancers, and stored in browser history. While the WebSocket auth was fixed to use a message protocol, the HTTP auth middleware still supports query parameter tokens.

**Recommendation:** Remove query parameter token extraction. Only accept tokens from `Authorization` header.

---

### M-2026-05: Refresh Token Cookie Missing SameSite Attribute

**Files:** `internal/user/auth_handler.go:48,74,135`

```go
c.SetCookie("refresh_token", resp.RefreshToken, int(7*24*time.Hour/time.Second), "/", domain, secure, true)
```

The `SameSite` attribute is not set. Modern browsers default to `SameSite=Lax`, which permits top-level cross-origin POST requests to send the cookie. Combined with the missing CSRF protection on `/api/refresh`, this enables session hijacking.

**Recommendation:** Set `SameSite=Strict` (or at minimum `SameSite=Lax`) on all cookies.

---

### M-2026-06: Refresh Endpoint Outside CSRF Protection

**Files:** `cmd/server/main.go:204`

The `/api/refresh` route is registered in `authH.RegisterRoutes`, which is called at line 204 — **before** the CSRF middleware is applied at line 207. The refresh endpoint reads the `refresh_token` from an `httpOnly` cookie (auto-sent by browsers) and has no CSRF protection. This enables cross-site request forgery.

**Recommendation:** Move `/api/refresh` behind CSRF middleware.

---

## 🟢 LOW (8 Findings)

| Finding | File |
|---|---|
| CSP allows `'unsafe-inline'` for styles | `middleware/security_headers.go:39` |
| WebSocket origin check allows empty Origin + broad prefixes | `pkg/websocket/hub.go:39-45` |
| No password complexity validation | `user/handler.go:97-101` |
| StoreID type assertion mismatch (`*int` → `int`) in report handlers | `report/handler.go:248-249` |
| Defaults to development mode when `ENV` unset | `config/config.go:45-48` |
| Error string comparison instead of `errors.Is()` | `sale/handler.go:203` |
| X-User-ID / X-User-Role response headers leak internal info | `middleware/auth.go:49-50` |
| No file upload validation (MIME, size) on import handlers | `product/handler.go:760` |

---

## Summary

| Severity | Count |
|---|---|
| 🔴 HIGH | 3 |
| 🟡 MEDIUM | 6 |
| 🟢 LOW | 8 |
| **Total** | **17** |

**Security Score:** ~65/100 (improved from 52/100 on June 29)
