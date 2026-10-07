# Audit: Queries That Cross Module Boundaries

| Field | Value |
|---|---|
| Date | 2026-10-07 |
| Question | Does any code run a database query against a table that belongs to a *different* module? |
| Where the rules live | `docs/adr/ADR_Modular_Monolith_Module_Boundaries.md` (§2.8 = who owns which table, §2.9 = how it's checked) |
| Where the checker lives | `internal/archtest/archtest_test.go` |
| How we checked | Ran the checker (`go test ./internal/archtest` — it passed), then independently listed every table each module's SQL touches and compared that list against the ownership rules |
| Not covered here | Import rules (which module may `import` which), and cross-module *writes* that go through the official "port" mechanism (those are allowed on purpose) |
| Status | **All seven steps of §4 are implemented** (2026-10-07) and verified with `go test ./internal/archtest` plus the `report`, `sale`, `consignment`, and `product` package tests. Findings 1–3 and checker holes (a)–(d) are fixed; the scanner now also accepts quoted/schema-qualified names (step 6), and the ADR records `cmd/dummy` as out of scope (step 7) alongside the consignment/appsettings hardening. §1–§3 below still describe the *before* state. |

---

## 0. A short glossary (read this first if you're new)

- **Module** — a self-contained folder of the codebase with one responsibility, e.g. `internal/sale` handles sales, `internal/product` handles products. The project rule is *"one module, one job, don't reach into someone else's folder."*
- **Table** — a database table, e.g. `sales`, `products`. Each table has exactly one **owner module** — only that module may query it directly.
- **Query** — a SQL statement. A **read** (`SELECT`) looks at data; a **write** (`INSERT` / `UPDATE` / `DELETE`) changes it.
- **Cross-module query** — a query written by module A that touches a table owned by module B. E.g. `internal/consignment` running `SELECT ... FROM products` — `products` belongs to `internal/product`, not to consignment.
- **Port** — the sanctioned workaround. If consignment needs product data, `product` publishes a small function (a "port") that consignment may call. The table itself stays private. Analogy: asking the kitchen for a dish instead of walking into the kitchen and cooking it yourself.
- **The checker (archtest)** — an automated test that reads every source file, extracts the table names out of each SQL string, and fails the build if a module touches a table it doesn't own. "Green" = the test passes.
- **Strict vs. context-level** — two strictness levels in the checker. *Strict* = the module may touch only the tables on its own whitelist, nothing else. *Context-level* = looser: the module may touch any table in its whole **group** (e.g. everything in the "transaksional"/transactional group), even tables owned by neighbouring modules. Almost every module is strict; a few are not yet.
- **Debt manifest (`crossContextDebt`)** — a "known problems" list inside the checker. A violation listed there doesn't fail the build; it just stays on the backlog until someone ports it properly. The test also fails if a listed entry becomes *outdated* (already fixed but never removed), so the list can't rot.
- **Read model / `mv_*`** — pre-aggregated reporting tables (materialized views) used by dashboards. The `report` module is allowed to read *any* table to build its charts, but must never change data.

---

## 1. Verdict

**No forbidden writes.** Every place where one module changes another module's data goes through a proper port (`StockAdjuster`, `MovementWriter`, `StockApplier`, ...). That part of the architecture is healthy.

**Reads are a different story.** Five groups of cross-module reads exist:

| # | Who reads whose table | Where | What we make of it |
|---|---|---|---|
| 1 | `consignment` reads `products` | `internal/consignment/repository.go:472`, `:484`, `:812`, `:953`, `:1549` | **Known problem.** Already listed in the debt manifest, so the checker tolerates it on purpose. Still should be fixed. |
| 2 | `consignment` reads `product_stock` | `internal/consignment/repository.go:497` (the `SearchAvailableProducts` function) | **Slipped through by accident.** Not listed in the debt manifest — it passes only because consignment hasn't been promoted to "strict" checking yet. |
| 3 | `sale` reads `mv_dashboard_totals` | `internal/sale/report_adapter.go:47` | **Invisible violation.** It *is* a rule breach, but the checker can't see the table at all (see §3a), so it never fails. |
| 4 | `report` reads `product_stock`, `categories`, `mv_daily_sales`, `mv_hourly_sales` | `internal/report/repository.go:374` and the `mv_*` queries | **Allowed on purpose.** The reporting module may read anything (see glossary) — and it never writes anything. |
| 5 | `audit` writes `audit_logs`, reads `stores`/`users` | `internal/audit/repository.go:45`, `:75` | **Allowed on purpose.** Logging is shared infrastructure used by every module; the ADR records this exception explicitly. |

Checked and found clean:

- `internal/shared/testdb.go` writes `stores` / `schema_migrations` — this only runs in test setup, not in the real application.
- `internal/shared/report.go` contains SQL text for `sales` / `sale_items`, but that SQL is actually executed by `internal/sale/report_adapter.go` — the owner of those tables. The wording lives in `shared`; the execution happens in the right place. That's the correct pattern.
- `pkg/*`, `middleware`, `permissions`, `wiring`, `config`, `secregtest`, `metrics` contain no SQL at all.
- Every strict module (`brand`, `category`, `customer`, `customergroup`, `inventory`, `platform`, `pricing`, `product`, `purchase`, `sale`, `shift`, `stockopname`, `storagelocation`, `store`, `supplier`, `uom`, `user`, `appsettings`) touches only its own tables.

---

## 2. The five cases, explained one by one

### 2.1 `consignment` reads `products` — a known, tracked problem

Five queries in the consignment module search products by name or SKU (product full-text search):

- `SearchAvailableProducts` — checks whether a product exists, then searches it (`repository.go:472`, `:484`)
- `ListConsignmentStockPaged` — looks up a product inside a filter (`repository.go:812`)
- `ListReceipts` — joins product details onto receipt rows (`repository.go:953`)
- `ListSettlements` — same idea for settlements (`repository.go:1549`)

Consignment does not own `products` — `product` does. This is written down in the debt manifest (`crossContextDebt` at `archtest_test.go:237`), which is why CI stays green. Think of it as a post-it note on the fridge: *"we know, we'll fix it."* The checker also forces us to delete the note once the fix lands, so it can't be forgotten.

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

### 2.3 `sale` reads `mv_dashboard_totals` — a violation the checker cannot see

`internal/sale/report_adapter.go:47`:

```sql
SELECT COALESCE(SUM(total_revenue), 0), COALESCE(SUM(transaction_count), 0)
FROM mv_dashboard_totals
```

The `sale` module is strict: it may touch only its six tables (`sales`, `sale_items`, `sale_payments`, `payment_methods`, `cart_sessions`, `cart_items`). `mv_dashboard_totals` is a reporting view owned by `report`.

It passes anyway for a boring technical reason: the checker keeps a hand-written list of known tables (`tableContext`), and `mv_dashboard_totals` was never added to it — and the checker's backup way of harvesting table names from the database migrations is broken (§3a). A table the checker doesn't know about is a table the checker ignores.

There's a nuance here: the surrounding *call* goes through a proper port (report's `GetAllCompletedSalesStats` interface, `internal/report/ports.go:19`) — but the SQL itself is written and run inside `sale`. The right module to host this query is `report`, which is allowed to read anything.

### 2.4 `report` reads — allowed by design

The reporting module reads `product_stock`, `categories`, `mv_daily_sales` and `mv_hourly_sales`. That's its job: dashboards need data from everywhere. The rule for `report` is *"read anything, change nothing"* — and we confirmed there is no `INSERT`, `UPDATE` or `DELETE` anywhere in `internal/report`.

### 2.5 `audit` writes — a deliberate exception

`internal/audit` inserts into and prunes `audit_logs`. The ownership table assigns `audit_logs` to the `user` module, but the ADR (§5.2) records audit as shared infrastructure that sits outside the module system entirely — every module is allowed to log.

---

## 3. Holes in the checker itself

The checker is good but has six blind spots.

### (a) The backup table-name harvester doesn't work

To catch tables that exist in the database but were forgotten in the hand-written list, the checker scans the migration files for `CREATE TABLE` / `CREATE VIEW` statements (`archtest_test.go:289`).

The problem: every one of the 65 statements in `000_baseline.sql` is written with a schema prefix —

```sql
CREATE TABLE IF NOT EXISTS public.users
```

— and the pattern grabs only the first word, so it captures **`public`** instead of `users`, every single time. The backup harvest reduces to the single useless entry `{"public"}`.

Consequence: a table created by a migration but missing from the hand-written list is **invisible** to the check — the code quietly skips any reference to it (`archtest_test.go:324-326`). That's exactly how finding §2.3 stays hidden.

### (b) `mv_dashboard_totals` is missing from the hand-written list

Its two sibling views (`mv_daily_sales`, `mv_hourly_sales`) are listed as belonging to the reporting group; this one isn't. Combined with hole (a), there is no path by which the checker could ever learn about it.

### (c) `consignment` hasn't been promoted to strict checking

Because consignment is still on the loose *context-level* rule, it could read **or write** any table in the transactional group — `sales`, `sale_items`, `sale_payments`, `shifts`, `cash_movements`, `cart_sessions`, `cart_items`, `product_stock`, `inventory_movements`, `payment_methods` — without the test failing. Today it only reads (§2.1, §2.2), so nothing is actively broken, but the guard rail is missing for this one module.

### (d) `appsettings` isn't registered as a module

The list of modules the checker walks over (`domainModules`, `archtest_test.go:19-23`) leaves out `appsettings`, so its SQL is never inspected — it's only covered by the import rule. Today it touches a single table (`app_settings`), so nothing is currently wrong; it's a latent gap.

### (e) Ways the scanner could be fooled (none used today)

- **Quoted table names:** `FROM "sales"` doesn't match the scanner's pattern because of the leading quote mark. Only `internal/shared/testdb.go:271` quotes a table name; no module code does.
- **Table names built at runtime:** something like `fmt.Sprintf("SELECT ... FROM %s", tableName)` would also slip past. We checked — every `fmt.Sprintf` SQL in the codebase only injects filter fragments or `$1` placeholders, never a table name.

### (f) The data seeder writes every table

`cmd/dummy` (run by `./seed-dev.sh`) inserts into essentially every table. It's a developer tool, not part of the application, and the module rules deliberately don't cover it. Noted for completeness only.

---

## 4. What we recommend doing

Ordered so the build stays green after every individual step:

1. **Fix the table-name harvester** — make the pattern accept an optional schema prefix (`public.users` → `users`), and add `mv_dashboard_totals` to the hand-written list as a reporting table.
   *Test-only change. Must come first, otherwise step 2's new failure stays invisible.*
2. **Deal with the fallout of step 1:** the hidden `sale → mv_dashboard_totals` breach will now fail the build. Move the `GetAllCompletedSalesStats` query into `internal/report` (report may read any table) while keeping the port's function signature unchanged, so no callers break.
3. **Port `SearchAvailableProducts` onto the existing ports**, the same way its sibling `ListAddTermProductOptions` already works: get the product search from the `product` side and the stock check from the `StockReader` port. This also clears the `products` reads in §2.1 if the search logic moves with it.
4. **Promote `consignment` to strict checking** (allowing only its own 13 `consignment_*` tables) and **delete the debt-manifest entry** once step 3 is done — the checker fails on stale entries, so this can't be left behind.
5. **Register `appsettings` in the module list.**

Optional polish, not urgent:

6. Teach the scanner to accept quoted table names (`FROM "sales"`).
7. State explicitly in the ADR that `cmd/dummy` is out of scope, so its omission reads as a decision rather than an oversight.

---

## 5. How to re-run this check yourself

```bash
# the built-in checker
go test ./internal/archtest -run TestModuleSQLTableOwnership -v   # table ownership
go test ./internal/archtest -run TestModuleImportBoundaries -v    # import rules

# independent: list every table one module's SQL touches
rg -o --no-filename '\b(FROM|JOIN|INTO|UPDATE)\s+[a-z_]+' internal/<module> -g '*.go' -g '!*_test.go'
```
