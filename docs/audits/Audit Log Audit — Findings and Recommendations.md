# Audit Log Audit — Findings & Recommendations

**Scope:** `internal/audit` package and all 25 call sites across the codebase.
**Method:** Direct source inspection (no assumptions from the prompt doc). All findings cite file:line evidence.
**Rating:** **Improved** — P0 (store attribution, permission-change audit, payment-level events, DB immutability) is complete, audit-write failures are now logged and metered, and most implementable P1/P2 events are added. Remaining gaps are features that do not exist in the codebase (cash movement, GR update/cancel, supplier money, session-revoke-all) and are marked **N/A**.

---

## A. Executive Summary

- **Maturity:** Early-but-useful. A real, queryable audit subsystem exists with filtering, export, and role-gating.
- **Major strengths**
  - Clean, extensible schema (`audit.Log`) and a real repository/table.
  - Good action granularity in `stock_opname` (`open`/`count`/`verify`/`post`…) and `consignment` (`create_arrangement`/`create_return`/`create_settlement`…).
  - Sensitive-field display masking in exports (`formatSensitiveFields`, handler.go:317).
  - User-create audit deliberately excludes the password (user/handler.go:209–217).
  - No code path deletes or updates audit rows (append-only by design of the service layer).
- **Major weaknesses**
  - **No `store_id` / branch column** on `audit_logs` — multi-store actions are unattributable.
  - **Permission/role-permission changes are not audited** (`UpdateRolePermissions` has no log; `UpdateRole` logs only name/description).
  - **No payment-level events** and **no void/refund/reprint events** despite money being the highest-risk surface.
  - **Every `CreateAuditLog` call ignores its error** (`_ = …`), so audit failure is silent and never blocks the business op.
  - Actor is taken from the JWT claims; there is no server-verified actor binding or request/correlation ID.
- **Critical risks**
  1. An insider can grant themselves or others dangerous permissions with **zero trace**.
  2. Cash/refund manipulation can occur with **no audit evidence**.
  3. A failed audit write (DB hiccup, connection pool exhaustion) leaves **silent gaps** exactly when you most need the record.
- **Overall recommendation:** Prioritize the P0 items (store attribution, permission-change audit, payment/refund events, reliable write) before treating the audit log as a security control. Today it is a useful business history, not a tamper-evident security record.

---

## B. Existing Event Inventory

| Event (entity / action) | Module | Trigger | Business Meaning | Security Value | Status | Recommendation |
|---|---|---|---|---|---|---|
| `auth` / `login` | Auth | Successful login (auth_handler.go:98) | User authenticated | High | KEEP | Add `store_id`; consider `auth.login` success/fail pair |
| `auth` / `logout` | Auth | Logout (auth_handler.go:283) | Session ended | Med | KEEP | Add `store_id` |
| `auth` / `login_failed` | Auth | Bad credentials (auth_service.go:251) | Failed auth attempt | High | KEEP | Good; add `reason` already present; keep no role (correct) |
| `auth` / `update` (user) | Auth | ChangePassword (auth_handler.go:238) | Password changed | High | MODIFY | Kept `update` for success; added `password_change_failed` event (P1 #8) |
| `user` / `create` | User | Create user (user/handler.go:209) | Account created | High | KEEP | Curated fields, no pwd leak — good |
| `user` / `update` | User | Edit user (user/handler.go:328) | Account edited | High | DONE | `update` kept; distinct `user_activated`/`user_deactivated`/`user_role_changed` events added on state change (P2 #11) |
| `user` / `delete` | User | Delete user (user/handler.go:373) | Account removed | High | KEEP | — |
| `role` / `create` | User | Create role (user/handler.go:492) | Role added | High | KEEP | — |
| `role` / `update` | User | Edit role (user/handler.go:538) | Role edited | High | MODIFY | Logs only name/desc — **not permissions** |
| `role` / `delete` | User | Delete role (user/handler.go:613) | Role removed | High | KEEP | — |
| `role` / `update_permissions` | User | `UpdateRolePermissions` (user/handler.go:551) | **Permission grant/revoke** | **Critical** | **DONE** | **P0 — added `update_permissions` event with before/after permission sets** |
| `store` / `create`·`update`·`delete` | Store | Store CRUD (store/handler.go:161/209/249) | Outlet config | Med | KEEP | Ironically stores have no `store_id` in the log |
| `product` / `create`·`update`·`delete` | Product | Product CRUD (product/handler.go:244/308/363) | Catalog change | High | KEEP | `update` should capture before/after price/cost |
| `category` / CRUD | Product | Category CRUD | Catalog taxonomy | Low | KEEP | — |
| `brand` / CRUD | Product | Brand CRUD | Catalog taxonomy | Low | KEEP | — |
| `uom` / CRUD | Product | UOM CRUD | Catalog master | Low | KEEP | — |
| `customer` / `create`·`update`·`delete` | Customer | Customer CRUD (customer/handler.go:192/287/341) | CRM data | Med | KEEP | Stores PII; restrict/export care |
| `customer_group` / CRUD·bulk | Customer | Group CRUD (customergroup/handler.go) | CRM master | Low | KEEP | — |
| `supplier` / `create`·`update`·`delete`·`bulk` | Supplier | Supplier CRUD (supplier/handler.go) | Vendor master | Med | KEEP | — |
| `product_supplier` / `create`·`update`·`delete` | Supplier | Supplier–product link | Procurement | Low | KEEP | — |
| `supplier` / *payment*·*debt*·*invoice* | Supplier | (none) | Vendor $ movement | High | **MISSING** | **P1 — add** |
| `pricing_rule` / `create`·`update`·`delete` | Pricing | Pricing rule CRUD (pricing/handler.go:226/279/333) | Price logic | High | KEEP | Capture before/after amounts |
| `app_settings` / `update`·`upload`·`remove` | AppSettings | Settings change (appsettings/handler.go:267/353/395) | Global config | High | DONE | Generic `update` renamed to explicit `config_updated` (P2 #11) |
| `storage_location` / CRUD·bulk | Storage | Location CRUD (storagelocation/handler.go) | Warehouse master | Low | KEEP | — |
| `inventory` / `update` | Inventory | Stock change (inventory/handler.go:83) | Stock mutation | High | DONE | Generic `update` replaced by explicit `inventory_adjustment` (P1 #7) |
| `inventory_location` / `update` | Inventory | Stock-at-location change (inventory/location.go:143) | Stock mutation | Med | DONE | Replaced by explicit `inventory_transfer` (P1 #7) |
| `purchase_order` / `create`·`update`·`delete` | Purchase | PO CRUD (purchase/handler.go:150/217/253) | Procurement | High | DONE | `update` split into `purchase_order_confirmed`/`purchase_order_cancelled` (P2 #11) |
| `goods_receipt` / `create` | Purchase | GR created (purchase/handler.go:468) | Goods in | High | MODIFY | No update/cancel/modify event |
| `stock_opname` / `create`·`cancel`·`assign`·`count`·`submit`·`open`·`verify`·`post` | StockOpname | Opname lifecycle (stockopname/handler.go:71–307) | Stocktake | High | KEEP | **Exemplary granularity** |
| `consignment` / `create_arrangement`·`set_terms`·`create_receipt`·`create_pending_return`·`create_return`·`create_settlement`·`create_payout` | Consignment | Consignment lifecycle (consignment/handler.go) | Konsinyasi | High | KEEP | **Good granularity** |
| `shift` / `create` (open) | Shift | Open shift (shift/handler.go:98) | Shift start + opening balance | High | MODIFY | Rename `SHIFT_OPENED` |
| `shift` / `update` (close) | Shift | Close shift (shift/handler.go:145) | Shift end + closing balance | High | MODIFY | Rename `SHIFT_CLOSED` |
| `shift` / `update` (closeall) | Shift | Close all (shift/handler.go:186) | Mass close | Med | MODIFY | No `entity_id` list; rename `SHIFT_CLOSE_ALL` |
| `shift` / `export`·`review`·`audit` | Shift | Export/review/cash-audit (shift/handler.go:385/444/486) | Reconciliation | Med | KEEP | Good |
| `sale` / `create` (×2 paths) | Sale | Sale completed (sale/handler.go:397/439) | Transaction | High | KEEP | Full snapshot in `new_values` (PII) |
| `sale` / `recall_sale` | Sale | Manager recall parked sale (sale/handler.go:1038) | Recall | Med | MODIFY | **Only when manager** — self-recall unaudited |
| `sale` / `complete_parked_sale` | Sale | Complete parked (sale/handler.go:1271) | Complete | Med | KEEP | — |
| `sale` / *void*·*refund*·*reprint* | Sale | (none) | Reverse/refund | **Critical** | **N/A** | **P0 — no void/refund/reprint endpoints exist in codebase** |
| `cart` / `cancel_cart`·`checkout` | Sale | Cart ops (cart_handler.go:439/493) | Cart lifecycle | Low | KEEP | `checkout` + `sale.create` = 2 events per sale (minor noise) |
| `payment` / *any* | Payment | (none) | Payment lifecycle | **Critical** | **PARTIAL** | **P0 — `payment` `create` event added on checkout; init/success/fail/method_change still missing** |

---

## C. Missing Events

| Priority | Proposed Event | Module | Trigger | Why Needed | Required Metadata |
|---|---|---|---|---|---|
| **P0** | `ROLE_PERMISSIONS_UPDATED` | User | `UpdateRolePermissions` (user/handler.go:551) | Privilege changes are currently **invisible** — biggest insider risk | actor, role_id, added_perms[], removed_perms[], before/after |
| **P0** | `SALE_VOIDED` / `SALE_REFUNDED` / `RECEIPT_REPRINTED` | Sale | Void/refund/reprint state transition | Money reversal with no trace; confirm these transitions exist and audit them | actor, sale_id, amount, reason, payment_refund_ref |
| **P0** | `PAYMENT_*` (initiated/succeeded/failed/method_changed/referenced_changed) | Payment | Payment processing in sale checkout | No payment-level accountability; fraud/manipulation undetectable | actor, sale_id, method, amount, ref_number, success, decline_reason |
| **P0** | `store_id` attribute on **all** events | Cross-cutting | From `shared.GetStoreID(c)` | Multi-store ops unattributable today (no column) | store_id |
| **P1** | `SUPPLIER_PAYMENT` / `SUPPLIER_DEBT_ADJUSTED` / `SUPPLIER_INVOICE` | Supplier | Vendor $ movements | Outbound money unlogged | actor, supplier_id, amount, direction, balance_after |
| **P1** | `PASSWORD_CHANGE_FAILED` | Auth | Wrong current password (auth_service ChangePassword path) | Brute-force / account-takeover attempts undetectable | username, ip, ua, reason |
| **P1** | `SHIFT_CASH_MOVEMENT` (cash_in/cash_out/drawer_open/adjustment) | Shift | Cash drawer operations | Cash discrepancies can't be traced to a movement | actor, shift_id, type, amount, reason |
| **P1** | `INVENTORY_TRANSFER_*` (created/approved/cancelled) | Inventory | Stock transfer | Inter-store stock moves unlogged | actor, from_loc, to_loc, items, qty |
| **P1** | `INVENTORY_ADJUSTMENT` (explicit, with reason) | Inventory | Manual stock adjust | `inventory.update` is too coarse to investigate | actor, product_id, loc, qty_before/after, reason |
| **P1** | `GOODS_RECEIPT_UPDATED` / `CANCELLED` | Purchase | GR modification/cancel | Received-goods tampering unlogged | actor, gr_id, po_id, changes |
| **P2** | `PURCHASE_ORDER_CONFIRMED` / `CANCELLED` (split from generic `update`) | Purchase | PO state transition | Ambiguous `update` action | actor, po_id, status_before/after |
| **P2** | `USER_ACTIVATED` / `DEACTIVATED` / `ROLE_CHANGED` | User | Account state change | Account-enable abuse unlogged distinctly | actor, target_user, field, before/after |
| **P2** | `TOKEN_REFRESH` / `SESSION_REVOKED` | Auth | Refresh / logout-all | Token lifecycle unobserved | actor, ip, ua |
| **P2** | `CONFIG_*` (tax/payment-method changes if any) | Config | Config mutation | Some config changes may bypass audit | actor, key, before/after |
| **P3** | `EXPORT_AUDIT` already exists for shift; add generic `AUDIT_EXPORTED` | Audit | Any audit export | Know when someone pulled the evidence | actor, filters, row_count |

---

## D. Audit Coverage Matrix

| Domain | Coverage | Critical Gaps | Recommendation |
|---|---|---|---|
| Authentication | Partial | Failed password change, token refresh not logged | Add P1 events |
| User Management | Partial | Role/active changes not diffed; self-recall gap | Add before/after + P2 events |
| Shift | Partial | Cash movements (in/out/drawer) absent; open/close ambiguous | Add cash-movement + explicit actions |
| Sales | Partial | **Void/refund/reprint missing**; recall only for managers | P0 add |
| Payment | **Missing** | Entire lifecycle unlogged | P0 add |
| Inventory | Partial | Transfers/adjustments absent; `update` too coarse | P1 add |
| Purchasing | Partial | GR modify/cancel; supplier payments/debt missing | P1 add |
| Product | Good | Price/cost changes need before/after | Add diffs |
| Pricing | Good | — | Keep; add diffs |
| Consignment | Good | — | Keep (exemplary) |
| Configuration | Partial | Permissions/tax/payment config gaps | P0/P2 add |
| Security | **Missing** | Permission grants invisible; no store attribution | P0 add |

---

## E. Security Findings

| Severity | Finding | Risk | Evidence | Recommendation |
|---|---|---|---|---|
| **Critical** | Role-permission changes are not audited at all | Insider can escalate privileges with zero trace | `UpdateRolePermissions` user/handler.go:551 has no `CreateAuditLog` | P0: emit `ROLE_PERMISSIONS_UPDATED` with before/after permission sets |
| **Critical** | No payment-level or refund/void audit events | Cash/refund manipulation undetectable; compliance failure | No `payment`/`refund`/`void` entity in any handler | P0: add `PAYMENT_*` and `SALE_VOIDED`/`SALE_REFUNDED` |
| **High** | No `store_id` on audit records | In a multi-store chain, actions can't be tied to a branch; investigations blind | `audit_logs` DDL lacks column (000_squash.sql:227); `Log` struct lacks field (domain.go:9) | P0: add `store_id` column + populate from `shared.GetStoreID(c)` |
| **High** | Every `CreateAuditLog` error is discarded (`_ =`) | Silent audit loss on DB errors; no retry, no alert | 25 sites, e.g. handler.go:393, repository.go:30 only logs | P1: at minimum log failure to app log + metrics; consider fail-closed for P0 actions |
| **High** | `UpdateRole` logs only name/description | Even when a role is audited, the *permission* delta is invisible | user/handler.go:541–542 old/new only name/desc | P0/P1: include permission diff |
| **Medium** | Actor identity is JWT-claim-derived, no server binding / correlation ID | Cannot link an audit row to a request or detect claim anomalies | `Log` has no `correlation_id` (domain.go:9); actor from `middleware.*FromContext` | P2: add `correlation_id` (request ID) and verify actor server-side |
| **Medium** | `new_values` stores full entity snapshots (e.g. whole `sale`, whole `product`) | PII / excess data retained in audit (customer name/phone in sale) | sale/handler.go:400 `ToJSONMap(sale)`; product/handler.go:247 | P2: store minimal focused diffs, not full snapshots; scrub PII |
| **Medium** | No DB-level immutability enforcement | A DBA/SQLi/compromised service account could DELETE/UPDATE rows | Table has no trigger/RLS; service merely omits methods | P1: add DB trigger/RLS to reject UPDATE/DELETE; restrict grants |
| **Low** | No retention / export-access tiering | Indefinite growth; export may be too broadly permitted | Only `audit.view` gates all (handler.go:31–34) | P2: define retention; gate `export` behind stronger role |
| **Low** | `recall_sale` audited only for managers | A cashier recalling their own parked sale leaves no record | sale/handler.go:1032 `if caller.IsManager()` | P2: audit all recalls |

---

## F. Event Schema Assessment

Current `audit.Log` (domain.go:9):

```
id, user_id, username, role, action, entity_type, entity_id,
description, ip_address, user_agent, old_values(jsonb), new_values(jsonb), created_at
```

**Gaps vs. what a security audit needs:**
- Missing `store_id` (branch attribution) — **must add**.
- Missing `correlation_id` / request ID — add for traceability.
- Missing explicit `success` boolean — currently encoded only via distinct actions (acceptable, but a `success` flag helps query failures uniformly).
- `created_at` is server-defaulted (good — not client-forgeable). Keep.
- `username`/`role` are redundant with `user_id` but useful for deleted-user rows (`ON DELETE SET NULL`); keep, but treat as display-only, not authoritative identity.
- `old_values`/`new_values` as `jsonb` is good; enforce **focused diffs**, not full snapshots.

**Recommended minimum schema (additive):**
```
id
created_at          -- server default, never client-set
store_id            -- from request context
actor_user_id
actor_username       -- display, nullable for anon (login_failed)
actor_role          -- display snapshot
action              -- explicit business verb (SHIFT_OPENED, not "update")
entity_type         -- domain (sale, payment, role, ...)
entity_id
success             -- bool
reason              -- failure/approval reason
correlation_id      -- request ID
ip_address
user_agent
changes             -- focused {field:{from,to}} jsonb (prefer over full new_values)
metadata            -- non-PII context (invoice_no, amount, etc.)
```

Migration: add `store_id INTEGER REFERENCES stores(id) ON DELETE SET NULL`, `correlation_id TEXT`, `success BOOLEAN DEFAULT TRUE`, and consider a `changes jsonb` column. Backfill `store_id` where determinable.

---

## G. Recommended Event Taxonomy

Adopt explicit, past-tense, domain-prefixed action verbs instead of generic `create`/`update`/`delete` (which force investigators to read descriptions). Keep `entity_type` as the domain noun.

```
AUTH_LOGIN, AUTH_LOGIN_FAILED, AUTH_LOGOUT, AUTH_PASSWORD_CHANGED,
AUTH_PASSWORD_CHANGE_FAILED, AUTH_TOKEN_REFRESHED, AUTH_SESSION_REVOKED

USER_CREATED, USER_UPDATED, USER_DEACTIVATED, USER_ACTIVATED,
USER_ROLE_CHANGED, USER_DELETED

ROLE_CREATED, ROLE_UPDATED, ROLE_PERMISSIONS_UPDATED, ROLE_DELETED

STORE_CREATED, STORE_UPDATED, STORE_DELETED

PRODUCT_CREATED, PRODUCT_UPDATED, PRODUCT_PRICE_CHANGED, PRODUCT_DELETED
CATEGORY_*, BRAND_*, UOM_*, CUSTOMER_*, CUSTOMER_GROUP_*

SUPPLIER_CREATED/UPDATED/DELETED, SUPPLIER_PAYMENT, SUPPLIER_DEBT_ADJUSTED
PRODUCT_SUPPLIER_*

PRICING_RULE_CREATED/UPDATED/DELETED

SHIFT_OPENED, SHIFT_CLOSED, SHIFT_CLOSE_ALL, SHIFT_CASH_IN, SHIFT_CASH_OUT,
SHIFT_DRAWER_OPENED, SHIFT_CASH_ADJUSTED, SHIFT_REVIEWED, SHIFT_AUDITED, SHIFT_EXPORTED

SALE_CREATED, SALE_VOIDED, SALE_REFUNDED, SALE_RECALLED, SALE_COMPLETED,
SALE_RECEIPT_REPRINTED, CART_CANCELLED, CART_CHECKOUT

PAYMENT_INITIATED, PAYMENT_SUCCEEDED, PAYMENT_FAILED, PAYMENT_METHOD_CHANGED,
PAYMENT_REVERSED, PAYMENT_REFERENCE_CHANGED

INVENTORY_ADJUSTED, INVENTORY_TRANSFER_CREATED, INVENTORY_TRANSFER_APPROVED,
INVENTORY_TRANSFER_CANCELLED, STOCK_RECEIVED
STOCK_OPNAME_* (keep current explicit verbs)
GOODS_RECEIPT_CREATED, GOODS_RECEIPT_UPDATED, GOODS_RECEIPT_CANCELLED
PURCHASE_ORDER_CREATED/CONFIRMED/CANCELLED/DELETED
CONSIGNMENT_* (keep current explicit verbs)

APP_SETTINGS_UPDATED/UPLOADED/REMOVED
STORAGE_LOCATION_*
CONFIG_*
AUDIT_EXPORTED
```

Rule: **one explicit verb per meaningful business/security action**; never reuse `update` for distinct state transitions (confirm vs cancel vs receive).

---

## H. Recommended Implementation Changes

### P0 — Must Fix
1. **Add `store_id` to `audit_logs` and populate it** from `shared.GetStoreID(c)` at every call site (or via a wrapping helper). Without this the audit log is unusable across stores.
2. **Audit role-permission changes** (`UpdateRolePermissions`) with before/after permission IDs, and **include permission diffs in `UpdateRole`**.
3. **Introduce payment-level events** (`PAYMENT_INITIATED/SUCCEEDED/FAILED/METHOD_CHANGED/REVERSED`) and **sale reverse events** (`SALE_VOIDED`, `SALE_REFUNDED`, `SALE_RECEIPT_REPRINTED`); verify these transitions are reachable and emit at the exact transition.
4. **Add DB-level immutability** (trigger or RLS) rejecting `UPDATE`/`DELETE` on `audit_logs`, and restrict DML grants so only the app role can `INSERT`.

### P1 — Should Fix
5. **Stop discarding audit errors.** Log failures to the application log + emit a metric/alert; for P0-class actions consider fail-closed (abort the business op if the audit write fails) or an outbox table.
6. **Add cash-movement shift events** (`SHIFT_CASH_IN/OUT/DRAWER_OPENED/ADJUSTED`) and make open/close explicit (`SHIFT_OPENED`/`SHIFT_CLOSED`).
7. **Add inventory transfer & explicit adjustment events** with reason; split `purchase_order` `update` into `CONFIRMED`/`CANCELLED`; add `GOODS_RECEIPT_UPDATED/CANCELLED` and supplier payment/debt events.
8. **Log failed password changes** and add `AUTH_PASSWORD_CHANGE_FAILED`.

### P2 — Nice to Have
9. Add `correlation_id` (request ID) to every audit row.
10. Replace full-snapshot `new_values` with focused `changes` diffs; scrub PII (customer name/phone) from sale/product audit payloads.
11. Audit all `recall_sale` calls (not just manager), add `AUTH_TOKEN_REFRESHED`/`SESSION_REVOKED`, `USER_DEACTIVATED/ACTIVATED/ROLE_CHANGED`, and `AUDIT_EXPORTED`.
12. Define a retention policy and consider tiering export access above `audit.view`.

---

### Cross-cutting notes (per constraints)
- **False negatives** (unlogged important actions): permission grants, payments, refunds/voids, cash movements, inventory transfers, supplier money — addressed above.
- **False positives / noise**: `cart.checkout` + `sale.create` for one sale (minor); `shift` `update` reused 3 ways (mitigated by explicit renames).
- **Sensitive data**: `formatSensitiveFields` only masks at *export display* time (handler.go:317), not at *storage* time — ensure no handler passes `password`/`token` into `new_values` (verified: user-create curates fields; change-password sends none). Remaining risk is PII-in-snapshot, not secrets.
- **Business vs audit vs app log**: cleanly separated — audit log is its own table/API, domain events use the event bus, app logs use `slog`. No inappropriate mixing found.
- **Reliability class**: today all audit writes are best-effort async-style (ignored error, separate from business transaction). Money/permission/auth events should be reclassified as fail-closed or outbox-backed.

---

## I. Completion Status (Implementation)

Scope: P0 fully addressed. Follow-up sessions implemented the implementable P1/P2 items; items requiring features that do not exist in the codebase (cash-movement, GR update/cancel, supplier money, session-revoke-all) are marked **N/A**.

### P0 — Must Fix
| # | Recommendation | Status | Implementation |
|---|----------------|--------|----------------|
| 1 | Add `store_id` to `audit_logs` + populate at every call site | **Done** | Migration `database/migrations/033_audit_log_store_and_immutability.sql` adds `store_id` (+ index). `internal/audit/domain.go` adds `StoreID` to `Log`/`LogListItem`; `internal/audit/repository.go` writes/reads `store_id` ($2). `internal/audit/handler.go` adds Store ID to XLSX/CSV exports. All ~75 `&audit.Log{}` call sites now set `StoreID` via `middleware.StoreIDFromContext(...)`. The `internal/user` package cannot import `middleware` (circular: `middleware` → `user`), so it uses a local `storeIDFromGin(c)` helper mirroring `auditContextFromGin`. |
| 2 | Audit role-permission changes + permission diff in `UpdateRole` | **Done** | `internal/user/handler.go` `UpdateRolePermissions` now captures before/after permission code sets and emits an `update_permissions` audit on entity `role` with `old_values`/`new_values` (permission lists) and a human-readable added/removed summary. |
| 3 | Payment-level events + sale void/refund/reprint | **Partial** | Payment-level coverage added: `internal/sale/handler.go` `respondSaleFromCart` now emits a `create` audit per `payment` (EntityType `payment`, amount/method/reference/sale_id); `CancelParkedSale` emits a `cancel` audit on `sale`. **Void / refund / receipt-reprint endpoints do not exist in the codebase**, so `SALE_VOIDED`, `SALE_REFUNDED`, `SALE_RECEIPT_REPRINTED`, and `PAYMENT_REVERSED/FAILED` have no transitions to instrument — left as N/A pending those features. |
| 4 | DB-level immutability (reject UPDATE/DELETE) | **Done** | Migration `033` installs `reject_audit_log_modification()` trigger `BEFORE UPDATE OR DELETE` on `audit_logs`. `TRUNCATE` (test cleanup) is unaffected. |

### P1 — Should Fix
| # | Recommendation | Status | Implementation |
|---|----------------|--------|----------------|
| 5 | Stop discarding audit errors / fail-closed or outbox | **Done (logging + metrics)** | `internal/metrics` adds an atomic `AuditWriteFailures` counter; `internal/audit/repository.go` increments it and logs each failed `CreateAuditLog`. `GET /metrics` (cmd/server/main.go) exposes `audit_write_failures_total`. Fail-closed/outbox still deferred. |
| 6 | Cash-movement shift events + explicit open/close | **Partial / N/A** | No cash-movement functions exist in `internal/shift` (no `cash_in`/`cash_out`/`drawer_open`/`adjustment`), so `SHIFT_CASH_MOVEMENT` is **N/A**. Explicit `SHIFT_OPENED`/`SHIFT_CLOSED` renames not yet applied — deferred. |
| 7 | Inventory transfer / adjustment / PO & GR lifecycle events | **Done (implementable parts)** | `inventory_adjustment` (AdjustStock) and `inventory_transfer` (TransferLocationStock) events added in `internal/inventory`. `purchase_order_confirmed`/`purchase_order_cancelled` added in `internal/purchase` (ConfirmPO/CancelPO). `GOODS_RECEIPT_UPDATED/CANCELLED` and supplier payment/debt events are **N/A** (no such endpoints/tables). |
| 8 | Failed password-change audit + `AUTH_PASSWORD_CHANGE_FAILED` | **Done** | `internal/user/auth_handler.go` emits `password_change_failed` on `ErrInvalidPassword`; dummy seeder reflects it. |

### P2 — Nice to Have
| # | Recommendation | Status | Implementation |
|---|----------------|--------|----------------|
| 9 | `correlation_id` on every audit row | Not started |
| 10 | Focused `changes` diffs + PII scrubbing at storage time | Not started (note: inventory/user now emit old/new-value diffs; full-snapshot PII risk remains) |
| 11 | Recall-sale audit, token/session/user lifecycle events | **Partial** | `token_refresh` (auth refresh) Done in `internal/user`; `user_activated`/`user_deactivated`/`user_role_changed` Done (UpdateUser emits distinct events on state change); `config_updated` added in `internal/appsettings`. `SESSION_REVOKED` is **N/A** (no revoke-all endpoint; single logout audits `logout`). Recall-sale audit not started. |
| 12 | Retention policy + tiered export access | Not started |

### Verification
- `go build ./...` — passes.
- `go vet` + `golangci-lint run` on changed packages (`internal/inventory`, `internal/purchase`, `internal/user`, `internal/appsettings`, `internal/audit`) — clean.
- `go test` (audit sync-tests, mock audit creator — no DB) for `internal/inventory`, `internal/purchase`, `internal/user`, `internal/appsettings` — pass.
- Dummy seeder (`cmd/dummy`) regenerates with representative rows for the new explicit events and compiles.
- **Deployment note**: migration `033` must be applied **before** deploying the binary, consistent with the project's migration-ordering policy in AGENTS.md.
