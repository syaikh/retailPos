# Transactional Outbox for Audit Delivery — Design (DEFERRED)

**Status:** ⬜ Deferred — not implemented. Kept for future use.
**Date:** 2026-08-30
**Owner:** backend

## Why it was deferred

The current audit design already writes the durable audit row into the **same
Postgres database as the business mutation** (`audit.NewRepository(p.DB)` in
`internal/wiring/wiring.go`, and `CreateAuditLogTx` runs inside the same
`pgx.Tx` as the sale/PO/inventory mutation). Because the audit store is the
same database the business already depends on, there is **no separate on-the-wire
SPOF** today. The transactional outbox only adds value when audit records must be
delivered to *external* consumers (SIEM, dashboards, a separate audit store)
that do not exist in this codebase yet. Since there are no external audit sinks
today, the outbox is deferred until one is introduced.

## Goal

Make external audit delivery **never block a transaction**, while preserving the
fail-closed "no sale without a durable audit record" guarantee.

Two facts we must keep straight:

1. **Durable accountability** must be atomic with the business mutation.
2. **External fan-out** (SIEM / dashboards / exports) must be asynchronous and
   best-effort — never able to roll back a sale.

The outbox separates exactly these two concerns.

## Current behavior (reference)

- `internal/audit/domain.go` — `TxCreator` (`Creator` + `CreateAuditLogTx`).
- `internal/audit/repository.go` — `CreateAuditLogTx(ctx, tx, log)` persists into
  the same DB transaction.
- Sale / purchase / inventory handlers wrap `mutation + audit` in one `InTx(...)`
  and commit together; `Notify*` methods publish events **post-commit**.

## Design

```
[Sale / PO / Inventory handler]
        │  InTx(tx):
        │      business mutation (sale, stock, PO)
        │      + INSERT audit_row INTO audit_outbox   (same DB, atomic)
        ▼
    tx commit  ──────────────────►  durable audit survives (no separate SPOF)
        │
        ▼  (post-commit, best-effort)
[Outbox worker goroutine]
        │  polls pending rows (status=new, attempts < N)
        │  ships each payload to external sinks (SIEM/dashboard/export)
        │  on success: delete row (or mark status=done)
        │  on failure: status=failed, attempts+1, next_retry = now + backoff
        ▼
   never affects the originating sale/PO/inventory
```

### Key properties

- **Atomic durability:** the `audit_outbox` row is written in the same
  transaction as the mutation, so the record and the action survive or roll back
  together. This keeps the fail-closed contract with the *same* SPOF as the
  sale itself (the DB).
- **Async fan-out:** external delivery happens after commit in the worker.
  Outages there merely accumulate retryable rows; they never block a sale.
- **Guaranteed eventual delivery:** rows are retained until the worker confirms
  success to each sink, or until a bounded retry budget is exhausted (then it
  surfaces an alert instead of silently dropping).

### Proposed schema

```sql
CREATE TABLE IF NOT EXISTS audit_outbox (
    id          BIGSERIAL PRIMARY KEY,
    action      varchar(100) NOT NULL,
    entity_type varchar(100),
    entity_id   integer,
    payload     jsonb NOT NULL,          -- the audit.Log as JSON
    status      varchar(20) NOT NULL DEFAULT 'new',   -- new|done|failed
    attempts    integer NOT NULL DEFAULT 0,
    next_retry  timestamptz NOT NULL DEFAULT now(),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_outbox_pending
    ON audit_outbox (next_retry) WHERE status = 'new';
```

### Worker (mirrors `report.RefreshCoordinator` pattern)

- A single goroutine polling `WHERE status = 'new' AND next_retry <= now()`
  with a small interval.
- Backoff on failure (`attempts`-based exponential), never fails the originating
  mutation and never triggers event-bus retries.
- On success, delete the row (keeps the table small).

### Integration points

- New handler/repository in `internal/audit` for the outbox table + worker.
- `cmd/server/main.go` starts the worker (like `RefreshCoordinator`).
- Optionally, `CreateAuditLogTx` can write to both `audit_logs` and
  `audit_outbox` inside the same tx, or `audit_logs` itself becomes the durable
  local record and the outbox only carries the fan-out copy.

## Trade-offs / consequences

- **More moving parts:** one table, one worker, one retry policy — real
  operational surface for a problem that does not exist yet.
- **Eventual consistency for external consumers:** dashboards/SIEM lag by the
  worker interval.
- **Do not use for local accountability:** the local `audit_logs` row should
  stay the source of truth; the outbox is only the fan-out mechanism.

## Decision

**Defer.** Implement only when an external audit consumer is introduced. If one
lands, the local durable `audit_logs` row stays the authority; the outbox worker
handles fan-out asynchronously so external delivery can never roll back a
sale/PO/inventory mutation.
