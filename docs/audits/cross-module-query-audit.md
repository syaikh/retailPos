# Audit: Queries That Cross Module Boundaries

| Field | Value |
|---|---|
| Date | 2026-10-07 |
| Question | Does any code run a database query against a table that belongs to a *different* module? |
| Where the rules live | `docs/adr/ADR_Modular_Monolith_Module_Boundaries.md` (§2.8 = who owns which table, §2.9 = how it's checked) |
| Where the checker lives | `internal/archtest/archtest_test.go` |
| How we checked | Ran the checker (`go test ./internal/archtest` — it passed), then independently listed every table each module's SQL touches and compared that list against the ownership rules |
| Not covered here | Import rules (which module may `import` which), and cross-module *writes* that go through the official "port" mechanism (those are allowed on purpose) |
| Status | **Remediated** — all seven steps of §4 were implemented and verified on 2026-10-07 (commit `9ae1862c`). Findings 1–3 are fixed, holes (a)–(d) are closed, the scanner accepts quoted/schema-qualified names (step 6), and the ADR records `cmd/dummy` as out of scope (step 7). See §4 for the per-step result. |
| Reading this doc | §1–§3 describe the state **as found** (before remediation); every finding and hole carries a *Status now* note. Code references point at the pre-remediation tree (`4dda0dcb`), so line numbers have moved. |

---

## 0. A short glossary (read this first if you're new)

- **Module** — a self-contained folder of the codebase with one responsibility, e.g. `internal/sale` handles sales, `internal/product` handles products. The project rule is *"one module, one job, don't reach into someone else's folder."*
- **Table** — a database table, e.g. `sales`, `products`. Each table has exactly one **owner module** — only that module may query it directly.
- **Query** — a SQL statement. A **read** (`SELECT`) looks at data; a **write** (`INSERT` / `UPDATE` / `DELETE`) changes it.
- **Cross-module query** — a query written by module A that touches a table owned by module B. E.g. `internal/consignment` running `SELECT ... FROM products` — `products` belongs to `internal/product`, not to consignment.
- **Port** — the sanctioned workaround. If consignment needs product data, `product` publishes a small function (a "port") that consignment may call. The table itself stays private. Analogy: asking the kitchen for a dish instead of walking into the kitchen and cooking it yourself.
- **The checker (archtest)** — an automated test that reads every source file, extracts the table names out of each SQL string, and fails the build if a module touches a table it doesn't own. "Green" = the test passes.
- **Strict vs. context-level** — two strictness levels in the checker. *Strict* = the module may touch only the tables on its own whitelist, nothing else. *Context-level* = looser: the module may touch any table in its whole **group** (e.g. everything in the "transaksional"/transactional group), even tables owned by neighbouring modules. At the time of the audit `consignment` was still context-level; since the remediation **every domain module is strict except `report`** (the read model).
- **Debt manifest (`crossContextDebt`)** — a "known problems" list inside the checker. A violation listed there doesn't fail the build; it just stays on the backlog until someone ports it properly. The test also fails if a listed entry becomes *outdated* (already fixed but never removed), so the list can't rot. **It is empty since the remediation** — no cross-module read is tolerated any more.
- **Read model / `mv_*`** — pre-aggregated reporting tables (materialized views) used by dashboards. The `report` module is allowed to read *any* table to build its charts, but must never change data.

---

## 1. Verdict

**No forbidden writes.** Every place where one module changes another module's data goes through a proper port (`StockAdjuster`, `MovementWriter`, `StockApplier`, ...). That part of the architecture is healthy.

**Reads are a different story.** Five groups of cross-module reads existed:

| # | Who reads whose table | Where | What we make of it | Status now |
|---|---|---|---|---|
| 1 | `consignment` reads `products` | `internal/consignment/repository.go:472`, `:484`, `:812`, `:953`, `:1549` | **Known problem.** Already listed in the debt manifest, so the checker tolerates it on purpose. Still should be fixed. | **Fixed** — all five queries now go through the `ProductMetaProvider` port; the debt entry is deleted (§4 step 3–4) |
| 2 | `consignment` reads `product_stock` | `internal/consignment/repository.go:497` (the `SearchAvailableProducts` function) | **Slipped through by accident.** Not listed in the debt manifest — it passes only because consignment hasn't been promoted to "strict" checking yet. | **Fixed** — stock exclusions moved behind `StockReader`, and `consignment` is strict (§4 step 3–4) |
| 3 | `sale` reads `mv_dashboard_totals` | `internal/sale/report_adapter.go:47` | **Invisible violation.** It *is* a rule breach, but the checker can't see the table at all (see §3a), so it never fails. | **Fixed** — the query moved into `internal/report` (its owner), and the view is registered in the checker (§4 step 1–2) |
| 4 | `report` reads `product_stock`, `categories`, `mv_daily_sales`, `mv_hourly_sales` | `internal/report/repository.go:374` and the `mv_*` queries | **Allowed on purpose.** The reporting module may read anything (see glossary) — and it never writes anything. | **Allowed, unchanged** |
| 5 | `audit` writes `audit_logs`, reads `stores`/`users` | `internal/audit/repository.go:45`, `:75` | **Allowed on purpose.** Logging is shared infrastructure used by every module; the ADR records this exception explicitly. | **Allowed, unchanged** |

Checked and found clean:

- `internal/shared/testdb.go` writes `stores` / `schema_migrations` — this only runs in test setup, not in the real application.
- `internal/shared/report.go` contains SQL text for `sales` / `sale_items`, but that SQL is actually executed by `internal/sale/report_adapter.go` — the owner of those tables. The wording lives in `shared`; the execution happens in the right place. That's the correct pattern.
- `pkg/*`, `middleware`, `permissions`, `wiring`, `config`, `secregtest`, `metrics` contain no SQL at all.
- Every strict module touches only its own tables. That list now includes `consignment` and `appsettings`, which the audit found unchecked (§3c, §3d).

---

## 2. The five cases, explained one by one

### 2.1 `consignment` reads `products` — a known, tracked problem

Five queries in the consignment module search products by name or SKU (product full-text search):

- `SearchAvailableProducts` — checks whether a product exists, then searches it (`repository.go:472`, `:484`)
- `ListConsignmentStockPaged` — looks up a product inside a filter (`repository.go:812`)
- `ListReceipts` — joins product details onto receipt rows (`repository.go:953`)
- `ListSettlements` — same idea for settlements (`repository.go:1549`)

Consignment does not own `products` — `product` does. This was written down in the debt manifest (`crossContextDebt`, then at `archtest_test.go:237`), which is why CI stayed green. Think of it as a post-it note on the fridge: *"we know, we'll fix it."* The checker also forces us to delete the note once the fix lands, so it can't be forgotten.

> **Status now: fixed.** `SearchAvailableProducts` no longer runs SQL at all — the service calls the new port methods `ProductMetaProvider.SearchProductOptions` and `ProductMetaProvider.ProductIDsByNameOrSKU` (implemented by `product.MetaLookup`), then filters the result against consignment's own tables through `Service.availableForTerms`. The three list filters (`ListConsignmentStockPaged`, `ListReceipts`, `ListSettlements`) dropped their `JOIN products` / `EXISTS ... FROM products` subqueries in favour of `product_id = ANY($n)`, with the IDs resolved through the same port; an empty ID list correctly matches nothing. The debt entry was deleted when `consignment` became strict (step 4).

### 2.2 `consignment` reads `product_stock` — slipped through unnoticed

The same `SearchAvailableProducts` function also runs:

```sql
SELECT ps.product_id FROM product_stock ps WHERE ...
```

(`repository.go:497`) to work out which products are already in stock elsewhere.

`product_stock` is owned by the `inventory` module. This read is **not** in the debt manifest — nobody wrote the post-it. It passes the checker only because consignment is still checked at the loose *context-level*: both `consignment` and `product_stock` sit in the same group ("transaksional"), so the loose rule allows it. Under the strict rule every other module follows, this would fail.

The proper fix already exists and is even wired up:

- `internal/consignment/ports.go:24` — the `StockReader` port (the official way to ask inventory about stock)
- `internal/inventory/consignment_adjuster.go:104`, `:123` — inventory's implementation of that port
- `internal/wiring/wiring.go:502` — where the two get connected at startup

A sibling function, `ListAddTermProductOptions` (`service.go:409`), already uses these ports correctly. `SearchAvailableProducts` is the one that was never migrated — it still reaches into both `products` and `product_stock` inside a single SQL statement.

> **Status now: fixed.** The single SQL statement is gone. Product lookup moved to the `product` side (see §2.1), and the stock exclusion now comes from `StockReader.StoreOwnedQuantities` — the same port `ListAddTermProductOptions` used. Both functions share one helper, `Service.availableForTerms`, so the eligibility rules can no longer drift apart. With step 4, `consignment` is checked strictly: a future `product_stock` read would fail CI outright.

### 2.3 `sale` reads `mv_dashboard_totals` — a violation the checker cannot see

`internal/sale/report_adapter.go:47`:

```sql
SELECT COALESCE(SUM(total_revenue), 0), COALESCE(SUM(transaction_count), 0)
FROM mv_dashboard_totals
```

The `sale` module is strict: it may touch only its six tables (`sales`, `sale_items`, `sale_payments`, `payment_methods`, `cart_sessions`, `cart_items`). `mv_dashboard_totals` is a reporting view owned by `report`.

It passes anyway for a boring technical reason: the checker keeps a hand-written list of known tables (`tableContext`), and `mv_dashboard_totals` was never added to it — and the checker's backup way of harvesting table names from the database migrations is broken (§3a). A table the checker doesn't know about is a table the checker ignores.

There's a nuance here: the surrounding *call* goes through a proper port (report's `GetAllCompletedSalesStats` interface, `internal/report/ports.go:19`) — but the SQL itself is written and run inside `sale`. The right module to host this query is `report`, which is allowed to read anything.

> **Status now: fixed.** The query now lives in `internal/report` as the private `Repository.getAllCompletedSalesStats`, so the SQL runs in the module that owns the view. The `GetAllCompletedSalesStats` method was removed from the `SaleStatsProvider` port (`internal/report/ports.go`) — the report module reads its own read model and needs no port for it — while the other two methods stayed, so no caller broke. The parity test moved with the query to `internal/report/dashboard_totals_test.go` (it also asserts the view agrees with the raw `sales` table), and the view is registered in `tableContext` so the checker can see it (§3b).

### 2.4 `report` reads — allowed by design

The reporting module reads `product_stock`, `categories`, `mv_daily_sales` and `mv_hourly_sales`. That's its job: dashboards need data from everywhere. The rule for `report` is *"read anything, change nothing"* — and we confirmed there is no `INSERT`, `UPDATE` or `DELETE` anywhere in `internal/report`.

> **Status now: unchanged.** Still allowed by design; the remediation only *added* to `report` (§2.3), never took the read allowance away.

### 2.5 `audit` writes — a deliberate exception

`internal/audit` inserts into and prunes `audit_logs`. The ownership table assigns `audit_logs` to the `user` module, but the ADR (§5.2) records audit as shared infrastructure that sits outside the module system entirely — every module is allowed to log.

> **Status now: unchanged.** Still an explicit, documented exception — the remediation did not touch it.

---

## 3. Holes in the checker itself

The checker is good but had six blind spots. **(a)–(d) are closed; (e) is partially closed; (f) is now documented.** Details per hole below.

### (a) The backup table-name harvester doesn't work

To catch tables that exist in the database but were forgotten in the hand-written list, the checker scans the migration files for `CREATE TABLE` / `CREATE VIEW` statements (`archtest_test.go:289`).

The problem: every one of the 65 statements in `000_baseline.sql` is written with a schema prefix —

```sql
CREATE TABLE IF NOT EXISTS public.users
```

— and the pattern grabs only the first word, so it captures **`public`** instead of `users`, every single time. The backup harvest reduces to the single useless entry `{"public"}`.

Consequence: a table created by a migration but missing from the hand-written list is **invisible** to the check — the code quietly skips any reference to it (`archtest_test.go:324-326`). That's exactly how finding §2.3 stays hidden.

> **Status now: fixed (step 1).** `createRe` strips an optional schema qualifier (and an optional quote mark), so `CREATE TABLE IF NOT EXISTS public.users` yields `users`. The harvest is back to being a real safety net.

### (b) `mv_dashboard_totals` is missing from the hand-written list

Its two sibling views (`mv_daily_sales`, `mv_hourly_sales`) are listed as belonging to the reporting group; this one isn't. Combined with hole (a), there is no path by which the checker could ever learn about it.

> **Status now: fixed (step 1).** `mv_dashboard_totals` is registered in `tableContext` as an `analitik` table, and the `sale` breach it exposed was removed (§2.3).

### (c) `consignment` hasn't been promoted to strict checking

Because consignment is still on the loose *context-level* rule, it could read **or write** any table in the transactional group — `sales`, `sale_items`, `sale_payments`, `shifts`, `cash_movements`, `cart_sessions`, `cart_items`, `product_stock`, `inventory_movements`, `payment_methods` — without the test failing. Today it only reads (§2.1, §2.2), so nothing is actively broken, but the guard rail is missing for this one module.

> **Status now: fixed (step 4).** `consignment` is in `strictModuleTables` with its own 13 `consignment_*` tables, and the `consignment → products` debt entry is gone (the checker fails on stale entries, so it could not be left behind).

### (d) `appsettings` isn't registered as a module

The list of modules the checker walks over (`domainModules`, `archtest_test.go:19-23`) leaves out `appsettings`, so its SQL is never inspected — it's only covered by the import rule. Today it touches a single table (`app_settings`), so nothing is currently wrong; it's a latent gap.

> **Status now: fixed (step 5).** `appsettings` is in `domainModules`, so both the import rule and the table-ownership rule now cover it.

### (e) Ways the scanner could be fooled (none used today)

- **Quoted table names:** `FROM "sales"` doesn't match the scanner's pattern because of the leading quote mark. Only `internal/shared/testdb.go:271` quotes a table name; no module code does.
- **Table names built at runtime:** something like `fmt.Sprintf("SELECT ... FROM %s", tableName)` would also slip past. We checked — every `fmt.Sprintf` SQL in the codebase only injects filter fragments or `$1` placeholders, never a table name.

> **Status now: partially fixed (step 6).** `sqlKeywordRe` now accepts quotes *and* a schema/alias qualifier (`FROM "sales"`, `FROM public.sales`, `FROM "public"."sales"`), pinned by `TestSQLTableRefPattern`. Runtime-built table names remain undetectable by construction — that limitation stays, but the audit re-checked the codebase and found no such query.

### (f) The data seeder writes every table

`cmd/dummy` (run by `./seed-dev.sh`) inserts into essentially every table. It's a developer tool, not part of the application, and the module rules deliberately don't cover it. Noted for completeness only.

> **Status now: documented (step 7).** The ADR (§5.4) now states explicitly that `cmd/dummy` and `internal/shared/testdb.go` are out of scope of the ownership rules — a decision, not an oversight.

---

## 4. What we recommended doing — and what happened

Ordered so the build stays green after every individual step. **All seven steps were carried out on 2026-10-07 (commit `9ae1862c`).**

1. **Fix the table-name harvester** — make the pattern accept an optional schema prefix (`public.users` → `users`), and add `mv_dashboard_totals` to the hand-written list as a reporting table.
   *Test-only change. Must come first, otherwise step 2's new failure stays invisible.*
   **Result:** done — `createRe` fixed, `mv_dashboard_totals` registered as `analitik`; the checker immediately started reporting the §2.3 breach, as predicted.
2. **Deal with the fallout of step 1:** the hidden `sale → mv_dashboard_totals` breach will now fail the build. Move the `GetAllCompletedSalesStats` query into `internal/report` (report may read any table) while keeping the port's function signature unchanged, so no callers break.
   **Result:** done — query now `Repository.getAllCompletedSalesStats` in `internal/report/repository.go`; `GetAllCompletedSalesStats` removed from the `SaleStatsProvider` interface (the remaining two methods are untouched, so no caller broke); parity test relocated to `internal/report/dashboard_totals_test.go`, and `DashboardStats_Seeded` now refreshes the MV before asserting.
3. **Port `SearchAvailableProducts` onto the existing ports**, the same way its sibling `ListAddTermProductOptions` already works: get the product search from the `product` side and the stock check from the `StockReader` port. This also clears the `products` reads in §2.1 if the search logic moves with it.
   **Result:** done — new port methods `ProductMetaProvider.SearchProductOptions` / `ProductIDsByNameOrSKU` implemented by `product.MetaLookup`; `Service.availableForTerms` now serves both `ListAddTermProductOptions` and `SearchAvailableProducts`; the three list filters switched to `product_id = ANY($n)`. Both `products` (§2.1) and `product_stock` (§2.2) reads are gone.
4. **Promote `consignment` to strict checking** (allowing only its own 13 `consignment_*` tables) and **delete the debt-manifest entry** once step 3 is done — the checker fails on stale entries, so this can't be left behind.
   **Result:** done — `consignment` added to `strictModuleTables`, `crossContextDebt` is now an empty map, and the stale-entry assertion still passes.
5. **Register `appsettings` in the module list.**
   **Result:** done — added to `domainModules`. Only `internal/wiring` (the composition root, itself not a domain module) imports it, so nothing was rejected by the tighter coverage.

Optional polish, not urgent:

6. Teach the scanner to accept quoted table names (`FROM "sales"`).
   **Result:** done — `sqlKeywordRe` accepts quotes plus schema/alias qualifiers; behaviour pinned by the new `TestSQLTableRefPattern`. The migration harvester accepts quotes too.
7. State explicitly in the ADR that `cmd/dummy` is out of scope, so its omission reads as a decision rather than an oversight.
   **Result:** done — ADR §5.4 now names `cmd/dummy` and `internal/shared/testdb.go` as deliberately out of scope; §5.2 gained the `consignment` strict row and §5.3 records the now-empty debt map with the ports that replaced it.

**Verification:** `go test ./internal/archtest` (ownership + import boundaries + the new pattern test), plus the `report`, `sale`, `consignment`, and `product` package tests against the dev database — all green. `go vet` and `gofmt -s` clean on every touched file. Run the package tests with `-p 1`: the packages share one test database and collide when run in parallel.

**Residual (accepted):** table names assembled at runtime remain invisible to the scanner (§3e) — re-checked, none exist; a future one would have to be reviewed by hand.

---

## 5. How to re-run this check yourself

```bash
# the built-in checker
go test ./internal/archtest -run TestModuleSQLTableOwnership -v   # table ownership
go test ./internal/archtest -run TestModuleImportBoundaries -v    # import rules
go test ./internal/archtest -run TestSQLTableRefPattern -v        # scanner pattern (quotes/schema)

# independent: list every table one module's SQL touches
rg -o --no-filename '\b(FROM|JOIN|INTO|UPDATE)\s+[a-z_]+' internal/<module> -g '*.go' -g '!*_test.go'
```

Useful follow-ups after any port:

- delete the `crossContextDebt` entry you just cleared — the test fails on stale entries;
- if you add a table, put it in `tableContext` *and* in the owning module's `strictModuleTables` row (and update the ADR §5.2 table).
