# Full Backend Review

## Summary

The Go backend is well-structured: modular-monolith boundaries, typed permission codes with an enforced baseline invariant, server-authoritative pricing, and rigorously idempotent migrations. Two CRITICAL issues block release though — an RBAC gap that lets a manager mint a superadmin account, and a baseline-migration replay failure that will abort every deploy on any database that already applied migration 058. A further twelve WARNING/SUGGESTION findings span pricing export scope, cash reconciliation, consignment concurrency, import performance, and dead/duplicated code.

## Issues Found

| Severity | File:Line | Issue |
|----------|-----------|-------|
| CRITICAL | `internal/user/handler.go:262-291, 360-398` (grants `database/migrations/000_baseline.sql:5063`) | No role ceiling on `CreateUser`/`UpdateUser` — a manager can create or promote a superadmin. |
| CRITICAL | `database/migrations/000_baseline.sql:4593, 2420-2425, 3008` vs `058_supplier_terms_store_scope.sql:50-57,94` | Baseline re-adds pre-058 constraints/indexes on schema 058 removed; replay on a 058'd DB aborts the whole baseline transaction, failing every deploy. |
| WARNING | `internal/pricing/adapter.go:204`, `repository.go:774-781` | Pricing-rule export is not store-scoped — any `pricing.view` holder dumps every store's rules. |
| WARNING | `internal/product/handler.go:67-70` | Product reads bypass `product.view` (`GET /products` anonymous; `GET /products/:id` any authenticated user). |
| WARNING | `internal/shift/repository.go:229`, `auto_close.go:50`, `handler.go:514` | Shift discrepancy ignores cash movements; recorded over/short is wrong whenever a drop/paid-in/paid-out occurs (frontend expected-cash includes them). |
| WARNING | `internal/sale/handler.go:287, 1041, 1331`; `pricing/resolver.go:286-287` | `customer_group_id` is client-supplied and never cross-checked against the customer's actual group. |
| WARNING | `internal/sale/handler.go:233-235` vs `:1058-1064` | ParkSale accepts a client invoice number; `sales.invoice_number` is UNIQUE + sequence-issued → collision/DoS. |
| WARNING | `internal/sale/cart_service.go:247-252`, `:466` | Quantity-tiered pricing is never re-applied after cart quantity edits (direct checkout re-resolves; cart path doesn't). |
| WARNING | `internal/consignment/service.go:1470-1518, 1330-1375` | No row locks in settlement/payout; concurrent payouts can over-pay payable and concurrent settlements can double-settle an item. |
| WARNING | `internal/platform/importexport/import/engine.go:220-289` | ~3 autocommitted statements per CSV row (cancel check + history + progress); 5k rows ≈ 15k round-trips. |
| WARNING | `deploy/podman-deploy.sh:562-570` | `seed` runs the unguarded bulk seeder (`-truncate=true` default) with no production guard. |
| WARNING | `internal/shift/repository.go:231` vs `internal/shift/service.go:45-66` | Hardcoded `discrepancyThreshold = 50000` duplicates the editable `shift_discrepancy_threshold` setting. |
| WARNING | `internal/sale/repository.go:307-314`, `internal/purchase/repository.go:445-452`, `internal/report/ranges.go:260`, `internal/audit/handler.go:381` | Date params validated four divergent ways; sales/purchase silently widen to unbounded history. |
| SUGGESTION | `cmd/server/main.go:226-240` | `/metrics` and `/swagger/*any` are anonymous — business totals + API surface disclosure. |
| SUGGESTION | `internal/consignment/checkout_provider.go:38-66` | Per-line `FOR UPDATE` + reduce + term lookup inside the checkout transaction (round-trips even for stores without consignment). |
| SUGGESTION | `internal/inventory/stock_deducer.go:70-71` | O(N) per-item UPDATEs in the checkout tx; batch lock already exists. |
| SUGGESTION | `cmd/server/main.go:103` | Postgres DSN built by string concat — URL-special chars in password break config. |
| SUGGESTION | `internal/purchase/domain.go:114,134`; `internal/sale/cart_handler.go:20` | Dead DTOs and a dead `CartService` interface with zero references. |

## Detailed Findings

### Security

**S1 — CRITICAL — Privilege escalation via role assignment [95%]**
`POST /admin/users` (`handler.go:166`) requires only `user.create`; `PUT /admin/users/:id` only `user.update`. Both accept a client `role_id` (`:185, :198`). `GetRoleByID` (`:276-279`) checks existence only — there is no role hierarchy, no "may not assign a role ≥ your own" rule, and `bindStoreScopedUser` pins *store* only, never role. The manager role holds `user.create`, `user.update`, and `role.view` (`000_baseline.sql:5063`), and `GET /admin/roles` (`:172`, `:710-712`) returns all roles including superadmin (id=1, `:4630`). A manager therefore: `GET /admin/roles` → `POST /admin/users` with `role_id:1` → new superadmin whose JWT is minted with all permissions (`auth_service.go:138`). `UpdateUser` also lets a manager change any *other* user's role to superadmin — its only guard blocks the caller changing **their own** role/store (`:378-384`). Adding `role.create` to manager (`:5063`) makes full role-copy → superadmin minting equally possible.
Fix: enforce a ceiling (only superadmin may assign superadmin; caller may not grant a role more privileged than their own); drop `role.create`/`role.update` from manager, or keep role CRUD superadmin-only.

**S2 — WARNING — Pricing export not store-scoped [95%]**
`ExportData` → `GetAllForExport` (`adapter.go:204` ↔ `repository.go:774-781`) reads `FROM pricing_rules ORDER BY id ASC` with no store filter, while the list path (`handler.go:327-345`) narrows store callers and the customer/product/supplier adapters scope via `StoreIDFromContext`. Export is reachable with only `pricing.view` (`importexport/handler/handler.go:48` `pricing_rules:export`). A multi-store estate leaks every store's margin schedule to any manager.
Fix: apply the same store scoping to the export query and to row injection; reject cross-store export for store-scoped callers.

**S3 — WARNING — Product reads bypass ProductView [80%]**
`GET /products` (`handler.go:68`) is anonymous; `GET /products/:id` (`:70`) is authenticated but not permission-gated, while create/update/delete require `product.*` perms. finance and inventory_staff are deliberately granted no `product.view` (`000_baseline.sql:5045, 5051`), yet any authenticated user can read `/products/:id`. Cost stays hidden (`presenter.go:12-19`), but catalogue/SKU is exposed and the gate is inconsistent with both the cost-gate design and the other product routes.
Fix: require `product.view` on the detail route; confirm the anonymous list route is intentional and document it.

**S4 — SUGGESTION — Anonymous `/metrics` and `/swagger/*any` [95%]**
Both registered outside `/api` (`main.go:226-240`). Metrics expose sales/dashboard counters; swagger enumerates the full API. Not a credential risk, but needs an auth gate (or admin-role restriction) in production.

### Business Logic

**B1 — WARNING — Cash reconciliation omits cash movements [95%]**
`CloseShiftTx` records `discrepancy = closing - opening - TotalCashSales` (`repository.go:229`), ignoring `CashMovementSummary.NetEffect = -drops + paid_ins - paid_outs` (`cash_movement.go:134`). Auto-close writes `expectedCash = opening + cashSales` as the closing balance (`auto_close.go:50`), and the audit endpoint uses the same (`handler.go:514`). The frontend's expected-cash, however, *does* include movements (`ShiftReport.svelte:228-232`). A mid-shift cash drop therefore produces a large false "short" that flips `needs_review` and skews reports and auto-close balances.
Fix: use `closing - opening - TotalCashSales - NetEffect` everywhere (one shared helper on the report), and make auto-close adopt the same expected-cash.

**B2 — WARNING — Client-supplied CustomerGroupID selects pricing rules [90%]**
`CustomerGroupID` flows straight from the request body into `ResolveCheckoutPrices` (`handler.go:287`, park `:1041`, complete `:1331`, cart `cart_handler.go:240`), whose resolver applies group-restricted rules (`pricing/resolver.go:286-287`, `repository.go:48`). Nothing validates it against `customer_id`'s real group. Direct-API callers can price a sale at any group's tier.
Fix: resolve the group server-side from the customer record (or verify req group == customer group) and pin it on the cart/sale.

**B3 — WARNING — ParkSale accepts client invoice_number [90%]**
CreateSale rejects client invoice numbers (`handler.go:233-236`); ParkSale silently accepts one (`:1058-1064`). `sales.invoice_number` is UNIQUE (`000_baseline.sql:2576-2578`) and drawn from `invoice_seq` (`repository.go:718-729`), so an authenticated `sale.park` holder can reserve future numbers and collide with (effectively block) subsequent real sales.
Fix: drop the client field; always call `GetNextInvoiceNumber`.

**B4 — WARNING — Cart quantity edits freeze quantity-tiered pricing [85%]**
`UpdateCartItemQuantity` recomputes the line from the **stored snapshot `UnitPrice`** (`cart_service.go:247-252`); checkout only sanity-checks `UnitPrice == Subtotal/Quantity` (`:466`) using frozen snapshots (`:462-474`). Direct checkout re-resolves (`handler.go:292`). Pricing rules with `minimum_quantity`/`maximum_quantity` (`resolver.go:290-293`) are therefore never re-applied when a cart line's quantity changes — drift and possible under/overcharge versus the direct path.
Fix: re-run `ResolveCheckoutPrices` after quantity changes (or at checkout) so both flows behave identically.

**B5 — WARNING — Consignment settlement/payout lack row locks [85%]**
`CreatePayout` re-reads the settlement, sums payouts, and checks `Amount <= TotalPayable - paidSoFar` with no `FOR UPDATE` (`service.go:1470-1481`; `GetSettlementByIDQuery` is a plain SELECT, `repository.go:1384-1392`). Two concurrent payouts can both pass the check and over-pay. `CreateSettlement` also reads unsettled sale items without locking them, and `consignment_settlement_items` has no UNIQUE on `consignment_sale_item_id` — concurrent settlements can double-settle an item's proceeds.
Fix: `SELECT … FOR UPDATE` the settlement row / unsettled rows in the tx; add a UNIQUE on `(consignment_sale_item_id)`.

### Performance

**P1 — SUGGESTION — Per-line DB statements in checkout [90%]**
`ResolveAndDeductConsignment` (`checkout_provider.go:38-66`) runs lock + reduce + term lookup per cart line inside the checkout transaction, round-tripping even for stores with no consignment (`repository.go:562` returns nil). Bounded by cart size, so low impact, but a batched ownership pre-check + single `UPDATE … FROM` would remove it.

**P2 — SUGGESTION — O(N) UPDATEs in StockDeducer [95%]**
`DeductStock` already batch-locks in one query (`stock_deducer.go:51-66`) then decrements per item (`:70-71`). Collapse into a single conditional `UPDATE … CASE` for one statement per checkout.

**P3 — WARNING — ~3 autocommitted statements per import row [90%]**
For each CSV row the engine issues `IsCancelRequested` (progress/pg_repo.go:241), history `SaveRow` (history/pg_store.go:52), and `UpdateProgress` (progress/pg_repo.go:74-88) — each autocommitted through the pool. The *entity* insert/update is batched (`engine.go:286-299`), but 5k rows make ~15k statements. Throttle progress/history writes (every N rows) and batch the history inserts.

### Deploy Safety

**D1 — CRITICAL — Baseline replay breaks on any database that applied 058 [95%]**
`058_supplier_terms_store_scope.sql` drops `suppliers.store_id` (`:94`) and replaces the `product_suppliers` UNIQUE and preferred-supplier index (`:50-57`). On every redeploy the baseline replays (podman-deploy.sh iterate-all-files, psql `-v ON_ERROR_STOP=1`, `:538-548`):
- `:4593` re-adds `suppliers_store_id_fkey` referencing the dropped column — the baseline never re-adds the column (only `CREATE TABLE IF NOT EXISTS`, which is a no-op on existing DBs). `ALTER TABLE … ADD CONSTRAINT … (store_id)` errors, the whole baseline (`BEGIN…COMMIT`) rolls back, `migrate()` fails, and `start_backend` refuses to roll out (`:442`). **Every subsequent deploy on the existing estate fails.**
- `:2420-2425` re-adds plain `UNIQUE (product_id, supplier_id)` — violates on any product/supplier with per-store rows (058 allows them), and even when it succeeds it reinstates semantics 058 removed.
- `:3008` recreates `idx_product_suppliers_one_preferred` on `(product_id)` — duplicate-key failure once two stores mark the same product preferred.

CI does not catch this: it always migrates a fresh database (ci.yml `:376-379`).
Fix: make these three baseline statements conditional on the 058-era schema (skip the suppliers FK when `store_id` is absent; recreate the NULLS-NOT-DISTINCT shapes 058 establishes) and add a CI job that applies migrations **twice** against a 058-shaped DB.

**D2 — WARNING — `podman-deploy.sh seed` is unguarded [85%]**
`seed()` (`deploy/podman-deploy.sh:562-570`) runs `go run cmd/dummy/main.go`, whose bulk seeder defaults to `-truncate=true` and truncates 41 tables with no ENV/HOST/DBNAME guard (unlike `scripts/reset-dev-db.sh`). An accidental `./podman-deploy.sh seed` against production destroys transactional data.
Fix: add the same hard-refuse guards used by reset-dev-db.sh.

**D3 — SUGGESTION — DSN string concat, no escaping [90%]**
`cmd/server/main.go:103` interpolates user/password directly into `postgres://`. A password containing `@ : / ?` produces an unparsable or wrong-target DSN at startup. Use `url.URL` + `url.UserPassword`.

### Duplication / Dead Code

**X1 — WARNING — Duplicate discrepancy threshold drifts [95%]**
`const discrepancyThreshold = 50000` (`shift/repository.go:231-232`) hardcodes the same value the app setting `shift_discrepancy_threshold` provides everywhere else (`service.go:45-66`, seeded `50000` at `000_baseline.sql:4996`, used by audit `handler.go:537`). Changing the setting via the UI updates the audit path but not `CloseShiftTx`/auto-close flagging.
Fix: route the threshold through the repo from the setting.

**X2 — WARNING — Date params validated four different ways [95%]**
- sale `repository.go:307-314` and purchase `repository.go:445-452`: parse failure silently **skips** the filter → a typo'd `start_date` returns the entire history.
- report `ranges.go:260`: 400 Bad Request.
- audit `handler.go:381-389`: silently returns "" (filter dropped).

Same user input, three different behaviors, one of them (unbounded history) an integrity/UX hazard.
Fix: shared validation helper; always reject malformed dates.

**X3 — SUGGESTION — Dead request DTOs and dead interface [95%]**
`CreatePurchaseOrderRequest`/`UpdatePurchaseOrderRequest` (`purchase/domain.go:114,134`) are never referenced — the handler binds inline anonymous structs (`handler.go:96,173`). `CartService` (`sale/cart_handler.go:20`) has zero implementations or callers; `Handler` holds `Service` directly. Remove.

**Dropped after verification:** `dbPool.Close()` ordering (verified correct — it runs *last* via LIFO defer), `sale with nil shift_id` (intentional optionality, no concrete harm proven), and eventbus at-most-once stock increment (documented design trade-off).

## Recommendation

**NEEDS CHANGES.** Two CRITICAL issues block release: the RBAC privilege escalation (S1) and the migration-058 baseline replay failure that breaks every subsequent deploy (D1). S1, B1, B2, B3, and the date-param inconsistency (X2) are quick, well-scoped fixes. D1 needs care to keep the baseline's "re-runnable on all three runner shapes" contract intact.

Want me to fix any of these? Suggestions: **(1)** S1 + cheap warnings (role ceiling, park invoice number, customer-group validation, shift-net-effect, price re-resolution), **(2)** D1 baseline repair + CI replay job, **(3)** everything. I won't touch the tree until you pick.