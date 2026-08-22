# Find Transaction — Feature Requirements

## Purpose

Allow a cashier (or any user with the `sale.lookup` permission) to locate a
**completed sale recorded by another cashier**, typically when a customer returns
with a receipt and needs help (verify, re-print). Return / void initiation is a
separate concern — see **Action scope (phased)** below.

This is a **cross-cashier** lookup. Unlike "My Transactions" (which is scoped to the
caller), Find Transaction is intentionally NOT clamped to the caller's own sales.
Access is gated at the route level by the `sale.lookup` permission, and the response
is a **redacted summary** — callers never receive sensitive fields (line items, cost,
full payment detail) for other cashiers' transactions.

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

The primary search input is the **invoice number** printed on the receipt
(`sale.invoice_number`, fuzzy `ILIKE` match). This is the only field a cashier reliably
has when a customer presents a receipt.

The shared sales filter also matches product name (via `sale_items`) and customer name,
but these are **secondary** affordances, not the intended receipt-lookup path.

## Top-panel filter widgets (business decision)

The Find Transaction panel reuses `TransactionFilters.svelte`, but only a subset of its
widgets is applicable to the receipt-lookup use case. Payment-method and amount-range
filters are hidden (`showPaymentMethods={false}`, `showAmountRange={false}`).

| Widget | Applicable? | Rationale |
|---|---|---|
| **Search (invoice number)** | ✅ Essential | The receipt's only hard identifier; exact match. This is the intended path. |
| **Date range** | ⚠️ Secondary | Backend defaults to the last 30 days. A receipt older than that will not appear until the window is widened, so the range has real (edge-case) value. |
| **Payment method** | ❌ Hidden | Cross-cashier by design; a cashier searching another's receipt rarely knows, and should not filter by, its payment type. Filtering can also produce **false negatives** (wrong filter hides the row). |
| **Amount range (min/max)** | ❌ Hidden | The invoice number is exact; an amount range is imprecise and can likewise hide the target row. |

## Tab visibility (role gating)

The Transactions page layout depends on the caller's permissions:

- **Cashier** (`sale.lookup`, no `report.view`): two tabs — "My Transactions" (own sales)
  and "Find Transaction" (cross-cashier, redacted, `completed`-only). This is where the
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

## Displayed columns (redacted summary)

`invoice_number`, date/time (Jakarta), cashier name, total, and a minimal customer
reference as needed. The full transaction table (customer/item/cost/payment detail) is
**not** rendered — the lookup presenter (`presentSaleLookup`) returns only redacted
fields, consistent with the `sale.lookup` permission model.

## Action scope (phased)

Find Transaction is first and foremost a **locator**. Actions on a found sale are
delivered in phases so each capability carries its own permission.

### Phase 1 — Locate + redacted summary (done)
- `GET /sales/lookup` (permission `sale.lookup`) returns the summary table only.
- Rows are clickable to drill into the redacted itemized detail (Phase 2).
- No action buttons on the summary itself.

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
