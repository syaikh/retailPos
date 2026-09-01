# ADR: Database Persistence Architecture Assessment

## Status

Accepted (PgBouncer deferred) — 2026-09-01

## Context

Assess whether introducing an ORM/database abstraction layer would add meaningful value to the Go modular-monolith POS backend.

The application uses PostgreSQL and contains business-critical modules including products, inventory, stock opname, purchasing, pricing, suppliers, consignment, payments, reporting, and audit logging.

Options considered:

- Keep the current approach (raw pgx)
- sqlc + pgx
- Bun + pgx
- GORM
- Ent
- Hybrid approaches

PgBouncer may be used in production, potentially with transaction pooling.

## Current Architecture

### Driver & Connection Pooling

| Aspect | Value |
|--------|-------|
| **Driver** | `jackc/pgx/v5` v5.9.2 (native pgx, not `database/sql`) |
| **Pool** | `pgxpool.Pool` backed by `jackc/puddle/v2` |
| **MaxConns** | 25 |
| **MinConns** | 5 |
| **MaxConnLifetime** | 30 minutes |
| **MaxConnIdleTime** | 5 minutes |
| **HealthCheckPeriod** | 15 seconds |
| **DSN** | `timezone=Asia/Jakarta` always appended |

The application uses pgx's native API, not `database/sql`. Any ORM integration must either use pgx's native interface or wrap `database/sql` — which would lose pgx-specific features like `CopyFrom`.

### Repository Layer

Repositories are concrete structs (no interfaces) accepting `shared.DBPool`:

```go
type DBPool interface {
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
    Begin(ctx context.Context) (pgx.Tx, error)
}
```

This interface is satisfied by both `*pgxpool.Pool` and `pgx.Tx`, enabling transparent transaction passing.

### SQL Patterns

All SQL is hand-written raw SQL. No query builder beyond the minimal `shared.QueryBuilder` (35 lines) which only constructs positional WHERE clauses. The codebase uses:

- Positional `$1..$N` parameters with `fmt.Sprintf` for dynamic queries
- `pgx.CopyFrom` for bulk inserts (9 table targets)
- `sql.Null*` types for nullable column scanning
- Dedicated `scanXxx()` functions for row mapping
- Jakarta timezone conversion on every timestamp

### Transaction Management

Two co-existing patterns:

1. **Self-contained**: Repository opens, defers rollback, commits
2. **Caller-injected**: `BeginTx()` + methods accept `pgx.Tx` parameter (for multi-repository transactions in service layer)

### Cross-Module Access

Uses port/provider interfaces wired via setter injection (`SetProductNameProvider`, `SetLocationRackProvider`, etc.). Cross-module reads never bypass the architectural boundary — enforced by AST-based architecture tests (`archtest_test.go`).

### Testing

Dual strategy:

- **Integration tests**: Real PostgreSQL via `pgxpool`, `TestMain` + `TruncateTestData` for setup
- **Mock tests**: `pgxmock/v4` for error branches, cache behavior, transaction failures
- No per-test transaction rollback isolation; relies on truncation + unique data generation

### Migrations

Simple file-based runner tracking `schema_migrations`. Baseline squash (`000_squash.sql`, 1615 lines) + 16 incremental migrations. No migration tool (no golang-migrate, no goose).

## PostgreSQL Feature Usage

The codebase makes heavy, idiomatic use of PostgreSQL-specific features:

| Feature | Usage | Locations |
|---------|-------|-----------|
| **Full-text search** | `tsvector`, `tsquery`, `plainto_tsquery`, GIN index, weighted columns | `000_squash.sql`, `product/query.go` |
| **pg_trgm** | Extension + 3 GIN trigram indexes | `000_squash.sql` |
| **JSONB** | `jsonb_agg(jsonb_build_object(...))` for inline child reads | `sale/repository.go` |
| **CTEs** | 5+ non-recursive CTEs for dashboards, shift summaries | `report/`, `sale/`, `user/` |
| **Recursive CTEs** | Org chart traversal, manager chain checks | `user/repository.go` |
| **Window functions** | `ROW_NUMBER() OVER` | migrations |
| **RETURNING** | 100+ occurrences, including `xmax = 0` trick | Throughout |
| **ON CONFLICT** | 77+ occurrences, including constraint-based upserts | Throughout |
| **FOR UPDATE** | 21+ occurrences, including `FOR UPDATE OF`, `SKIP LOCKED` implied | `inventory/`, `shift/`, `sale/`, `stockopname/`, `consignment/`, `purchase/` |
| **Partial indexes** | 9+ partial indexes (WHERE clauses) | `000_squash.sql` |
| **Covering indexes** | 3 INCLUDE indexes | `000_squash.sql` |
| **Array operations** | `ARRAY_AGG ... FILTER`, `ANY($1)` for slice parameters | `user/`, `store/`, `pricing/`, `inventory/` |
| **LATERAL joins** | In `v_products_full` view | `000_squash.sql` |
| **Materialized views** | 3 views with `REFRESH CONCURRENTLY` | `000_squash.sql`, `report/` |
| **generate_series** | Date series for daily charts | `report/repository.go` |
| **COPY protocol** | `pgx.CopyFrom` for 9 table targets | `sale/`, `purchase/`, `stockopname/`, `inventory/`, `user/`, `supplier/` |

**Key finding**: The vast majority of these features would require raw SQL escape hatches in any ORM. The `RETURNING (xmax = 0) AS is_insert` trick, `CopyFrom` for bulk writes, recursive CTEs for org charts, and `jsonb_agg` for inline child reads are all deeply idiomatic PostgreSQL patterns that ORMs either don't support or support poorly.

## PgBouncer Compatibility

Currently no PgBouncer is configured. The application relies on pgx's native connection pooling.

If PgBouncer with transaction pooling were introduced:

| Concern | Impact | Recommendation |
|---------|--------|----------------|
| **Prepared statements** | PgBouncer transaction pooling does not support prepared statements across transactions | Disable pgx statement caching or use `DefaultQueryExecMode = QueryExecModeSimpleProtocol` |
| **Session state** | `SET`, `LISTEN/NOTIFY`, advisory locks lost between transactions | Avoid session-state features (none currently used) |
| **`pgxpool` config** | `poolCfg.ConnConfig.DefaultQueryExecMode` should be `pgx.QueryExecModeSimpleProtocol` | Required for PgBouncer transaction mode |
| **Connection lifetime** | PgBouncer manages its own pool; pgxpool settings become secondary | Reduce pgxpool `MaxConns` to match PgBouncer `default_pool_size` |
| **Health checks** | PgBouncer has its own `server_check_query` | Keep pgxpool `HealthCheckPeriod` for local pool health |

If adding PgBouncer, configure pgx to use simple query protocol:

```go
poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
poolCfg.ConnConfig.PreferSimpleProtocol = true
```

### Dummy Seeder Incompatibility

The dummy seeder (`cmd/dummy/`) uses `lib/pq` via `database/sql` (not pgx) and is incompatible with PgBouncer transaction pooling due to:

1. **Prepared statements** — `tx.PrepareContext()` creates server-side prepared statements that PgBouncer cannot route across transactions
2. **`SET session_replication_role`** — Session-level state that would reset between transactions
3. **Commit cadence** — Sales worker commits every 500 rows, re-prepares statements each batch

**Resolution:** Bypass PgBouncer for the seeder. It is a batch tool for local development, not a production service. Connect directly to PostgreSQL:

```
Development:
  Seeder → PostgreSQL (port 5433)
  App    → PostgreSQL (port 5433)

Production:
  App → PgBouncer (port 6432) → PostgreSQL (port 5432)
```

See `docs/design/pgbouncer-implementation-plan.md` for full implementation details.

This is compatible with the current codebase because all queries use positional parameters (`$1..$N`) which work with simple protocol. The `CopyFrom` bulk path uses the PostgreSQL COPY protocol which bypasses query parsing entirely.

## Architecture Evaluation

### Persistence Boundary

The current architecture is clean and well-structured:

```
Handler (HTTP)
  ↓
Application Service (business logic, orchestration)
  ↓
Repository (concrete struct, shared.DBPool)
  ↓
Persistence (raw SQL, pgx native API)
  ↓
PostgreSQL
```

Transaction ownership is split:

- Simple operations: Repository owns transactions (self-contained)
- Complex cross-module operations: Service owns transactions via `BeginTx()` + injected `pgx.Tx`

This is a sound pattern. The service layer controls transaction scope for multi-repository operations, while repositories can still self-contain simple transactions.

### ORM Model Exposure

ORM models should not be exposed outside the persistence layer. The current design already keeps domain entities (`domain.go` structs) separate from database concerns. Introducing ORM models would create a third layer (ORM model ↔ domain model ↔ API model) that adds complexity without benefit.

## Comparison

| Criteria | Current (raw pgx) | sqlc + pgx | Bun + pgx | GORM | Ent |
|----------|-------------------|------------|-----------|------|-----|
| **Productivity** | Good | Best | Good | High for CRUD, poor for complex | Medium |
| **Type safety** | Manual | Best (compile-time) | Good | Medium (reflection) | Good |
| **Complex SQL** | Excellent | Excellent | Poor | Poor | Poor |
| **PostgreSQL features** | Excellent | Excellent | Poor-Medium | Poor | Poor |
| **FTS** | Excellent | Excellent | Poor | Poor | Poor |
| **Transactions** | Excellent | Excellent | Good | Medium | Medium |
| **PgBouncer** | Excellent | Excellent | Good | Poor | Poor |
| **Performance** | Excellent | Excellent | Good | Poor | Good |
| **Maintainability** | Good | Excellent | Good | Medium | Medium |
| **Complexity** | Low | Low-Medium | Low | Medium | High |
| **Portability** | PostgreSQL-only | PostgreSQL-only | Agnostic (theoretically) | Agnostic | Agnostic |
| **Learning curve** | Low | Low | Low | Medium | High |
| **Migration effort** | None | Medium | High | Very High | Very High |

### Why Current Approach Works

1. The codebase is already well-architected: clean module boundaries, port/provider decoupling, composition root, architecture tests
2. PostgreSQL feature usage is deep: FTS, JSONB, CTEs, recursive CTEs, COPY, FOR UPDATE OF, materialized views — all would require raw SQL escape hatches in any ORM
3. The `CopyFrom` bulk path: 9 table targets use pgx's COPY protocol for performance. No ORM supports this
4. The scan boilerplate is manageable: dedicated `scanXxx()` functions with `rowScanner` interface are clear and testable
5. Testing is already solid: pgxmock integration + real DB tests
6. The `shared.DBPool` interface is minimal and correct: 4 methods that cover all needs

### Why sqlc + pgx Is the Best Hybrid (If Any Change Is Made)

sqlc generates type-safe Go code from SQL queries. It would:

- Eliminate scan boilerplate (auto-generated scan functions)
- Keep all SQL explicit (no magic, no reflection)
- Maintain full PostgreSQL feature support (you write the SQL, sqlc generates Go)
- Integrate seamlessly with pgx (same driver, same connection pool)
- Work with PgBouncer (same underlying driver)
- Allow incremental adoption (one query at a time, no big-bang rewrite)

However, the current approach already works well and the marginal benefit of sqlc does not justify the migration effort for a mature codebase.

### Why Other Approaches Are Not Preferred

- **Bun**: Would lose `CopyFrom`, recursive CTEs, `RETURNING (xmax = 0)`, `ARRAY_AGG ... FILTER`, and most PostgreSQL-specific features. Would require raw SQL escape hatches for 60%+ of queries.
- **GORM**: Reflection-based, N+1 risk, poor PostgreSQL feature support, poor PgBouncer compatibility (uses `database/sql`), would require complete rewrite.
- **Ent**: Different query model (graph traversal), steep learning curve, poor PostgreSQL feature support, complete rewrite required.

## Recommendation

### Keep the Current Approach

The current raw pgx architecture is the right choice for this codebase.

### Optional Incremental Improvement: sqlc

If the team wants to reduce scan boilerplate without changing the architecture:

1. **Phase 1**: Set up sqlc configuration, generate for 1 simple module (e.g., `uom`)
2. **Phase 2**: Migrate simple CRUD modules (`brand`, `category`, `customer`, `customergroup`, `store`)
3. **Phase 3**: Migrate complex modules (`product`, `sale`, `inventory`, `purchase`)
4. **Phase 4**: Migrate report module (CTE-heavy queries)
5. **Never migrate**: `CopyFrom` bulk writes (keep as raw pgx calls)

Each phase is independent and can be validated with existing tests.

### PgBouncer Configuration

If PgBouncer is added in production:

```go
poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
poolCfg.ConnConfig.PreferSimpleProtocol = true
```

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Scan boilerplate grows | Accept as explicit cost; consider sqlc incrementally |
| New developers unfamiliar with pgx | Document patterns in AGENTS.md; `scanXxx()` pattern is consistent |
| No database portability | Not a concern for a PostgreSQL-specific POS system |
| pgxmock maintenance | pgxmock tracks pgx releases; pin version in go.mod |
| PgBouncer compatibility | Use simple query protocol; test with PgBouncer in staging |

## DECISION

```
ORM:
No

Recommended approach:
Raw pgx/v5 (current) with optional incremental sqlc adoption for scan boilerplate reduction

Driver:
jackc/pgx/v5 v5.9.2

Database:
PostgreSQL

PgBouncer:
Compatible with transaction pooling (use QueryExecModeSimpleProtocol); not currently deployed

Primary reason:
The codebase uses 14+ PostgreSQL-specific features (FTS, JSONB aggregation, recursive CTEs, CopyFrom bulk writes, FOR UPDATE OF, materialized views, partial indexes, ARRAY_AGG FILTER) that would require raw SQL escape hatches in any ORM. The current architecture is already well-structured with clean module boundaries, port/provider decoupling, and a composition root. Adding an ORM would increase complexity without meaningful benefit.

Migration:
None

Confidence:
High
```
