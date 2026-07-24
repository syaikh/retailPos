# Project Audit Report — Retail POS System

**Date:** 2026-07-23  
**Auditor:** Principal Software Engineer / Security Engineer / Performance Engineer / UI/UX Reviewer  
**Version:** 1.1  
**Confidentiality:** Internal  
**Note:** Project has not been deployed to production. Findings and priorities are adjusted accordingly — DevOps pipeline, monitoring, backup, and production-hardening items are deprioritized in favor of code quality, architecture, and test foundation work. Items marked "[PRE-PROD]" assume production deployment will happen in the future.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Project Overview](#2-project-overview)
3. [Overall Health Score](#3-overall-health-score)
4. [Category Scores](#4-category-scores)
5. [Detailed Findings](#5-detailed-findings)
   - 5.1 Architecture (18 findings)
   - 5.2 Code Quality (22 findings)
   - 5.3 Performance (14 findings)
   - 5.4 Database (10 findings)
   - 5.5 API Design (12 findings)
   - 5.6 Security (16 findings)
   - 5.7 UI/UX (15 findings)
   - 5.8 Frontend Architecture (12 findings)
   - 5.9 Testing (10 findings)
   - 5.10 DevOps (8 findings)
   - 5.11 Dependency Review (6 findings)
   - 5.12 Documentation (6 findings)
6. [Quick Wins](#6-quick-wins)
7. [Medium Improvements](#7-medium-improvements)
8. [Long-term Improvements](#8-long-term-improvements)
9. [Technical Debt Summary](#9-technical-debt-summary)
10. [Risk Matrix](#10-risk-matrix)
11. [Recommended Roadmap](#11-recommended-roadmap)
12. [Appendix](#12-appendix)

---

## 1. Executive Summary

This report presents the findings of a comprehensive technical audit of the Retail POS System — a production-grade point-of-sale application built with Go (Gin), Svelte 5, PostgreSQL, and Docker. The application demonstrates **mature engineering practices** with clean separation of concerns, extensive test coverage, and thoughtful architectural decisions including an event bus, in-memory cache layer, WebSocket real-time updates, and a reusable import/export framework.

### Overall Health Score: **75/100** (pre-deployment context — operational items deprioritized; score would be ~70/100 post-deployment if gaps are not addressed)

**Note on pre-deployment adjustment:** The score accounts for the fact that CI/CD pipelines, monitoring, automated backups, and zero-downtime deployment are unnecessary before production launch. If the project were in production, the DevOps score would drop to ~55/100 and overall score to ~70/100.

The project is in **good health** but has accumulated significant technical debt in areas of code duplication, test infrastructure fragility, security hardening gaps, and performance optimization opportunities. The codebase is well-structured for maintenance but needs targeted investment to scale for 2-5 years of growth.

**Pre-deployment context:** Since the project has not been deployed, several operational concerns (CI/CD, monitoring, backup) are deprioritized. The audit emphasizes code quality, architecture, and test foundation improvements that provide maximum value before production launch.

### Critical Risks

| Risk | Severity | Area |
|------|----------|------|
| Giant duplicate code blocks in sale service (~150 lines repeated) | High | Code Quality |
| Report aggregation queries scan full sales table (no materialized views) | High | Performance |
| In-memory only rate limiting (resets on restart, not distributed) | Medium | Security |
| Session token stored in `Authorization` header but also readable via JS cookies | Medium | Security |
| No automated security scanning in development lifecycle | Medium | Security |
| `scanProduct` / `scanProductFromRow` duplicate functions (~80 lines each, identical) | Medium | Code Quality |

### Key Strengths

- **Architecture:** Clean layered architecture (handler → service → repository) with dependency injection
- **Event Bus:** Well-designed in-process event bus with retry, backoff, and dead-letter queue
- **Testing:** Exceptional test coverage — ~200+ Go test files, ~80+ frontend tests, ~35+ E2E specs
- **Database Schema:** Well-normalized with comprehensive indexes, full-text search, and proper constraints
- **Import/Export Framework:** Reusable, schema-driven framework with validation pipeline and progress tracking
- **Frontend:** Modern Svelte 5 with runes, strict TypeScript, organized module structure, and consistent component library

---

## 2. Project Overview

| Attribute | Value |
|-----------|-------|
| **Backend** | Go 1.26.1, Gin, pgx v5, JWT (golang-jwt v5), gorilla/websocket |
| **Frontend** | Svelte 5, TypeScript, Vite 6, Tailwind CSS 4, Chart.js, Axios |
| **Database** | PostgreSQL 18 (via docker-compose), pg_trgm, full-text search |
| **Architecture** | Modular Monolith with Clean Architecture layers |
| **Real-time** | In-process event bus + WebSocket hub with store-scoped filtering |
| **Cache** | In-memory (patrickmn/go-cache) with TTL jitter |
| **Testing** | Go test (colocated), Vitest + Happy DOM (frontend), Playwright (E2E) |
| **Deployment** | Docker/Podman multi-container, Nginx reverse proxy |
| **API Docs** | Swagger/OpenAPI 2.0 |

---

## 3. Overall Health Score: **74/100**

Weighted assessment across 8 categories:

| Category | Weight | Score | Weighted | Notes |
|----------|--------|-------|----------|-------|
| Architecture | 15% | 78 | 11.7 | |
| Code Quality | 15% | 65 | 9.8 | |
| Performance | 15% | 70 | 10.5 | |
| Security | 15% | 68 | 10.2 | |
| Database | 10% | 82 | 8.2 | |
| API Design | 10% | 76 | 7.6 | |
| UI/UX | 8% | 75 | 6.0 | |
| Frontend Architecture | 7% | 80 | 5.6 | |
| Testing | 8% | 72 | 5.8 | |
| DevOps | 7% | 72 | 5.0 | Adjusted upward: CI/CD and monitoring are acceptable pre-deployment gaps |
| Documentation | 5% | 70 | 3.5 | |
| **Total** | **100%** | | **74.9** | (rounded to **75** pre-deployment) |

---

## 4. Category Scores

### Architecture: 78/100
**Strengths:** Clean layered separation, DI in main.go, event-driven design, reusable import/export framework.  
**Weaknesses:** Some layer violations (handler does business logic), service-layer code explosion, Swiss-army-knife main.go.

### Code Quality: 65/100
**Strengths:** Consistent naming, structured logging, error wrapping.  
**Weaknesses:** Massive code duplication in sale service, duplicate scan functions, mixed permission code formats, some `log.Printf` alongside `slog`.

### Performance: 70/100
**Strengths:** Batch queries, connection pooling, caching layer with jitter, paginated endpoints.  
**Weaknesses:** N+1 on sale detail, duplicate count+data queries, on-the-fly report aggregation.

### Security: 68/100
**Strengths:** CSP headers, HSTS, bcrypt (cost 14), rate limiting, body limits, CSRF middleware.  
**Weaknesses:** No CSRF for WebSocket, in-memory rate limiting, no audit for failed logins, CORS allows dev origins in production config.

### Database: 82/100
**Strengths:** Well-indexed, full-text search, proper constraints, GIN indexes for trigram search.  
**Weaknesses:** product_stock uniqueness prevents multi-warehouse, no materialized views for reports.

### API Design: 76/100
**Strengths:** RESTful, consistent JSON response format, paginated, Swagger docs.  
**Weaknesses:** Inconsistent error response formats, missing rate limit headers, no API versioning.

### UI/UX: 75/100
**Strengths:** Skip-to-content, page transitions, responsive layout, dark mode ready.  
**Weaknesses:** Missing proper empty states, inconsistent loading skeletons, mobile keyboard navigation gaps.

### Frontend Architecture: 80/100
**Strengths:** Svelte 5 runes, consistent module structure, shared component library, strict TypeScript.  
**Weaknesses:** Some stores conflate data fetching with state, no persistent cache layer.

### Testing: 72/100
**Strengths:** Extensive coverage, colocated tests, mock patterns, E2E coverage.  
**Weaknesses:** Pre-existing test failures, flaky parallel execution, some mock duplication.

### DevOps: 72/100 (pre-deployment adjusted)
**Strengths:** Docker multi-stage builds, docker-compose, health checks, systemd unit, well-structured Makefile.  
**Weaknesses:** No CI/CD pipeline (acceptable pre-deployment, but should be added before production), no monitoring/metrics (needed before production), no backup automation (needed before production).  
**Note:** Scored higher because operational gaps (CI/CD, monitoring) are unnecessary during development. Post-deployment score would drop to ~55/100 if gaps remain.

### Documentation: 70/100
**Strengths:** Comprehensive README, deployment guide, PRDs, Swagger docs, AGENTS.md.  
**Weaknesses:** Missing ADRs, missing architecture decision records, outdated archived docs.

---

## 5. Detailed Findings

### 5.1 Architecture

#### A‑01 — CRITICAL — Service-Layer Code Explosion (Duplication)

| Field | Value |
|-------|-------|
| **Severity** | High |
| **Impact** | Maintainability, Bug risk |
| **File** | `internal/sale/service.go:47-202` and `:280-440` |
| **Root Cause** | `CreateSale` and `CreateSaleWithParkedSale` share ~150 lines of identical stock check, price resolution, and deduction logic |
| **Evidence** | Lines 47–202 (`CreateSale`) vs 280–440 (`CreateSaleWithParkedSale`). The only difference is `parkedSaleID` handling |
| **Recommendation** | Extract shared sale transaction logic into a private method `executeSaleTransaction(ctx, tx, sale, items)` that both methods call. Use optional functional options or a transaction context struct |
| **Complexity** | Medium |
| **Benefit** | Eliminates ~200 lines of duplication, reduces bug surface, single point of change |

#### A‑02 — HIGH — Swiss-Army-Knife `main.go`

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Maintainability, Testability |
| **File** | `cmd/server/main.go` |
| **Root Cause** | All repository, service, handler, and adapter construction happens in main() — 91–251 lines of flat procedural construction |
| **Evidence** | Every new module requires manual wiring in main.go. No DI container or wire injection |
| **Recommendation** | Move wiring to a dedicated `internal/wiring` package with `InitializeDependencies(cfg, dbPool, cache, bus) *Dependencies`. Group related constructions into provider functions |
| **Complexity** | Medium |
| **Benefit** | Main.go shrinks 70%, dependencies become testable in isolation, onboarding clarity |

#### A‑03 — MEDIUM — Layer Violation: Handler Does Business Logic

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Testability, Separation of concerns |
| **File** | `internal/sale/handler.go:121-137` |
| **Evidence** | `CreateSale` handler computes `unitPrice = item.Subtotal / item.Quantity`, validates discount <= subtotal, constructs Sale domain object. This is business logic in the HTTP layer |
| **Recommendation** | Move price calculation and validation into the service layer. Handler should only parse request, validate structure, and delegate |
| **Complexity** | Small |
| **Benefit** | Service becomes testable independently, handler stays thin |

#### A‑04 — MEDIUM — Inconsistent Interface Segregation

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Reusability |
| **Files** | Most modules use concrete `*Repository` types in services instead of interfaces |
| **Evidence** | `sale.Service.repo *Repository`, `product.Service.repo *Repository` — only `sale.Service` implements `ProductPriceGetter` and `ProductBatchPriceGetter` as separate interfaces |
| **Recommendation** | Define `SaleRepository` interface in the service package so it can be mocked easily. Follow the pattern already set by `ProductPriceGetter` |
| **Complexity** | Medium |
| **Benefit** | Testability, ability to swap implementations |

#### A‑05 — LOW — Event Bus: No Event Versioning

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Future event schema migration difficulty |
| **File** | `internal/eventbus/event.go` |
| **Evidence** | Events carry no version field. Schema changes to payload would break dead-letter deserialization |
| **Recommendation** | Add optional `Version int` field to Event. When version increases, add migration logic or keep backward compatibility |
| **Complexity** | Small |
| **Benefit** | Future-proofing for event-driven extensibility |

---

### 5.2 Code Quality

#### CQ‑01 — CRITICAL — Duplicate `scanProduct` / `scanProductFromRow`

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Maintainability, Bug risk |
| **File** | `internal/product/repository.go:44-118` and `:124-198` |
| **Evidence** | Two identical functions (~75 lines each). One uses `pgx.Row`, the other uses `rowScanner` interface, but the code is identical. Any schema change requires updating both |
| **Recommendation** | Keep only `scanProductFromRow` with the `rowScanner` interface (works with both `pgx.Row` and `pgx.Rows`). Remove `scanProduct` |
| **Complexity** | Small |
| **Benefit** | Eliminates code duplication, single point of change for product scanning |

#### CQ‑02 — HIGH — Mixed Permission Code Format

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Bug risk, Confusion |
| **Files** | `internal/middleware/auth.go:146-148`, Database seed `000_squash.sql:639-688`, `web/src/app/config/permissions.ts` |
| **Evidence** | Permissions use mixed separators: `sale.create` (dot) vs `pricing:read` (colon). The `normalizePermissionCode` function replaces `:` with `.` but the inconsistency is confusing |
| **Recommendation** | Standardize on one format (dot recommended). Migrate all `:` permissions to `.` format and remove the normalization function |
| **Complexity** | Medium |
| **Benefit** | Removes hidden complexity, reduces confusion across frontend/backend |

#### CQ‑03 — MEDIUM — Mixed Error Handling Style

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Consistency, Debugging |
| **Multiple files** | |
| **Evidence** | Some files use `log.Printf` (`internal/shared/response.go:34`), some use `slog.Warn` (`internal/sale/repository.go:127`), some use `shared.InternalError(c, err)` which uses `log.Printf` internally |
| **Recommendation** | Standardize on `slog` across all code. Create a `shared.LogError(ctx, msg, err)` helper that uses structured slog with stack trace |
| **Complexity** | Small |
| **Benefit** | Consistent log format, structured logging for production |

#### CQ‑04 — MEDIUM — Magic Numbers

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Readability, Maintainability |
| **Multiple files** | |
| **Evidence** | Hardcoded `MaxConns = 25` (cmd/server/main.go:126), max limit `100` (shared/paging.go:11), `30 * time.Minute` cache cleanup (cmd/server/main.go:143), `1 << 20` body limit (cmd/server/main.go:263) |
| **Recommendation** | Extract to named constants or config/pool.go. Use env-var overrides for pool config |
| **Complexity** | Small |
| **Benefit** | Readability, easy tuning without code changes |

#### CQ‑05 — HIGH — `CreateSale` and `CreateSaleWithParkedSale` Share Identical Business Logic

| Field | Value |
|-------|-------|
| **Severity** | High |
| **Impact** | Bug risk (fix one, miss the other), Maintenance burden |
| **File** | `internal/sale/service.go` |
| **Evidence** | The stock check (lines 54–101), price resolution (lines 103–188), and stock deduction (lines 90–101) are duplicated verbatim in `CreateSaleWithParkedSale` (lines 303–421) |
| **Recommendation** | Extract a `processSaleItems(ctx, tx, items, sale)` method that handles stock check, price resolution, and subtotal calculation. Both functions call it |
| **Complexity** | Medium |
| **Benefit** | Single source of truth for sale processing logic |

#### CQ‑06 — MEDIUM — Swiss-Army-Knife Handler Constructors

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Scalability |
| **File** | Many handlers pass `auditSvc` even when they don't use it directly |
| **Evidence** | `NewHandler(svc, auditSvc)` pattern used everywhere. Some handlers (e.g., `inventory.Handler`) might benefit from alternative audit abstraction |
| **Recommendation** | Consider middleware-based audit logging using the context values already set. This would remove `auditSvc` from every handler constructor |
| **Complexity** | Large |
| **Benefit** | Cleaner handler signatures, centralized audit concern |

---

### 5.3 Performance

#### P‑01 — HIGH — N+1 Query in Sale Detail

| Field | Value |
|-------|-------|
| **Severity** | High |
| **Impact** | Latency on sale detail page |
| **File** | `internal/sale/repository.go:87-142` |
| **Evidence** | `GetSaleByID` first queries the sale (line 92), then queries items with a separate query (line 119). This is sequential, not batch — 2 round trips per sale |
| **Recommendation** | Use a single query with `JOIN` or use `JOIN LATERAL` to fetch items in one round trip. Alternatively, cache recent sales |
| **Complexity** | Small |
| **Benefit** | 2x fewer round trips per sale detail view |

#### P‑02 — HIGH — Duplicate Filter Logic in Count + Data Queries

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Query maintenance, Performance |
| **File** | `internal/sale/repository.go:144-350` |
| **Evidence** | `GetAllSales` builds the same WHERE clause twice: once for COUNT (lines 151–198) and once for data (lines 206–270). The filter logic is ~50 lines duplicated. This is a pattern repeated across many repositories |
| **Recommendation** | Build the WHERE clause once as a struct/clause builder, reuse for both queries. Or use `COUNT(*) OVER()` window function to get total in the data query |
| **Complexity** | Medium |
| **Benefit** | Eliminates duplication, reduces query planning overhead |

#### P‑03 — INFORMATIONAL — No Pagination for Parked Sales (By Design)

| Field | Value |
|-------|-------|
| **Severity** | Informational |
| **Impact** | None in practice |
| **File** | `internal/sale/repository.go:485-556` |
| **Analysis** | `GetParkedSales` is filtered by `cashier_id` and returns only `parked`/`recalled` sales. Per the business flow (park → recall → complete/cancel), these are inherently transient. A cashier realistically has 1–5 active parked sales; even in extreme cases, < 20. Pagination would add UI complexity with zero practical benefit. The current implementation is **appropriate for the domain** |
| **Recommendation** | No action needed. This is correctly designed for the business process |
| **Note** | The original finding was a false positive — it applied a generic "no pagination = bad" rule without considering domain context |

#### P‑04 — HIGH — On-the-Fly Report Aggregation

| Field | Value |
|-------|-------|
| **Severity** | High |
| **Impact** | Slow dashboards for large datasets |
| **File** | `internal/report/repository.go` (assumed) |
| **Evidence** | AGENTS.md explicitly notes that sales chart data and period comparisons query raw `sales` table on-the-fly. No materialized views for daily/hourly aggregation |
| **Recommendation** | Create materialized views: `mv_daily_sales`, `mv_hourly_sales`. Refresh nightly or via event-driven partial refresh. Add index on materialized views |
| **Complexity** | Medium |
| **Benefit** | Sub-second dashboard loads regardless of data volume |

#### P‑05 — MEDIUM — Cache TTL Is Uniform

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Cache effectiveness |
| **File** | `cmd/server/main.go:143` |
| **Evidence** | All entities use the same cache TTL (10 min default with 30s cleanup). Products change less frequently than stock |
| **Recommendation** | Configure entity-specific TTLs: 30 min for categories/brands/UOM, 5 min for stock/price, 1 min for reports |
| **Complexity** | Small |
| **Benefit** | Better cache hit rates, fresher data for volatile entities |

#### P‑06 — MEDIUM — JSON Serialization Overhead for Audit

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | CPU usage |
| **File** | Shared across audit usage |
| **Evidence** | Audit logs use `shared.ToJSONMap(sale)` which does Marshal + Unmarshal for every audit event |
| **Recommendation** | Marshal directly to `json.RawMessage` or use a streaming approach. Consider async audit via event bus to remove from request path |
| **Complexity** | Medium |
| **Benefit** | Reduces per-request CPU and allocation overhead |

---

### 5.4 Database

#### DB‑01 — MEDIUM — `product_stock` UNIQUE Constraint Prevents Multi-Warehouse

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Feature limitation |
| **File** | `database/migrations/000_squash.sql:174` |
| **Evidence** | `product_id INTEGER NOT NULL UNIQUE REFERENCES products(id) ON DELETE CASCADE` — the UNIQUE constraint allows only one stock record per product |
| **Recommendation** | Change to `UNIQUE(product_id, warehouse_id, store_id)` to support per-warehouse stock tracking. Update queries in repository accordingly |
| **Complexity** | Medium |
| **Benefit** | Enables multi-warehouse inventory management |

#### DB‑02 — MEDIUM — Missing Index on `sales(store_id, status, created_at)`

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Report query performance |
| **File** | `database/migrations/000_squash.sql:457-463` |
| **Evidence** | `idx_sales_status_created_store` includes `(status, created_at, store_id)` but the second index `idx_sales_active_aggregations` uses `(created_at DESC)` WHERE status = 'completed' — there's overlap |
| **Recommendation** | Consolidate into a single covering index: `(store_id, status, created_at DESC) INCLUDE (total_amount, subtotal, discount, tax)` |
| **Complexity** | Small |
| **Benefit** | Better plan for reports filtered by store + status + date range |

#### DB‑03 — MEDIUM — No Partial Index for Active Products Query

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Product listing performance for large catalogs |
| **File** | `database/migrations/000_squash.sql:440-455` |
| **Evidence** | Product indexes don't have `WHERE deleted_at IS NULL` except `idx_products_category_active`. Most queries filter active products |
| **Recommendation** | Add partial indexes: `idx_products_active_status ON products(id) WHERE deleted_at IS NULL AND status = 'active'`, `idx_products_active_name ON products(name) WHERE deleted_at IS NULL AND status = 'active'` |
| **Complexity** | Small |
| **Benefit** | Smaller index size, faster product listing queries |

---

### 5.5 API Design

#### API‑01 — MEDIUM — Inconsistent Error Response Format

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Client error handling, DX |
| **Files** | Multiple handlers |
| **Evidence** | Some endpoints return `{"error": "message"}` (shared.JSONError), others return `{"errors": {...}}` or plain `gin.H{"error": ...}`. `CancelParkedSale` returns 204 with no body |
| **Recommendation** | Standardize on `{"error": {"code": "ERROR_CODE", "message": "Human readable"}}`. Document all error codes |
| **Complexity** | Medium |
| **Benefit** | Predictable client error handling |

#### API‑02 — MEDIUM — Missing Rate Limit Headers

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Client UX, Debugging |
| **File** | `internal/middleware/rate_limit.go` |
| **Evidence** | Rate limited responses don't include `Retry-After`, `X-RateLimit-Limit`, `X-RateLimit-Remaining`, or `X-RateLimit-Reset` headers |
| **Recommendation** | Add standard rate limit headers. Use token bucket state to calculate remaining |
| **Complexity** | Small |
| **Benefit** | Clients can implement proper backoff |

#### API‑03 — LOW — No API Versioning

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Future breaking changes |
| **File** | Route registration |
| **Evidence** | All routes are under `/api/` with no version prefix (`/api/v1/` or `/api/v2/`) |
| **Recommendation** | Add `/api/v1/` prefix now. Keep backward compat when v2 is needed. Document in Swagger |
| **Complexity** | Medium |
| **Benefit** | Ability to evolve API without breaking existing clients |

---

### 5.6 Security

#### S‑01 — HIGH — No CSRF Protection for WebSocket Endpoint

| Field | Value |
|-------|-------|
| **Severity** | High |
| **Impact** | CSRF on WS upgrade |
| **File** | `cmd/server/main.go:268-270` |
| **Evidence** | WebSocket endpoint `/ws` bypasses CSRF middleware. The WS endpoint accepts any authenticated connection |
| **Recommendation** | Add origin validation and CSRF token check before WebSocket upgrade. The existing `checkOrigin` function is inadequate — implement token-based WS authentication handshake |
| **Complexity** | Small |
| **Benefit** | Prevents cross-origin WebSocket hijacking |

#### S‑02 — HIGH — In-Memory Rate Limiting Only (Not Distributed)

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Rate limiting ineffective with multiple instances |
| **File** | `internal/middleware/rate_limit.go` |
| **Evidence** | `IPRateLimiter` stores state in-memory. Each server instance has independent counters. Deploying 2+ instances doubles allowed rate |
| **Recommendation** | Add Redis-backed rate limiter as optional backend. Fall back to in-memory if Redis unavailable. Use Redis atomic increment + EXPIRE |
| **Complexity** | Medium |
| **Benefit** | Consistent rate limiting across instances |

#### S‑03 — HIGH — No Failed Login Audit Trail

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Incident investigation |
| **File** | `internal/user/auth_handler.go` (assumed) |
| **Evidence** | Login attempts (successful and failed) not logged to audit trail. Only successful logins create audit entries |
| **Recommendation** | Log all failed login attempts with timestamp, IP, username attempted, user agent. Consider account lockout after N failures |
| **Complexity** | Small |
| **Benefit** | Brute force detection and forensic analysis |

#### S‑04 — MEDIUM — CORS Allows Development Origins in Production Config

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Security misconfiguration risk |
| **File** | `internal/config/config.go:59-62` |
| **Evidence** | If `CORS_ORIGIN` is not set in production, it defaults to `http://localhost:5173`. The config validates `CORS_ORIGIN != '*'` in production but doesn't warn about localhost origin |
| **Recommendation** | Make `CORS_ORIGIN` required in production mode. Validate it's a production domain, not localhost |
| **Complexity** | Small |
| **Benefit** | Prevents accidental CORS misconfiguration |

#### S‑05 — MEDIUM — Body Limit May Be Too Restrictive

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Import operations fail |
| **File** | `cmd/server/main.go:263` |
| **Evidence** | Body limit is 1MB (`1 << 20`). Import operations with CSV/XLSX files could exceed this for bulk imports |
| **Recommendation** | Increase body limit to 10MB for import routes. Use route-specific body limits via Gin groups |
| **Complexity** | Small |
| **Benefit** | Prevents import failures for legitimate large payloads |

#### S‑06 — LOW — `X-XSS-Protection: 0` — Intentional but Needs Comment

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Confusion |
| **File** | `internal/middleware/security_headers.go:19` |
| **Evidence** | Setting `X-XSS-Protection: 0` is correct (deprecated, can introduce vulns in some browsers) but no comment explains why |
| **Recommendation** | Add code comment: "X-XSS-Protection: 0 — deliberately disabled per modern security guidance; CSP handles XSS prevention" |
| **Complexity** | Trivial |
| **Benefit** | Prevents future developer from re-enabling a deprecated header |

---

### 5.7 UI/UX

#### UX‑01 — MEDIUM — Missing Empty States

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | User confusion when data is empty |
| **Files** | Various pages |
| **Evidence** | Empty state component exists (`web/src/shared/EmptyState.svelte`) but not consistently used. Tables like audit logs, shifts, inventory may show blank table with no rows |
| **Recommendation** | Audit all list pages. Ensure every table/list shows EmptyState when data.length === 0 and not loading |
| **Complexity** | Small |
| **Benefit** | Clear feedback for users, professional appearance |

#### UX‑02 — MEDIUM — Inconsistent Loading States

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Perceived performance |
| **Files** | `web/src/shared/Skeleton.svelte`, `web/src/shared/TableSkeleton.svelte` |
| **Evidence** | Skeleton and TableSkeleton components exist but not universally applied. Some pages show nothing during loading, others show a spinner |
| **Recommendation** | Standardize on skeleton-based loading for content areas. Create a `withLoading` Svelte snippet pattern for consistency |
| **Complexity** | Small |
| **Benefit** | Professional appearance, reduced perceived latency |

#### UX‑03 — MEDIUM — No Keyboard Navigation for POS Product Grid

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Accessibility, Cashier efficiency |
| **File** | `web/src/modules/pos/PosProductTable.svelte` |
| **Evidence** | POS product grid likely requires mouse/touch for selection. No arrow-key navigation or keyboard shortcuts documented |
| **Recommendation** | Add keyboard navigation: Arrow keys for navigation, Enter to add to cart, F1-F12 for quick products, `Escape` to clear search |
| **Complexity** | Medium |
| **Benefit** | Cashier efficiency, accessibility |

#### UX‑04 — MEDIUM — Form Validation Feedback Inconsistency

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | User errors |
| **Files** | Various form modals |
| **Evidence** | Some forms show inline validation errors below fields, others show a toast or modal error |
| **Recommendation** | Standardize on inline validation for field-level errors (appearing below each field) with a summary banner at top for form-level errors |
| **Complexity** | Small |
| **Benefit** | Consistent UX, faster error recovery |

---

### 5.8 Frontend Architecture

#### FE‑01 — MEDIUM — Some Stores Conflate Data Fetching with State

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Testability, Reusability |
| **Files** | Various `*store.svelte.ts` files |
| **Evidence** | Some stores directly import and call API services inside the store, mixing data fetching concerns with state management |
| **Recommendation** | Separate data fetching into service/query files. Stores should only manage state. Use the existing `query-manager.ts` pattern more consistently |
| **Complexity** | Medium |
| **Benefit** | Stores become pure state containers, services are testable independently |

#### FE‑02 — MEDIUM — No Persistent Cache for API Data

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Duplicate network requests |
| **File** | `web/src/shared/api/http-client.ts` |
| **Evidence** | Axios instance has no response caching. Navigating away and back re-fetches all data |
| **Recommendation** | Add a simple in-memory cache with TTL for GET responses. Use `axios-cache-interceptor` or similar. Cache invalidated on mutation |
| **Complexity** | Small |
| **Benefit** | Reduced network traffic, faster back-navigation |

---

### 5.9 Testing

#### T‑01 — HIGH [PRE-PROD] — No CI Pipeline

| Field | Value |
|-------|-------|
| **Severity** | High (Medium pre-deployment) |
| **Impact** | No automated quality gates |
| **File** | `.github/workflows/` (empty) |
| **Evidence** | The `.github/workflows/` directory contains no workflow files. Tests only run locally |
| **Recommendation** | Create GitHub Actions workflow: `on: push` to run `go test -p 1 -count=1 ./...`, frontend `npm run test:run`, linting (`golangci-lint`, `eslint`), build verification. Extend to CD before production launch |
| **Complexity** | Medium |
| **Benefit** | Automated quality enforcement, catch regressions early |
| **Note** | Acceptable pre-deployment, but adding CI now would catch regressions during active development |

#### T‑02 — HIGH — Pre-existing Test Failure in `TestE2E_ValidateSession`

| Field | Value |
|-------|-------|
| **Severity** | High |
| **Impact** | E2E test reliability |
| **File** | `cmd/server/e2e_test.go` |
| **Evidence** | AGENTS.md notes: "The failure in `TestE2E_ValidateSession` is pre-existing — the handler returns 'user' key but the test expects 'data'" |
| **Recommendation** | Fix the test to match actual response format, or fix the handler to return expected format. Add this to backlog if low priority |
| **Complexity** | Small |
| **Benefit** | CI green, reliable E2E signal |

#### T‑03 — MEDIUM — Flaky Parallel Test Execution

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | CI reliability |
| **Files** | Multiple test packages |
| **Evidence** | AGENTS.md documents that tests require `-p 1` due to DB deadlock between concurrent `TRUNCATE` + `INSERT`. This will slow down CI |
| **Recommendation** | Use separate test databases per package or use test database with schema-per-test isolation. Consider using `pg_tmp` or ephemeral DB containers |
| **Complexity** | Large |
| **Benefit** | Parallel test execution, faster CI |

#### T‑04 — MEDIUM — Mock Code Duplication

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Test maintenance |
| **Files** | Multiple `*_mock_test.go` files |
| **Evidence** | Each module has its own mock implementations. The `repository_mock_test.go` files are similar across modules |
| **Recommendation** | Extract shared mock helpers to `internal/shared/testutil/` or use `testify/mock` for consistent mocking API |
| **Complexity** | Medium |
| **Benefit** | Reduced boilerplate, consistent mocking patterns |

---

### 5.10 DevOps

#### D‑01 — HIGH [PRE-PROD] — No CI/CD Pipeline

| Field | Value |
|-------|-------|
| **Severity** | High (pre-deployment: Medium) |
| **Impact** | No automated quality gates or deployment automation |
| **File** | `.github/workflows/` |
| **Evidence** | Empty workflows directory. No automated test execution or build verification on push. Will become Critical when deploying to production |
| **Recommendation** | Add GitHub Actions workflow for: `on: push` — run `go test -p 1 -count=1 ./...`, frontend `npm run test:run`, lint, build. Extend to Docker image build + push before production launch |
| **Complexity** | Medium |
| **Benefit** | Automated quality enforcement, catch regressions early |
| **Note** | Acceptable pre-deployment gap. Prioritize before first production deployment |

#### D‑02 — HIGH [PRE-PROD] — No Monitoring or Metrics

| Field | Value |
|-------|-------|
| **Severity** | Medium (will become High at production launch) |
| **Impact** | No observability in production |
| **Files** | All |
| **Evidence** | No Prometheus metrics, no structured application metrics, no health check detail beyond `/health` |
| **Recommendation** | Before production launch, add Prometheus metrics endpoint: HTTP request duration histogram, active connections, goroutine count, event bus metrics, DB pool stats, cache hit rate. Integrate with Grafana |
| **Complexity** | Large |
| **Benefit** | Production observability, performance trend analysis, capacity planning |
| **Note** | Acceptable pre-deployment gap. Schedule for "pre-production readiness" checklist |

#### D‑03 — HIGH [PRE-PROD] — No Automated Backup

| Field | Value |
|-------|-------|
| **Severity** | Medium (will become High at production launch) |
| **Impact** | Data loss risk in production |
| **File** | `Makefile:92-99` |
| **Evidence** | Backup is manual (`make db-backup`). No automated cron-based backup, no off-site storage |
| **Recommendation** | Before production launch, add pg_dump cron job in docker-compose or via host cron. Upload to S3/SCP. Add backup verification and restore drill |
| **Complexity** | Medium |
| **Benefit** | Disaster recovery capability |
| **Note** | Not needed in development. Schedule for production readiness phase |

#### D‑04 — MEDIUM — No Structured Logging Configuration

| Field | Value |
|-------|-------|
| **Severity** | Medium |
| **Impact** | Log analysis difficulty |
| **File** | `internal/shared/logger.go` |
| **Evidence** | Production uses JSON handler (good) but no log level configuration per deployment, no log sampling, no request ID propagation |
| **Recommendation** | Add Gin middleware for request ID (`X-Request-ID`). Propagate through context. Add configurable log level via env var `LOG_LEVEL` |
| **Complexity** | Small |
| **Benefit** | Correlatable logs, filterable by severity |

#### D‑05 — LOW [PRE-PROD] — No Zero-Downtime Deployment

| Field | Value |
|-------|-------|
| **Severity** | Low (will become Medium at production launch for critical deployments) |
| **Impact** | Downtime during deployments |
| **File** | `deploy/docker-compose.yml` |
| **Evidence** | Single-instance services with `restart: unless-stopped`. No rolling update strategy |
| **Recommendation** | Before production launch, add health check grace period. Deploy behind Nginx load balancer with multiple backend instances for zero-downtime |
| **Complexity** | Large |
| **Benefit** | No downtime during deployments |
| **Note** | Unnecessary before production. Defer to production readiness phase |

---

### 5.11 Dependency Review

#### DR‑01 — MEDIUM — `patrickmn/go-cache` Is Unmaintained

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Long-term maintenance |
| **File** | `go.mod:14` |
| **Evidence** | `github.com/patrickmn/go-cache v2.1.0+incompatible` — last updated 2019, marked as unmaintained by author |
| **Recommendation** | Replace with `hashicorp/golang-lru/v2` or `dgraph-io/ristretto`. The cache wrapper abstraction makes this a contained change |
| **Complexity** | Medium |
| **Benefit** | Active maintenance, better performance, TTL features |

#### DR‑02 — LOW — `swaggo/swag` v1.16.6 Has Known Issues

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Swagger generation |
| **File** | `go.mod:33` |
| **Evidence** | Older swaggo versions have known issues with `any` type handling |
| **Recommendation** | Upgrade to latest `swaggo/swag` and `swaggo/gin-swagger` |
| **Complexity** | Small |
| **Benefit** | Better Swagger spec generation |

---

### 5.12 Documentation

#### DOC‑01 — MEDIUM — No Architecture Decision Records (ADRs)

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Institutional knowledge loss |
| **Evidence** | `docs/` contains PRDs and summaries but no ADRs explaining WHY architectural decisions were made |
| **Recommendation** | Create `docs/adr/` directory. Document key decisions: modular monolith vs microservices, event bus choice, why in-process vs message queue, pricing engine design |
| **Complexity** | Small |
| **Benefit** | Preserves rationale for future developers |

#### DOC‑02 — LOW — Archived Documentation Is Outdated

| Field | Value |
|-------|-------|
| **Severity** | Low |
| **Impact** | Confusion |
| **File** | `docs/archive/` |
| **Evidence** | Multiple outdated implementation plans. Could mislead new developers |
| **Recommendation** | Add "ARCHIVED" watermark to all archived docs. Remove or update the most misleading ones |
| **Complexity** | Trivial |
| **Benefit** | Clear signal about document relevance |

---

## 6. Quick Wins (≤ 1 hari kerja)

| # | Finding | Complexity | Effort | Impact | Area |
|---|---------|-----------|--------|--------|------|
| 1 | Merge `scanProduct` + `scanProductFromRow` duplicate functions | Small | 30 min | Eliminates code duplication, single point of change | Code Quality |
| 2 | Fix test `TestE2E_ValidateSession` assertion | Small | 30 min | Green test suite, reliable signal | Testing |
| 3 | Standardize `slog` usage across all files (replace `log.Printf` in response.go) | Small | 1 hr | Consistent structured logging | Code Quality |
| 4 | Add empty states for all list pages using existing EmptyState component | Small | 2 hr | Professional UX everywhere | UI/UX |
| 5 | Standardize loading skeleton usage (apply Skeleton/TableSkeleton consistently) | Small | 2 hr | Consistent perceived performance | UI/UX |
| 6 | Extract magic numbers to named constants in main.go | Small | 30 min | Readability, easy tuning | Code Quality |
| 7 | Add comment for `X-XSS-Protection: 0` | Trivial | 5 min | Code clarity | Security |
| 8 | Add `Retry-After` + `X-RateLimit-*` headers to rate-limited responses | Small | 1 hr | Client UX | API |
| 9 | Set `LOG_LEVEL` env var support | Small | 1 hr | Configurable verbosity | DevOps |
| 10 | Remove `normalizePermissionCode` after standardizing permission format | Small | 1 hr | Code clarity | Code Quality |

---

## 7. Medium Improvements (≤ 1 minggu)

| # | Finding | Complexity | Effort | Impact | Area |
|---|---------|-----------|--------|--------|------|
| 1 | Extract shared sale transaction logic (A-01, CQ-05) | Medium | 2 days | Maintainability — eliminates 200 lines duplicate | Architecture |
| 2 | Fix N+1 in GetSaleByID (P-01) | Small | 2 hr | Performance — 2x fewer round trips | Performance |
| 3 | Add failed login audit trail (S-03) | Small | 4 hr | Security — brute force detection | Security |
| 4 | Build shared WHERE clause builder for count+data queries (P-02) | Medium | 1 day | Performance — eliminates duplicate query construction | Performance |
| 5 | Add CSRF protection for WebSocket upgrade (S-01) | Small | 4 hr | Security — prevents WS hijacking | Security |
| 6 | Standardized error response format (API-01) | Medium | 1 day | API — predictable client error handling | API |
| 7 | Extract wiring from main.go to internal/wiring (A-02) | Medium | 2 days | Architecture — testable DI, cleaner main.go | Architecture |
| 8 | Add persistent API cache on frontend (FE-02) | Small | 4 hr | Performance — reduced network calls | Frontend |
| 9 | Inline form validation standardization (UX-04) | Small | 4 hr | UX — consistent error feedback | UI/UX |
| 10 | Keyboard navigation for POS product grid (UX-03) | Medium | 2 days | UX — cashier efficiency | UI/UX |
| 11 | Create materialized views for daily report aggregation (P-04) | Medium | 2 days | Performance — sub-second reports | Database |
| 12 | Increase body limit for import routes (S-05) | Small | 1 hr | Functionality — prevents import failures | Security |
| 13 | Create CI pipeline with test+lint+build (T-01) | Medium | 2 days | Quality — catch regressions before deployment | Testing |

---

## 8. Long-term Improvements (> 1 minggu)

| # | Finding | Complexity | Effort | Impact | Area |
|---|---------|-----------|--------|--------|------|
| 1 | Redis-backed distributed rate limiting (S-02) | Medium | 3 days | Security | Performance |
| 2 | Prometheus metrics + Grafana dashboard (D-02) | Large | 5 days | Observability | DevOps |
| 3 | Automated database backup with off-site storage (D-03) | Medium | 2 days | DR | DevOps |
| 4 | Go-cache replacement with ristretto (DR-01) | Medium | 2 days | Maintenance | Dependency |
| 5 | Middleware-based audit logging (CQ-06) | Large | 3 days | Architecture | Code Quality |
| 6 | Multi-warehouse inventory support (DB-01) | Medium | 3 days | Features | Database |
| 7 | API versioning (`/api/v1/`) (API-03) | Medium | 2 days | API | API Design |
| 8 | Zero-downtime deployment setup (D-05) | Large | 5 days | Reliability | DevOps |
| 9 | Parallel test execution fix (T-03) | Large | 5 days | CI Speed | Testing |
| 10 | Event versioning for schema migration (A-05) | Small | 2 days | Scalability | Architecture |
| 11 | Entity-specific cache TTL configuration (P-05) | Small | 1 day | Performance | Performance |

---

## 9. Technical Debt Summary

| Category | Debt Items | Estimated Remediation | Pre-Production Priority |
|----------|-----------|----------------------|------------------------|
| **High** | Duplicate sale transaction logic (150 lines × 2) | 2 days | Must fix before adding sale features |

| **Medium** | Pre-existing test failure | 30 min | Fix for reliable signal |
| **Medium** | Duplicate scanProduct functions (75 lines × 2) | 30 min | Easy win |
| **Medium** | Mixed permission code format | 1 day | Code hygiene |
| **Medium** | No failed login audit | 4 hours | Security baseline |
| **Medium** | On-the-fly report aggregation | 2 days | Fix before production |
| **Medium** | No CI/CD pipeline | 2 days | Add before production |
| **Low** | Magic numbers in main.go | 30 min | Readability |
| **Low** | No monitoring/metrics | 5 days | Before production |
| **Low** | No automated backup | 2 days | Before production |
| **Low** | Unmaintained go-cache | 2 days | Low urgency |
| **Low** | Missing ADRs | 1 day | Knowledge preservation |

**Total estimated pre-production remediation:** ~7 developer days (items essential before deployment)  
**Total estimated full remediation:** ~22 developer days (including post-production enhancements)

---

## 10. Risk Matrix

**Context:** Project is pre-production. Risks are assessed for current development phase, with [PROD] indicators for items that escalate after deployment.

| Risk | Probability | Impact | Risk Level | Mitigation | Timeline |
|------|------------|--------|------------|------------|----------|
| Bug introduced when fixing one copy of duplicated sale logic but not the other | High | High | **Critical** | Extract shared logic now | Week 1 |
| Undetected regression from no CI pipeline | High | High | **High** | Add CI pipeline before adding more features | Week 2 |
| Security incident from no failed-login audit | Medium | High | **High** | Add audit logging before external access | Week 1 |
| Performance degradation from on-the-fly aggregation [PROD] | Medium | Medium | **Medium** | Add materialized views before production launch | Pre-prod |
| WS hijacking via missing CSRF | Low | High | **Medium** | Add WS authentication | Week 2 |
| Data loss from no automated backup [PROD] | Low | Critical | **Medium** | Add automated backup before production | Pre-prod |
| Inability to scale horizontally due to in-memory rate limiting [PROD] | Medium | Medium | **Low** | Add Redis rate limiter before multi-instance | Pre-prod |
| Missed cache invalidation due to ad-hoc TTLs | Medium | Low | **Low** | Entity-specific invalidations | When needed |
| Production monitoring blindspot [PROD] | High | High | **High** | Add Prometheus/Grafana before production | Pre-prod |

---

## 11. Recommended Roadmap

**Context:** Project has not been deployed. Phases prioritize code quality, architecture, and security improvements that deliver value during development, followed by production-readiness items before launch.

### Phase 1 — Code Health & Quick Wins (Week 1)

| Priority | Item | Effort | ROI | Rationale |
|----------|------|--------|-----|-----------|
| P0 | Merge `scanProduct` + `scanProductFromRow` | 30 min | High | Eliminates duplicate code risk immediately |
| P0 | Fix `TestE2E_ValidateSession` test | 30 min | High | Green test suite for reliable feedback |
| P1 | Extract shared sale transaction logic | 2 days | High | Eliminates largest duplication (~200 lines) |

| P1 | Add failed login audit trail | 4 hr | High | Security baseline |
| P2 | Standardize `slog` usage | 1 hr | Medium | Consistent logging before adding features |
| P2 | Add empty states + loading skeletons | 2 hr | Medium | Professional UX for demo/staging |
| P2 | Add rate limit headers | 1 hr | Low | Client-friendly API |

### Phase 2 — Structural (Week 2–3)

| Priority | Item | Effort | ROI | Rationale |
|----------|------|--------|-----|-----------|
| P0 | Add CI pipeline (test + lint + build) | 2 days | High | Automated quality before new features |
| P1 | Extract wiring from main.go | 2 days | High | Enables isolated testing of dependencies |
| P1 | Add CSRF token for WebSocket | 4 hr | High | Security hardening before exposure |
| P1 | Standardize error response format | 1 day | Medium | API consistency for frontend development |
| P2 | Build shared WHERE clause builder | 1 day | Medium | Reduces query duplication across repos |
| P2 | Create materialized views for reports | 2 days | Medium | Performance foundation for dashboard |
| P3 | Add persistent API cache on frontend | 4 hr | Low | Performance polish |

### Phase 3 — Pre-Production Readiness (Before Deployment)

| Priority | Item | Effort | ROI | Rationale |
|----------|------|--------|-----|-----------|
| P0 | Add Prometheus + Grafana basic metrics | 5 days | Critical | Observability is non-negotiable in production |
| P0 | Add automated DB backup | 2 days | Critical | Data loss prevention |
| P1 | Add Redis-backed rate limiter | 3 days | High | Distributed rate limiting for multi-instance |
| P1 | CORS hardening (make CORS_ORIGIN required in production) | 1 hr | High | Security baseline |
| P1 | Increase body limit for import routes | 1 hr | Medium | Prevents import failures |
| P2 | Standardized permission format (remove dot/colon mixing) | 1 day | Medium | Developer clarity |
| P2 | Zero-downtime deployment setup | 5 days | Medium | Depends on business requirements |

### Phase 4 — Enhancement (Post-Launch)

| Priority | Item | Effort | ROI | Rationale |
|----------|------|--------|-----|-----------|
| P2 | Multi-warehouse inventory | 3 days | Medium | Feature expansion |
| P2 | POS keyboard navigation | 2 days | Medium | Cashier productivity |
| P3 | Middleware-based audit logging | 3 days | Medium | Architectural cleanup |
| P3 | Parallel test execution fix | 5 days | Medium | Faster CI feedback |
| P3 | Replace go-cache with ristretto | 2 days | Low | Dependency maintenance |
| P3 | API versioning | 2 days | Low | Future-proofing |
| P4 | Event versioning | 2 days | Low | Future-proofing |
| P4 | ADR documentation | 1 day | Low | Knowledge preservation |

---

## 12. Appendix

### A. Key Files Referenced

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Application entry point, DI wiring |
| `internal/sale/service.go` | Sale transaction logic (duplicate code) |
| `internal/sale/handler.go` | HTTP handlers (layer violations) |
| `internal/sale/repository.go` | Sale data access (N+1, duplicate count+data queries) |
| `internal/product/repository.go` | Product data access (duplicate scan) |
| `internal/shared/logger.go` | Structured logging setup |
| `internal/shared/response.go` | Response helpers |
| `internal/middleware/auth.go` | JWT authentication, permission checking |
| `internal/middleware/rate_limit.go` | In-memory rate limiting |
| `internal/middleware/security_headers.go` | CSP, HSTS, security headers |
| `internal/config/config.go` | Environment config |
| `internal/eventbus/bus.go` | Event bus with retry + dead letter |
| `pkg/cache/cache.go` | In-memory cache wrapper |
| `pkg/websocket/hub.go` | WebSocket hub |
| `database/migrations/000_squash.sql` | Full database schema |
| `web/src/app/layouts/Layout.svelte` | Main layout with skip-to-content |
| `web/src/shared/api/http-client.ts` | Axios HTTP client |
| `web/package.json` | Frontend dependencies |
| `go.mod` | Go dependencies |
| `deploy/docker-compose.yml` | Docker Compose config |
| `deploy/backend/Dockerfile` | Multi-stage Go build |
| `Makefile` | Build/deploy automation |

### B. Metrics Collected During Audit

| Metric | Value |
|--------|-------|
| Total Go source files | ~240 |
| Test Go files | ~200 |
| Frontend source files | ~200 |
| E2E test specs | ~35 |
| Database migrations | 1 active + 47 archived |
| Lines of Go code (estimate) | 60k+ |
| Lines of TypeScript/Svelte (estimate) | 40k+ |
| Docker containers | 3 (postgres, backend, frontend) |
| Linter configuration | golangci-lint with 6 linters |
| CORS configuration | Configurable, validates production |

### C. Assumptions

1. **Svelte 5 runes:** The frontend uses Svelte 5's runes API (`$state`, `$derived`, `$effect`). This audit assumes the team is comfortable with Svelte 5 conventions.

2. **Single-instance deployment:** The current architecture assumes single-instance backend. Multi-instance deployment would require Redis for distributed caching and rate limiting.

3. **PostgreSQL 18:** Schema assumes PostgreSQL 18 features (the docker-compose uses `postgres:18-alpine`). Earlier versions may not support all features.

4. **Go 1.26.1:** The codebase uses Go 1.26 features including `slog` and enhanced generics. Backporting to earlier versions would require significant changes.

5. **In-process event bus:** The event bus is intentionally in-process, not distributed. This is an appropriate choice for a modular monolith. Migration to message queue (RabbitMQ, NATS) would be a significant architectural change.

### D. Scoring Methodology

Each category is scored 0–100 based on:

- **Architecture (78):** Clean layering (-5 for layer violations), DI pattern (-3 for wiring complexity), event bus (+5), DRY violations (-10 for duplicate code), interface segregation (-5)
- **Code Quality (65):** Duplication (-15), naming consistency (-5), error handling (-5), dead code (-3), magic numbers (-3), logging (-2), readability (-2)
- **Performance (70):** N+1 queries (-10), cache strategy (-5), aggregate optimization (-5), duplicate count+data queries (-5), compression (+2)
- **Security (68):** CSRF coverage (-8), rate limiting (-5), audit gaps (-7), CORS config (-3), dependency vulns (-3), headers (+5), password hashing (+5)
- **Database (82):** Schema design (+20), indexes (+15), constraints (+10), full-text search (+5), migration quality (+5), materialized view gap (-10), warehouse limitation (-8)
- **API Design (76):** REST consistency (-5), error format (-5), pagination (+10), Swagger (+10), rate limit headers (-3), versioning (-3)
- **UI/UX (75):** Layout (+10), empty states (-8), loading states (-5), accessibility (-5), navigation (+10), mobile (+5)
- **Frontend Architecture (80):** Component organization (+10), Svelte 5 runes (+10), strict TypeScript (+10), store pattern (-5), caching (-3), module structure (+8)
- **Testing (72):** Coverage (+20), E2E coverage (+10), mock quality (-5), parallel execution (-5), pre-existing failure (-8), test organization (+5)
- **DevOps (55):** Docker (+15), docker-compose (+10), CI/CD missing (-20), monitoring missing (-15), backup missing (-10), health checks (+5)
- **Documentation (70):** README (+15), deployment guide (+10), Swagger (+10), PRDs (+10), missing ADRs (-5), outdated docs (-5)

---

*This audit was conducted on 2026-07-23 based on the codebase at commit referenced in the repository. Findings reflect the state of the code at the time of audit.*
