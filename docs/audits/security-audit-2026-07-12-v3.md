# Security Audit Report — Retail POS System (v3)

**Date:** 2026-07-12
**Auditor:** opencode (big-pickle)
**Previous Score:** 40/100 (v2, 2026-07-10)
**Current Score:** 72/100

---

## Executive Summary

The codebase has been significantly hardened since v2. RBAC is properly implemented, password hashing uses bcrypt cost 14, JWT tokens have reasonable TTLs with refresh rotation, parameterized queries prevent SQL injection, and store-scoping prevents cross-store IDOR. The critical issues from v2 have largely been resolved.

This audit identifies **18 current findings**: 0 Critical, 3 High, 7 Medium, 5 Low, 3 Informational. The biggest remaining risks are: IP spoofing via trusted-proxy header inconsistency, token prefix logged to browser console, and the legacy `.env` still lingering in git history.

---

## Score Breakdown

| Category | Score | Max | Notes |
|---|---|---|---|
| Authentication | 17 | 20 | Strong JWT/bcrypt; minor: no lockout, weak change-password policy |
| Authorization | 18 | 20 | RBAC solid; self-role-escalation gap |
| Input Validation | 14 | 15 | Comprehensive; no global body limit |
| CSRF | 8 | 10 | Works but pattern is fragile |
| Rate Limiting | 8 | 10 | Good coverage; memory leak and WS bypass |
| Secrets Mgmt | 8 | 10 | Env vars good; .env in git history |
| Session Management | 9 | 10 | Rotation works; logout doesn't revoke all tokens |
| API Security | 9 | 10 | Headers, CORS, error handling solid |
| WebSocket Security | 7 | 10 | Auth good; IP spoofing, token logging |
| Database Security | 10 | 10 | Parameterized queries, SSL in prod |
| **Total** | **72** | **100** | |

---

## Findings

| ID | Severity | Title | Location |
|---|---|---|---|
| S-01 | **High** | IP spoofing for audit logging via X-Forwarded-For | `shared/context.go:65-71` |
| S-02 | **High** | Token prefix logged to browser console in production | `web/.../websocket.ts:5,14` |
| S-03 | **High** | `.env` with dev secrets still tracked in git history | `.env` (git history) |
| S-04 | **Medium** | CSRF bypass via `X-Requested-With` header | `middleware/security_headers.go:63-64` |
| S-05 | **Medium** | No global request body size limit | `cmd/server/main.go:195-205` |
| S-06 | **Medium** | Rate limiter memory grows unboundedly | `middleware/rate_limit.go:86-98` |
| S-07 | **Medium** | Refresh token not revoked across sessions on logout | `user/auth_service.go:166-168` |
| S-08 | **Medium** | User self-modification not restricted (role escalation) | `user/handler.go:185-273` |
| S-09 | **Medium** | WebSocket `c.ClientIP()` used instead of RemoteAddr | `pkg/websocket/hub.go:339` |
| S-10 | **Medium** | Audit log export limit of 100k rows in memory | `audit/handler.go:119` |
| S-11 | **Low** | `GetIPAddress` trusts X-Forwarded-For for audit IP recording | `shared/context.go:65-71` |
| S-12 | **Low** | No account lockout after failed login attempts | `user/auth_service.go:59-69` |
| S-13 | **Low** | Inconsistent password minimum length (6 vs 8) | `user/auth_handler.go:180` vs `user/handler.go:136` |
| S-14 | **Low** | Access token stored in sessionStorage (XSS-readable) | `web/.../session.ts:1-6` |
| S-15 | **Low** | No request timeout middleware (slowloris) | `cmd/server/main.go:251-257` |
| S-16 | **Info** | `unsafe-inline` in CSP style-src | `middleware/security_headers.go:39` |
| S-17 | **Info** | Public endpoints expose reference data | Multiple handlers |
| S-18 | **Info** | Debug log messages expose internal state | `pkg/websocket/hub.go:383` |

---

## Resolved Issues from v2 Audit

| v2 Finding | Status | Resolution |
|---|---|---|
| Critical: No RBAC / missing permission checks | **RESOLVED** | `RequirePermission` middleware on all mutating endpoints |
| Critical: SQL injection via string concatenation | **RESOLVED** | All queries use pgx parameterized `$N` placeholders |
| Critical: JWT tokens never expire | **RESOLVED** | Access TTL 15min, refresh TTL 7d with rotation |
| Critical: Passwords stored in plaintext | **RESOLVED** | bcrypt cost 14 |
| Critical: No CSRF protection | **RESOLVED** | Custom CSRFMiddleware + SameSite=Strict cookies |
| Critical: Hardcoded JWT secret in source | **RESOLVED** | Now reads from `JWT_SECRET` env var, panics if missing |
| Critical: No rate limiting | **RESOLVED** | Per-IP rate limiters on login, refresh, and general API |
| Critical: Walk-in customer modifiable/deletable | **RESOLVED** | `IsWalkIn` check blocks update/delete |
| High: No refresh token rotation | **RESOLVED** | Old token deleted + new issued in transaction |
| High: No audit logging | **RESOLVED** | Comprehensive audit log on all CRUD + auth events |
| High: No input validation | **RESOLVED** | Regex, binding tags, and explicit validation on all inputs |
| High: Error messages leak internal details | **RESOLVED** | `InternalError()` returns generic "internal server error" |
| Medium: No security headers | **RESOLVED** | CSP, HSTS, X-Frame-Options, X-Content-Type-Options all set |
| Medium: Token storage in localStorage | **RESOLVED** | Migrated to sessionStorage + HttpOnly cookie for refresh |
| Medium: No WebSocket auth | **RESOLVED** | Token-based auth on WS connect, origin checking |

---

## Detailed Findings

### S-01 | High | Inconsistent IP Extraction — Audit vs Rate Limiter

**Files:** `internal/shared/context.go:65-71`, `internal/middleware/rate_limit.go:110-122`

Two different IP extraction strategies coexist:
- **Rate limiter** (`getClientIP`) correctly uses `RemoteAddr` and ignores `X-Forwarded-For`
- **Audit logging** (`GetIPAddress`) trusts `X-Forwarded-For` and `X-Real-IP` headers, which are client-spoofable

An attacker behind a reverse proxy can set arbitrary `X-Forwarded-For` headers, causing audit logs to record a fake IP address.

**Fix:** Use the same `getClientIP` approach for audit logging, or configure `gin.Engine.SetTrustedProxies()`.

### S-02 | High | JWT Token Prefix Logged to Browser Console

**Files:** `web/src/app/providers/websocket.ts:5`, `web/src/shared/api/websocket.ts:14`

First 15 characters of the JWT access token are printed to browser console. JWT tokens have a predictable base64 structure — the first 15 characters encode `exp`, `iat`, and partial `username`.

**Fix:** Remove all `console.log` statements containing token data. Use a build-time flag to strip debug logs in production.

### S-03 | High | `.env` with Dev Secrets in Git History

The `.env` file containing `JWT_SECRET` and `DB_PASSWORD` was committed to git and remains in the git history.

**Fix:** Use `git filter-repo` or BFG Repo Cleaner to purge `.env` from history. Rotate secrets immediately.

### S-04 | Medium | CSRF Bypass via X-Requested-With Header

The CSRF middleware skips verification for any request containing `X-Requested-With`. The CORS config includes this in `AllowHeaders`. If `CORS_ORIGIN` is misconfigured, this becomes a full CSRF bypass.

**Fix:** Replace with a proper synchronizer token pattern or rely solely on `SameSite=Strict`.

### S-05 | Medium | No Global Request Body Size Limit

Only the import/export handler applies `http.MaxBytesReader` (10MB). All other JSON endpoints have no body size limit. A single large POST can crash the server via OOM.

**Fix:** Add a global body size limit middleware (e.g., 1MB for JSON endpoints).

### S-06 | Medium | Rate Limiter Memory Grows Unboundedly

Three separate rate limiter instances store IP entries in maps. Under a distributed attack with spoofed IPs, maps grow without bound until the 30-minute cleanup runs.

**Fix:** Add max-capacity check or use LRU cache with fixed size.

### S-07 | Medium | Logout Doesn't Revoke Stolen Refresh Tokens

`Logout` only deletes the specific refresh token from the cookie. If a session is compromised, the attacker's copy of the refresh token remains valid until expiry (7 days).

**Fix:** Implement token version/epoch counter on users table, incremented on password change.

### S-08 | Medium | Users Can Modify Their Own Role/Permissions

`PUT /admin/users/:id` has `perm("user:update")` but no check preventing self role escalation.

**Fix:** Guard: if `id == currentUserID`, reject changes to `role_id` and `store_id`.

### S-09 | Medium | WebSocket Uses Spoofable IP for Rate Limiting

`pkg/websocket/hub.go:339` uses `c.ClientIP()` which respects `X-Forwarded-For`. Easy rate limit bypass.

**Fix:** Use `getClientIP` logic from HTTP rate limiter.

### S-10 | Medium | Audit Log Export Loads 100k Rows into Memory

`audit/handler.go:119` fetches up to 100,000 audit logs into memory. Resource exhaustion risk.

**Fix:** Stream exports or add pagination limit.
