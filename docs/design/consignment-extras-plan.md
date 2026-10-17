# Consignment Extras Plan — PRD §15 Deferred Items

> **Status:** Phases 1–3 implemented (backend + frontend); docs finalised
> **Source:** `docs/roadmap/upcoming-features.md:586`, `docs/prd/PRD-Konsinyasi-Supplier-v7-Consolidated-Full.md:805-816` (§15 Open / Future Considerations)
> **Prior plan:** `docs/archived-plans/consignment-supplier-plan.md:213-215` (deferral list only)

## Context

The core consignment workflow is shipped (arrangement → receipt → stock → sale → pending return → return → settlement → payout). PRD §15 listed ten open items that were deliberately deferred. This plan sequences the three that matter most and parks the rest with explicit re-trigger conditions.

**Already mitigated (do not rebuild):**

- §15.9 partial fix — "Return All" with reason `"termination"` is implemented (`docs/design/consignment-bulk-return-to-end.md`, status: Implemented; `ReasonTermination` at `internal/consignment/domain.go:68`). It eases bulk return before end, but the store still has no *sovereign* force-release path and no dedicated permission.
- §15.4 groundwork — receipt/return document numbers (`CR-`, `RT-`) are already stored (BR-11), so printing needs rendering only, no schema change.

## Scope

| Phase | PRD item | Verdict |
|-------|----------|---------|
| 1 — Forced-return release + permission | §15.9 | **Shipped** (data-lock risk) |
| 2 — Print documents (receipt/return) | §15.4 | **Shipped** (quick win) — browser print; print-agent deferred |
| 3 — Physical reconciliation | §15.10 | **Shipped** — schema + service + UI |
| Analytics dashboard | §15.5 | Parked — re-trigger on user request |
| Expiry-by-shelf-life rules | §15.1 | Parked — re-trigger only if perishables are in scope |
| Refund/void beyond eligible customer return | §15.3 | Parked — Sales Return PRD covers the real path (`PRD-Sales-Return.md:697`) |
| Payment flows beyond payout record | §15.8 | Parked — no concrete flow exists yet |
| Damage-reason classification, audit-trail detail, access-right detail | §15.2/§15.6/§15.7 | Parked — fold into whichever phase first needs them |

---

## Phase 1 — Forced-Return Release (§15.9)

### Problem

BR-05b: a SKU's ownership = available stock + pending returns of one supplier. If a supplier stops visiting with stock (or pending returns) still on the books, that SKU is **locked against every other consignment supplier** until the stock is manually returned. "Return All / termination" helps but is user-driven, has no distinct audit signal, and grants no elevated permission.

### Design

1. **New permission `consignment.force_return`**
   - Migration `065_consignment_force_return.sql` following the guarded pattern of `061_add_store_cg_import_export_perms.sql` (`WHERE NOT EXISTS` inserts into `permissions` + `role_permissions`).
   - Grant to `superadmin` and `manager` (the store boss). The plan originally said `admin`, but no such role exists — the system has six roles (superadmin, manager, supervisor, cashier, inventory_staff, finance), and PRD §15.9 frames forced return as a **store** action ("forced return oleh toko"). Finance-only roles are excluded.
   - Must be permanently re-runnable (migration replay contract, AGENTS.md).

2. **Backend endpoint `POST /consignment/arrangements/:id/force-return`**
   - Handler guards `consignment.force_return`; store scope via existing `claimsStore` pattern.
   - Single transaction: create a formal return with `reason = "termination"` covering **all remaining ownership** (available + pending return rows, pending rows marked `returned`), reusing the bulk-return path proven in `consignment-bulk-return-to-end.md`.
   - **Unsettled sales are untouched** — they still settle through the normal settlement/payout flow. Force-return only releases *unsold* stock ownership.
   - On success, arrangement may then pass the existing `CNS-201` end-arrangement check (`internal/consignment/handler.go:676`); do **not** auto-end — let the user end explicitly so the flow stays observable.
   - Distinct audit/event marker (e.g. return `notes` or a `forced` flag) so a forced release is distinguishable from a supplier-initiated return in reports and future audit-trail work (§15.7).

3. **Frontend**
   - "Force Return All Stock" button on arrangement detail, rendered only with the new permission; confirmation modal reusing the `consignment-bulk-return-to-end` modal pattern (list of products + quantities, amber warning styling).
   - Replaces the guidance banner path when the user holds `force_return`; the existing "Return All" remains for unprivileged users.
   - i18n entries in `web/src/shared/i18n/en.ts` + `id.ts`.

### Tests

- Service: force return clears available + pending ownership in one tx; CNS-201 end check passes afterwards; unsettled sales still settleable; `ErrNothingToReturn` when nothing owned; arrangement stays active; store-scope mismatch rejected; SKU lock released for a subsequent supplier arrangement (BR-05b release scenario).
- Handler: permission middleware wiring (`consignment.force_return`; supervisor/cashier denied).
- Frontend: `web/src/modules/consignment` vitest for button visibility + modal flow.

---

## Phase 2 — Print Documents (§15.4)

### Problem

Receipts and returns exist only on screen. Suppliers/couriers expect paper proof of handover.

### Design

- **No backend or schema work** — `CR-`/`RT-` numbers, items, and totals already returned by detail endpoints (`consignment-receipt-detail`, `consignment-return-detail`).
- **Printable rendering on the detail pages** of `web/src/modules/consignment/`:
  - "Print" button on Receipt detail and Return detail.
  - Template contents: store header + logo, document type/number, date, supplier name, arrangement ref, item table (SKU, name, qty, price/share terms), signature blocks for store & supplier.
- **Transport:** decision taken — **browser print** (`window.print()` on a print-styled overlay), not the print-agent. The agent's payload model (`tools/print-agent/internal/receipt/model.go`) is sale-receipt-shaped; carrying an arbitrary consignment document would break or complicate silent mode, and these are hand-over proofs, not POS receipts. `VITE_PRINT_AGENT_URL` semantics are untouched. Print-agent support can be added later if silent-mode users demand it.

### Implementation (as shipped)

- `web/src/modules/consignment/stores/consignmentPrint.svelte.ts` — writable queue holding the current document (`{ kind: "receipt" | "return", doc }`).
- `web/src/modules/consignment/components/ConsignmentPrintOverlay.svelte` — A4 print view (store header + logo, document title/number, date, supplier, arrangement ref, handled-by, item table, notes, store/supplier signature blocks, printed-at). Mounted once at app root in `src/app/main.svelte` beside `ReceiptPrintOverlay`.
- `web/src/modules/consignment/lib/print.ts` — `printConsignmentDocument(doc)` queues, opens the browser dialog, then clears (mirrors `previewPrint` in `$shared/services/print-service.ts`).
- `web/src/app.css` — `.consignment-print-container` (off-screen on screen, A4 in `@media print`) plus the print-hide rule now keeps both printable containers: `#app > *:not(.thermal-receipt-container):not(.consignment-print-container)`.
- Print buttons on the Receipt detail modal (`ReceiptEntry.svelte`) and Return detail modal (`ReturnPage.svelte`). Not permission-gated — anyone who can open the detail can print the hand-over proof.

### Tests

- `__tests__/ConsignmentPrintOverlay.svelte.test.ts` — source-structure guards (container class, receipt/return branches, totals, signature blocks).
- `lib/print.test.ts` — queue → `window.print()` → clear, with fake timers.
- ReceiptEntry / ReturnPage guards extended for the print button.
- No E2E for MVP; optional Playwright print-preview check later.

---

## Phase 3 — Physical Reconciliation (§15.10)

### Problem

System `consignment_stock` (available + pending return) can drift from what is physically on the shelf (damage, miscount, theft). Shipped as the **Rekonsiliasi** tab: it counts available consignment stock against the system and posts the correction.

### Design — shipped

> **Implemented.** Spike: `docs/design/consignment-reconciliation-spike.md` (D1–D8 signed off).
> Findings verified against the tree, and the candidate approach was corrected: a posted
> reconciliation moves **both** `consignment_stock` and `product_stock` via
> `internal/inventory.ConsignmentAdjuster.ApplyConsignmentDelta`
> (`internal/inventory/consignment_adjuster.go:37`), never `consignment_stock` alone.
> Migration `066_consignment_reconciliation.sql`; backend in `internal/consignment/{domain,repository,service,handler}.go`;
> UI in `web/src/modules/consignment/components/ReconciliationPage.svelte` (Reconcile tab).

Decisions as implemented:

- **D1 — permission:** new `consignment.reconcile`, granted to `superadmin`, `manager`, `inventory_staff`; `consignment.view` also granted to `inventory_staff`.
- **D2 — scope:** per arrangement.
- **D3/D3a/D3b — liability:** missing stock with reason `lost`/`stolen` computes a **store-owed liability** (`unitPrice × qty − computeStoreShare(...)`, `internal/consignment/repository.go:1863`) tracked on `consignment_settlements.total_liability` and added to the supplier's next settlement. Reasons `damaged`/`expired`/`miscount`/`other` write off without liability; positive variance never creates liability.
- **D4 — state machine:** `draft → counted → posted`. `SaveReconciliationCount` accepts a `draft` only; `PostReconciliation` accepts a `counted` only.
- **D5 — positive variance:** allowed with a required reason (e.g. `unrecorded_receipt`); increases supplier ownership.
- **D6 — count basis:** available-only (pending returns are the supplier's already-separated bucket).
- **D7 — settlement block:** none — posting does not block settlement; liability rides the next settlement.
- **D8 — count→post ownership drift:** posting re-validates every changed product against the locked ledger. A SKU released or re-owned by another supplier since the count is refused (`ErrReconciliationStale`, 409) rather than re-binding the owner columns; a fully-released SKU may only be re-created by a surplus (D5) that is not store-owned. See spike D8.

Reason vocabulary: `damaged, expired, miscount, lost, stolen, unrecorded_receipt, other` (not `customer_return`, not `termination`).

- **Do not reuse stock opname blindly**: stock opname adjusts `product_stock` + `inventory_adjustments` (`internal/stockopname/service.go:865`), but consignment stock is *supplier-owned* and lives in `consignment_stock`; the reconciliation pairs both ledgers through the consignment adjuster and tags the movement `consignment_reconciliation`.

### Tests (shipped)

- `internal/consignment/reconciliation_test.go`: `TestReconciliationReasonValidation`, `TestService_ReconciliationLifecycle` (liability 24000 for price 10000 × shortfall 3 − 20% share), `TestService_ReconciliationSurplus`, `TestService_ReconciliationValidation`, `TestService_ReconciliationStaleOwnership` (D8: re-owned and released SKUs are refused at post); plus `TestHandler_ReconciliationAuthorization` in `handler_rbac_test.go`. DB-backed tests run in CI (no local PostgreSQL).
- Frontend: `web/src/modules/consignment/services/__tests__/consignment-service.test.ts` (client calls) and `components/__tests__/ReconciliationPage.svelte.test.ts` (source-structure guards).

---

## Cross-cutting requirements

- **Migrations:** next free number is `067`; all runners replay every file — guarded DDL only; AGENTS.md migration-ordering table carries the `065`/`066` rows.
- **Architecture:** domain → repository → service → handler in `internal/consignment/`; no cross-module imports (depguard).
- **Permissions:** register codes in migration + role grants; frontend gates via existing permission store.
- **Docs on completion:** `docs/roadmap/upcoming-features.md:586` updated (shipped items moved out of the deferred note); `docs/guides/user-manual.md` consignment section extended (print, force return, reconcile).
- **Verification:** targeted package tests during development only (`go test ./internal/consignment/...`, `npx vitest run src/modules/consignment`); full suites run in CI per AGENTS.md. Never auto-commit.

## Still-parked items — re-trigger conditions

| Item | Re-trigger when |
|------|-----------------|
| §15.5 analytics dashboard | Users request aggregation beyond settlement/stock pages |
| §15.1 expiry-by-shelf-life | Perishable categories enter product scope |
| §15.3 refund/void extras | A void path appears that `PRD-Sales-Return.md` doesn't cover |
| §15.8 payment flows beyond payout | A concrete deduction/offset flow is specified by the business |
| §15.2/§15.6/§15.7 (reasons, access, audit) | First phase that needs them lands; fold in then |
