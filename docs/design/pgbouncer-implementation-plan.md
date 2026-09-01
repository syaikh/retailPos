# PgBouncer Implementation Plan

## Status

Deferred — 2026-09-01. PgBouncer is not needed now. The current pgxpool with MaxConns=25 is sufficient for a single-instance modular monolith. Revisit when deploying multiple app instances or hitting PostgreSQL connection limits. The plan is ready for when that time comes.

## Context

PgBouncer may be introduced in production to multiplex application connections onto a smaller pool of PostgreSQL backend processes. This document covers the implementation details, mode selection, and compatibility with the existing codebase including the dummy seeder.

## Architecture

```
Application (pgx/v5) → PgBouncer → PostgreSQL
```

## Pooling Mode Selection

PgBouncer supports three modes:

| Mode | How it works | Fit for this codebase |
|------|-------------|----------------------|
| **Session** | Client holds server connection for entire session | Safe but defeats purpose — need as many server connections as clients |
| **Transaction** | Client gets server connection only during `BEGIN`–`COMMIT`/`ROLLBACK` | **Best fit** — all transactions are explicitly bounded |
| **Statement** | Each individual statement gets a server connection | Impossible — codebase uses multi-statement transactions extensively |

**Decision: Transaction pooling.**

### Why transaction pooling works

- Every transaction in the codebase is explicitly bounded: `Begin()` → work → `Commit()`/`Rollback()`
- No session state is used: no `SET`, no `LISTEN/NOTIFY`, no advisory locks
- `CopyFrom` bulk writes use the PostgreSQL COPY protocol, which bypasses PgBouncer's query routing entirely
- Cross-module transactions use `BeginTx()` + injected `pgx.Tx` — clean boundaries

## Configuration

### PgBouncer config (`pgbouncer.ini`)

```ini
[databases]
retail_pos = host=localhost port=5432 dbname=retail_pos

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt

pool_mode = transaction
default_pool_size = 20
min_pool_size = 5
reserve_pool_size = 5
reserve_pool_timeout = 3
max_client_conn = 100
max_db_connections = 25

server_reset_query = DISCARD ALL
server_idle_timeout = 300
client_idle_timeout = 0
query_wait_timeout = 120
```

### pgx configuration in `cmd/server/main.go`

```go
poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
poolCfg.ConnConfig.PreferSimpleProtocol = true
```

This tells pgx to send all queries via the simple query protocol instead of the extended protocol (which uses `PREPARE`/`EXECUTE`). Required because PgBouncer transaction mode cannot route prepared statements across transactions.

**Tradeoff:** One extra round-trip per query for planning (no cached plan). Negligible for short POS queries.

### Environment variables

```bash
# Production
DATABASE_URL=postgres://pos:password@pgbouncer-host:6432/retail_pos?sslmode=require&timezone=Asia/Jakarta

# Development (no PgBouncer)
DATABASE_URL=postgres://pos:admin123@localhost:5433/retail_pos?sslmode=disable&timezone=Asia/Jakarta
```

## Compatibility Matrix

| Feature | PgBouncer Transaction Mode | Status |
|---------|---------------------------|--------|
| Positional parameters (`$1..$N`) | Works with simple protocol | Compatible |
| `CopyFrom` (COPY protocol) | Bypasses PgBouncer query routing | Compatible |
| `RETURNING` clause | Works with simple protocol | Compatible |
| `ON CONFLICT` upserts | Works with simple protocol | Compatible |
| `SELECT ... FOR UPDATE` | Works (within transaction) | Compatible |
| Recursive CTEs | Works with simple protocol | Compatible |
| JSONB operations | Works with simple protocol | Compatible |
| `ARRAY_AGG ... FILTER` | Works with simple protocol | Compatible |
| Materialized view refresh | Single statement, no session state | Compatible |
| `generate_series` | Single statement, no session state | Compatible |
| `setval()` | Single statement, no session state | Compatible |
| `pgxmock` testing | No PgBouncer in test env | Compatible |

## Dummy Seeder Compatibility

### Problem

The dummy seeder (`cmd/dummy/`) uses three patterns incompatible with PgBouncer transaction pooling:

1. **Prepared statements** — `tx.PrepareContext()` creates server-side prepared statements that PgBouncer cannot route across transactions
2. **`SET session_replication_role`** — Session-level state that would reset between transactions
3. **Commit cadence** — Sales worker commits every 500 rows, re-prepares statements each batch

### Recommendation: Bypass PgBouncer for the seeder

The seeder is a batch tool for local development, not a production service. It connects directly to PostgreSQL, bypassing PgBouncer entirely.

```
Development:
  Seeder → PostgreSQL (port 5433)
  App    → PostgreSQL (port 5433)

Production:
  App → PgBouncer (port 6432) → PostgreSQL (port 5432)
```

Two separate connection strings. Zero risk. This is the standard approach — batch jobs bypass connection poolers.

### Alternative: Rewrite seeder (if production seeding needed)

If the seeder ever needs to run in production (e.g., data migration):

1. Replace `tx.PrepareContext()` + `stmt.ExecContext()` with direct `tx.ExecContext()` + inline SQL
2. Replace `SET session_replication_role` with per-table `TRUNCATE ... CASCADE` (or use `DELETE FROM`)
3. `setval()` calls work fine — single statements, no session state

This is a low-effort change (~20 lines across `main.go` and `daily.go`).

## Changes Required

### Must change (1 file)

`cmd/server/main.go` — add simple protocol configuration:

```go
poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
poolCfg.ConnConfig.PreferSimpleProtocol = true
```

### Must NOT change

- `internal/shared/dbpool.go` — interface remains identical
- All repositories — zero changes
- All tests — zero changes (pgxmock doesn't use PgBouncer)
- Dummy seeder — bypasses PgBouncer, uses direct connection

### Infrastructure (production deployment)

- Deploy PgBouncer container alongside PostgreSQL
- Update `DATABASE_URL` in production environment
- Configure PgBouncer `pool_mode = transaction`
- Set `default_pool_size` based on expected concurrent connections

## Performance Expectations

| Metric | Without PgBouncer | With PgBouncer |
|--------|-------------------|----------------|
| PostgreSQL processes | 25 per app instance | 10-20 total |
| Connection establishment | ~5-10ms (fork) | ~0.1ms (in-process) |
| Per-query latency | Baseline | +0.1-0.5ms (routing overhead) |
| Memory usage | ~250MB per app (25 conns × 10MB) | ~100-200MB total |
| Under high load | Connections queue at PostgreSQL | Connections queue at PgBouncer (faster) |

**Net effect:** Performance stays flat or improves under load. Fewer PostgreSQL processes = less memory contention, less context switching. The only "cost" is the simple protocol flag disabling prepared statement caching, which saves virtually nothing for short POS queries.

## Risks

| Risk | Mitigation |
|------|------------|
| Simple protocol performance | Negligible for short queries; benchmark before/after |
| PgBouncer becomes single point of failure | Deploy with failover or use session pooling as fallback |
| `server_reset_query = DISCARD ALL` overhead | Standard best practice; minimal cost |
| Seeder incompatible | Bypass PgBouncer (direct connection) |
