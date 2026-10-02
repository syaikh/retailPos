# Master Data Store Boundary Audit & Fix Plan

> **Status:** Proposed — not started
> **Created:** 2026-09-30
> **Scope:** Master-data modules other than `products`, reviewed against the store boundary.
> **Related:** [Store Scoping: All Non-Superadmin Roles](./store-scoping-all-roles.md) · [Storage-Location Store Boundary Plan](./storage-location-store-boundary-plan.md) · [Store-First & Finance Role](./store-first-and-finance-role.md) · [Role/Permission Audit](./role-permission-audit.md)

---

## 1. Method

`cmd/server/main.go:174-201` registers every module on a single `/api` group carrying
`authMiddleware` + `CSRFMiddleware` + `RequireStoreID()`. `internal/middleware/auth.go:166-175`
restricts the `RequireStoreID` bypass to `superadmin` alone — manager was already removed from the
bypass list, so `docs/design/store-scoping-all-roles.md` §1 is **done**.

That means every non-superadmin JWT already *carries* a store id. The open question for each module
is narrower: **does the handler read `shared.GetStoreID(c)` and does the repository filter on it?**

Two schema shapes exist for a nullable `store_id`:

- **NULL = global / shared** — `products`, `suppliers`, `pricing_rules`, `warehouses`. A NULL row is
  an HQ row visible to every store.
- **NULL = single-store artifact** — `audit_logs`, `sales`, `shifts`, `import_jobs`, whose scope is
  derived from the parent transaction.

Judging which shape a table uses is part of this audit.

### 1.1 Archtest constraints that bound the fix

`internal/archtest/archtest_test.go:143-200` holds the **strict** module→table allowlists. Every
Tier 3 module except `user` is already single-table, so store scoping is a same-table predicate and
**needs no new cross-module port** (unlike `storagelocation`, which had to route warehouse→store
resolution through `store.ExistenceProvider`):

| Module | May touch only |
|---|---|
| `pricing` | `pricing_rules` |
| `supplier` | `suppliers` |
| `shift` | `shifts`, `cash_movements` |
| `user` | `users` |

---

## 2. Classification

### Tier 1 — Global by design (no boundary needed)

No `store_id` column; shared catalog/reference data referenced *by* `products`. A store-owned brand
or unit would break cross-store product references, so global is correct.

`brands` · `categories` · `units_of_measure` · `customer_groups` · `payment_methods` · `tax_classes` ·
`roles` · `permissions` · `role_permissions` · `app_settings` · `stores` (is the store)

`internal/brand/handler.go`, `category/handler.go`, `uom/handler.go`, `customergroup/handler.go`
never call `GetStoreID` — the only `StoreID` occurrences are audit-log literals. **Intentional.**

### Tier 2 — Store-scoped and correctly enforced (use as reference patterns)

| Module | Pattern to copy |
|---|---|
| `products` | Create/Update **stamp** `StoreID` from claims (`product/handler.go:258,317`) so a body `store_id` cannot be spoofed; reads use `store_id IS NULL OR = $n` (`repository.go:217,253,273`). **Gold standard.** |
| `storage_locations` | All 7 methods scope; `ErrStoreForbidden` → 403 via `writeError` (`handler.go:40-51`). |
| `customers` | All 8 methods scope; repo `store_id = $2 OR is_walk_in` (`repository.go:53,85`); writes `AND store_id = $n`. |
| `stores` / `warehouses` | `requireOwnStore` (`store/handler.go:44-50`) at `:156,184,255,314`; `Create` unscoped by intent. |
| `consignment` | 12 of 13 methods scope; 7 `NOT NULL store_id` headers, repos filter `store_id = $n`. |

### Tier 3 — Real gaps

| # | Module | Severity | Gap |
|---|---|---|---|
| 1 | `pricing` | **Critical** | 6 unguarded mutations, client-supplied `store_id`, reachable by supervisor |
| 2 | `supplier` | **Critical** | Gated on `pricing.*` — no `supplier.*` codes exist; cashier reads all, supervisor mutates all |
| 3 | `user` | High | Create/Update accept body `store_id` — authz, not just data leak |
| 4 | `shift` | High | `OpenShift` takes body store; by-id reads use user-not-store scope |
| 5 | `consignment` | Low | `EditReceipt` is the single unscoped method |
| 6 | `audit` | Low | `GetAuditLogByID` cross-store read by id |
| 7 | schema | Low | 4 store-scoped tables have `store_id` but no FK to `stores(id)` |

---

## 3. Tier 3 findings in detail

### 3.1 `pricing` — 6 unguarded mutations (highest blast radius)

Reads are scoped; every write is not.

| Scoped | Unscoped |
|---|---|
| `ListRules` (read `:146`, pass `:157`) | `CreateRule:212` |
| `GetRule` (post-hoc 403 `:190-198`) | `UpdateRule:254` |
| `CheckConflicts` (stamp `:471`, filter `:484-495`) | `DeleteRule:309` |
| | `SubmitForApproval:362` |
| | `ApproveRule:386` |
| | `RejectRule:410` |

Consequences:

1. `Rule.StoreID` is body-bound (`domain.go:86`) and NULL means **global** — a manager can create a
   NULL-store rule that overrides pricing in every store.
2. All six routes gate on a single permission each (`handler.go:61-67`): `approve`, `reject`, and
   `submit` all require only `pricing.update`. There is **no separation of duties** — a store-scoped
   manager can author, submit, and approve their own rule. *Fixed by 2c/2d + `054_pricing_rule_created_by.sql`:
   approve/reject need their own codes, and a creator can no longer approve their own rule.
   `submit` is gone entirely — see 2g.*
3. `CheckConflicts` is scoped but `Create` is not, so conflict detection is advisory: posting straight
   to `POST /pricing-rules` bypasses it entirely. *Fixed by 2h's reset: a rule only becomes live via
   approval, and approval is where `CheckConflicts` runs.*
4. A manager can `PUT`/`DELETE`/`approve` a **foreign** store's rule by id. *Fixed by 2b
   (`authorizeStoreRule`) and 2f (`ruleForAction` on every single-rule route).*
5. **The permission that unlocks this is held below manager.** From the baseline grants
   (`000_baseline.sql:5046-5052`): **supervisor** holds `pricing.create`, `pricing.update`, and
   `pricing.delete`. So a shift lead — not a store boss — can rewrite pricing globally. This is the
   sharpest finding in the audit.

### 3.2 `supplier` — gated on the wrong permissions, plus a dead column

Two independent defects in one module.

**(a) There are no `supplier.*` permission codes.** `internal/permissions/permissions.go` defines
none, and `000_baseline.sql` grants none. All 13 supplier routes are gated on pricing permissions
(`internal/supplier/handler.go:47-62`):

| Route group | Permission used |
|---|---|
| `GET /suppliers`, `GET /suppliers/:id`, `GET /suppliers/:id/products`, `GET /products/:id/suppliers` | `pricing.view` |
| `POST /suppliers`, `PUT /suppliers/:id`, `PUT /suppliers/bulk`, all `/products` link writes, `POST .../preferred` | `pricing.create` / `pricing.update` |
| `DELETE /suppliers/:id`, `DELETE /suppliers/bulk` | `pricing.delete` |

Consequence, from the same grants: **cashier holds `pricing.view`** (`000_baseline.sql:5006-5013`), so
any cashier can `GET /suppliers` and read every supplier in the chain — name, code, contact person,
email, phone, address, notes. **Supervisor holds `pricing.create`/`update`/`delete`**, so a shift lead
can create, edit, and delete any supplier globally. Supplier administration was never its own
permission surface.

**(b) `Supplier.StoreID` is dead.** `domain.go:37` declares it, `000_baseline.sql:1586` has the
column, and it has **zero SQL references** — the only two hits repo-wide are `domain.go:37` and a
test. `GetByID:55-77`, `GetByIDs:79`, `GetByCode:125`, `GetAll:192`, `Create:149`, `Update:166`,
`Delete:181`, `GetAllForExport:453` all omit `store_id` from their column lists. The seeder never sets
it, so every seeded row is `store_id = NULL`.

This outranks the original placement in this audit: the store boundary is moot until the module stops
being reachable by a cashier. See §5 for the scoping decision; the permission defect is fixed in
Wave 2c regardless of which option is chosen.

### 3.3 `user` — body-bound `store_id`

Only `ListUsers:161` scopes (repo filter is permissive: `AND (u.store_id IS NULL OR u.store_id = $n)`,
`repository.go:160,196` — global users visible to all, which is intended for HQ accounts).

Unscoped: `CreateUser` (`StoreID: req.StoreID`, `handler.go:220`; only validated as "required for
operational role" at `:199`), `UpdateUser` (`:341-342`), `DeleteUser:498`, `GetSubordinates:533`,
`GetManager:550`, `GetOrgChart:563`.

A store-scoped admin can mint or reassign an account in **any** store. Unlike the other findings this
is an authorization defect, not a read-scope leak: it manufactures identities outside the caller's
boundary. Note manager holds `user.create`/`user.update`/`user.view` but **not** `user.delete`
(`000_baseline.sql:5030-5036`), so the delete path is currently superadmin-only — the create/update
paths are the live exposure.

### 3.4 `shift` — body store on write, user-not-store scope on read

- `OpenShift:83-124` reads `store_id` from the **request body** (`StoreID *int`, `:85` → `:105`/`:124`),
  while `ListShifts:234` / `ExportShifts:262` scope by claims (repo `AND (s.store_id IS NULL OR
  s.store_id = $n)`, `repository.go:354`). A cashier can open a shift in another store.
- `GetActiveShift:198`, `GetShiftByID:410`, `ReviewShift:433`, `AuditShift:473`,
  `CreateCashMovement:540`, `ListCashMovements:576`, `GetShiftReport:592` rely on
  `ownership.Scope` (`handler.go:61-67`). `internal/ownership/ownership.go:29-33` resolves
  **user-level** ownership only — there is no store dimension. Any holder of the shift permission can
  read or audit another store's shift by id.

### 3.5 `consignment` — `EditReceipt`

`handler.go:275` calls `h.svc.EditReceipt(...)` with no `shared.GetStoreID(c)` and no
`claimsStore`; the service (`service.go:719`) validates only the edit window and item structure. The
sole inconsistency in an otherwise fully-scoped 698-line handler. `GetArrangement` already
post-checks via `checkArrangementStore` (`service.go:150`) — the same guard is what `EditReceipt`
needs.

### 3.6 `audit` — `GetAuditLogByID`

`ListAuditLogs:98` and `ExportAuditLogs:164` both scope (`AND (al.store_id IS NULL OR
al.store_id = $n)`, `repository.go:171,211`). `GetAuditLogByID:120` is a cross-store read by id.

### 3.7 Schema — 4 missing FKs

| Table | `store_id` | FK to `stores(id)` |
|---|---|---|
| `customers` | `integer DEFAULT 1 NOT NULL` | ❌ |
| `users` | `integer` | ❌ |
| `goods_receipts` | `integer NOT NULL` | ❌ |
| `purchase_orders` | `integer NOT NULL` | ❌ |

The other 21 store-scoped tables all have `REFERENCES stores(id)`. These four can hold orphaned store
references. `customers` additionally hard-codes the baseline store invariant at the DDL level
(`DEFAULT 1`), unlike every other store-scoped master table.

---

## 4. Sequenced implementation plan

Sequencing principle: **blast radius first, cheapest pattern-reuse first within a wave.** Each wave
is independently shippable, independently revertable, and ends green.

### Wave 0 — Decisions and baseline (no code)

Three business decisions are needed before Waves 2 and 5. Record them here so they are not decided
implicitly inside a diff.

- [x] **D1 — Pricing global rules** (§3.1 / Wave 2a). *Answered and implemented — see 2a:* Who may create a `store_id IS NULL` pricing
      rule? Proposed: superadmin only.
- [x] **D2 — Supplier role mapping** (§3.2 / Wave 2c). *Answered and implemented — see 2c:* Which role may read, and which may
      create/update/delete suppliers? Proposed: `supplier.view` to manager + supervisor +
      superadmin; create/update/delete to manager + superadmin.
- [ ] **D3 — Supplier store scoping** (§5). Option A / B / C. *Can be deferred until after Wave 2c
      lands* — see the sequencing note in §5.
- [x] **D4 — Was cross-store user creation a supported practice?** (Wave 3) Manager holds
      `user.create`/`user.update`; if HQ used managers to provision users in other branches, enforcing
      the boundary changes a workflow, not just a bug. **Answered: no.** It is not a supported
      workflow, so the boundary is a bug fix rather than a workflow change. Enforced in Wave 3 as
      `bindStoreScopedUser` — see that wave for the exact rules.
- [x] Add a `store_id` scoping note to `docs/design/role-permission-audit.md` and
      `store-first-and-finance-role.md` stating the enforced rule: **all roles except superadmin are
      scoped to their assigned store** (already true of `RequireStoreID`; this documents it).
- [ ] Record which of the 4 FK-gap tables hold orphaned `store_id` values today
      (`SELECT ... WHERE store_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM stores WHERE id = ...)`),
      because Wave 6 will fail on real rows.
- [ ] Confirm `suppliers` has no `store_id` outliers before Wave 5 (expected: all NULL).

**Exit criteria:** D1–D4 answered in writing.

### Wave 1 — Single-method holes (pattern already proven)

Deliberately first: smallest diffs, both copy a guard that already exists in the same file, and they
validate the audit itself before anything expensive is touched.

- [x] **1a `consignment.EditReceipt`** (`handler.go:275`) — `shared.GetStoreID(c)` passed to the service;
      signature extended with `claimsStore *int`; the receipt is checked against the caller's store
      **before** the edit-window validation, so a foreign receipt cannot be probed for window state.
      Superadmin (`nil`) bypasses. `ErrStoreForbidden` → 403 in `writeError`.
- [x] **1b `audit.GetAuditLogByID`** (`handler.go:120`) — `shared.GetStoreID(c)` passed through
      handler → service → repository; repository gains a `storeID *int` param and adds
      `AND (al.store_id IS NULL OR al.store_id = $n)` when non-nil, matching the list query. A foreign
      row surfaces as `pgx.ErrNoRows` → **404, deliberately not 403**, so the endpoint cannot be used as
      an existence oracle for audit-log ids across the chain. Non-`ErrNoRows` errors return 500 — a DB
      outage must not be reported as "not found".
- [x] Tests: `TestService_EditReceipt_StoreBoundary` (403 foreign / 200 own / 200 superadmin) and
      `TestAuditRepository_GetAuditLogByID_StoreBoundary` +
      `TestHandler_GetAuditLog_StoreBoundary` (the latter asserts 404-not-403 through the handler, since
      a repository-only test would not pin the status code).

**Exit criteria:** 2 handlers, 0 new ports, 0 migrations.

### Wave 2 — `pricing` mutations + supplier permission surface (highest blast radius)

Do this as one unit. The two modules share the defect: supplier administration was built on
`pricing.*` permissions, so splitting them would leave a cashier reading the whole supplier chain
while pricing is being fixed.

- [x] **2a Decide the global-rule rule.** **Decided:** `store_id = NULL` (global) → **superadmin only**;
      `store_id = <caller>` → any store-scoped role holding the permission. A store-scoped role supplying a
      **foreign** id → 403. A store-scoped role supplying NULL or omitting the field → the API **pins the rule
      to the caller's own store** (201), rather than 403.

      *Why pin instead of 403:* JSON `null` is indistinguishable from an omitted field, so rejecting only the
      explicit-null form would be arbitrary. The security invariant is "a store-scoped role can never produce
      a global rule", which pinning enforces. Because the silent coercion could misreport the saved scope,
      `PricingRulesPage.svelte` hides the "All Stores" option for non-superadmin and defaults the form to the
      user's own store — the backend 403 remains the enforcement, the UI is not the control.
- [x] **2b Handler guard** — extracted `authorizeStoreRule` (`isAdmin` + `GetStoreID`) and
      `bindStoreScopedRule` (create/update store pinning) from the `ListRules` / `GetRule` idiom, applied to
      `CreateRule`, `UpdateRule`, `DeleteRule`, `SubmitForApproval`, `ApproveRule`, `RejectRule`. Every
      transition loads the existing row first and scope-checks **the existing row**, not just the incoming
      body, so a supervisor cannot `PUT` their own rule with a foreign `store_id` to move it out of their store.

      `ruleForAction` is the authorization gate, so it tolerates **only** `ErrRuleNotFound`: a transient DB
      error returns 500 and stops the request rather than being mistaken for a missing row and letting the
      caller through unchecked.
- [x] **2c Add real `supplier.*` permission codes** and re-gate `internal/supplier/handler.go:47-62`.
      Seed in `000_baseline.sql` in dot notation. Suggested split mirroring the rest of the catalog:

      | Code | Grants |
      |---|---|
      | `supplier.view` | read suppliers, their product links, and the supplier picker |
      | `supplier.create` | `POST /suppliers` |
      | `supplier.update` | `PUT /suppliers/:id`, `PUT /suppliers/bulk`, all product-link writes, `set-preferred` |
      | `supplier.delete` | `DELETE /suppliers/:id`, `DELETE /suppliers/bulk` |
      | `product.cost.view` | **already exists** — gate the `unit_cost` field in supplier/product-link responses on it (see below) |

      Role mapping **confirmed with the user** and applied: manager + superadmin for
      create/update/delete, manager + supervisor + superadmin for view, and `pricing.create` /
      `pricing.update` / `pricing.delete` **removed from the supervisor grant** (it keeps
      `pricing.view`) so a shift lead stops rewriting global pricing. Cashier, finance, and
      inventory staff get no supplier code.

      Implemented. Grants: cashier 19, finance 6, inventory 17, manager 84, superadmin 90,
      supervisor 56 — 272 total. `TestBaselineRoleGrantsMatchPolicy`
      (`internal/permissions/baseline_test.go`) pins this so a later seed edit that re-grants
      pricing.write to a shift lead fails in CI rather than in production.
- [x] **2d Split approve/reject from edit.** `approve` and `reject` no longer share `pricing.update`;
      both now demand `pricing.approve` (manager + superadmin).

      Self-approval is blocked on **`pricing_rules.created_by`**, the rule's author — not its last
      editor, since the author is who wrote the price. The column did not exist, so
      `054_pricing_rule_created_by.sql` adds it as a nullable FK `ON DELETE SET NULL`: nullable so
      pre-existing rules stay approvable (a NULL author is not a self-approval), and `SET NULL` so
      deleting a user cannot leave an unapprovable rule or block the user delete. `CreateRule` sets
      it from the verified token; `Repository.Update` never writes it, so a later edit cannot
      launder authorship and make the rule approvable by whoever rewrote it.

      **Imports stamp it too.** `BulkInsertPricingRules` reads the author off the payload, and the
      import engine injects the importer's id into each row as `_user_id` beside the existing
      `_store_id` — same provenance, same reasoning: the identity comes from the token, never the
      CSV, which has no such column. Without this an imported rule would have a NULL author, which
      the guard reads as *legacy* and therefore approvable by anyone, making the import path a way
      to mint a price and then approve it yourself. `BulkUpdatePricingRules` still never writes
      `created_by`, so a re-import cannot reattribute an existing rule to whoever ran it. Covered by
      `TestMapToEntityTakesAuthorFromInjectedClaim` and
      `TestBulkInsertPricingRulesRecordsTheAuthor`.

      Superadmin is exempt: it is the escalation path for a rule whose author over-authored it, and
      blocking it would strand a rule permanently. `authorizeApproval`
      (`internal/pricing/handler.go`) implements this; `TestAuthorizeApproval` covers all six
      caller/author pairings including the NULL-author legacy case.

      **The edit route was a second way in, and is now closed.** `Rule.Status` carries a
      `json:"status"` tag, so `PUT /api/pricing-rules/:id` bound the status from the request body and
      wrote it straight through — a caller holding only `pricing.update` could approve their own rule
      with `{"status":"approved"}` and never reach either the `pricing.approve` gate or
      `authorizeApproval`. Since manager holds both codes, the new control was inert against exactly
      the role it targets. `UpdateRule` now takes `Status` from the already-loaded `existing` row, and
      `service.Update` re-reads it unconditionally rather than only when the caller left the field
      blank, so no caller can reintroduce it. `is_active` deliberately stays body-bound: deactivation
      is a legitimate edit and cannot activate a rule alone, since resolution requires
      `is_active = true AND status = 'approved'`. `TestUpdateCannotGrantApproval` covers both
      directions and pins the deactivation behaviour.

      This is also why `service.Update` no longer swallows a missing row: it previously discarded a
      `GetByID` error and reported success for a rule that did not exist.
- [x] **2e `unit_cost` exposure.** Confirmed leaking: four supplier responses serialized
      `ProductSupplier` directly (`GetProductsBySupplier`, `GetSuppliersByProduct`, `LinkProduct`,
      `UpdateProductSupplier`), so `supplier.view` alone exposed a purchase price.

      It was latent rather than live — every role currently holding `supplier.view` also holds
      `product.cost.view` — but the grants are unrelated and would diverge the moment a role is
      added. `internal/supplier/presenter.go` now mirrors `internal/product/presenter.go`: the field
      is **omitted**, not nulled, when the caller lacks `product.cost.view`, so a consumer cannot
      distinguish a hidden cost from a zero one. There is no superadmin bypass here; the seed
      grants superadmin `product.cost.view` already.
- [x] **2g Every rule starts `pending` and inactive.** Decided: creation never grants approval, and
      there is no immediate-active bypass on any path. `service.Create` no longer defaults an empty
      status to `approved`; it pins `StatusPending` + `IsActive = false` and ignores both in the body
      rather than rejecting the request, so a client round-tripping a rule object is not punished for
      echoing back the status it was given. The frontend create form omits `status` entirely, which is
      precisely how a UI-created rule used to land approved and active. `Approve` is the only
      transition that activates: `pending → approved` with `IsActive = true`. `Reject` also pins
      inactive, so a rejected rule cannot reach the till even if a later change made a pending rule
      active. Resolution (`GetActiveRules`) already required `is_active = true AND status = 'approved'`,
      which is what makes the self-approval check meaningful; `TestGetActiveRulesRequiresApprovedNotJustActive`
      forces `is_active` on behind the service's back to prove `status` alone is the gate.

      **The import path was a triple bypass, and it was the worst of the three.** `pricing_rules:import`
      is gated on `pricing.create`, so a *manager* can import. The adapter never populated
      `RuleImportPayload.StoreID` (the struct had the field; `MapToEntity` ignored the engine's
      injected `_store_id`), so every imported rule was inserted with `store_id = NULL` — a *global*
      rule, created by a store-scoped role, which is the thing 2a exists to prevent. And
      `BulkInsertPricingRules` hardcoded `StatusApproved` while taking `is_active` straight from the
      CSV, so imports skipped the approval workflow and activated on insert. `MapToEntity` now reads
      `_store_id` the way `internal/customer/adapter.go` and `internal/product/adapter.go` do, and the
      bulk insert pins pending + inactive.

      `BulkUpdatePricingRules` was worse than the insert, in a way the create-path decision does not
      name but the same two invariants cover. It wrote `status = 'approved'` and `store_id` **from the
      payload** — so an import could approve a pending rule, and since the adapter left `StoreID` nil it
      also *re-homed* an approved rule to global — and its `WHERE product_id = ? AND pricing_type = ?
      AND name = ?` had **no store predicate at all**, so a manager could update a rule belonging to
      any store by name. `status` is now absent from `SET` (only `Approve`/`Reject` may move it) and
      `store_id` moved from `SET` into the `WHERE` as `store_id IS NOT DISTINCT FROM $n`, so an import
      can only ever touch a rule inside the importer's own scope and can neither approve nor re-home it.

      **The schema default was the loophole under all of the code.** `pricing_rules.status` defaulted
      to `'approved'` and `is_active` to `true`, so any `INSERT` omitting the two columns — a future
      service, a `psql` fix, a script — produced an approved and active rule on insert. Every Go path
      now pins the values explicitly, which left the default unreachable from the application and
      reachable from the database, i.e. a strictly larger hole than the one just closed.
      `055_pricing_rule_new_rows_start_pending.sql` sets both defaults to `'pending'`/`false`, and the
      baseline `CREATE TABLE` is amended to match so a fresh install agrees. Changing a default touches
      new rows only, so already-approved rules stay active — the "existing rules remain until replaced"
      half of the decision, and it needs no data backfill.

      `repo.Create` still honours a caller-supplied status, and that is deliberate: its only production
      caller is `service.Create`, which pins both fields, and the test fixtures need to create
      approved rules directly. Pinning it in the repository would buy nothing and cost every
      fixture. `cmd/dummy`'s seeder also still writes approved active rules through raw SQL — it is a
      dev-only generator of *historical* sales, not a workflow entry point, and forcing its rules to
      pending would produce a dev database whose history references unapproved prices.

- [x] **2g `draft` retirement + approval state machine.** Found while implementing 2b, not a
      store-boundary bug: `SubmitRule` existed so a creator could send a `draft` for approval, but
      nothing else used `draft` — `CreateRule` inserted `approved`, and nothing could ever create
      one. So the "submit" path was reachable only by rows that predated the default, and the
      workflow had three states where two were real.

      **Decided:** `draft` and `submit` are retired, leaving `pending` / `approved` / `rejected`.
      A new rule is created `pending` + `is_active = false` and becomes live on approval, so the
      one row that actually exists in production — `approved` on create — is the invariant being
      fixed, not preserved.
      - `057_pricing_retire_draft_status.sql` drops the `chk_pricing_status` check and re-adds it
        over `(pending, approved, rejected)`. A `DROP CONSTRAINT IF EXISTS` + `ADD CONSTRAINT`
        pair rather than `ALTER ... DROP CONSTRAINT` so a second runner pass is a no-op; the
        constraint is verified in replay by re-inserting a `draft` row, which the new definition
        rejects. `055_pricing_rule_new_rows_start_pending.sql` had already moved the column
        defaults, so this migration only narrows the enum the check accepts.
      - `statusDraft`, `SubmitRule`, the `POST /pricing-rules/{id}/submit` route and the whole
        `dokumentasi/7-submit-flow.md` walkthrough go with it. Regenerating Swagger confirmed the
        route is the only removed path and the `status` enum is the only parameter change.
      - Frontend: the table's submit action, the draft status pill, the draft filter and the
        `submitPricingRule` service method are gone; `statusDraft`/`submit`/`ruleSubmittedApproval`/
        `failedToSubmitApproval` had no other consumer, so the i18n keys went too.
- [x] **2h `UpdateRule` does not demote an approved rule.** The inverse hole: `UpdateRule` wrote
      whatever the body said, so an approved rule could be re-saved with new economics and stay
      `approved` + `is_active` — approval bypassed by editing, with no second reviewer.
      **Decided:** an economic edit on an approved rule resets it to `pending` + `is_active = false`;
      a non-economic edit (name, description, priority) keeps its state. "Economic" is the same
      field set the audit's §3.1 notes call out as price-affecting, so the rule is stated in one
      place in the service rather than duplicated in the handler. The service is the enforcement
      point because the import path writes through it, so an import cannot demote silently either.
- [x] **2f `GetRule`'s existing post-hoc 403** (`:190-198`) becomes a load-time guard shared with 2b.

      `GetRule` and `UpdateRule` were the last two handlers that hand-rolled
      "load the row, then check the store boundary" instead of calling
      `ruleForAction`, so all seven single-rule routes now share one gate. The
      duplication was not only a second copy to keep in step: both collapsed
      *every* load error into a 404, so a dropped connection reported "pricing
      rule not found" and the real error never reached the logs. `ruleForAction`
      already had the right split — `ErrRuleNotFound` is a 404, anything else is
      a 500 that stops the request.

      The mock's `GetByID` returned `pgx.ErrNoRows` while production returns
      `ErrRuleNotFound`. That divergence was harmless while the two handlers
      only checked `err != nil`, and would have become a trap here:
      `errors.Is(pgx.ErrNoRows, ErrRuleNotFound)` is false, so the mock would
      have produced a 500 for every not-found case and made the 404 paths
      untestable. The mock now returns the domain sentinel.
      `TestPricingHandler_LoadFailureIsNotReportedAsNotFound` pins both outcomes
      for `GET` and `PUT`.

      This is the same shape as `requireOwnStore` in `internal/store/handler.go`
      and the planned `ensureStoreScope` in the storage-location plan: the check
      belongs at load, so a caller cannot observe a row before it is authorized.
- [ ] Tests: matrix over (superadmin / manager-A / supervisor-A / cashier-A / manager-B) × (global
      rule / own / foreign) × (create / update / delete / approve / reject), plus a supplier
      matrix asserting a cashier gets 403 on `GET /suppliers`.
      *Partly landed:* `internal/pricing/routes_permission_test.go` now asserts every pricing route
      demands *exactly* its own code (grant all-but-one → 403), which is the route-level half of the
      matrix, plus a dedicated unauthorized-approval case. `workflow_test.go` covers the state machine
      and both import paths. The full cross-role × cross-store matrix is still open.

**Exit criteria:** pricing and supplier are internally consistent; `CheckConflicts` can no longer be
bypassed; no sub-manager role can mutate either module.

**Risk:** High. This changes the role→permission mapping for two modules and removes permissions from
supervisor. User-visible. Consider splitting 2c/2d into their own PR behind a role-permission audit
and a `role-permission-audit.md` update.

### Wave 3 — `user` store assignment (authz)

One helper, `bindStoreScopedUser(c, requested, existing)` (`handler.go`), carries 3a and 3b. It is
deliberately **not** a copy of pricing's `bindStoreScopedRule`, because a user with `store_id IS
NULL` is a legitimate *global* (HQ) identity while a global pricing rule is superadmin-only:
`ListUsers` already exposes `store_id IS NULL` rows to every store-scoped caller, and
`permissions.OperationalRoles` treats a store-less non-operational account as valid. So a global row
is *in scope* here, and what is forbidden is writing a store that is not the caller's own.

| Case | Superadmin | Store-scoped caller |
|------|-----------|---------------------|
| Create, `store_id` omitted | global (HQ) user | pinned to the caller's store |
| Create, `store_id` = own | allowed | allowed |
| Create, `store_id` = foreign | allowed | **403** |
| Update own-store row, `store_id` omitted | store preserved | store preserved |
| Update own-store row, `store_id` = foreign | allowed (moves it) | **403** |
| Update own-store row, `store_id` = own | allowed | allowed |
| Update **global** row, `store_id` omitted | stays global | stays global |
| Update **global** row, `store_id` = any store | allowed | **403** "global user requires superadmin" |
| Update a foreign row | allowed | **403** |
| No store claim | n/a | **403** (fails closed) |

- [x] **3a `CreateUser`** — `bindStoreScopedUser` runs *before* the "required for operational role"
      check, so a manager provisioning a cashier is stamped with their own store instead of being
      400'd for omitting a field they may not choose. The check itself is preserved and still
      governs superadmin's global creates.
- [x] **3b `UpdateUser`** — the bound value replaces the old
      `if req.StoreID != nil { existing.StoreID = req.StoreID }`. Two behaviours fell out of routing
      both paths through one helper:
      - **A global row is not captured by naming a store.** The first cut only rejected a *foreign*
        store on a global row, which let a manager pull an HQ account into their own store by naming
        it — the exact escalation the rule exists to stop, one account at a time. Any named store on
        a global row is now refused.
      - **An omitted `store_id` inherits, for superadmin too.** JSON cannot distinguish an omitted
        field from an explicit `null`, so the superadmin path originally cleared the store on every
        partial update and tripped the operational-role guard. This was caught by the pre-existing
        `TestHandler_UpdateUser` suite, not by a new test.
      The "cannot remove `store_id` from an operational role" guard is kept, now reading the
      already-resolved effective store.
- [x] **3c Read paths** — `GetSubordinates`, `GetManager` and `GetOrgChart` take `claimsStore *int`
      through handler → service → repository, matching the Wave 1a naming. Each adds
      `(store_id IS NULL OR store_id = $n)`, reusing `ListUsers`' permissive shape.
      - `GetOrgChart` filters the **outer** `SELECT`, not the recursive CTE: the walk still
        traverses foreign nodes so filtering output cannot truncate the shown tree.
      - A foreign `GetManager` row is reported as `ErrManagerNotFound` → **404, not 403**, matching
        Wave 1b so the endpoint is not an existence oracle for user ids across stores.
      - `DeleteUser` is guarded in the handler, not the SQL: it now loads the target and refuses a
        foreign store **403** (it is a write, not a probe), and a load failure returns 404 without
        deleting. This changes a pre-existing tolerance — the load used to be best-effort, only for
        the audit description, so `TestAuditHandler_DeleteUser_GetUserError` asserted 200 and had to
        be rewritten to assert the delete does not run.
- [x] UI: `UserFormModal.svelte` disables the store select for non-superadmin and lists only the
      caller's own store; `UsersPage.svelte` opens the form on that store. Same treatment as Wave 2a,
      same caveat: the backend 403 is the control, the UI is not.
- [x] Tests — `TestUserRepository_StoreBoundary` (own-store + global returned, foreign dropped, nil
      claim unscoped, foreign manager 404) and `TestMockHandler_{Create,Update,Delete}User_StoreBoundary`
      + `TestMockHandler_ReadPaths_PassStoreClaim`.

**Exit criteria:** no identity can be created or moved outside the caller's store. Met.

### Wave 4 — `shift` store boundary

- [ ] **4a `OpenShift`** (`handler.go:83-124`) — stop trusting the body `store_id`; pass
      `shared.GetStoreID(c)`. Decide whether to keep accepting the field at all (reject a non-nil
      value that differs from claims → 403; silently overwrite is the alternative).
- [ ] **4b Extend `ownership.Scope` with a store dimension**, or introduce a sibling
      `ownership.StoreScope`. `internal/ownership/ownership.go:29-33` is user-level by construction
      and its doc comment says "shifts today; sales, stock opnames, ... later" — this is the intended
      extension point. Preserve `CanAccessAll` semantics: superadmin (`nil`) bypasses.

      **Blast radius — 3 consumers, not 1** (verified by direct grep, not the call graph, which
      conflates `ownership.Resolve` with `pricing.Resolver.Resolve`):

      | Consumer | Uses | Impact of a `Scope` change |
      |---|---|---|
      | `shift` | `Scope` in repo + service signatures (`repository.go:348,500`, `service.go:19,21,116,120,129`; `handler.go:30,31,37,61`) | Full — signatures change |
      | `sale` | `ownership.Resolve` + `OwnID` only (`handler.go:649,702`) | Low, if `Resolve`'s signature is unchanged |
      | `product` | `ownership.CanAccessAll` only (`presenter.go:15`, `adapter.go:285`) | None |

      Additive route preferred: keep `Scope` as-is and add `StoreID *int` to it, so `sale` and
      `product` compile unchanged. Also note `shift/repository.go:590,607,621` call
      `GetShiftByID(ctx, ownership.Scope{}, shiftID)` with a **zero-value** scope — those three
      internal paths currently bypass all filtering and must be given a real scope.
- [ ] **4c Apply to** `GetActiveShift:198`, `GetShiftByID:410`, `ReviewShift:433`,
      `AuditShift:473`, `CreateCashMovement:540`, `ListCashMovements:576`, `GetShiftReport:592`.
      Repository filters match the existing `ListShifts` shape (`repository.go:354`).
- [ ] Tests: cashier-A cannot open a shift in store B; cannot read/review/audit store B's shift by id
      even with a `shift.review` permission.

**Exit criteria:** cash reconciliation is store-closed.

**Risk:** Medium. `ownership` is shared infrastructure, but the additive shape limits the blast radius
to `shift`. Note supervisor holds `shift.audit` and `shift.review`
(`000_baseline.sql:5046-5052`) — the same class of over-grant as Wave 2c, worth re-checking.

### Wave 5 — `supplier` store scoping (blocked on the §5 decision)

Prerequisite: **Wave 2c must land first.** Creating `supplier.*` codes while leaving the store
boundary undecided would mean re-mapping permissions twice.

- [ ] Execute the chosen option: **wire up** (5a) or **remove** (5b), per §5.
- [ ] Whichever option: update `docs/design/role-permission-audit.md` and the README supplier
  section so the decision is recorded and does not get silently reversed.

**Exit criteria:** no column promises a boundary the code does not enforce, and supplier reads are
limited to roles that should have them.

### Wave 6 — Schema integrity (FK gaps)

- [ ] New migration `058_store_fk_integrity.sql`. Per AGENTS.md the baseline is amended in place, but
      the number is not free: 054–057 are taken by the pricing author column, the pricing status
      defaults, the supervisor pricing-grant revoke, and the `draft` status retirement. Per AGENTS.md the baseline is amended in place, but
  new migrations start at `054_*.sql`; a *constraint* added post-baseline belongs in a new migration so
  it replays in lexical order on every runner.
- [ ] Add `REFERENCES stores(id)` to `customers`, `users`, `goods_receipts`, `purchase_orders`.
  **Must be re-runnable** (`ON DELETE SET NULL` to match the 21 existing conventions, guarded with a
  `DO $$ ... IF NOT EXISTS (SELECT 1 FROM pg_constraint ...) $$` block, or `ALTER TABLE ... DROP
  CONSTRAINT IF EXISTS` then re-add).
- [ ] Decide `customers.store_id`'s `DEFAULT 1` — either drop the default (callers must pass a store)
  or keep it and document the baseline invariant.
- [ ] The Wave 0 orphan audit must be clean **before** this migration, or it will fail on real rows.
  Orphan cleanup is data migration and out of scope here — escalate if non-empty.

**Exit criteria:** all 25 store-scoped tables have a real FK; `pg_constraint` count matches the
baseline's documented total plus the 4 new ones.

**Risk:** Low for the constraint itself, but it will hard-fail at apply time if orphans exist.

---

## 5. Supplier decision: `StoreID` wire-up vs. drop

> **Read §3.2 first.** The permission defect (all 13 routes on `pricing.*`) is independent of this
> decision, outranks it in severity, and is fixed in Wave 2c regardless of the option chosen.

### Option A — Make suppliers store-scoped (wire the column up)

**Pros**

- **Fixes a latent confidentiality gap.** A manager sees every supplier's contact details and unit
  cost, across all stores.
- **Consistent inside the Referensi context.** `suppliers` sits alongside `customers`,
  `warehouses`, and `storage_locations` in the ADR table — all three of which *are* enforced.
  Suppliers are the odd one out.
- **The column and the DDL already exist.** `000_baseline.sql:1586`; no migration needed to add it,
  only to index/backfill.
- **The architecture already supports it.** `supplier` is a strict module touching only `suppliers`
  (`archtest_test.go:164-166`), so the filter is a same-table predicate — no new port, no
  cross-module call. The `product_suppliers` join is untouched.
- **Prevents the "shared supplier, store-specific terms" leak.** Consignment deals
  (`consignment_arrangements.store_id NOT NULL`) and purchase orders
  (`purchase_orders.store_id NOT NULL`) are store-scoped while their supplier is global, so
  per-store pricing/offers to one supplier become visible to all stores.

**Cons**

- **One supplier must become N rows to model a real distributor.** A vendor supplying 30 stores
  needs 30 supplier records. That is operationally absurd and a real source of bad data.
- **Breaks the seeded dev data.** Every existing supplier row is `store_id = NULL`; the seeder
  (`cmd/dummy/consignment.go:48-92`) picks the *first N suppliers globally* to flag as consignment
  and then creates arrangements across **different stores** for them. Store-scoping invalidates both
  steps and requires a seeder rewrite.
- **Migration is not additive.** It is a data migration: split or backfill every existing row,
  rewrite `consignment_arrangements`/`purchase_orders`/`goods_receipts` FKs, and handle the
  `product_suppliers` join (a store-owned supplier cannot serve a global product's preferred-supplier
  slot without a store dimension of its own). This dwarfs every other wave in this plan.
- **Buys little beyond the permission fix.** The store boundary that actually matters is *already*
  enforced one level down: `consignment_arrangements` and `purchase_orders` both carry
  `store_id NOT NULL` and filter on it. The deal is store-scoped even though the supplier record is
  not. Once Wave 2c restricts `supplier.view`/`update` to manager+, most of the exposure is closed
  without any schema work.
- **Front-end and export surface.** `web/src/modules/supplier/` and the import/export adapters
  (`BulkInsertSuppliers:375`, `GetAllForExport:453`, `BulkUpdateSuppliers:403`) all need a store
  dimension, plus a `SUP-XXXXXX` code-generation change (currently a single global `supplier_seq`,
  `repository.go:30-37`).
- **Highest cost, lowest marginal security gain** once the permission fix is in.

### Option B — Remove `Supplier.StoreID` (declare suppliers global)

**Pros**

- **Honest.** The schema stops advertising a boundary that does not exist. There is no way for a
  future reader to infer supplier rows are store-private.
- **Cheap and safe.** Column is already unread and unwritten — nothing in SQL, service, export, or
  seeder touches it. Removal is a `DROP COLUMN IF EXISTS` and a field deletion.
- **Matches current behaviour exactly.** Zero runtime risk, no data migration, no frontend change, no
  seeder rewrite, no code-generation change.
- **Consistent with the other Referensi globals** (`customer_groups`, `tax_classes`,
  `payment_methods`), which are also global with no `store_id`.

**Cons**

- **Does not fix the information leak.** A manager still reads every supplier's contact details and
  unit cost. If that matters, Option B trades a code-cleanliness problem for a real
  confidentiality gap.
- **Leaves commercial terms global.** A distributor's price to Store A is visible to Store B's
  manager — the per-store `unit_cost` in `product_suppliers` (`internal/shared/supplier.go`) is a
  *link*-level cost with no store dimension, so this gap survives the column removal entirely.
- **Reversal cost is asymmetric.** Re-adding the column later is trivial, but if the business later
  wants store-owned suppliers, the data migration in Option A still has to happen — just later, with
  more production data.
- **Needs a written decision.** Otherwise a future contributor re-adds the column, assuming it works.

### Option C — Hybrid: global supplier, store-scoped commercial terms *(recommended)*

Keep `suppliers` global; put the store boundary where the money is.

**Pros**

- **Matches the domain.** A supplier is a real-world trading partner; its identity (name, address,
  contact) is not a per-store attribute. Store-specific facts are the *terms*.
- **Mirrors what consignment already does.** `consignment_arrangements` is the store-scoped join over
  a global supplier — Option C generalises that existing, working pattern rather than inventing a new
  one.
- **No supplier data migration.** Supplier rows stay as they are; the column is dropped, not split.
- **Closes the real leak.** Store-specific prices/offers stop being visible across stores.
- **Cheaper than Option A, stronger than Option B.**

**Cons**

- **Requires a `store_id` on `product_suppliers`** (currently owned by `internal/product`; a strict
  module — `archtest_test.go:186-192`). That is a schema change to a module already hardened, plus
  a `ProductSupplierStore` port change consumed by `internal/supplier` (`ports.go`, wired via
  `SetProductSupplierStore`, `repository.go:42`).
- **`is_preferred` becomes store-relative.** Today a product has one preferred supplier globally;
  with store-scoped links, each store has its own. The `GetPreferredSupplier:257` /
  `SetPreferredSupplier:261` / `HasPreferredSupplier:324` contract changes, and purchase-order
  supplier selection follows.
- **Still two work items, not one** — but both are additive columns with backfill, not row splits.
- **Does not restrict who may *edit* a supplier.** A manager can still edit the global supplier's
  contact details. Mitigate with a permission (e.g. `supplier.edit_global`) if that matters, which
  is much cheaper than store-scoping the table.

### Recommendation

**Option C**, with Wave 2c's permission split as the primary fix and Option C's schema change as the
follow-up.

Rationale: Option A's data cost is high and its security benefit is largely already delivered by the
store-scoped `consignment_arrangements` / `purchase_orders` joins — plus the permission fix that must
happen either way. Option B is cheap but leaves the commercially sensitive per-store pricing exposed.
Option C fixes the part that actually matters (terms) with additive columns and no row splits, and
reuses a pattern the system already proves in consignment.

Sequencing insight: **2c first, then re-evaluate.** The permission fix is cheap, independently
correct, and removes most of the exposure. Once cashier and supervisor lose supplier access, decide
whether the residual cross-store visibility among managers actually justifies a data migration.
That is a business decision best made with the fix already in place, not before.

If the business genuinely wants per-store supplier *identity* — e.g. the same vendor is a different
legal entity per store, with different tax IDs — then Option A is correct and this becomes a
migration project, not a wave.

**Regardless of option: record the decision in `docs/design/` and remove or wire the column in the
same PR.** A dead column that reads as a security boundary is the worst of both worlds.

**Interim mitigation (any option, and it is not interim — it stands on its own):** Wave 2c's
`supplier.*` codes. Concretely, the verified state today is that **cashier holds `pricing.view`**
(`000_baseline.sql:5006-5013`) and therefore can read the entire supplier chain, and **supervisor
holds `pricing.create`/`update`/`delete`** (`000_baseline.sql:5046-5052`) and therefore can mutate
it globally. Both are one line of SQL in `000_baseline.sql` to stop, independent of the store
scoping decision.

---

## 6. Out of scope

- **Tier 1 modules** — global by design; adding a boundary would be a regression.
- **Tier 2 modules** — already correct; they are the reference patterns, not work items.
- **Cross-store analytics/reports** — out of scope for this audit; `report.RefreshCoordinator`
  materialised views are store-agnostic by construction and need a separate decision.
- **Frontend** — all scoping is server-side; 403s flow through existing `getApiErrorMessage` toasts.
  Two exceptions where the UI would otherwise misreport what the API actually stored:
  - Wave 2a/2b: `PricingRulesPage.svelte` hides the "All Stores" option for non-superadmin and defaults
    the store field to the user's own store, so the form cannot claim a global scope that create/update
    will pin to one store.
  - Wave 2c changes what a cashier can see in the supplier UI, so a role-visibility pass over
    `web/src/modules/supplier/` is expected alongside it. Done: `SuppliersPage.svelte` now reads
    `supplier.create/update/delete/view` instead of the borrowed `pricing.*`, and the `/suppliers`
    route gate is `supplier.view`.
  - Wave 2d adds a `pricing.approve` capability the pricing table has no prop for. Done:
    `PricingRulesTable.svelte` takes `canApprove` and gates the approve/reject menu items on it,
    leaving `canEdit` for submit. (At the time of 2d, submit was intentionally still
    `pricing.update`; 2g retired the submit path entirely, so only `canApprove` remains.)
- **Permission catalog beyond Wave 2c/2d** — `role-permission-audit.md` is updated in Wave 0, but
  no other module's role mapping is reworked here. The supervisor over-grant on
  `shift.audit`/`shift.review` (Wave 4) is flagged, not fixed.
- **Orphan-row data migration** (Wave 6 blocker) — escalate, do not silently write.

## 7. Verification

No local full lint/test runs (AGENTS.md) — CI runs gofmt/vet/golangci-lint/`go test`/archtest + e2e
(4 shards, full stack). Targeted local runs only on request:

- `go test ./internal/pricing/... ./internal/supplier/... ./internal/permissions/...`
- `go test ./internal/user/... ./internal/shift/... ./internal/ownership/...`
- `go test ./internal/consignment/... ./internal/audit/...`
- `go test ./internal/archtest/...`
- `npx playwright test <spec>` from repo root

E2E follows the AGENTS.md conventions: behavior specs assert via `api-driver.ts` with `apiAs(request,
role)`, fresh `ApiDriver` per test in `beforeEach`, `Date.now()` suffixes for shard safety, and
`tests/e2e/db-helper.ts` cleanup ordered FK-safe (locations → warehouses → stores).

**Existing regression spec to extend, not duplicate:** `tests/e2e/storage-locations-api.spec.ts`
already establishes the two-store `managerA`/`managerB` + `superadmin` control pattern this plan
should copy per module.

## 8. Rollout risks (deployed behavior changes)

These are silent on upgrade — no error, no failed migration, just a narrower permission set. Each needs
a release note, and the first two need a decision on how to handle existing rows.

- **Existing `store_id IS NULL` pricing rules become superadmin-only.** Every non-superadmin seed user has
  `store_id = 1`, so a shop that already created global rules loses non-superadmin management access to
  them on deploy. Nothing errors; the rules stop appearing in the management list for those roles
  (`ListRules` already filtered with `AND store_id = $n`). Decide: backfill the NULL rules to a specific
  store, or accept superadmin-only and say so in the release note. The rules still *apply* at checkout
  for every store — only management access narrows.
- **The 428-gate / global-rule interaction.** A shop with NULL-store rules and a store-scoped admin
  following the installation guide will be unable to manage them until either the rules are backfilled
  or they are granted superadmin. This is the intended tightening, not a bug, but it will look like one.
- **`UpdateRule` now 404s a missing id** (it previously returned 200 echoing the body, because
  `repo.Update` ignores `RowsAffected`). A client that relied on upsert-like behavior will now see a 404.
- **`GetRule` 403s store-scoped callers on global rules.** No UI impact: globals were never listed for
  those roles, and `getPricingRule` (`pricing-service.ts:65`) is an unused export.
- **Two `GetByID` reads per update/workflow transition** (handler + service). Not a hot path; noted only
  so a future optimization does not remove the handler-side one and silently reopen the hole.

**Never auto-commit.**
- **Supervisors lose pricing write access.** `pricing.create` / `pricing.update` / `pricing.delete`
  are removed from the supervisor grant, and cashiers, finance, and inventory staff lose the supplier
  routes they reached through the borrowed `pricing.*` codes. Supervisors keep `pricing.view` and
  `supplier.view`. Intended, but it is a capability removal for an existing role — worth a release note
  and a check that no shift-lead workflow depended on authoring pricing.
- **Pricing rules authored before `054_pricing_rule_created_by.sql` have no recorded author**, so a
  manager may approve them even if they wrote it. This is a deliberate trade: recording an author for
  historical rows would mean trusting `audit_logs`, which are deletable and record the last editor as
  well as the creator. The window closes for any rule created after the migration.
- **Managers can no longer approve a rule they authored**, and the UI does not hide the button — the
  response is a 403 with `you cannot approve your own pricing rule`. `created_by` is `json:"-"`, so the
  client cannot know who to disable the action for. If this proves noisy, exposing the author id to
  the client is the fix; the server-side check is the part that matters.

## 9. `sale.lookup` is intentionally absent from superadmin (not a defect)

`RequirePermission` does not bypass for superadmin (`internal/middleware/auth.go:100-119`) — it reads
the caller's grant list and nothing else. Superadmin holds 90 of 91 codes: **`sale.lookup` is
absent**, and `GET /sales/lookup` is gated strictly on it (`internal/sale/handler.go:81`). So
superadmin gets a 403 on cross-cashier lookup.

That is **correct, and was recorded as a decision** in `031_revoke_sale_lookup_manager.sql`
(preserved in `database/migrations/archive/pre-squash-migrations.tar.gz`):

> Find Transaction is a cashier-only capability. Managers, admins, and superadmins already see every
> cashier's sales in "My Transactions" via `report.view` (`ownership.CanAccessAll`), so the Find
> Transaction tab is both redundant and a weaker redacted subset for them. The frontend hides the tab
> bar entirely for `report.view` holders; this migration removes the now-unused manager grant so the
> permission is cashier-only.

Both halves of that decision are live: `TransactionsPage.svelte:48` gates the tab on `sale.lookup` and
hides the whole tab bar for `report.view` holders, and `useRBAC.test.ts:12` asserts superadmin's
matrix as `ALL_PERMISSIONS` minus `sale.lookup`. No supported client ever calls the route as
superadmin, so the 403 is unreachable in normal use — and granting it would widen the API surface
against a deliberate decision.

`TestBaselineRoleGrantsMatchPolicy` now *asserts the exclusion* (`notGranted(superadmin, SaleLookup)`)
rather than tolerating it as a gap, so the decision cannot be quietly reversed in either direction.
