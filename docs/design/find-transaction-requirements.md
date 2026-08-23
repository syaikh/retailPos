# Find Transaction — Feature Requirements

## Purpose

Allow a cashier (or any user with the `sale.lookup` permission) to locate a
**completed sale recorded by another cashier**, typically when a customer returns
with a receipt and needs help (verify, re-print). Return / void initiation is a
separate concern — see **Action scope (phased)** below. Entry is **by invoice number
only** — there is no browsable/recent-sales list (see **Proof-of-purchase gate**).

This is a **cross-cashier** lookup. Unlike "My Transactions" (which is scoped to the
caller), Find Transaction is intentionally NOT clamped to the caller's own sales.
Access is gated at the route level by the `sale.lookup` permission, and the response
is a **redacted summary** — callers never receive sensitive fields (line items, cost,
full payment detail) for other cashiers' transactions.

### Proof-of-purchase gate (anti-fraud)

The receipt is the **proof-of-purchase token**. Requiring the invoice number at lookup is
a deliberate anti-fraud control: a customer must *present evidence* of a transaction
before any action (verify, re-print, future return). A browsable list or no-receipt search
(by customer name, amount, or item) would let a person reconstruct or impersonate a
purchase, so neither is provided. A lost receipt means the customer is **not entitled** to
a lookup/action — that is the control working as intended, not a gap. Genuine exceptions
(trusted customer, manager override) belong behind `manager`-level auth and are out of
scope (return/void is **On Hold**).

## Scope of results

Only **`completed`** sales are surfaced.

- Held / discarded carts live in `cart_sessions` (status `cancelled`), not in `sales`,
  and never appear.
- Other non-completed sale statuses (`parked`, `cancelled`, `recalled`) have **no
  customer-service value** in this screen:
  - `parked` (held) belongs to the held-sales **recall** flow, not to receipt lookup.
  - `cancelled` / `recalled` are audit/log concerns, not something a customer presents
    a receipt for.

The lookup query pins `status = 'completed'`, so the count (`total`) is also scoped to
completed sales.

## Search field

The search input is **invoice-number first** — the number printed on the receipt
(`sale.invoice_number`, fuzzy `ILIKE` match), the only field a cashier reliably has when
a customer presents a receipt. A query is **always required**: the screen never renders an
unfiltered list of sales.

The lookup also matches product name (via `sale_items`) and customer name as **secondary**
behavior, but these are fallbacks, not a browse path. Invoice lookup is **not constrained
by date** — searching a specific invoice resolves it regardless of the default 30-day
window. The backend **does** apply its default 30-day `start_date`/`end_date` window when
those params are omitted, so the frontend makes invoice lookup date-independent by
widening `start_date` to the epoch (e.g. `2000-01-01`) whenever a search term is present
(see `FindTransaction.svelte` `buildFilters()`). No backend change was required.

## Top-panel filter widgets

Find Transaction renders a **single search box** (invoice number). The other filter
widgets used by "My Transactions" are intentionally NOT shown:

| Widget | Applicable? | Rationale |
|---|---|---|
| **Search (invoice number)** | ✅ The only widget | The receipt's hard identifier; an explicit query is always required (no default list). Invoice lookup is date-independent. |
| **Date range** | ❌ Removed | Its only purpose was widening the window for an old receipt when the exact invoice was unknown — but no-receipt lookup is excluded by the proof-of-purchase gate, so the widget has no valid use here. |
| **Payment method** | ❌ Hidden | Cross-cashier by design; a cashier searching another's receipt rarely knows, and should not filter by, its payment type. Filtering can also produce **false negatives**. |
| **Amount range (min/max)** | ❌ Hidden | The invoice number is exact; an amount range is imprecise and can hide the target row. |

## Tab visibility (role gating)

The Transactions page layout depends on the caller's permissions:

- **Cashier** (`sale.lookup`, no `report.view`): two tabs — "My Transactions" (own sales)
   and "Find Transaction" (cross-cashier, redacted, `completed`-only, **invoice-lookup only — no browsable list**). This is where the
    cross-cashier capability is needed.
- **Manager / admin / superadmin** (`report.view`): **no tab bar at all**. `GET /sales`
  (My Transactions) already returns every cashier's sales — full detail, all statuses —
  for these roles via `ownership.CanAccessAll(permissions.ReportView)`. A separate
  "My Transactions" tab would be redundant (it already shows all cashiers), and the Find
  Transaction tab only adds a redacted subset, so **both** tabs are hidden and the
  transactions list is rendered as a single, untabbed all-cashier view. This reverts
  higher roles to the pre-"Find Transaction" single-view layout.
- **Plain user** (neither): "My Transactions" only (own sales), no Find Transaction tab.

`sale.lookup` is granted to the **cashier** role only; the manager grant is revoked
(migration `031_revoke_sale_lookup_manager.sql`). Admin/superadmin never held it.

This keeps the cross-cashier lookup where it is needed (cashiers) and gives higher roles a
single, uncluttered all-cashier view.

## Result presentation (redacted)

A search yields **zero, one, or a few** matched sales (invoice numbers are unique, so
normally exactly one). Matches are shown as a minimal result — `invoice_number`,
date/time (Jakarta), cashier name, total — and clicking drills into the **redacted
itemized detail drawer** (Phase 2). There is **no full transaction table** and no
pagination: the lookup presenter (`presentSaleLookup`) returns only redacted fields,
consistent with the `sale.lookup` permission model. If several fuzzy matches return,
present a short pick-list; a single match may open the drawer directly.

## Action scope (phased)

Find Transaction is first and foremost a **locator**. Actions on a found sale are
delivered in phases so each capability carries its own permission.

### Phase 1 — Locate + redacted summary (done)
- `GET /sales/lookup` (permission `sale.lookup`) is invoked on an **explicit invoice
  search** — it does **not** render a default list of all sales.
- The matched result drills into the redacted itemized detail (Phase 2).
- No action buttons on the result itself.

### Phase 2 — Option B: cashier view-detail + reprint (done)
A cashier who locates a sale can drill into a **read-only itemized receipt** and
**re-print** it. Implemented.

- **Drill-down detail**: `GET /sales/lookup/:id` (permission `sale.detail`) returns the
  redacted itemized detail (line items for the receipt, payments without reference).
  Cross-cashier, no ownership gate — access control is the `sale.detail` permission itself.
  Granted to cashier, manager, admin, and superadmin (migration
  `032_sale_detail_and_receipt_print.sql`).
- **Reprint**: gated by `receipt.print` (new). Granted to the same roles. In the Find
  Transaction drawer the Print (reprint) button is shown only in lookup mode and only when
  the caller holds `receipt.print`.
- **Redaction rules for the detail view** (`presentSaleLookupDetail`):
  - Cost / margin are **never** sent (items use `saleItemWithoutCost` — no `Cost` field).
  - Customer PII is **omitted entirely** (the detail carries no `customer_name`); the
    reprinted receipt shows the generic walk-in label.
  - Payment tender **reference numbers are omitted** (payments carry only method + amount).
  - Store scoping is preserved (the underlying `GetSaleByID` still scopes by `store_id`).
- **Manager / admin / superadmin** do not see the Find Transaction tab (they have
  `report.view` → single untabbed all-cashier view), so they reach full detail through that
  list, not this redacted endpoint. Holding `sale.detail`/`receipt.print` keeps the
  permission hierarchy coherent.
- **Receipt branding (header/footer, store address/phone):** `GET /api/settings` is readable
  by **any authenticated user** (the `app_settings.view` gate was removed from the GET route;
  only the Settings *management UI* stays gated by `app_settings.view` via the frontend
  `routePermissions` map). This is deliberate: the payload is non-sensitive global config
  required for receipt rendering, and `cashier` must NOT be granted `app_settings.view`
  (per `permission-matrix-final.md` least-privilege — it would also expose the Settings menu).
  With the relaxed GET, a cashier's reprinted (and POS) receipts include the store branding
  and receipt text.

### Return / void — On Hold (pending review)
Return and void flows of completed sales are **explicitly deferred** (no such mechanism
exists anywhere in the codebase today — only parked-sale recall/cancel). They require
manager-auth gates and are treated as net-new functionality. **Status: On Hold — pending
review.** Reprint is allowed without it; return/void remains excluded from Phase 2.

## Non-goals

- Not a self-service "my own sales" view (that is the existing transactions screen).
- Not a held-cart recall screen (separate feature).
- Not an analytics / reporting filter (no export, no amount aggregation).
- Not a return / void flow — **On Hold — pending review** (see Action scope above).
- Not a browsable / recent-sales list of other cashiers' transactions (proof-of-purchase gate — see Purpose).
