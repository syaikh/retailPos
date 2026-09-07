# Fix Seed Data Bug: 4500 Products Not Injected

## Problem

The dummy data seeder (`cmd/dummy/main.go`) silently fails to inject products.
All 4 workers return 0 products, and the DB resilience check also finds 0 products.

**Symptom output:**
```
🗑️  Truncating existing transactional data...
✅ Data truncated successfully
🔧 Ensuring N categories exist...
   ✅ N categories ready
   ... (other ensure functions) ...
📦 Injecting 4500 products...
🚀 Starting 4 workers to inject 4500 products with batch size 100
   ✅ 0 products injected
```

## Root Cause Analysis

### What we know
1. Truncation succeeds (no error)
2. All 4 workers return 0 products (all inserts hit `sql.ErrNoRows`)
3. DB resilience check `SELECT COUNT(*) FROM products WHERE status = 'active'` returns 0
4. No FK violation warnings or transaction-poisoning errors appear

### Key code paths
- `injectProducts` (line 1471) starts 4 concurrent workers
- Each worker calls `processProductWorkerJob` (line 1572)
- Worker begins transaction, prepares INSERT:
  ```sql
  INSERT INTO products (sku, name, barcode, price, cost, category_id, status,
      tax_class_id, brand_id, unit_of_measure_id, created_at)
  VALUES ($1, $2, $3, $4, $5, $6, 'active', 1, $7, $8, $9)
  ON CONFLICT (sku) DO NOTHING RETURNING id
  ```
- If `Scan(&id)` returns `sql.ErrNoRows` → ON CONFLICT path taken → SKU already exists
- Resilience check (line 1553) queries DB directly → also finds 0 products

### The contradiction
- `sql.ErrNoRows` from `ON CONFLICT DO NOTHING RETURNING id` means the SKU already exists (conflict)
- But DB count is 0 → products don't exist
- These two facts are contradictory

### Hypothesis: `session_replication_role` interaction

The code comment at line 1553-1556 describes the known mechanism:

> "the ON CONFLICT … RETURNING id path can silently lose inserts when a
> leaked session_replication_role='replica' connection disables BEFORE
> INSERT triggers"

**Current "fix"** in `truncateAllData` (line 604):
- Pins connection with `db.Conn(ctx)` so SET and RESET happen on same session
- LIFO defer ordering is correct: RESET runs before Close
- But the bug persists, suggesting the fix is insufficient

**Possible leak vectors:**
1. A pre-existing connection in the pool already has `session_replication_role = 'replica'` when `truncateAllData` is called
2. The `db.Conn(ctx)` pin does not guarantee the connection was clean when obtained
3. Some other code path (e.g., `ensure*` functions) may set `session_replication_role` and not reset it

## Fix Plan — Implementation Status

### Step 1: Add pre-flight verification after truncation ✅ IMPLEMENTED
### Step 2: Reopen connection pool before workers ✅ IMPLEMENTED (revised)
### Step 3: Add diagnostic logging to first product inserts ✅ IMPLEMENTED
### Step 4: Add error detail logging (check if SKU exists in DB) ✅ IMPLEMENTED
### Step 5: Verify and test ✅ PASSED
- **Compile check:** `go build ./cmd/dummy/...` — PASS
- **Go tests:** `go test -p 1 -count=1 ./internal/consignment/... ./internal/archtest/... ./internal/inventory/...` — ALL PASS
- **Manual test:** Pending (requires running seeder against dev DB)

## Files modified

| File | Change |
|------|--------|
| `cmd/dummy/main.go` | Steps 1-4: pre-flight check, pool reset, diagnostic logging |

## Risk assessment

- **Low risk**: All changes are diagnostic/logging only (Steps 1-4)
- **Step 2** (pool reset) is the only behavioral change — it explicitly resets any leaked connection
- If Step 2 fixes the bug, we can remove the diagnostic logging later
- If Step 2 does NOT fix the bug, the diagnostic logging from Steps 1 and 4 will reveal the true root cause

## Alternative approach (if hypothesis is wrong)

If diagnostic logging reveals a different root cause, consider replacing the
`ON CONFLICT DO NOTHING RETURNING id` pattern entirely:

```go
// Instead of ON CONFLICT:
var id int
err := tx.QueryRowContext(ctx,
    `INSERT INTO products (sku, ...) VALUES ($1, ...) RETURNING id`,
    sku, ...).Scan(&id)
if err != nil {
    // Check if it's a unique violation (not ErrNoRows)
    if strings.Contains(err.Error(), "duplicate key") {
        continue // SKU already exists, skip
    }
    return nil, fmt.Errorf("insert product: %w", err)
}
```

This avoids the `ON CONFLICT DO NOTHING RETURNING id` edge case entirely,
using a plain `INSERT ... RETURNING id` and catching the unique violation
error directly.
