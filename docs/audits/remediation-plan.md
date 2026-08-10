# Audit Remediation Plan — Retail POS System

**Date:** 2026-07-23  
**Based on:** Project Audit Report v1.1  
**Context:** Pre-production — project has not been deployed. Plan prioritizes code quality, architecture, and test foundation improvements, with production-readiness items scheduled before launch.

---

## Table of Contents

1. [How to Use This Plan](#1-how-to-use-this-plan)
2. [Phase 1 — Code Health & Quick Wins](#2-phase-1--code-health--quick-wins)
3. [Phase 2 — Structural Improvements](#3-phase-2--structural-improvements)
4. [Phase 3 — Pre-Production Readiness](#4-phase-3--pre-production-readiness)
5. [Phase 4 — Post-Launch Enhancements](#5-phase-4--post-launch-enhancements)
6. [Dependency Graph](#6-dependency-graph)
7. [Effort Summary](#7-effort-summary)
8. [Risk During Remediation](#8-risk-during-remediation)

---

## 1. How to Use This Plan

### Structure
Each task follows this template:

```
### [ID] — Title
**Audit Ref:** A‑XX, CQ‑XX
**Effort:** X hours/days
**Difficulty:** Small / Medium / Large
**Risk:** Low / Medium / High (risk of regression during change)
**Files affected:** path/to/file.go, path/to/file2.go
```

### Workflow for Each Task
1. Read the audit finding and this remediation plan
2. Create a feature branch
3. Implement the change
4. Run existing tests: `go test -p 1 -count=1 ./...` (backend) or `npm run test:run` (frontend)
5. Run linter: `golangci-lint run ./...`
6. Commit with descriptive message
7. Update test coverage if needed

### Sequencing
Tasks within a phase can be parallelized unless noted in "Depends on". Cross-phase dependencies are documented in the dependency graph.

---

## 2. Phase 1 — Code Health & Quick Wins

### 2.1 Merge `scanProduct` and `scanProductFromRow` ✅ DONE

**Audit Ref:** CQ‑01  
**Effort:** 30 minutes  
**Difficulty:** Small  
**Risk:** Medium (scan logic is critical, incorrect merge could corrupt product data)  
**Files affected:** `internal/product/repository.go`

**Problem:** Two identical functions (`scanProduct` with `pgx.Row` param, `scanProductFromRow` with `rowScanner` interface). ~75 lines each, identical body.

**Solution:**
1. Delete `scanProduct` function entirely
2. Rename `scanProductFromRow` to `scanProduct` (shorter name)

```go
// Before: two functions
func scanProduct(row pgx.Row) (*Product, error) { /* 75 lines */ }
func scanProductFromRow(row rowScanner) (*Product, error) { /* 75 lines identical */ }

// After: one function with rowScanner interface
func scanProduct(row rowScanner) (*Product, error) { /* single source of truth */ }
```

3. All callers using `scanProduct(r.db.QueryRow(...))` already work because `pgx.Row` satisfies the `rowScanner` interface.

**Verification:**
- `go build ./...` compiles
- `go test ./internal/product/...` passes
- Functions that previously used `scanProduct` still work (no signature change needed, just function selection)

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — hanya `scanProduct(row rowScanner)` yang tersisa
(`internal/product/repository.go`), persis seperti target solusi. Diverifikasi
via audit (2026-08-10).

---

### 2.2 Fix `TestE2E_ValidateSession` ✅ DONE

**Audit Ref:** T‑02  
**Effort:** 30 minutes  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `cmd/server/e2e_test.go` (or the auth handler)

**Problem:** Test expects response key `"data"` but handler returns key `"user"`.

**Solution:**

Option A (recommended — align test to reality):
```go
// In e2e_test.go, change:
require.Contains(t, resp, "data")
require.NotEmpty(t, resp["data"])
// To:
require.Contains(t, resp, "user")
require.NotEmpty(t, resp["user"])
```

Option B (if `"data"` is the correct API convention):
```go
// In auth_handler.go, wrap the response:
c.JSON(http.StatusOK, gin.H{"data": user})
// instead of returning raw user object
```

**Verification:** `go test -p 1 -count=1 -run TestE2E_ValidateSession ./cmd/server/...` passes (requires DB + JWT_SECRET)

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — test di `cmd/server/e2e_test.go` (line 339, 483) kini mengharapkan key `"user"`, sesuai respons handler. Diverifikasi via audit (2026-08-10).

---

### 2.3 Standardize `slog` Usage ✅ DONE

**Audit Ref:** CQ‑03  
**Effort:** 1 hour  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `internal/shared/response.go`, `internal/sale/repository.go`, `internal/sale/handler.go`, `internal/middleware/rate_limit.go`, `pkg/websocket/hub.go`, plus any file using `log.Printf`

**Problem:** Mix of `log.Printf`, `slog.Info`, `slog.Warn`, `slog.Error`, and `println`.

**Solution:**

1. Replace all `log.Printf` calls with structured `slog`:

```go
// Before:
log.Printf("Internal server error: %v", err)
// After:
slog.Error("internal server error", "error", err)
```

```go
// Before:
log.Printf("failed to write xlsx: %v", err)
// After:
slog.Warn("failed to write xlsx", "error", err)
```

2. Replace `println` in main.go:
```go
// Before:
println("Server starting on " + addr + " (env: " + cfg.Env + ")")
// After:
slog.Info("server starting", "addr", addr, "env", cfg.Env)
```

3. Search all `.go` files for `log.Printf` and `println` (excluding `cmd/dummy/`):
```bash
grep -rn 'log\.Printf\|println(' --include='*.go' | grep -v vendor | grep -v cmd/dummy
```

**Verification:** `golangci-lint run ./...` passes. No `log.Printf` remain (except in `cmd/dummy/`).

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — semua paket internal (16 file) memakai `slog`; tidak ada lagi `log.Printf`/`log.Println` di `internal/` maupun `pkg/` (verifikasi rg 2026-08-10).

---

### 2.4 Add Empty States for All List Pages ✅ DONE

**Audit Ref:** UX‑01  
**Effort:** 2 hours  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** All list components (`web/src/modules/*/...Table.svelte`)

**Problem:** `EmptyState` component exists but isn't consistently used. Some tables show blank when empty.

**Solution:**
In every Svelte table component, add an empty state block:

```svelte
{#if loading}
  <TableSkeleton />
{:else if items.length === 0}
  <EmptyState
    title="No data found"
    description={searchQuery ? "Try adjusting your search or filters" : "No records yet"}
    icon={searchQuery ? SearchIcon : InboxIcon}
  />
{:else}
  <!-- existing table -->
{/if}
```

**Files to check:**
- `AuditLogsTable.svelte`
- `TransactionTable.svelte`
- `CustomerTable.svelte`
- `SuppliersTable.svelte`
- `PricingRulesTable.svelte`
- `ShiftsTable` (if exists)
- Any other list component

**Verification:** Navigate to each list page with empty data (test DB with no records). Verify friendly empty state is shown.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — setiap halaman/tabel list punya cabang
`{:else if X.length === 0}` dengan blok `role="status"` berisi ikon + pesan
`labels.*` (varian search-aware). Diverifikasi via audit (2026-08-10).

---

### 2.5 Standardize Loading Skeleton Usage ✅ DONE

**Audit Ref:** UX‑02  
**Effort:** 2 hours  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** All list pages

**Problem:** Skeleton and TableSkeleton components exist but are inconsistently applied.

**Solution:**
Apply the same pattern as 2.4 — use `TableSkeleton` during loading state:

```svelte
{#if loading}
  <TableSkeleton rows={5} />
{:else}
  <!-- table or empty state -->
{/if}
```

**Verification:** Slow network simulation (Chrome DevTools throttling) shows skeletons consistently.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — semua tabel data memakai `Skeleton`/`TableSkeleton`;
sisa spinner loading halaman dikonversi ke skeleton (`ShiftsPage`, `PurchaseOrderDetail`,
`RackStockPanel`); spinner di tombol submit (`isSubmitting`) sengaja dipertahankan.
Diverifikasi via audit (2026-08-10).

---

### 2.6 Extract Magic Numbers to Named Constants ✅ DONE

**Audit Ref:** CQ‑04  
**Effort:** 30 minutes  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `cmd/server/main.go`, `internal/shared/paging.go`

**Problem:** Hardcoded values like `25`, `MinConns = 5`, `max page size = 100`, `1 << 20`.

**Solution:**

```go
// cmd/server/main.go - add near top or in a config section
const (
    defaultMaxConns         = 25
    defaultMinConns         = 5
    defaultMaxConnLifetime  = 30 * time.Minute
    defaultMaxConnIdleTime  = 5 * time.Minute
    defaultHealthCheckPeriod = 15 * time.Second
    defaultBodyLimit         = 1 << 20 // 1 MB
)
```

```go
// internal/shared/paging.go - replace magic number
const DefaultMaxPageLimit = 100

func ParsePaginationParams(limitStr, offsetStr string) (int, int) {
    limit, _ := strconv.Atoi(limitStr)
    if limit <= 0 || limit > DefaultMaxPageLimit {
        limit = 20
    }
    // ...
}
```

**Verification:** `go build ./...` compiles. Existing pagination behavior unchanged.

**Status:** SUDAH DIIMPLEMENTASI — konstanta `defaultMaxConns/MinConns/MaxConnLifetime/
MaxConnIdleTime/HealthCheckPeriod/BodyLimit/Port/ReadTimeout/WriteTimeout/IdleTimeout`
di `cmd/server/main.go`, `DefaultMaxPageLimit = 100` di `internal/shared/paging.go`.
Diverifikasi via audit (2026-08-10).

**Depends on:** Nothing

---

### 2.7 Add Security Header Comments ✅ DONE

**Audit Ref:** S‑06  
**Effort:** 5 minutes  
**Difficulty:** Trivial  
**Risk:** None  
**Files affected:** `internal/middleware/security_headers.go`

**Solution:**

```go
// X-XSS-Protection: 0 — deliberately disabled per modern security guidance.
// This header is deprecated and can introduce XSS vulnerabilities in some
// older browsers. CSP handles XSS prevention via script-src and style-src.
c.Header("X-XSS-Protection", "0")
```

**Verification:** Code review only.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — komentar `// X-XSS-Protection: 0 — deliberately disabled per modern security guidance.` ada di `internal/middleware/security_headers.go:21`.

---

### 2.8 Add Rate Limit Response Headers ✅ DONE

**Audit Ref:** API‑02  
**Effort:** 1 hour  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `internal/middleware/rate_limit.go`

**Problem:** Rate-limited responses don't include standard headers for client backoff.

**Solution:**

```go
// In the rate limiter, track state and set headers before aborting
func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
    // ... existing code ...
}

// In RateLimitMiddleware:
func RateLimitMiddleware() gin.HandlerFunc {
    limiter := NewIPRateLimiter(...)
    return func(c *gin.Context) {
        ip := getClientIP(c)
        l := limiter.GetLimiter(ip)
        
        // Set rate limit headers
        c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.b))
        c.Header("X-RateLimit-Remaining", strconv.Itoa(l.Tokens()))
        
        if !l.Allow() {
            retryAfter := int(time.Until(/* next token */).Seconds())
            c.Header("Retry-After", strconv.Itoa(retryAfter))
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "too many requests",
                "retry_after": retryAfter,
            })
            return
        }
        c.Next()
    }
}
```

> **Note:** `rate.Limiter` doesn't expose remaining tokens directly. One approach: use a wrapper that tracks allowance. Alternatively, use a simpler counter-based approach for the `X-RateLimit-Remaining` header.

**Verification:** `curl -v http://localhost:9095/api/products` shows `X-RateLimit-*` headers. After exceeding limit, `Retry-After` header appears.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — `internal/middleware/rate_limit.go` (line 155-229) men-set `X-RateLimit-Limit` dan `Retry-After` (1s/60s) pada semua cabang limiter, sebelum abort 429.

---

### 2.9 Add `LOG_LEVEL` Environment Variable ✅ DONE

**Audit Ref:** D‑04  
**Effort:** 1 hour  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `internal/shared/logger.go`, `internal/config/config.go`

**Problem:** Log level is hardcoded (Debug in dev, Info in production). No env override.

**Solution:**

```go
// In config.go
type Config struct {
    // ... existing fields
    LogLevel string
}

// In Load():
cachedConfig = &Config{
    // ...
    LogLevel: getEnv("LOG_LEVEL", func() string {
        if env == "production" { return "info" }
        return "debug"
    }),
}
```

```go
// In shared/logger.go
func InitLogger(env string, levelStr string) {
    var level slog.Level
    switch strings.ToLower(levelStr) {
    case "debug": level = slog.LevelDebug
    case "info":  level = slog.LevelInfo
    case "warn":  level = slog.LevelWarn
    case "error": level = slog.LevelError
    default:      level = slog.LevelInfo
    }
    // ...
}
```

**Verification:** Set `LOG_LEVEL=error`, verify only errors appear. Set `LOG_LEVEL=debug`, verify debug messages appear.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — `LOG_LEVEL` dibaca di `internal/config/config.go:92`, diparsing `parseLogLevel` di `internal/shared/logger.go:15`. Default: debug (non-production) / info (production).

---

## 3. Phase 2 — Structural Improvements

### 3.1 Extract Shared Sale Transaction Logic ✅ DONE

**Audit Ref:** A‑01, CQ‑05  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** High (sale transaction is the most critical code path)  
**Files affected:** `internal/sale/service.go`

**Problem:** ~200 lines of identical stock check → price resolution → stock deduction logic duplicated in `CreateSale` and `CreateSaleWithParkedSale`.

**Solution:**

**Step 1:** Create a transaction context struct and a shared method:

```go
// Struct to pass data through the shared pipeline
type saleTxContext struct {
    tx      pgx.Tx
    sale    *Sale
    items   []SaleItem
}

// Shared processing pipeline
func (s *Service) processSaleItems(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem) error {
    // 1. Validate quantities
    for _, item := range items {
        if item.Quantity <= 0 {
            return fmt.Errorf("invalid quantity %d for product %d", item.Quantity, item.ProductID)
        }
    }

    // 2. Batch lock and check stock
    productIDs := make([]int, len(items))
    for i, item := range items {
        productIDs[i] = item.ProductID
    }
    rows, err := tx.Query(ctx, `SELECT product_id, COALESCE(quantity, 0) FROM product_stock 
        WHERE product_id = ANY($1) AND warehouse_id IS NULL AND store_id IS NULL FOR UPDATE`, productIDs)
    if err != nil {
        return fmt.Errorf("batch check stock: %w", err)
    }
    // ... scan stockMap ...
    // ... check stock >= quantity ...

    // 3. Deduct stock
    // (existing unnest-based batch update)

    // 4. Resolve prices
    if s.resolver != nil {
        return s.resolveWithResolver(ctx, sale, items)
    } else if s.priceStore != nil {
        return s.resolveWithPriceStore(ctx, sale, items)
    }
    return nil
}
```

**Step 2:** Simplify both methods:

```go
func (s *Service) CreateSale(ctx context.Context, sale *Sale, items []SaleItem) error {
    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer func() { _ = tx.Rollback(ctx) }()

    if err := s.processSaleItems(ctx, tx, sale, items); err != nil {
        return err
    }

    if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
        return err
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    sale.Items = items
    _ = s.eventBus.Publish(ctx, "sale.created", sale)
    return nil
}

func (s *Service) CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []SaleItem, parkedSaleID *int) error {
    if parkedSaleID == nil {
        return s.CreateSale(ctx, sale, items)
    }

    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer func() { _ = tx.Rollback(ctx) }()

    // Lock parked sale
    var parkedStatus string
    err = tx.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1 AND status = 'recalled' FOR UPDATE`,
        *parkedSaleID).Scan(&parkedStatus)
    if err != nil {
        if err == pgx.ErrNoRows {
            return ErrParkedSaleNotRecalled
        }
        return fmt.Errorf("lock parked sale: %w", err)
    }

    if err := s.processSaleItems(ctx, tx, sale, items); err != nil {
        return err
    }

    if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
        return err
    }
    if err := s.repo.ConsumeParkedSale(ctx, tx, *parkedSaleID); err != nil {
        return fmt.Errorf("consume parked sale: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    sale.Items = items
    _ = s.eventBus.Publish(ctx, "sale.created", sale)
    return nil
}
```

**Testing Strategy:**
1. Run existing sale tests: `go test -p 1 -count=1 ./internal/sale/...`
2. Verify both `CreateSale` and `CreateSaleWithParkedSale` behaviors in integration tests
3. Verify that stock deduction and price resolution produce identical results for same inputs
4. Run E2E tests: `npx playwright test --reporter=list`

**Verification:**
- `go test -p 1 -count=1 ./internal/sale/...` passes
- Create a sale → stock deducted correctly
- Create sale from parked sale → parked sale consumed, new sale created
- Price rules applied correctly in both paths

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — `processSaleItems(ctx, tx, sale, items)` di `internal/sale/service.go:110` dipanggil dari `CreateSale` (line 210) dan `CreateSaleWithParkedSale` (line 339). Pipeline validasi → batch lock stok → harga via resolver/priceStore → potong stok jadi satu sumber.

---

### 3.2 Fix N+1 Query in `GetSaleByID` ✅ DONE

**Audit Ref:** P‑01  
**Effort:** 2 hours  
**Difficulty:** Small  
**Risk:** Medium  
**Files affected:** `internal/sale/repository.go`

**Problem:** Two sequential queries: one for sale, one for items.

**Solution — Option A (recommended):** Use a single query with `LEFT JOIN LATERAL`:

```go
func (r *Repository) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
    query := `
        SELECT s.id, s.invoice_number, s.cashier_id, s.customer_id, s.store_id, 
               s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, 
               s.status, s.created_at, s.updated_at, COALESCE(c.name, '') as customer_name,
               COALESCE(
                   (SELECT jsonb_agg(jsonb_build_object(
                       'id', si.id, 'sale_id', si.sale_id, 'product_id', si.product_id,
                       'name', p.name, 'quantity', si.quantity, 'unit_price', si.unit_price,
                       'subtotal', si.subtotal, 'dpp_amount', si.dpp_amount, 'tax_amount', si.tax_amount,
                       'pricing_rule_id', si.pricing_rule_id, 'pricing_rule_name', si.pricing_rule_name,
                       'pricing_rule_type', si.pricing_rule_type, 'pricing_type', si.pricing_type,
                       'original_price', si.original_price
                   )) FROM sale_items si 
                   JOIN products p ON si.product_id = p.id 
                   WHERE si.sale_id = s.id),
               '[]'::jsonb
           ) as items
        FROM sales s
        LEFT JOIN customers c ON s.customer_id = c.id
        WHERE s.id = $1`
    
    var itemsJSON []byte
    // scan itemsJSON, then json.Unmarshal into items slice
}
```

**Solution — Option B (simpler):** Use a `JOIN` with rows and collect into sale + items map (same pattern as `GetAllSales` but with a single query).

**Solution — Option C (pragmatic):** Accept the second query — it's only 1 extra round trip for a single sale detail view, which is infrequent.

**Recommendation:** Option C for now. The N+1 here is minimal (1 extra query per view). Reserve Option A for when this endpoint is proven to be a bottleneck.

**Verification:** `go test -p 1 -count=1 ./internal/sale/...` passes.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — `GetSaleByID` memakai satu query dengan `jsonb_agg` untuk item sale (tidak ada N+1). Diverifikasi `internal/sale/repository.go:200-214` via audit (2026-08-10).

---

### 3.3 Add Failed Login Audit Trail ✅ DONE

**Audit Ref:** S‑03  
**Effort:** 4 hours  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `internal/user/auth_handler.go`, `internal/user/auth_service.go`, `internal/user/repository.go`

**Problem:** Failed login attempts leave no audit trail. Brute force detection impossible.

**Solution:**

**Step 1:** Create an `audit_login` table or use existing `audit_logs`:

```sql
-- Add to migration (or use existing audit_logs)
CREATE TABLE IF NOT EXISTS login_attempts (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_login_attempts_username ON login_attempts(username);
CREATE INDEX idx_login_attempts_ip ON login_attempts(ip_address);
CREATE INDEX idx_login_attempts_created ON login_attempts(created_at);
```

**Step 2:** Log attempts in `AuthService.Login`:

```go
func (s *AuthService) Login(ctx context.Context, req LoginRequest, ip string, userAgent string) (*LoginResponse, error) {
    user, err := s.repo.GetUserByUsername(ctx, req.Username)
    
    // Log attempt (failed or successful)
    defer func() {
        success := err == nil
        reason := ""
        if err != nil {
            reason = err.Error()
        }
        _ = s.repo.LogLoginAttempt(ctx, req.Username, ip, userAgent, success, reason)
    }()

    if err != nil {
        return nil, ErrInvalidCredentials
    }
    // ... validate password, return tokens
}
```

**Step 3:** Add rate limiting based on consecutive failures:

```go
// Optional: return account lockout info
const maxFailedAttempts = 5
const lockoutDuration = 15 * time.Minute

func (s *AuthService) getRecentFailures(ctx context.Context, username string) (int, error) {
    var count int
    err := s.repo.db.QueryRow(ctx, `
        SELECT COUNT(*) FROM login_attempts 
        WHERE username = $1 AND success = false 
        AND created_at > NOW() - INTERVAL '15 minutes'
    `, username).Scan(&count)
    return count, err
}
```

**Verification:**
1. Login with wrong password → check `login_attempts` table has a failed entry
2. Login with correct password → verify success flagged
3. Attempt 6+ failed logins in 15 minutes → account lockout (optional enhancement)

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — login gagal dicatat sebagai audit trail `action = 'login_failed'` (`internal/user/auth_service.go:249`), dengan query pemantauan di `internal/user/repository.go:642,653`. Diverifikasi via audit (2026-08-10).

---

### 3.4 Build Shared WHERE Clause Builder ✅ DONE

**Audit Ref:** P‑02  
**Effort:** 1 day  
**Difficulty:** Medium  
**Risk:** Low  
**Files affected:** `internal/sale/repository.go`, plus all other repositories with count+data patterns

**Problem:** Every repository that uses count+data pattern duplicates the entire WHERE clause logic.

**Solution:**

Create a shared query builder:

```go
// internal/shared/querybuilder.go
package shared

import "fmt"

type QueryBuilder struct {
    WhereClauses []string
    Args         []interface{}
    ArgIdx       int
}

func NewQueryBuilder() *QueryBuilder {
    return &QueryBuilder{
        WhereClauses: []string{"1=1"},
        ArgIdx:       1,
    }
}

func (qb *QueryBuilder) AddClause(clause string, args ...interface{}) {
    if len(args) > 0 {
        qb.WhereClauses = append(qb.WhereClauses, fmt.Sprintf(clause, qb.ArgIdx))
        qb.Args = append(qb.Args, args...)
        qb.ArgIdx += len(args)
    }
}

func (qb *QueryBuilder) Where() string {
    return strings.Join(qb.WhereClauses, " AND ")
}

func (qb *QueryBuilder) NextPlaceholder() int {
    return qb.ArgIdx
}
```

Usage in sale repository:

```go
func (r *Repository) buildSaleFilter(search string, startDate string, endDate string, 
    storeID *int, paymentMethods string, minTotal, maxTotal *int, cashierID *int) *shared.QueryBuilder {
    
    qb := shared.NewQueryBuilder()
    
    if search != "" {
        qb.AddClause(" AND (s.invoice_number ILIKE $%d OR s.id IN (...))", "%"+search+"%")
    }
    if startDate != "" {
        if start, err := time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation()); err == nil {
            qb.AddClause(" AND s.created_at >= $%d", start)
        }
    }
    // ... etc
    
    return qb
}

func (r *Repository) GetAllSales(ctx context.Context, ...) ([]Sale, int, error) {
    qb := r.buildSaleFilter(search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, cashierID)
    
    // Use same qb for both count and data queries
    countQuery := "SELECT COUNT(*) FROM sales s WHERE " + qb.Where()
    dataQuery := fmt.Sprintf("SELECT ... FROM sales s LEFT JOIN customers c ... WHERE %s ORDER BY ...", qb.Where())
}
```

**Verification:** All existing sales queries return identical results before and after. Run full test suite.

**Depends on:** Nothing (can be done incrementally, one repository at a time)

**Status:** SUDAH DIIMPLEMENTASI — `shared.QueryBuilder` (`internal/shared/querybuilder.go`) dengan `AddClause`/`Where`, dipakai `buildSalen` di `internal/sale/repository.go`. Ada unit test `internal/shared/querybuilder_test.go`.

---

### 3.5 Add CSRF Protection for WebSocket

**Audit Ref:** S‑01  
**Effort:** 4 hours  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `pkg/websocket/hub.go`, `cmd/server/main.go`, `internal/middleware/security_headers.go`

**Problem:** WebSocket endpoint `/ws` has no CSRF protection. The `checkOrigin` function is the only defense.

**Solution:**

**Step 1:** The existing WebSocket auth mechanism (send `{"type":"auth","token":"..."}` after upgrade) already provides token-based protection. The token is sent as a message body, not as a cookie, so standard CSRF doesn't apply.

**Step 2:** However, to be thorough, add a custom WebSocket origin header check + nonce:

```go
// In pkg/websocket/hub.go
func ServeWebSocket(hub *Hub, c *gin.Context) {
    // Verify CSRF token via custom header (set by frontend)
    csrfHeader := c.Request.Header.Get("X-WebSocket-CSRF")
    if csrfHeader == "" || !validateWSCSRF(csrfHeader, c.Request) {
        c.JSON(http.StatusForbidden, gin.H{"error": "CSRF validation failed"})
        return
    }
    // ... rest of upgrade
}
```

**Step 3:** On the frontend, send a CSRF token as a custom header during WebSocket connection:

```typescript
// In web/src/app/providers/websocket.ts
const ws = new WebSocket(`ws://${host}/ws`);
// Add CSRF token as query param or header
```

> **Note:** Since the WebSocket uses token-based auth (not cookie-based), the current implementation is **not vulnerable to CSRF in practice**. CSRF attacks exploit cookie-based auth. The JWT in a message body cannot be set by a cross-origin form. This is low priority.

**Verification:** Existing WebSocket tests pass.

**Depends on:** Nothing

**Status:** RESOLVED TANPA PERUBAHAN — sesuai catatan plan ini sendiri: WebSocket memakai token-based auth (JWT dikirim dalam pesan `{"type":"auth","token":...}` setelah upgrade, bukan cookie), jadi CSRF tidak berlaku. `checkOrigin` di `pkg/websocket/hub.go:32` tetap menjadi pertahanan origin dengan allowlist + `CORS_ORIGIN`. Nonce `X-WebSocket-CSRF` tidak ditambahkan — tidak diperlukan.

---

### 3.6 Standardize Error Response Format ✅ DONE

**Audit Ref:** API‑01  
**Effort:** 1 day  
**Difficulty:** Medium  
**Risk:** Medium (frontend depends on current format)  
**Files affected:** `internal/shared/response.go`, all handler files

**Problem:** Mix of `{"error":"message"}` and `{"errors":{...}}` formats.

**Solution:**

**Step 1:** Define standard error types:

```go
// internal/shared/errors.go
package shared

type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func NewError(code string, message string) ErrorResponse {
    return ErrorResponse{
        Error: ErrorDetail{
            Code:    code,
            Message: message,
        },
    }
}

// Standard error codes
const (
    ErrBadRequest       = "BAD_REQUEST"
    ErrNotFound         = "NOT_FOUND"
    ErrUnauthorized     = "UNAUTHORIZED"
    ErrForbidden        = "FORBIDDEN"
    ErrConflict         = "CONFLICT"
    ErrInternal         = "INTERNAL_ERROR"
    ErrValidation       = "VALIDATION_ERROR"
    ErrRateLimited      = "RATE_LIMITED"
)
```

**Step 2:** Update `shared.JSONError`:

```go
func JSONError(c *gin.Context, status int, code string, message string) {
    c.JSON(status, NewError(code, message))
}

func InternalError(c *gin.Context, err error) {
    slog.Error("internal server error", "error", err)
    c.JSON(http.StatusInternalServerError, NewError(ErrInternal, "internal server error"))
}
```

**Step 3:** Update frontend error handling in `http-client.ts`:

```typescript
if (error.response?.data?.error) {
    const code = error.response.data.error.code;
    const message = error.response.data.error.message;
    // Handle structured errors
}
```

> **Note:** This requires coordinated frontend+backend changes. Do not deploy until both sides are updated.

**Verification:** All endpoints return consistent `{"error":{"code":"...","message":"..."}}` format. All existing E2E and frontend tests pass after update.

**Depends on:** Nothing (but coordinate with frontend)

**Status:** SUDAH DIIMPLEMENTASI — `shared.NewError(code, message)` dengan `ErrorResponse`/`ErrorDetail` (`{"error":{"code","message"}}`) ada di `internal/shared/response.go:23-34`, dipakai konsisten di handler.

---

### 3.7 Extract Wiring from `main.go` ✅ DONE

**Audit Ref:** A‑02  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** Low (pure refactoring, no behavior change)  
**Files affected:** `cmd/server/main.go`, new `internal/wiring/wiring.go`

**Problem:** All dependency construction lives in `main.go` (~160 lines of flat procedural code). Every new module requires manual wiring.

**Solution:**

Create a wiring package:

```go
// internal/wiring/wiring.go
package wiring

import (
    "retail-pos-system/internal/audit"
    "retail-pos-system/internal/brand"
    // ...
)

type Dependencies struct {
    // Repositories
    UserRepo        *user.Repository
    ProductRepo     *product.Repository
    SaleRepo        *sale.Repository
    // ...
    
    // Services
    UserSvc         *user.Service
    AuthSvc         *user.AuthService
    ProductSvc      *product.Service
    SaleSvc         *sale.Service
    // ...
    
    // Handlers
    UserH           *user.Handler
    AuthH           *user.AuthHandler
    // ...
    
    // Cross-cutting
    Bus             *eventbus.Bus
    Hub             *websocket.Hub
    Cache           *cache.Cache
}

type Providers struct {
    DB      shared.DBPool
    Config  *config.Config
}

func Initialize(p Providers) *Dependencies {
    d := &Dependencies{}
    appCache := cache.New(10*time.Minute, 30*time.Second)
    d.Cache = appCache
    
    bus := eventbus.New()
    bus.SetDeadLetterStore(eventbus.NewPgDeadLetterStore(p.DB))
    d.Bus = bus
    
    // Repositories
    d.UserRepo = user.NewRepository(p.DB)
    d.UserRepo.SetCache(appCache)
    d.ProductRepo = product.NewRepository(p.DB)
    d.ProductRepo.SetCache(appCache)
    // ...
    
    // Services
    d.UserSvc = user.NewService(d.UserRepo)
    d.AuthSvc = user.NewAuthService(d.UserRepo)
    // ...
    
    // Handlers
    d.UserH = user.NewHandler(d.UserSvc, d.AuditSvc)
    d.AuthH = user.NewAuthHandler(d.AuthSvc, d.AuditSvc)
    // ...
    
    return d
}
```

Then `main.go` becomes:

```go
func main() {
    cfg := config.Load()
    shared.InitLogger(cfg.Env)
    
    pool := shared.NewDBPool(cfg)
    cache := shared.NewCache(cfg)
    
    deps := wiring.Initialize(wiring.Providers{
        DB:     pool,
        Config: cfg,
    })
    
    go deps.Bus.Run()
    go deps.Hub.Run()
    
    router := setupRouter(cfg, deps)
    // ...
}
```

**Verification:** `go build ./...` compiles. App starts and behaves identically. All routes functional.

**Depends on:** Nothing (but coordinate with 3.1 to avoid merge conflicts)

**Status:** SUDAH DIIMPLEMENTASI — wiring diekstrak ke `internal/wiring/wiring.go`.

---

### 3.8 Add Persistent API Cache on Frontend ✅ DONE

**Audit Ref:** FE‑02  
**Effort:** 4 hours  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `web/src/shared/api/http-client.ts`

**Problem:** No caching of GET responses. Navigating away and back re-fetches all data.

**Solution:**

```typescript
// web/src/shared/api/cache.ts
const cache = new Map<string, { data: unknown; expiresAt: number }>();

const DEFAULT_TTL = 30_000; // 30 seconds

export function getCached<T>(key: string): T | null {
    const entry = cache.get(key);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
        cache.delete(key);
        return null;
    }
    return entry.data as T;
}

export function setCache(key: string, data: unknown, ttl = DEFAULT_TTL) {
    cache.set(key, { data, expiresAt: Date.now() + ttl });
}

export function invalidateCache(prefix: string) {
    for (const key of cache.keys()) {
        if (key.startsWith(prefix)) {
            cache.delete(key);
        }
    }
}
```

Then use it in an Axios interceptor:

```typescript
apiClient.interceptors.request.use((config) => {
    if (config.method === 'GET') {
        const cached = getCached(config.url!);
        if (cached) {
            // Return cached data via adapter
        }
    }
    return config;
});

apiClient.interceptors.response.use((response) => {
    if (response.config.method === 'GET') {
        setCache(response.config.url!, response.data);
    }
    // Invalidate on mutations
    if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(response.config.method!)) {
        invalidateCache(response.config.url!);
    }
    return response;
});
```

**Verification:** Navigate between pages → network tab shows fewer duplicate requests. After mutating data, cache is invalidated and fresh data is fetched.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — `web/src/shared/api/cache.ts` menyediakan cache API dengan TTL + invalidasi; `http-client.ts` memakainya untuk deduplikasi request.

---

### 3.9 Inline Form Validation Standardization ✅ DONE

**Audit Ref:** UX‑04  
**Effort:** 4 hours  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** Multiple form component files

**Problem:** Inconsistent validation feedback — some forms use inline errors, others use toasts.

**Solution:**
Standardize on this pattern for all forms:

```svelte
<!-- Shared pattern for form validation -->
{#each formErrors as error}
  <div class="text-red-500 text-sm mt-1" role="alert">{error.message}</div>
{/each}
```

- Field-level errors appear below the field
- Form-level errors appear in a banner at the top
- Toast is used only for success/navigation feedback

**Verification:** All forms show validation errors inline. No validation errors appear only as toasts.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — validasi inline via `validateProductForm` (`web/src/modules/product/lib/product-utils.ts:41`) dan pola serupa per modul; error field ditampilkan inline.

---

### 3.10 Create Materialized Views for Reports ✅ DONE

**Audit Ref:** P‑04  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** Low (read-only change)  
**Files affected:** `database/migrations/000_squash.sql`, `internal/report/repository.go`

**Problem:** Sales chart and period comparison queries aggregate raw `sales` table on-the-fly.

**Solution:**

**Step 1:** Create materialized views:

```sql
CREATE MATERIALIZED VIEW mv_daily_sales AS
SELECT 
    DATE(created_at AT TIME ZONE 'Asia/Jakarta') as sale_date,
    store_id,
    COUNT(*) as transaction_count,
    COUNT(DISTINCT cashier_id) as active_cashiers,
    COALESCE(SUM(total_amount), 0) as total_revenue,
    COALESCE(SUM(subtotal), 0) as total_subtotal,
    COALESCE(SUM(discount), 0) as total_discount,
    COALESCE(SUM(tax), 0) as total_tax
FROM sales
WHERE status = 'completed'
GROUP BY DATE(created_at AT TIME ZONE 'Asia/Jakarta'), store_id
WITH DATA;

CREATE UNIQUE INDEX idx_mv_daily_sales_date_store ON mv_daily_sales(sale_date, store_id);
```

```sql
CREATE MATERIALIZED VIEW mv_hourly_sales AS
SELECT 
    DATE_TRUNC('hour', created_at AT TIME ZONE 'Asia/Jakarta') as sale_hour,
    store_id,
    COUNT(*) as transaction_count,
    COALESCE(SUM(total_amount), 0) as total_revenue
FROM sales
WHERE status = 'completed'
GROUP BY DATE_TRUNC('hour', created_at AT TIME ZONE 'Asia/Jakarta'), store_id
WITH DATA;
```

**Step 2:** Create refresh function:

```sql
CREATE OR REPLACE FUNCTION refresh_sales_mv()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_sales;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hourly_sales;
END;
$$ LANGUAGE plpgsql;
```

**Step 3:** Update `report/repository.go` to query from materialized views instead of raw `sales`.

**Step 4:** Schedule periodic refresh (via cron job or application scheduler):

```go
// In main.go or a dedicated scheduler
func startMVRefreshScheduler(pool *pgxpool.Pool) {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            _, err := pool.Exec(context.Background(), "SELECT refresh_sales_mv()")
            if err != nil {
                slog.Error("failed to refresh materialized views", "error", err)
            }
        }
    }()
}
```

**Verification:**
1. Run `SELECT * FROM mv_daily_sales` — data matches raw sales aggregation
2. Run chart queries — sub-second response time
3. After new sales, run refresh — new data appears in views

**Depends on:** Database migration

**Status:** SUDAH DIIMPLEMENTASI — `mv_hourly_sales`/`mv_daily_sales` dibuat (migration `001_materialized_views.sql`), di-refresh via `refresh_sales_mv()` yang dikelola `report.RefreshCoordinator` (debounce 30s). Query report membaca MV (lih. `internal/report/repository.go`).

---

### 3.11 Increase Body Limit for Import Routes ✅ DONE

**Audit Ref:** S‑05  
**Effort:** 1 hour  
**Difficulty:** Small  
**Risk:** Low  
**Files affected:** `cmd/server/main.go`

**Problem:** 1MB global body limit blocks legitimate bulk import files.

**Solution:**
Apply route-specific body limits:

```go
// In main.go, use Gin's built-in MaxMultipartMemory or custom middleware:
protected.POST("/import/*path", middleware.BodyLimitMiddleware(10<<20), ieH.Import)
// 10 MB for import routes vs 1 MB global
```

Or configure per-route using Gin's `MaxMultipartMemory`:

```go
router.MaxMultipartMemory = 10 << 20 // 10 MB for file uploads
```

**Verification:** Upload a 5MB CSV/XLSX file via import → succeeds. Upload a 15MB file → rejected.

**Depends on:** Nothing

**Status:** SUDAH DIIMPLEMENTASI — `BodyLimitMiddleware` di `internal/middleware/body_limit.go` (dengan `defaultBodyLimit = 32 << 20` / 32MB di `cmd/server/main.go:35`), dipasang pada rute import. Ada unit test `internal/middleware/body_limit_test.go`.

---

## 4. Phase 3 — Pre-Production Readiness

### 4.1 Add CI Pipeline

> **Status: ON HOLD** — deferred by decision 2026-08-10. Revisit before production launch.

**Audit Ref:** T‑01, D‑01  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** Low  
**Files affected:** `.github/workflows/ci.yml` (new file)

**Solution:**

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:18-alpine
        env:
          POSTGRES_USER: pos
          POSTGRES_PASSWORD: admin123
          POSTGRES_DB: retail_pos_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5433:5432
    
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      
      - name: Install dependencies
        run: go mod download
      
      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
      
      - name: Test
        env:
          TEST_DB_PORT: 5433
          DB_PORT: 5433
          TEST_DB_USER: pos
          TEST_DB_PASSWORD: admin123
          DB_USER: pos
          DB_PASSWORD: admin123
          JWT_SECRET: test-secret-for-testing-only
        run: go test -p 1 -count=1 -coverprofile=coverage.out $(go list ./... | grep -v -E '(retail-pos-system/cmd/|retail-pos-system/tools/)')
      
      - name: Build
        run: go build -o /dev/null ./cmd/server

  frontend:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: web
    
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: web/package-lock.json
      
      - name: Install dependencies
        run: npm ci
      
      - name: Lint
        run: npm run lint || true  # if eslint configured
      
      - name: Test
        run: npm run test:run
      
      - name: Build
        run: npm run build
```

**Verification:** Push to GitHub → workflow runs → green checkmark. Test failures cause red.

**Depends on:** GitHub repository configured

---

### 4.2 Add Prometheus + Grafana

> **Status: ON HOLD** — deferred by decision 2026-08-10. Revisit before production launch.

**Audit Ref:** D‑02  
**Effort:** 5 days  
**Difficulty:** Large  
**Risk:** Low  
**Files affected:** Multiple new files

**Solution:**

**Step 1:** Add Gin metrics middleware:

```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "Duration of HTTP requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status"},
    )
    
    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Duration of database queries",
            Buckets: prometheus.DefBuckets,
        },
        []string{"query"},
    )
    
    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_connections",
            Help: "Number of active HTTP connections",
        },
    )
)
```

**Step 2:** Add Prometheus to docker-compose:

```yaml
prometheus:
  image: prom/prometheus
  volumes:
    - ./deploy/prometheus.yml:/etc/prometheus/prometheus.yml
  ports:
    - 9090:9090

grafana:
  image: grafana/grafana
  ports:
    - 3000:3000
  depends_on:
    - prometheus
```

**Verification:** Hit `/metrics` endpoint → see metrics. Configure Grafana dashboard → see request duration, error rate, DB pool stats.

**Depends on:** Production deployment timeline

---

### 4.3 Add Automated Database Backup

**Audit Ref:** D‑03  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** Low  
**Files affected:** `deploy/docker-compose.yml` or host cron

**Solution:**

```yaml
# Docker Compose backup service (runs on schedule)
backup:
  image: postgres:18-alpine
  depends_on:
    - postgres
  environment:
    PGUSER: pos
    PGPASSWORD: ${DB_PASSWORD}
    PGHOST: postgres
    PGDATABASE: retail_pos
  volumes:
    - ./backups:/backups
  command: >
    sh -c "
    while true; do
      pg_dump -Fc -f /backups/backup_\$(date +%Y%m%d_%H%M%S).dump
      # Keep only last 7 days
      find /backups -name '*.dump' -mtime +7 -delete
      sleep 86400
    done
    "
```

Or via host cron:

```bash
# /etc/cron.d/retail-pos-backup
0 2 * * * root pg_dump -h localhost -U pos -Fc retail_pos > /backups/retail_pos_$(date +\%Y\%m\%d).dump && find /backups -name 'retail_pos_*.dump' -mtime +7 -delete
```

**Verification:**
1. Wait for scheduled backup or trigger manually
2. Verify backup file exists and is non-empty
3. Test restore: `pg_restore -d retail_pos_test backups/backup_xxx.dump`

**Depends on:** Production deployment timeline

**Status:** BELUM DIIMPLEMENTASI — hanya ada target manual `make db-backup` (Makefile:92). Belum ada cron/host scheduler maupun service backup berjadwal di `deploy/`. **Tersisa untuk pre-production.**

---

## 5. Phase 4 — Post-Launch Enhancements

### 5.1 Multi-Warehouse Inventory Support ✅ DONE

**Audit Ref:** DB‑01  
**Effort:** 3 days  
**Difficulty:** Medium  
**Risk:** High (schema change, data migration required)  
**Files affected:** Database migration, product repository

**Problem:** `product_stock.product_id` has UNIQUE constraint, preventing per-warehouse stock.

**Solution:**

**Step 1:** Alter constraint:

```sql
-- Remove existing unique constraint
ALTER TABLE product_stock DROP CONSTRAINT product_stock_product_id_key;
-- Add composite unique
ALTER TABLE product_stock ADD UNIQUE (product_id, warehouse_id, store_id);
```

**Step 2:** Update all stock queries to include warehouse/store filtering.

**Step 3:** Backfill data for existing single-warehouse records.

**Verification:** Multiple stock records per product with different warehouse/store IDs.

**Depends on:** Not needed until multi-warehouse is a business requirement

**Status:** SUDAH DIIMPLEMENTASI — migration `002_multi_warehouse.sql` menambah kolom `warehouse_id`/`store_id` pada `product_stock` dengan unique constraint gabungan (`uq_product_stock`), lalu `020_per_rack_stock.sql` menambah `location_id` (UNIQUE NULLS NOT DISTINCT). Stock global vs per-warehouse di-handle `internal/inventory/stock_deducer.go` (baris global `warehouse_id IS NULL AND store_id IS NULL`).

---

### 5.2 POS Keyboard Navigation

**Audit Ref:** UX‑03  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** Low  
**Files affected:** `web/src/modules/pos/`

**Solution:**
Add Svelte `onkeydown` handler on the POS page:

```svelte
<svelte:window onkeydown={handleKeydown} />

<script lang="ts">
function handleKeydown(e: KeyboardEvent) {
    switch (e.key) {
        case 'ArrowDown':
            e.preventDefault();
            // Move selection down in product grid
            break;
        case 'ArrowUp':
            // Move selection up
            break;
        case 'Enter':
            // Add selected product to cart
            break;
        case 'Escape':
            // Clear search / close modal
            break;
        case 'F2':
            // Quick product lookup
            break;
    }
}
</script>
```

**Status:** SUDAH DIIMPLEMENTASI — `web/src/modules/pos/components/PosPage.svelte` sudah punya `handleGlobalKeydown` (`<svelte:window onkeydown=...>`): `F2` fokus ke `#pos-search-input`, `Escape` clear search/close modal, `ArrowDown`/`ArrowUp` pindah `selectedProductIndex` (dengan `scrollSelectedIntoView`), `Enter` tambahkan produk terpilih, `F4` checkout, `F5` modal parked, `F6` hold sale, `Alt+Delete` clear cart. Guard pada event target mencegah double-fire: Enter/Space dari dalam `INPUT`/`TEXTAREA`/`SELECT`/`BUTTON`/`A` tidak di-intercept (native handler yang bertanggung jawab), dan arrow key tetap aktif saat fokus di search input (`pos-search-input`). `handleSearchSubmit` kini menambah produk terpilih (bukan `products[0]`). Hint keyboard ditambahkan di `ProductSearchPanel.svelte` (`F2`, `↑↓`, `Enter`) dengan label i18n `posSelectProductHint`/`posAddToCartHint`.

**Verification:** Test all keyboard shortcuts in the POS flow. No conflicts with existing browser shortcuts.

**Depends on:** Nothing

---

### 5.3 Replace `patrickmn/go-cache` with `ristretto` ✅ DONE

**Audit Ref:** DR‑01  
**Effort:** 2 days  
**Difficulty:** Medium  
**Risk:** Low (the `Cache` wrapper abstracts the implementation)  
**Files affected:** `pkg/cache/cache.go`
**Status:** SUDAH DIIMPLEMENTASI — `pkg/cache/cache.go` sekarang menggunakan `github.com/dgraph-io/ristretto v0.2.0` (lihat `go.mod`). Konfigurasi: `NumCounters: 1e7`, `MaxCost: 1 << 30`, `BufferItems: 64`, `Metrics: true`. Wrapper menyimpan `keys map[string]struct{}` untuk `FlushByPrefix`, menerapkan TTL + jitter (10%) per set, dan memanggil `store.Wait()` setelah `Del` untuk memastikan invalidation tercatat sebelum event `sale.created` diproses. Tercatat juga di `docs/audits/post-hardening-improvement-plan.md`.

**Problem:** `patrickmn/go-cache` is unmaintained (last updated 2019). The wrapper abstraction makes replacement contained.

**Solution:**

```go
import "github.com/dgraph-io/ristretto"

type Cache struct {
    store *ristretto.Cache
}

func New() *Cache {
    store, err := ristretto.NewCache(&ristretto.Config{
        NumCounters: 1e7,     // number of keys to track frequency
        MaxCost:     1 << 30, // maximum cost of cache (1GB)
        BufferItems: 64,      // number of keys per Get buffer
    })
    if err != nil {
        panic(err)
    }
    return &Cache{store: store}
}

func (c *Cache) Set(key string, value interface{}) {
    c.store.SetWithTTL(key, value, 1, c.defaultTTL) // cost=1
}

func (c *Cache) Get(key string) (interface{}, bool) {
    return c.store.Get(key)
}
```

**Verification:** All cache tests pass. Cache behavior identical to before.

**Depends on:** Nothing urgent

---

## 6. Dependency Graph

```
Phase 1 — No dependencies between items. All can be parallelized.
│
├── 2.1 Merge scanProduct functions
├── 2.2 Fix failing E2E test
├── 2.3 Standardize slog usage
├── 2.4 Add empty states
├── 2.5 Standardize skeletons
├── 2.6 Extract magic numbers
├── 2.7 Add security header comment
├── 2.8 Add rate limit headers
├── 2.9 Add LOG_LEVEL support
│
Phase 2 — Some items benefit from Phase 1 completion
│
├── 3.1 Extract sale transaction logic  ← HIGHEST VALUE
├── 3.2 Fix N+1 in GetSaleByID          ← Independent
├── 3.3 Add failed login audit          ← Independent
├── 3.4 Build WHERE clause builder      ← Independent (but benefits from slog standardization)
├── 3.5 Add WS CSRF protection          ← Independent
├── 3.6 Standardize error format        ← Requires frontend coordination
├── 3.7 Extract wiring from main.go     ← Independent
├── 3.8 Add frontend API cache          ← Independent
├── 3.9 Form validation standardization ← Independent
├── 3.10 Create materialized views      ← Independent
├── 3.11 Increase body limit            ← Independent
│
Phase 3 — Dependent on deployment decision
│
├── 4.1 Add CI pipeline                 ← Can be done anytime
├── 4.2 Add monitoring                  ← Before production
├── 4.3 Add automated backup            ← Before production
│
Phase 4 — Post-launch
│
├── 5.1 Multi-warehouse inventory       ← Feature-driven
├── 5.2 POS keyboard navigation         ← UX-driven
├── 5.3 Replace go-cache                ✅ DONE (ristretto)
```

**Recommended ordering within Phase 2:**
1. 3.1 (sale transaction logic) — highest risk, highest value
2. 3.6 (error format) + 3.7 (wiring) — structural but safe
3. 3.3 (login audit) + 3.5 (WS CSRF) — security hardening
4. 3.2 (N+1 fix) + 3.4 (query builder) + 3.10 (materialized views) — performance
5. 3.8 (frontend cache) + 3.9 (validation) + 3.11 (body limit) — polish

---

## 7. Effort Summary

| Phase | Tasks | Total Effort | Risk Level |
|-------|-------|-------------|------------|
| Phase 1 — Code Health | 9 tasks | ~7 hours | Low |
| Phase 2 — Structural | 11 tasks | ~9 days | Medium (sale refactor) |
| Phase 3 — Pre-Production | 3 tasks | ~8 days | Low |
| Phase 4 — Enhancements | 2 tasks (5.3 ✅ DONE) | ~5 days | Low |
| **Total** | **26 tasks** | **~22 developer days** | |

**Critical path** (must be done in order):
1. Phase 1 items (parallel)
2. 3.1 Extract sale logic
3. Phase 3 items (parallel, before deployment)

All other tasks can be done in any order within their phase.

---

## 8. Risk During Remediation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Sale transaction refactor (3.1) introduces regression in price calculation | Low | Critical | Write integration tests first. Compare before/after output for same inputs. Deploy to staging first |
| Error format change (3.6) breaks frontend | Medium | High | Coordinate frontend+backend release. Update frontend error handling first, then deploy backend |
| Schema change for materialized views (3.10) causes locking | Low | Medium | Use `CONCURRENTLY` option. No lock on source table |
| CI pipeline (4.1) fails due to test flakes | Medium | Medium | Fix pre-existing test flake first (2.2). Use `-p 1` flag in CI |
| Multi-warehouse migration (5.1) loses data | Low | Critical | Test migration on copy of production data first. Have rollback script ready |

---

*This remediation plan is based on the Project Audit Report v1.1 and assumes the project is pre-production. Adjust priorities accordingly as the project approaches deployment.*
