# Architecture Audit — Retail POS System

**Date:** 2026-07-12
**Auditor:** opencode (big-pickle)
**Previous Score:** 5.5/10 (2026-07-10)
**Current Score:** 5.0/10

---

## Executive Summary

Score downgraded from 5.5 to 5.0. Multiple previous findings remain open, CI/CD is still missing (empty workflows directory), and new duplication/discovery issues were found. On the positive side, several significant cleanups were completed: dead `web/src/lib/` (34 files), dead SvelteKit routes, dead migration directory, product repository god file split, and CSV injection sanitization.

---

## Score Breakdown

| Area | Score | Notes |
|---|---|---|
| Backend Structure | 7/10 | Clean layers, good module separation, but no interfaces |
| Frontend Structure | 6/10 | Modular Svelte 5, but 3 HTTP clients remain |
| Dead Code | 8/10 | Major cleanup done — `shared.JakartaLocation()` is minor |
| Code Duplication | 3/10 | Timezone duplication is the worst offender (12 copies) |
| God Objects | 5/10 | Product split done, but `report/handler.go` (681) and `user/handler.go` (486) remain |
| Error Handling | 6/10 | Generally proper, but some silent drops |
| Configuration | 6/10 | Centralized config exists, but hardcoded values persist |
| Database Schema | 7/10 | Clean migrations, but no migration tracking |
| Testing | 6/10 | Good file count, but all real-DB, no mocked unit tests |
| CI/CD | 1/10 | Empty workflows directory — no pipelines |
| **Overall** | **5.0/10** | |

---

## Previous Audit Findings Status

| Previous Finding | Status | Notes |
|---|---|---|
| A-01 DB connection pooling | ✅ RESOLVED | `pgxpool` properly used |
| A-02 Jakarta timezone duplication (12 init functions) | ❌ STILL OPEN | 12 duplicate `var jakartaLoc` + `init()` remain. `shared.JakartaLocation()` exists but zero callers |
| A-03 Dead `web/src/lib/` (34 files) | ✅ RESOLVED | Directory fully deleted |
| A-04 God file `product/repository.go` (1204 lines) | ✅ RESOLVED | Split into repository.go, query.go, bulk.go |
| A-05 Duplicate config (false positive) | — FALSE POSITIVE | Was already centralized |
| A-06 Backend DI in `main.go` | ❌ STILL OPEN | 281 lines, 30+ manual dependencies |
| A-07 Triple API client (frontend) | ❌ STILL OPEN | Now 2 clients + authApi in auth-service (3 HTTP contexts) |
| A-08 Error swallowing | ❌ STILL OPEN | Several silent error drops remain |
| A-09 Hardcoded values | ❌ STILL OPEN | `pos-service.ts:40` '2025-01-01', `testdb.go:18` admin123 |
| A-10 No interfaces for repo dependencies | ❌ STILL OPEN | middleware/auth.go:21 depends on concrete type |
| A-11 No E2E tests | ✅ RESOLVED | 28 E2E specs, 74 frontend unit tests, 54 backend test files |
| A-12 No integration tests | ⚠️ PARTIAL | Backend tests use real DB — no mocked unit tests |
| A-13 No CI/CD | ❌ STILL WORSE | `.github/workflows/` is empty |
| A-14 CSV injection sanitization | ✅ RESOLVED | Already fixed |
| A-15 Dead migration directory | ✅ RESOLVED | Fully deleted |
| A-16 Frontend SvelteKit routes dead code | ✅ RESOLVED | Fully deleted |
| A-17 Dual `cn()` utility | ✅ RESOLVED | Removed |
| A-18 `report/handler.go` god handler (653 lines) | ❌ STILL OPEN | Grew to 681 lines |
| A-19 EventBus `interface{}` payloads | ❌ STILL OPEN | Low priority |

---

## New Findings

| ID | Finding | Severity |
|---|---|---|
| N-01 | `shared.JakartaLocation()` exists but is never called — dead abstraction | Low |
| N-02 | `auth-service.ts` has duplicate `refreshAccessToken` / `refreshTokenSilently` (identical logic) | Medium |
| N-03 | Frontend auth module creates separate `authApi` axios instance duplicating `apiClient` | Medium |
| N-04 | `report/handler.go` grew from 653 to 681 lines (no split performed) | Medium |
| N-05 | `user/handler.go` (486 lines) contains inline validation, bcrypt imports, email regex — mixed concerns | Medium |
| N-06 | `sale/handler.go` (393 lines) contains inline request type definitions and manual JSON construction | Medium |
| N-07 | Backend test files (54) are real-DB tests, not mocked — true unit tests absent | Medium |
| N-08 | `main.go:81-84` duplicates timezone loading already done by config.Load() | Low |
| N-09 | `testdb.go:38` uses ignored return value `_, _ =` | Low |

---

## Top 5 Priorities

1. **Consolidate timezone duplication** — Adopt `shared.JakartaLocation()` across all 12 packages (2-3 hours)
2. **Add CI pipeline** — Lint, test, build workflows in `.github/workflows/` (1-2 days)
3. **Split `report/handler.go`** — Into chart, dashboard, and export handlers (2-3 days)
4. **Standardize frontend HTTP client** — Remove `authApi`, merge refresh functions (2-3 days)
5. **Add interface abstractions** — Enable mocked unit tests for faster development (ongoing)

---

## Strengths

1. Clean module structure — Handler → Service → Repository layering consistent across all 10 domain packages
2. No circular dependencies — Clean direction: cmd → handler → service → repository → eventbus
3. Well-designed import/export framework — generic, schema-driven, and extensible
4. Good test structure — 54 backend test files, 74 frontend unit tests, 28 E2E specs
5. Role-based access control — Clean permission middleware with RequirePermission, RequireAnyPermission, AdminOnly
6. EventBus decoupling — Sale events properly trigger stock adjustments asynchronously via listeners
7. WebSocket real-time — Hub pattern with proper listener architecture for live updates
8. Product split completed — product/repository.go successfully split into CRUD, query, and bulk files
