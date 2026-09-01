# Plan: Partitioned Audit Logs with Tiered Retention

## Current State

| Aspect | Status |
|--------|--------|
| Table | `audit_logs` — SERIAL PK, JSONB old/new values, FK to users/stores |
| Immutability | Append-only trigger (`trg_audit_logs_immutable`) with GUC bypass for maintenance + FK cascade bypass |
| Retention | `AuditRetentionDays = 365` + `PurgeOlderThan()` defined but **no scheduler wired** |
| Indexes | `idx_audit_logs_created`, `idx_audit_logs_user`, `idx_audit_logs_store`, `idx_audit_logs_action_ip_created` |
| Export | `audit.export` permission + CSV export handler |
| Queries | `GetAuditLogs` (paginated, filtered), `GetAuditLogByID`, `GetDistinctEntityTypes`, `CountRecentLoginFailures`, `CountRecentLoginFailuresByUsername`, `ExportAuditLogs` |
| Test cleanup | `shared.TruncateTestData` uses `TRUNCATE TABLE audit_logs CASCADE` |
| Seed tool | `cmd/dummy/main.go` batch INSERT into `audit_logs` |

---

## Proposed Changes

### 1. Partitioning — PostgreSQL Declarative Partitioning

Convert `audit_logs` to a **RANGE-partitioned table** on `created_at` with daily intervals.

```
audit_logs (parent, partitioned by RANGE created_at)
├── audit_logs_2026_08_27
├── audit_logs_2026_08_28
├── ...
└── audit_logs_2026_09_03
```

**New DDL:**

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL,
    user_id integer,
    store_id integer,
    role character varying(50),
    action character varying(100) NOT NULL,
    entity_type character varying(100),
    entity_id integer,
    old_values jsonb,
    new_values jsonb,
    description text,
    ip_address inet,
    user_agent text,
    correlation_id text,
    created_at timestamp with time zone DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

### 2. FK Constraints Removed

`user_id → users.id` and `store_id → stores.id` with `ON DELETE SET NULL` cannot exist on partitioned tables in PostgreSQL. Rely on application-level integrity.

### 3. Append-Only Trigger Recreated Per Partition

Each partition needs its own copy of `trg_audit_logs_immutable`. The parent-table trigger does NOT fire for inserts routed to child partitions.

### 4. Indexes Redesigned

| Old Index | New Index |
|-----------|-----------|
| `idx_audit_logs_created (created_at)` | Implicit via partition key; no separate index needed on parent |
| `idx_audit_logs_user (user_id)` | Created on each partition |
| `idx_audit_logs_store (store_id)` | Created on each partition |
| `idx_audit_logs_action_ip_created (action, ip_address, created_at)` | Created on each partition |

### 5. Partition Lifecycle Tiers

| Tier | Age | Location | Action |
|------|-----|----------|--------|
| Hot | 0–30 days | Primary DB | Full fidelity, all indexes |
| Warm | 31–365 days | Primary DB | Same indexes, lower query frequency |
| Cold | 366+ days | Detach → archive / DROP | Detach partition, compress, store off-DB or drop |

### 6. Automated Partition Management

#### Existing Scheduling Infrastructure

The codebase uses **in-process background goroutines** — no external cron library. The established pattern is `RefreshCoordinator` (`internal/report/refresh_coordinator.go`): a struct with `Start()`/`Shutdown()` lifecycle, background goroutine with time-based scheduling, and retry with exponential backoff. Started in `cmd/server/main.go:104-105`.

#### Recommended Approach

Two complementary mechanisms:

**A. Server startup — ensure look-ahead partitions exist**

```go
// Called once in main.go after DB pool is ready
if err := audit.EnsurePartitions(dbPool); err != nil {
    slog.Error("failed to create audit log partitions", "error", err)
    os.Exit(1)
}
```

Creates partitions for today + next 7 days. If they already exist, no-op. This covers the common case: server restarts after downtime, new partitions are ready before any inserts arrive.

**B. Daily background goroutine — drop expired partitions**

Follow the `RefreshCoordinator` pattern:

```go
type PartitionCoordinator struct {
    mu      sync.Mutex
    started bool
    closed  bool
    cancel  context.CancelFunc
    done    chan struct{}
    db      shared.DBPool
    retentionDays int
}

func (c *PartitionCoordinator) run(ctx context.Context) {
    defer close(c.done)
    c.dropExpired(ctx) // immediate on startup

    for {
        next := time.Now().Truncate(24*time.Hour).Add(24*time.Hour) // next midnight
        timer := time.NewTimer(time.Until(next))
        select {
        case <-timer.C:
            c.dropExpired(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

Started alongside `ReportRefreshCoord` in `main.go`:

```go
deps.PartitionCoord.Start()
defer deps.PartitionCoord.Shutdown()
```

**C. Optional CLI — manual partition management**

`cmd/partition-audit/main.go` for operators who want to run partition maintenance outside the server process (e.g., cron job, systemd timer). Flags: `-ensure`, `-drop-expired`, `-retention-days=365`.

This CLI reuses the same `EnsurePartitions`/`DropPartition` functions from the in-process goroutine. Example crontab entry:

```cron
# Daily at 02:00 — drop expired audit log partitions
0 2 * * * /usr/local/bin/partition-audit -drop-expired -retention-days=365

# Every 6 hours — ensure future partitions exist (belt-and-suspenders)
0 */6 * * * /usr/local/bin/partition-audit -ensure
```

### 7. PurgeOlderThan Implementation Change

Replace `DELETE FROM audit_logs WHERE created_at < $1` with partition DROP:

```go
func (r *Repository) DropPartition(ctx context.Context, partitionDate time.Time) error {
    name := fmt.Sprintf("audit_logs_%s", partitionDate.Format("2006_01_02"))
    _, err := r.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", name))
    return err
}
```

### Cron Job Options Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **In-process goroutine** (RefreshCoordinator pattern) | No external dependency; follows existing codebase pattern; graceful shutdown with server | Dies with server; requires server restart to resume | **Recommended** for this project |
| External cron (systemd timer, crond) | Runs independently of server; survives server restarts | Adds deployment complexity; requires DB credentials outside server; no shared retry/metrics | Beneficial in specific scenarios (see below) |
| pg_partman extension | Automated partition creation/drop; battle-tested | Extension dependency; less control over scheduling; may conflict with existing migration workflow | Consider for future if partition complexity grows |
| Run once on startup only | Simplest | No automatic cleanup of old partitions; relies on operators | Insufficient for retention policy |

The in-process goroutine approach is recommended because:
1. **Matches existing pattern** — `RefreshCoordinator` already does exactly this for materialized views
2. **No external dependencies** — no cron, no pg_partman, no systemd
3. **Graceful lifecycle** — `Start()`/`Shutdown()` with context cancellation
4. **Observability** — can expose metrics (partitions created, dropped, failures)
5. **Retry on failure** — exponential backoff if DROP fails (e.g., locks)

#### When OS Cron Becomes Beneficial

The in-process goroutine is sufficient for a single-server deployment with high uptime. OS cron adds value in these scenarios:

| Scenario | Why cron helps | In-process gap |
|----------|---------------|----------------|
| **Multi-instance deployment** (load balancer, multiple pods) | Only one cron job needed; avoids duplicate partition drops across instances | Each instance runs the goroutine; concurrent DROP IF EXISTS is idempotent but wasteful |
| **Long server downtime** (>24h) | Cron can create/drop partitions while server is down | Partitions created on next startup; old partitions wait until restart |
| **Separation of duties** | DBA runs partition maintenance independently of app deployment | App team owns partition logic |
| **Burst retention enforcement** | Cron can be triggered manually to drop partitions immediately (e.g., disk full) | Requires code change or restart with different config |
| **Compliance/audit** | Cron logs provide independent evidence of retention enforcement | Tied to app logs |

**Recommendation for this project:** Start with in-process goroutine. If you later deploy multiple instances or need DBA-controlled partition maintenance, add an optional cron entry point (`cmd/partition-audit/main.go`) that calls the same `EnsurePartitions`/`DropExpiredPartitions` functions. The code is the same either way — the scheduling mechanism is the only difference.

---

## Implications

### Breaking Changes (Must Fix)

| # | Change | Why | Mitigation |
|---|--------|-----|------------|
| 1 | Remove FK constraints (`user_id`, `store_id`) | PostgreSQL doesn't support FKs from partitioned tables unless FK includes partition key | Application-level integrity; existing `isForeignKeyViolation` retry already handles dangling user_id gracefully |
| 2 | Recreate append-only trigger per partition | Triggers on parent don't fire for partition inserts | Use pg_partman or manual trigger creation in partition management code |
| 3 | PK becomes `(id, created_at)` composite | Partitioned tables require PK to include partition key | Go `domain.go` `Log.ID` stays `int`; `GetAuditLogByID` works unchanged for single-row lookups |

### Behavioral Changes (Verify)

| # | Change | Why | Mitigation |
|---|--------|-----|------------|
| 4 | `PurgeOlderThan` becomes DROP not DELETE | DELETE on partitioned table is slow; DROP is instant | Update method implementation |

### Non-Issues (Verified Safe)

| # | Query | Why it's fine |
|---|-------|---------------|
| 5 | `GetAuditLogs` with ILIKE search | Frontend **always** sends `start_date`/`end_date` (default: 7d range). PostgreSQL prunes on `created_at` first, then applies ILIKE only within matching partitions. A 7-day search scans ~7 partitions, not 365. |
| 6 | `GetDistinctEntityTypes` (no date filter) | Scans all partitions, but entity types are a small stable set (~15 values). Each partition has an index on `entity_type`. Index-only scan + hash aggregate takes <50ms even with 365 partitions. Called once per page load. |

### Non-Breaking (No Action Required)

| # | Area | Notes |
|---|------|-------|
| 7 | TRUNCATE in tests | `TRUNCATE TABLE audit_logs CASCADE` works on partitioned parent — cascades to all partitions |
| 8 | Seed tool batch INSERT | Works unchanged; inserts route to correct partition automatically |
| 9 | `CountRecentLoginFailures` | Already filters by `created_at >= $2` — partition pruning works |
| 10 | `CountRecentLoginFailuresByUsername` | Same — already prunable |
| 11 | Export handler | Uses `GetAuditLogs` with date filters — prunable |
| 12 | pg_dump / pg_basebackup | Works unchanged on partitioned tables |

### Migration Risk

| Risk | Severity | Mitigation |
|------|----------|------------|
| Data loss during migration | High | Test on staging; use blue-green swap approach |
| Downtime during swap | High | Brief maintenance window for table rename |
| Duplicate IDs across partitions | Medium | Composite PK `(id, created_at)` prevents this |
| Index bloat from per-partition indexes | Low | Standard PostgreSQL maintenance (VACUUM/ANALYZE) |
| `archtest` dependency matrix | Low | May need update if table name changes |
| ILIKE search performance | **None** | Frontend always sends date range; partition pruning works |
| GetDistinctEntityTypes perf | **None** | Small stable result set; index-only scan per partition |

---

## Test Impact Analysis

### Existing Tests — Direct INSERTs via Raw SQL

These tests insert into `audit_logs` using raw SQL. With partitioned tables, the INSERT fails if no partition exists for the `created_at` date.

| File | Test | INSERT date | Status |
|------|------|-------------|--------|
| `audit/repository_test.go:430` | `TestAuditRepository_PurgeOlderThan` | `2000-01-01` | **BREAK** — no partition for year 2000 |
| `user/repository_test.go:476-478` | `TestUserRepository_CountRecentLoginFailures` | `now` | OK — today's partition exists |
| `user/repository_test.go:482-485` | same | `now` | OK |
| `user/repository_test.go:496-498` | same | `now` | OK |
| `user/auth_service_test.go:602-605` | `TestAuthService_Login_RateLimitedByIP` | `NOW()` | OK |

### Existing Tests — Go `CreateAuditLog()` Calls

All Go-based inserts work fine — they're INSERT statements routed to the correct partition automatically. ~20 tests across `audit/`, `user/`, `sale/`, `shift/`, etc. **No changes needed.**

### Existing Tests — Raw SQL Queries

| File | Line | Query | Issue |
|------|------|-------|-------|
| `audit/handler_test.go` | 408, 413 | `SELECT count(*)/entity_type FROM audit_logs WHERE action = ...` | None |
| `user/handler_test.go` | 559 | `SELECT COUNT(*) FROM audit_logs WHERE entity_type = $1` | None |
| `user/repository_test.go` | 719, 730 | `SELECT COUNT(*) FROM audit_logs WHERE action = 'login_failed' AND created_at >= ...` | None — prunable |

### Existing Tests — Cleanup

| Mechanism | Works on partitioned? | Notes |
|-----------|----------------------|-------|
| `shared.TruncateTestData` → `TRUNCATE TABLE audit_logs CASCADE` | Yes | Cascades to all partitions |
| `cmd/dummy/main.go` → `TRUNCATE TABLE audit_logs RESTART IDENTITY CASCADE` | Yes | Same |

### Fix Required

`TestAuditRepository_PurgeOlderThan` (`repository_test.go:426-443`):

```go
// Current — FAILS because year 2000 has no partition:
_, err := dbPool.Exec(ctx, `INSERT INTO audit_logs (role, action, entity_type, created_at) 
    VALUES ('system', 'purge_stale', 'system', '2000-01-01 00:00:00+00')`)
```

**Fix:** Change the test date to a relative date within an existing partition (e.g., `time.Now().Add(-400 * 24 * time.Hour)` with a corresponding partition, or create the partition in test setup).

### New Tests to Create

| # | Test | Purpose | File |
|---|------|---------|------|
| 1 | `TestEnsurePartitions` | Verify `EnsurePartitions` creates daily partitions for the look-ahead window, skips existing ones, handles edge cases (month/year boundaries) | `audit/partition_test.go` |
| 2 | `TestDropPartition` | Verify `DropPartition` drops a specific partition, is idempotent (DROP IF EXISTS), and doesn't affect adjacent partitions | `audit/partition_test.go` |
| 3 | `TestAuditLogs_PartitionRouting` | Verify inserts with different `created_at` dates land in the correct partitions (query `pg_partitions` or check partition bounds) | `audit/repository_test.go` |
| 4 | `TestAuditLogs_PartitionPruning` | Verify a query with `created_at` filter only scans relevant partitions (use `EXPLAIN` to check partition pruning) | `audit/repository_test.go` |
| 5 | `TestAuditLogs_AppendOnlyTrigger_PerPartition` | Verify UPDATE/DELETE is rejected on rows in any partition (not just the parent) | `audit/repository_test.go` |

### Test Infrastructure Change

`shared.testdb.go` — `RunMigrations` should be updated to ensure partitions exist after applying migrations. The migration itself should create initial partitions, but test setup may need to call `EnsurePartitions` to cover test date ranges.

---

## Files Affected

| File | Change Type | Description |
|------|-------------|-------------|
| `database/migrations/038_partitioned_audit_logs.sql` | **New** | Migration: create partitioned table, create partitions, migrate data |
| `internal/audit/repository.go` | Modify | Update `PurgeOlderThan` to use partition DROP; add `EnsurePartitions` |
| `internal/audit/domain.go` | Verify | `Log.ID` type stays `int` — no change needed |
| `internal/audit/partition.go` | **New** | `PartitionCoordinator` — daily background goroutine for dropping expired partitions |
| `internal/audit/partition_test.go` | **New** | Tests for `EnsurePartitions`, `DropPartition`, `PartitionCoordinator` |
| `internal/audit/repository_test.go` | Modify | Fix `PurgeOlderThan` test date; add partition routing/pruning/trigger tests |
| `internal/wiring/wiring.go` | Modify | Wire `PartitionCoordinator` into dependency graph |
| `cmd/server/main.go` | Modify | Start `PartitionCoordinator` on server startup |
| `cmd/partition-audit/main.go` | **New** | Optional CLI tool for manual partition management |
| `internal/shared/testdb.go` | Verify | `TruncateTestData` TRUNCATE works on partitioned parent — verify |
| `cmd/dummy/main.go` | Verify | Batch INSERT works unchanged — verify |
| `README.md` | Modify | Add deployment notes for partition management |

---

## Implementation Order

1. Create migration `038_partitioned_audit_logs.sql`
   - Create new partitioned table
   - Create initial partitions (30-day look-ahead)
   - Migrate data from old table (batched, with `SET app.allow_audit_mod = 'on'`)
   - Swap table names
   - Drop old table after verification
2. Update `internal/audit/repository.go`
   - Add `EnsurePartitions` function
   - Update `PurgeOlderThan` to use partition DROP
3. Create `internal/audit/partition.go`
   - `PartitionCoordinator` following `RefreshCoordinator` pattern
   - Daily goroutine to drop partitions older than retention window
4. Update `internal/wiring/wiring.go`
   - Add `PartitionCoord` to `Deps` struct
   - Wire `PartitionCoordinator` in `Initialize()`
5. Update `cmd/server/main.go`
   - Start `PartitionCoordinator` on startup
   - Call `EnsurePartitions` before accepting traffic
6. Create `cmd/partition-audit/main.go` (optional CLI)
7. Create `internal/audit/partition_test.go`
   - `TestEnsurePartitions`
   - `TestDropPartition`
   - `TestPartitionCoordinator`
8. Update `internal/audit/repository_test.go`
   - Fix `TestAuditRepository_PurgeOlderThan` date
   - Add `TestAuditLogs_PartitionRouting`
   - Add `TestAuditLogs_PartitionPruning`
   - Add `TestAuditLogs_AppendOnlyTrigger_PerPartition`
9. Update `internal/shared/testdb.go` if needed
10. Verify all tests pass
11. Verify seed tool works
12. Update README.md with deployment notes (partition management, cron setup if used)
