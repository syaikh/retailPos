# Improvement Plan — Retail POS System

**Date:** July 15, 2026
**Baseline Scores:** Security 85/100, Architecture 6.5/10, Performance 6.3/10, UI/UX 7.0/10, Test Coverage 90.6%

---

## Cross-Cutting Conflicts

### 1. Dashboard Caching vs WebSocket Real-Time — PARTIAL CONFLICT
- Dashboard (`Home.svelte`) fetches `/api/dashboard/live` on mount, then WebSocket `sale_created` events update revenue in real-time
- Caching the API endpoint with 30-60s TTL would return stale data on initial page load
- **Resolution:** Cache with **10-15s TTL only** — reduces DB hits (every page load → every 10s) while keeping near-real-time feel. WebSocket updates bypass the cache entirely and update in-memory state directly.

### 2. Optimistic UI Updates vs Server-Side Validation — DIRECT CONFLICT
- Sale creation validates stock via `FOR UPDATE` row locking (`sale/service.go:57`)
- Price is validated server-side (`ErrPriceMismatch`)
- Optimistic updates (showing success before server confirms) would mislead users if the server rejects due to stock or price changes
- **Resolution:** Don't do optimistic updates for write operations. Only apply optimistic UI for reads (e.g., category name edit showing immediately while save happens in background). For POS checkout, keep current pattern: spinner → server confirm → UI update.

### 3. Streaming Exports vs WriteTimeout — COMPLEMENTARY (not conflicting)
- Current 60s `WriteTimeout` kills large exports (`main.go:273`)
- Streaming helps but doesn't fully solve it for very large datasets
- **Resolution:** Do both — increase `WriteTimeout` to 120s AND implement streaming exports with per-row flush.

### 4. Gzip vs Security Headers — NO CONFLICT
- Gzip compresses response body; security headers add response headers
- Just insert gzip middleware before security headers in the chain

### 5. i18n Wiring vs Code Duplication — COMPLEMENTARY
- `labels.ts` already exists with 100+ Indonesian labels but is barely used
- Wiring it up simultaneously fixes i18n inconsistency AND reduces hardcoded string duplication

---

## Phase 1: Quick Wins (1-2 days, high ROI)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 1.1 | Performance | Add gzip middleware (`github.com/gin-contrib/gzip`) | 60-70% smaller JSON responses | 30 min |
| 1.2 | Security | Set `lang="id"` on `<html>` in `app.html:2` | Correct a11y lang | 1 min |
| 1.3 | Security | Enforce consistent 8-char password minimum (`auth_handler.go:189`) | Auth hardening | 5 min |
| 1.4 | Performance | Increase `WriteTimeout` to 120s (`main.go:273`) | Prevent export timeouts | 1 min |
| 1.5 | Architecture | Extract `AuditCreator` interface to `shared/audit.go`, delete 7 duplicate definitions | Eliminates code duplication | 30 min |
| 1.6 | Architecture | Use `shared.GetStoreID(c)` in all 24 handler storeID extractions | Consistent pattern | 1 hr |
| 1.7 | Security | Fix username enumeration — return same error for inactive/invalid (`auth_service.go:65-67`) | Security hardening | 10 min |
| 1.8 | Architecture | Clean up stale binaries (`seeder`, `server`, `dummy`, `*.test`) + add to `.gitignore` | Repo hygiene | 10 min |

---

## Phase 2: Performance Quick Fixes (2-3 days)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 2.1 | Performance | Add `pg_trgm` GIN indexes on `audit_logs(username, role, action)`, `customers(name, phone, email)`, `users(username, email)` | ILIKE searches go from seq scan → index scan | 1 hr |
| 2.2 | Performance | Cache dashboard endpoints with 10-15s TTL (product price cache pattern) | Reduces 3-5 DB queries per page load | 2 hrs |
| 2.3 | Performance | Replace correlated subquery in `GetSalesForExport` (`sale/repository.go:309-312`) with batch LEFT JOIN | O(n) → O(1) subqueries | 1 hr |
| 2.4 | Performance | Merge `/dashboard/stats` + `/dashboard/live` into single CTE endpoint | One DB round-trip instead of two | 2 hrs |
| 2.5 | Performance | Add `pg_trgm` GIN index on `sales` for product name search in sale history | Sale search performance | 30 min |

---

## Phase 3: UI/UX i18n Consistency (2-3 days)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 3.1 | UI/UX | Wire `labels.ts` into all components — replace hardcoded strings | Consistent Indonesian UI | 1-2 days |
| 3.2 | UI/UX | Add `aria-expanded` to mobile cart toggle (`PosPage.svelte:489`) | a11y | 5 min |
| 3.3 | UI/UX | Add `aria-label` to POS product table (`PosProductTable.svelte:53`) | a11y | 5 min |
| 3.4 | UI/UX | Add `aria-label` to cart quantity inputs (`CartPanel.svelte:98-124`) | a11y | 10 min |
| 3.5 | UI/UX | Fix CheckoutModal/CustomerSelectModal to use shared Modal component | Consistency | 2 hrs |

---

## Phase 4: Architecture Refactoring (3-4 days)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 4.1 | Architecture | Extract `report/handler.go` date range logic (260 lines) into `report/ranges.go` | Reduces handler from 701→441 lines | 2 hrs |
| 4.2 | Architecture | Extract `product/repository.go` scan-to-struct logic into shared helper | Eliminates 58-line duplication between `GetProductByID`/`GetProductBySKU` | 1 hr |
| 4.3 | Architecture | Define `EventBus` interface once in `eventbus/`, import in `product/sale/report/inventory` | Eliminates 4 duplicate interface defs | 30 min |
| 4.4 | Architecture | Define repository interfaces in service layer (replace concrete `*Repository` deps) | Better testability, cleaner boundaries | 3-4 hrs |
| 4.5 | Architecture | Extract `pkg/websocket/hub.go` timezone init to use `shared.JakartaLocation()` | Eliminates last timezone duplication | 15 min |
| 4.6 | Architecture | Extract `product/service.go` dead code (`resolveCategoryID`, `resolveBrandID`, `resolveUnitOfMeasureID`, `strPtr`) | Dead code cleanup | 15 min |

---

## Phase 5: Security Hardening (1-2 days)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 5.1 | Security | Validate `CORS_ORIGIN` format — reject `*` in production | Prevents CORS wildcard | 15 min |
| 5.2 | Security | Use separate `JWT_SECRET_REFRESH` env var (or HMAC-based derivation) | Stronger refresh token isolation | 30 min |
| 5.3 | Security | Add `SecurityHeadersMiddleware` for WebSocket upgrade endpoint | Consistent headers | 15 min |
| 5.4 | Security | Remove `.env` from working directory if tracked, ensure `.gitignore` covers it | Secrets hygiene | 10 min |

---

## Phase 6: Performance Deep (3-4 days)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 6.1 | Performance | Stream CSV/XLSX exports with per-row flush instead of buffering | Prevents memory spikes + timeouts | 3-4 hrs |
| 6.2 | Performance | Cache report queries (`GetPeriodComparison`, `GetDualChartData`) with 30s TTL | Reduces expensive CTE execution | 2 hrs |
| 6.3 | Performance | Optimize `GetPeriodComparison` — reduce 6 CTE subqueries to 2 with date range table | Faster period comparisons | 3 hrs |
| 6.4 | Performance | Batch stock deduction in `sale/service.go:82-87` instead of per-item UPDATE | Fewer DB round-trips per sale | 2 hrs |

---

## Phase 7: UI/UX Polish (2-3 days)

| # | Area | Task | Impact | Effort |
|---|------|------|--------|--------|
| 7.1 | UI/UX | Add dark/light mode toggle with `prefers-color-scheme` support | User preference | 1-2 days |
| 7.2 | UI/UX | Add `aria-invalid` + `aria-describedby` on form validation errors | Better screen reader UX | 2 hrs |
| 7.3 | UI/UX | Add pause-on-hover to toast auto-dismiss | UX polish | 30 min |
| 7.4 | UI/UX | Add toast stack limit (max 5 visible) | Prevents screen flooding | 30 min |
| 7.5 | UI/UX | Responsive POS table columns (replace `table-fixed` with responsive widths) | Better tablet/mobile UX | 2 hrs |

---

## Summary

| Phase | Focus | Timeline | Est. Impact |
|-------|-------|----------|-------------|
| 1 | Quick wins | 1-2 days | Security 85→88, Architecture 6.5→6.8 |
| 2 | Performance quick | 2-3 days | Performance 6.3→7.0 |
| 3 | i18n + a11y | 2-3 days | UI/UX 7.0→7.5 |
| 4 | Architecture refactoring | 3-4 days | Architecture 6.5→7.5 |
| 5 | Security hardening | 1-2 days | Security 85→90 |
| 6 | Performance deep | 3-4 days | Performance 7.0→7.5 |
| 7 | UI/UX polish | 2-3 days | UI/UX 7.5→8.0 |

**Total estimated:** 15-21 days

### Projected Scores After All Phases

| Domain | Current | Projected |
|--------|---------|-----------|
| Security | 85/100 | 90-92/100 |
| Architecture | 6.5/10 | 7.5-8.0/10 |
| Performance | 6.3/10 | 7.5/10 |
| UI/UX | 7.0/10 | 8.0/10 |
| Test Coverage | 90.6% | 92%+ |
