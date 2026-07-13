# Performance Audit — Retail POS System

**Date:** 2026-07-12
**Auditor:** opencode (big-pickle)
**Previous Score:** 4.5/10 (2026-07-10)
**Current Score:** 6.0/10

---

## Executive Summary

Significant improvements since the previous audit. N+1 queries in sale listing, frontend code splitting, aggregation indexes, product full-text search, and dead-letter event handling are all resolved. However, two critical systemic issues remain: zero caching across the entire backend, and a dashboard period comparison query that scans the sales table 6 times per request.

---

## Findings Summary

| Severity | Count |
|---|---|
| Critical | 2 |
| High | 6 |
| Medium | 12 |
| Low | 6 |

---

## Top 5 Bottlenecks by Impact

1. **P-01: Zero Caching** — Every endpoint hits DB for semi-static data. Adding in-memory TTL cache for payment methods, categories, brands, and dashboard stats would reduce DB load by 60-70%.
2. **P-02: Period Comparison 6x Table Scan** — The dashboard's most expensive query scans the sales table 6 times. Collapsing into 2 CTEs with conditional aggregation would cut query time from ~500ms to ~50ms.
3. **P-05: Export Memory Loading** — Loading 5000+ products or 100K+ sales into memory for export is an OOM risk. Streaming export would fix this.
4. **P-03: O(n^2) Sale Item Matching** — Using a map instead of linear scan would reduce list endpoint time from O(n^2) to O(n).
5. **P-04: User List N+1 Roles** — Fixing this reduces the user list from N+1 queries to 2 queries.

---

## All Findings

### CRITICAL

| ID | Title | Location | Impact |
|----|-------|----------|--------|
| P-01 | Zero Caching Layer | Entire backend | Every request hits DB |
| P-02 | Period Comparison Scans Sales Table 6x | `report/repository.go:44-110` | Dashboard loads 500ms+ |

### HIGH

| ID | Title | Location | Impact |
|----|-------|----------|--------|
| P-03 | O(n^2) Sale Item Matching in List | `sale/repository.go:313-317` | O(sales x items) per page |
| P-04 | N+1 Role Lookup in User List | `user/repository.go:218-223` | N+1 per user row |
| P-05 | Export Loads Entire Table into Memory | `product/query.go:229`, `sale/repository.go:329` | OOM risk |
| P-06 | Sale Items N+1 per Price Validation | `sale/service.go:61-62` | N+1 queries per item |
| P-07 | Category Name Resolution N+1 | `product/service.go:51-63` | N queries per category |
| P-08 | No DB Connection Pool Tuning | `cmd/server/main.go:111` | Default pool may bottleneck |

### MEDIUM

| ID | Title | Location |
|----|-------|----------|
| P-09 | Correlated Subquery in Sales Export | `sale/repository.go:332` |
| P-10 | ILIKE on Sales Search Fields | `sale/repository.go:144,198,340` |
| P-11 | ILIKE on Customer Search | `customer/repository.go:120,142` |
| P-12 | v_products_full LATERAL Join Per Row | `database/migrations/025:50-55` |
| P-13 | Stock Deduction Loop Not Batched | `sale/service.go:42-72` |
| P-14 | Import Reads Entire File into Memory | `import/engine.go:76` |
| P-15 | Import Goroutine Leak on Shutdown | `import/engine.go:362` |
| P-16 | Event Bus 1000-Buffer Silent Drop | `eventbus/bus.go:73,109-112` |
| P-17 | Rate Limiter Read-Modify-Write Pattern | `middleware/rate_limit.go:74-99` |
| P-18 | BulkUpsertProduct Not Truly Batched | `product/bulk.go:24-113` |
| P-19 | Preview State In-Memory Growth | `import/engine.go:51-53` |
| P-20 | StockAdjustedListener Full View Query | `pkg/websocket/listener.go:82` |

### LOW

| ID | Title | Location |
|----|-------|----------|
| P-21 | Sale Items Inconsistent Column Selection | `sale/repository.go:300 vs 110` |
| P-22 | Payment Methods Not Cached | `sale/repository.go:415-438` |
| P-23 | No Prepared Statement Reuse | Global |
| P-24 | GetSaleByID Two Queries Instead of JOIN | `sale/repository.go:77,109` |
| P-25 | No WebSocket Event Deduplication | `eventbus/bus.go` |
| P-26 | No Bundle Analyzer Configured | `web/vite.config.js` |

---

## Resolved from Previous Audit

| Previous Issue | Status | Resolution |
|---------------|--------|------------|
| N+1 queries in sale listing | **RESOLVED** | Batch loading with chunking |
| Frontend no code splitting | **RESOLVED** | Lazy import() for all routes |
| No vendor chunk splitting | **RESOLVED** | Manual chunks in vite.config.js |
| No product full-text search | **RESOLVED** | GIN index + tsvector |
| Missing aggregation indexes | **RESOLVED** | Covering index on sales |
| No event retry mechanism | **RESOLVED** | Exponential backoff with dead-letter store |
| WebSocket no rate limiting | **RESOLVED** | IP rate limiter + per-user connection cap |
| No graceful shutdown | **RESOLVED** | Hub + EventBus shutdown with drain |
| Unique barcode constraint too strict | **RESOLVED** | Partial unique index on active products only |
| No dead-letter queue for events | **RESOLVED** | dead_letter_events table + PgDeadLetterStore |
