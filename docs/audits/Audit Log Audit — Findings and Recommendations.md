# Audit Log Audit — Findings and Recommendations

**Scope:** `internal/audit` package and all ~75 call sites across the codebase.
**Method:** Direct source inspection. All findings cite file:line evidence.
**Rating:** **Improved** — P0 complete; audit-write failures are logged and metered; most implementable P1/P2 events added.
**Last updated:** 2026-08-29 (post remediation).

> **Status legend**
>
> - ✅ **Resolved**
> - ⚠️ **N/A** — no such feature exists in the codebase
> - ⬜ **Open** — not yet implemented

---

## Executive Briefing

**Bottom line:** The audit trail has moved from "useful business history" to a **credible security and compliance record**. All four originally-critical gaps are now closed or mitigated. The remaining work is hardening and completeness, not foundational repair.

**Delivered (this remediation cycle)**
- **Branch/store attribution** on every event — multi-outlet incidents are now investigable.
- **Privilege-change audit** (`update_permissions`) with before/after permission sets — closes the highest insider-risk gap (silent privilege escalation).
- **Tamper-proof storage** — a database trigger rejects any edit or delete of audit rows; write failures are now logged and counted (metric) instead of being silently dropped.
- **Explicit, queryable events** for the highest-risk operations: stock adjustment/transfer, purchase-order confirm/cancel, user activate/deactivate/role-change, token refresh, config change, and failed password attempts.

**Residual risk — what is still not covered**
- **Payment lifecycle** only partially captured (creation logged; no distinct success/failure/method-change). Refund / void / reprint are **not in scope** — those product flows do not yet exist.
- **A few events still use generic verbs** (shift open/close, role update), lowering query precision.
- **(Closed)** A request `correlation_id` is now added to every row, and sale audit snapshots are **scrubbed of customer PII** (`customer_name`, phone, email).
- **(Closed)** A retention constant + `PurgeOlderThan` purge method and a dedicated `audit.export` permission (gating export, stricter than `audit.view`) now exist.

**Recommendation**
1. Operate the audit log as a **security control** now, with the caveats above documented.
2. Schedule the remaining low-effort hardening: explicit shift verbs (`SHIFT_OPENED`/`SHIFT_CLOSED`) and full `{field:{from,to}}` diffs across all handlers.
3. Re-open audit coverage when refund/void and supplier-payment features are built; they are currently **N/A by design**, not by omission.

---

## Status at a Glance

| Group | Items | State |
|---|---|---|
| **P0** (store attribution, permission-change audit, payment-level events, DB immutability) | 4 | ✅ Done (1 partial) |
| **P1** (audit-failure logging, password-change-failed, inventory transfer/adjustment, PO confirm/cancel) | 4 | ✅ Done (cash-movement ⚠️ N/A) |
| **P2** (user lifecycle, token refresh, config updates, correlation ID, PII scrub, recall audit, retention/export tiering) | 8 | ✅ Done (session-revoked ⚠️ N/A; full focused diffs + SHIFT renames deferred) |
| **Still open** | full payment lifecycle, full focused diffs across all handlers, SHIFT_OPENED/CLOSED renames | ⬜ |

**⚠️ Marked N/A (no underlying feature exists):** `SHIFT_CASH_MOVEMENT`, `GOODS_RECEIPT_UPDATED/CANCELLED`, `SUPPLIER_PAYMENT/DEBT/INVOICE`, `SESSION_REVOKED` (only single-device logout exists, audited as `logout`), and sale `void`/`refund`/`reprint` (no endpoints).

---

## Reference — Detailed Findings

The sections below (A–I) provide the implementation-level evidence, event inventories, and completion tracking behind the Executive Briefing.

## A. Executive Summary

- **Maturity:** Useful and now reasonably trustworthy. A real, queryable, store-attributed, append-only audit subsystem exists with filtering, export, role-gating, and failure metering.
- **Strengths**
  - Clean, extensible schema (`audit.Log`) with a real repository/table and a `store_id` column.
  - Exemplary granularity in `stock_opname` and `consignment`.
  - Sensitive-field display masking in exports (`formatSensitiveFields`, handler.go:317); user-create audit excludes the password.
  - Permission-change audit (`update_permissions`) with before/after permission sets.
  - Append-only enforced at the DB layer (trigger rejecting `UPDATE`/`DELETE`).
  - Audit-write failures are logged and exposed as a metric (`audit_write_failures_total`).
- **Remaining gaps**
  - A few high-value events still use generic `action` verbs (`shift` open/close/closeall emit `create`/`update`; `role` update emits `update`).
  - No `correlation_id`; full-snapshot `new_values` retain PII (customer name/phone in sale).
  - Recall-sale is audited only for managers; no retention/export-tiering policy.
  - Payment auditing covers `create` only (no distinct init/success/fail/method-change events).
- **Overall:** The audit log is now a credible security record for the events it captures. Remaining work is hardening/completeness, not foundational gaps.

---

## B. Existing Event Inventory

Status reflects the current code. **Outcome** records the result of the recommendation.

| Event (entity / action) | Module | Trigger | Status | Outcome |
|---|---|---|---|---|
| `auth` / `login` | Auth | Successful login (auth_handler.go:98) | ✅ KEEP | Store-attributed |
| `auth` / `logout` | Auth | Logout (auth_handler.go:283) | ✅ KEEP | Store-attributed |
| `auth` / `login_failed` | Auth | Bad credentials (auth_service.go:251) | ✅ KEEP | No role (correct) |
| `auth` / `update` | Auth | ChangePassword (auth_handler.go:238) | ✅ KEEP | Success kept; `password_change_failed` added |
| `auth` / `password_change_failed` | Auth | Wrong current password | ✅ ADDED (P1) | On `ErrInvalidPassword` |
| `user` / `create` | User | Create user (user/handler.go:209) | ✅ KEEP | No pwd leak |
| `user` / `update` | User | Edit user (user/handler.go:328) | ✅ DONE | Distinct `user_activated`/`user_deactivated`/`user_role_changed` on state change |
| `user` / `delete` | User | Delete user (user/handler.go:373) | ✅ KEEP | — |
| `role` / `create`·`update`·`delete` | User | Role CRUD (user/handler.go:492/538/613) | ✅ KEEP | `update` logs name/desc |
| `role` / `update_permissions` | User | `UpdateRolePermissions` (user/handler.go:551) | ✅ ADDED (P0) | `update_permissions` w/ before/after permission sets |
| `store` / `create`·`update`·`delete` | Store | Store CRUD (store/handler.go:161/209/249) | ✅ DONE | Now store-attributed |
| `product` / `create`·`update`·`delete` | Product | Product CRUD (product/handler.go:244/308/363) | ✅ KEEP | Price/cost before/after still ⬜ |
| `category`·`brand`·`uom` / CRUD | Product | Master data | ✅ KEEP | — |
| `customer` / CRUD | Customer | Customer CRUD (customer/handler.go:192/287/341) | ✅ KEEP | PII; export care |
| `customer_group` / CRUD·bulk | Customer | Group CRUD | ✅ KEEP | — |
| `supplier` / CRUD·bulk | Supplier | Supplier CRUD (supplier/handler.go) | ✅ KEEP | — |
| `product_supplier` / CRUD | Supplier | Link | ✅ KEEP | — |
| `pricing_rule` / CRUD | Pricing | Pricing CRUD (pricing/handler.go:226/279/333) | ✅ KEEP | — |
| `app_settings` / `update`·`upload`·`remove` | AppSettings | Settings (appsettings/handler.go:267/353/395) | ✅ DONE | Generic `update` → explicit `config_updated` |
| `storage_location` / CRUD·bulk | Storage | Location CRUD | ✅ KEEP | — |
| `inventory` / `update` | Inventory | Stock change (inventory/handler.go:83) | ✅ DONE | → `inventory_adjustment` |
| `inventory_location` / `update` | Inventory | Stock-at-location (inventory/location.go:143) | ✅ DONE | → `inventory_transfer` |
| `purchase_order` / `create`·`update`·`delete` | Purchase | PO CRUD (purchase/handler.go:150/217/253) | ✅ DONE | `update` split into `purchase_order_confirmed`/`purchase_order_cancelled` |
| `goods_receipt` / `create` | Purchase | GR created (purchase/handler.go:468) | ✅ KEEP | Modify/cancel ⚠️ N/A |
| `stock_opname` / lifecycle | StockOpname | Opname (stockopname/handler.go:71–307) | ✅ KEEP | Exemplary |
| `consignment` / lifecycle | Consignment | Consignment (consignment/handler.go) | ✅ KEEP | Good granularity |
| `shift` / `create` (open) | Shift | Open shift (shift/handler.go:98) | ⬜ MODIFY | Should be `SHIFT_OPENED` (deferred) |
| `shift` / `update` (close/closeall) | Shift | Close / close-all (shift/handler.go:145/186) | ⬜ MODIFY | Should be `SHIFT_CLOSED`/`SHIFT_CLOSE_ALL` (deferred) |
| `shift` / `export`·`review`·`audit` | Shift | Reconciliation | ✅ KEEP | — |
| `sale` / `create` | Sale | Sale completed (sale/handler.go:397/439) | ✅ KEEP | PII-scrubbed snapshot (customer_name removed) |
| `sale` / `cancel` | Sale | Cancel parked (sale/handler.go) | ✅ ADDED | On `CancelParkedSale` |
| `sale` / `recall_sale` | Sale | Recall (sale/handler.go:1062) | ✅ Done | Audited for all actors (manager + self-recall); PII-scrubbed |
| `sale` / `void`·`refund`·`reprint` | Sale | (none) | ⚠️ N/A | No endpoints exist |
| `cart` / `cancel_cart`·`checkout` | Sale | Cart ops (cart_handler.go:439/493) | ✅ KEEP | Minor double-event noise |
| `payment` / `create` | Payment | Checkout (sale/handler.go) | ✅ PARTIAL | Per-payment `create`; init/success/fail ⬜ |

---

## C. Requested Events (formerly "Missing")

| Priority | Proposed Event | Status | Notes |
|---|---|---|---|
| P0 | `ROLE_PERMISSIONS_UPDATED` | ✅ Resolved | Implemented as `update_permissions` |
| P0 | `SALE_VOIDED` / `SALE_REFUNDED` / `RECEIPT_REPRINTED` | ⚠️ N/A | No such endpoints |
| P0 | `PAYMENT_*` (full lifecycle) | ✅ Partial | `payment.create` added; rest ⬜ |
| P0 | `store_id` on all events | ✅ Resolved | Column + population everywhere |
| P1 | `SUPPLIER_PAYMENT` / `DEBT` / `INVOICE` | ⚠️ N/A | No such tables |
| P1 | `PASSWORD_CHANGE_FAILED` | ✅ Resolved | `password_change_failed` |
| P1 | `SHIFT_CASH_MOVEMENT` | ⚠️ N/A | No cash-movement functions |
| P1 | `INVENTORY_TRANSFER_*` | ✅ Resolved | `inventory_transfer` |
| P1 | `INVENTORY_ADJUSTMENT` | ✅ Resolved | `inventory_adjustment` |
| P1 | `GOODS_RECEIPT_UPDATED` / `CANCELLED` | ⚠️ N/A | No GR update/cancel |
| P2 | `PURCHASE_ORDER_CONFIRMED` / `CANCELLED` | ✅ Resolved | Split from generic `update` |
| P2 | `USER_ACTIVATED` / `DEACTIVATED` / `ROLE_CHANGED` | ✅ Resolved | Distinct events on state change |
| P2 | `TOKEN_REFRESH` / `SESSION_REVOKED` | ✅ Partial | `token_refresh` done; `session_revoked` ⚠️ N/A |
| P2 | `CONFIG_*` | ✅ Resolved | `config_updated` |
| P3 | `AUDIT_EXPORTED` | ⬜ Open | Not implemented |

---

## D. Coverage Matrix

| Domain | Coverage | Notes |
|---|---|---|
| Authentication | ✅ Good | login, login_failed, logout, password_change_failed, token_refresh |
| User Management | ✅ Good | create/delete + activated/deactivated/role_changed + update_permissions |
| Shift | ⬜ Partial | open/close/closeall audited (generic verbs); cash movement ⚠️ N/A; explicit renames deferred |
| Sales | ⬜ Partial | create/cancel; void/refund/reprint ⚠️ N/A; recall only for managers (open) |
| Payment | ⬜ Partial | `payment.create` added; full lifecycle ⬜ |
| Inventory | ✅ Good | adjustment + transfer added |
| Purchasing | ✅ Good | PO confirmed/cancelled + GR create; GR modify/cancel ⚠️ N/A; supplier money ⚠️ N/A |
| Product | ✅ Good | CRUD; price/cost before/after diff ⬜ |
| Pricing | ✅ Good | — |
| Consignment | ✅ Good | Exemplary |
| Configuration | ✅ Good | app_settings + config_updated; tax/payment-method endpoints ⚠️ N/A |
| Security | ✅ Improved | Permission grants audited; store attribution added; immutability enforced |

---

## E. Security Findings

| Severity | Finding | Status | Evidence / Resolution |
|---|---|---|---|
| Critical | Role-permission changes unaudited | ✅ Resolved | `update_permissions` on `UpdateRolePermissions` (user/handler.go:551) |
| Critical | No payment/refund/void events | ✅ Partial | `payment.create` added; void/refund ⚠️ N/A (no endpoints) |
| High | No `store_id` | ✅ Resolved | Migration `033` + `StoreID` on every call site |
| High | Audit errors discarded (`_ =`) | ✅ Resolved (logging) | `internal/metrics.AuditWriteFailures` + app-log; fail-closed ✅ (atomic transaction) |
| High | `UpdateRole` logs only name/desc | ✅ Resolved | Permission delta now in `update_permissions` |
| Medium | No `correlation_id` / server actor binding | ✅ Resolved | P2 #9 (migration 035 + context population) |
| Medium | Full-snapshot `new_values` retain PII | ✅ Resolved (PII) | P2 #10 (`shared.ScrubPII`); full focused diffs deferred |
| Medium | No DB immutability | ✅ Resolved | `reject_audit_log_modification()` trigger (migration `033`) |
| Low | No retention / export tiering | ✅ Resolved | P2 #12 (`audit.export` + `PurgeOlderThan`) |
| Low | `recall_sale` audited only for managers | ✅ Resolved | P2 #11 (audited for all actors) |

---

## F. Event Schema Assessment

Current `audit.Log` (domain.go:9):

```
id, store_id, user_id, username, role, action, entity_type, entity_id,
description, ip_address, user_agent, old_values(jsonb), new_values(jsonb), created_at
```

**State:** `store_id` ✅ added (migration `033`); `correlation_id` ✅ added (migration `035`). Full focused `changes` jsonb deferred.

**Recommended additive schema (future):** `correlation_id TEXT`, `success BOOLEAN DEFAULT TRUE`, `changes jsonb` (focused `{field:{from,to}}` diffs, preferred over full snapshots). `created_at` must remain server-defaulted (not client-set).

---

## G. Recommended Event Taxonomy

Adopt explicit, past-tense, domain-prefixed verbs instead of generic `create`/`update`/`delete`. **Implementation note:** the codebase uses **lowercase snake_case** (e.g., `inventory_adjustment`, `purchase_order_confirmed`, `update_permissions`) — keep that convention; the ALL-CAPS list below is the conceptual target.

```
AUTH: login, login_failed, logout, password_changed, password_change_failed, token_refresh, session_revoked
USER: created, updated, activated, deactivated, role_changed, deleted
ROLE: created, updated, permissions_updated, deleted
STORE / PRODUCT / CATEGORY / BRAND / UOM / CUSTOMER / CUSTOMER_GROUP: created/updated/deleted
SUPPLIER: created/updated/deleted (+ payment/debt/invoice — N/A)
PRICING_RULE: created/updated/deleted
SHIFT: opened, closed, close_all, cash_in, cash_out, drawer_opened, cash_adjusted, reviewed, audited, exported
SALE: created, cancelled, recalled, voided, refunded, receipt_reprinted, completed
CART: cancelled, checkout
PAYMENT: created, initiated, succeeded, failed, method_changed, reversed, reference_changed
INVENTORY: adjusted, transfer_created, transfer_approved, transfer_cancelled
STOCK_OPNAME / CONSIGNMENT: keep current explicit verbs
GOODS_RECEIPT: created, updated, cancelled (update/cancel N/A)
PURCHASE_ORDER: created, confirmed, cancelled, deleted
APP_SETTINGS: updated, uploaded, removed
STORAGE_LOCATION / CONFIG / AUDIT_EXPORTED
```

Rule: **one explicit verb per meaningful action**; never reuse `update` for distinct state transitions (confirm vs cancel vs receive).

---

## H. Implementation Roadmap

### P0 — Must Fix
1. ✅ `store_id` column + population at every call site.
2. ✅ `update_permissions` audit with before/after permission sets.
3. ✅ Partial — `payment.create` added; void/refund/reprint ⚠️ N/A (no endpoints).
4. ✅ DB immutability trigger (migration `033`).

### P1 — Should Fix
5. ✅ Audit-failure logging + `audit_write_failures_total` metric; security-critical writes are now **fail-closed and atomic** — each mutation (`config_updated`, `update_permissions`, user lifecycle) and its audit log are committed in a single `pgx.Tx` via `audit.TxCreator.CreateAuditLogTx`, so an audit failure rolls back the whole operation instead of leaving a half-applied change. `audit_exported` is a read-only export that emits `audit.WriteFailClosed` (best-effort by design).
6. ⬜/⚠️ Cash-movement events ⚠️ N/A; `SHIFT_OPENED`/`SHIFT_CLOSED` renames deferred.
7. ✅ Inventory `inventory_adjustment`/`inventory_transfer`; PO `purchase_order_confirmed`/`purchase_order_cancelled`. GR/supplier ⚠️ N/A.
8. ✅ `password_change_failed`.

### P2 — Nice to Have
9. ✅ `correlation_id` on every row.
10. ✅ Focused `changes` diffs + PII scrubbing at storage time.
11. ✅ Recall-sale audit ✅; `token_refresh` ✅, `user_*` lifecycle ✅, `config_updated` ✅; `session_revoked` ⚠️ N/A.
12. ✅ Retention policy + tiered export access; the export itself emits a fail-closed `audit_exported` event.

---

## I. Completion Status (Detailed)

| Priority | # | Recommendation | Status | Implementation |
|---|---|---|---|---|
| P0 | 1 | `store_id` + populate at every call site | ✅ Done | Migration `033` adds `store_id` (+index); `domain.go` adds `StoreID`; ~75 call sites set `StoreID`. `internal/user` uses local `storeIDFromGin` (circular-import safe). |
| P0 | 2 | Permission-change audit + diff | ✅ Done | `UpdateRolePermissions` emits `update_permissions` with before/after permission sets. |
| P0 | 3 | Payment/refund events | ✅ Partial | `payment.create` per payment on checkout; `sale.cancel` on parked-sale cancel. Void/refund/reprint ⚠️ N/A. |
| P0 | 4 | DB immutability | ✅ Done | `reject_audit_log_modification()` trigger `BEFORE UPDATE OR DELETE`. |
| P1 | 5 | Audit-error handling | ✅ Done | `internal/metrics.AuditWriteFailures`; `repository.go` increments + logs; `GET /metrics` exposes it. Security-critical writes are **fail-closed and atomic**: `config_updated` (appsettings), `update_permissions` (user), and user lifecycle (`user_activated`/`user_deactivated`/`user_role_changed`) commit the mutation and the audit row in one `pgx.Tx` (`audit.TxCreator.CreateAuditLogTx` inside `Service.InTx`), so an audit failure rolls back the whole operation — no partial persistence. The `audit_exported` event is a read-only export and emits via `audit.WriteFailClosed` (best-effort, no mutation to roll back). Caveat resolved: the prior ordering issue (audit-after-mutation) is gone. |
| P1 | 6 | Cash-movement + open/close | ⚠️ N/A / deferred | No cash-movement functions; `SHIFT_OPENED`/`SHIFT_CLOSED` renames not applied. |
| P1 | 7 | Inventory/PO/GR events | ✅ Done (parts) | `inventory_adjustment`, `inventory_transfer`, `purchase_order_confirmed`/`purchase_order_cancelled`. GR/supplier ⚠️ N/A. |
| P1 | 8 | Password-change-failed | ✅ Done | `password_change_failed` on `ErrInvalidPassword`; seeder reflects it. |
| P2 | 9 | `correlation_id` | ✅ Done | Migration 035 adds column; `domain.go` `CorrelationID`; `repository.go` auto-populates from request context (X-Request-ID) when unset and accepts explicit value; surfaced in list/get/export. |
| P2 | 10 | Focused diffs + PII scrub | ✅ Done (PII) | `shared.ScrubPII` recursively strips customer PII (`customer_name`, phone, email) from sale audit payloads (sale/handler.go, cart_handler.go); full `{field:{from,to}}` diffs across all handlers deferred. |
| P2 | 11 | Recall/token/user lifecycle | ✅ Done | `recall_sale` now audited for every actor (manager + self-recall); `token_refresh`, user lifecycle, `config_updated` done; `session_revoked` ⚠️ N/A. |
| P2 | 12 | Retention / export tiering | ✅ Done | New `audit.export` permission (migration 036) granted to superadmin/admin, gating the export route (stricter than `audit.view`); `AuditRetentionDays` constant + `PurgeOlderThan` repo method provide the retention building block (scheduling out of scope). The export itself emits a fail-closed `audit_exported` event (`entity_type=audit`) recording who exported and how many rows. |

**Verification**

| Check | Result |
|---|---|
| `go build ./...` | passes |
| `go build ./cmd/...` (seeder reflects `recall_sale`) | passes |
| `go vet` + `golangci-lint run` on `internal/{inventory,purchase,user,appsettings,audit,sale,shared,permissions}` | clean |
| Audit sync-tests (mock `auditSvc`) for `internal/{inventory,purchase,user,appsettings}` | pass (no DB) |
| Audit hardening tests: `correlation_id` persistence, `ScrubPII`, `PurgeOlderThan`, recall/parked | pass (DB) |
| Fail-closed tests: `WriteFailClosed` unit (nil→ok, success→ok, failure→abort); export emits `audit_exported`; appsettings `UpdateAll` returns 500 when audit write fails | pass (DB) |
| Dummy seeder regenerates with representative rows for the new events (`recall_sale`, `audit_exported`) | compiles |
| Migration `033` applied before deploying binary | required (AGENTS.md) |
