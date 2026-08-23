# Transactions History — Refresh & Notification Deep-link Plan

Status: **Design agreed. Part A implemented; Part B not yet coded.**

## Part A — Notification → Transaction Detail (IMPLEMENTED)

Clicking the "new transaction" notification opens that sale's detail drawer instead of just the list.

- `web/src/app/layouts/NotificationBell.svelte` — `sale_created` websocket handler now sets
  `navigateTo: data.id ? \`/transactions?txn=${data.id}\` : '/transactions'`.
- `web/src/modules/sales/components/TransactionsPage.svelte` — on mount and on every route
  change, reads `?txn=<id>` and opens `TransactionDrawer` for that sale:
  1. from the loaded list first,
  2. then `GET /api/sales/:id` (history mode),
  3. then `GET /api/sales/lookup/:id` (lookup mode) as fallback.
- No backend change: `SaleCreatedEvent` already carries the sale `id`.
- `svelte-check`: 0 errors.

## Part B — Refresh & Find Transaction redesign (AGREED DESIGN, NOT CODED)

### 1. Find Transaction → search-only (no default table)
Decision: remove the default store-wide table from the Find Transaction tab. Keep a single
invoice-focused search box; on a match, open the existing lookup detail drawer directly
(or a one-row result card). Skip the list/pagination.

- Matches the spec's **intended path**: the invoice number on the receipt is "the only
  field a cashier reliably has" (`find-transaction-requirements.md:31-38`). Product/customer
  matches stay as secondary search behavior but are not surfaced as a browsable list.
- Invoice numbers are unique → effectively a 1-result lookup → no table/pagination needed.
- Removes the initial heavy "load all completed sales" query on tab open.

**Receipt-as-proof / anti-fraud rationale (business control, not just UX):**
The receipt is the **proof-of-purchase token**. Requiring the invoice number at lookup is a
deliberate anti-fraud gate — the customer must *present evidence* of a transaction before
any action (reprint, verify, future return). Browsing or searching without a receipt
(customer name / amount / item) would become a way to reconstruct or impersonate a purchase.
Therefore NOT providing no-receipt lookup is the feature working as intended, not a gap.
- Redaction bounds exposure at the other end: even with a valid invoice, a cashier gets
  only redacted detail + reprint — never cost/margin or customer PII.
- Genuine lost-receipt exceptions (trusted customer, manager override) belong behind
  `manager`-level auth — out of scope; return/void itself is **On Hold** per spec.

### 2. Refresh / live-update scope (consequent simplification)
Because Find Transaction no longer has a persistent table, it needs **neither a refresh
button nor a websocket banner** (a search re-runs on submit). This retires both for that tab.

| Tab                                     | Refresh button        | Websocket banner / live update |
|-----------------------------------------|-----------------------|--------------------------------|
| Cashier · My Transactions (own sales)   | Yes (essential)       | No — store broadcast would miscount other cashiers' sales (own-scope list) |
| Cashier · Find Transaction (search-only)| No (search re-runs)   | No — no persistent table to update |
| Manager · My Transactions (all sales)   | Yes                   | Yes (optional) — scope matches |

### 3. Manual Refresh button (backbone, where a table exists)
- One ghost icon button (`⟳`, ~36px) in a **slim toolbar row directly above the table**
  (right-aligned), NOT inside the filters card.
- Re-fetches **only the active tab's current query** (preserves filters/sort/page). Never
  both tabs at once.
- Paired with a muted **"Updated HH:MM WIB"** label on its left (freshness cue).
- States: idle → spinner while loading (disabled) → optional brief `✓`.
- Applies to **My Transactions** and **Manager all-sales** (the tabs with tables).

### 4. Websocket banner (Manager all-sales only; Phase 2 optional)
- A strip listening to the existing `sale_created` socket event:
  "🟢 N new transactions since HH:MM — [View]". Clicking refreshes the list.
- Accurate only on the **Manager** all-sales view (matches the store-scoped broadcast).
- NOT required: the Refresh button already covers freshness. Banner = passive convenience.

### 5. Explicitly excluded
- **No blind auto-polling** (DB load + disrupts scrolling/filtering on shared terminals).
- **No banner on cashier tabs** (own-scope miscounts; Find Transaction has no table).
- **No browse / no-receipt lookup on Find Transaction** (anti-fraud gate — see §1).

### 6. Redundancy resolution
Button ≠ banner: the socket announces only *new* sales; refunds/voids/status edits are NOT
announced, so the button still catches those. On the Manager view they're complementary;
on My Transactions the button is the only correct tool (banner would miscount).

## Placement mockup (text)
Slim toolbar between `TransactionFilters` and `TransactionTable`, **on tabs that have a
table** (My Transactions, Manager). Left = result count + "Updated HH:MM WIB";
right = `( ⟳ ) Refresh`. Find Transaction shows only the search box → drawer on match.

## Design evolution note
- 2026-08-23 (a): banner proposed for Find Transaction (cross-cashier scope matches
  broadcast). **Superseded** by the search-only decision (§1): with no persistent table,
  the Find Transaction banner is moot.
- 2026-08-23 (b): originally a manual Refresh button was proposed for Find Transaction
  too; retired for the same reason (no persistent list to refresh).
- The `find-transaction-requirements.md` spec was updated to the search-only design
  (no default table; single search box; proof-of-purchase / anti-fraud gate). This is a
  change to a shipped feature and needs product sign-off before implementation.
- Part A (notification → detail drawer) is already implemented and unchanged.

## Pinned behavioral decisions (2026-08-23)
Resolving the open calls from the refresh-design review:

- **#2 Date-range widget on Find Transaction:** REMOVED. Invoice lookup is
  date-independent; the widget's only purpose (widening the window for an unknown old
  receipt) is moot once no-receipt lookup is excluded. Backend must not clamp an invoice
  search by the default 30-day window.
- **#3 Refresh on My Transactions / Manager:** resets to **page 0** (newest first; default
  sort `created_at desc`) so a cashier's newest sale / latest status change surfaces at top.
  Filters and sort are otherwise preserved.
- **#4 "Updated HH:MM" timestamp:** stamped on every successful list load **and** manual
  refresh, using **Jakarta time** (`formatDateTimeInJakarta`), labeled "Updated HH:MM WIB".
  Never local browser time.
- **#5 Cross-cashier notification landing:** DEFERRED. Part A already opens the drawer in
  lookup mode for any `?txn=<id>` regardless of tab; landing such notifications on the Find
  Transaction context is a future enhancement, not part of this plan.

## Final decisions
- **Manager websocket banner: SHIP as Phase 2** — passive freshness on the all-sales view;
  its scope matches the store-scoped `sale_created` broadcast. All other behavioral calls
  are pinned above.
- The `find-transaction-requirements.md` spec update (search-only Find Transaction) is
  approved as the target design and accompanies implementation.

## Sign-off
- **Status:** Design approved — 2026-08-23.
- **Part A** (notification → transaction detail drawer): implemented; `svelte-check` clean.
- **Part B** (refresh + Find Transaction redesign): **implemented and verified**.
  - Find Transaction → search-only (no default table); invoice lookup; proof-of-purchase /
    anti-fraud gate; date-range widget removed.
  - My Transactions + Manager: manual Refresh button + "Updated HH:MM WIB" (Jakarta time).
  - Manager all-sales: websocket "N new transactions" banner (Phase 2).
  - Excluded: auto-polling; banner on own-scope / Find Transaction tabs; no-receipt browse.
- **Verification (2026-08-23):**
  - Frontend: `svelte-check` 0 errors/0 warnings; `npm run build` OK;
    `FindTransaction` + `TransactionsPage` source-structure guard tests **37/37 pass**.
  - Backend: no change required — lookup accepts a widened `start_date` window
    (`buildSaleFilter` parses it with no clamp). Added
    `TestLookup_InvoiceSearchIgnores30DayWindow` (internal/sale) which passes; it locks
    the date-independent invoice lookup against future regressions.
 - **Outstanding (user actions):** manual browser smoke test; commit when ready
   (not auto-committed per repo policy).

## Post-implementation review (2026-08-24)
A code review of the uncommitted changes flagged four items; all resolved except the
intentional design choice noted in #4.
- **#1 Broken sort (fixed):** removing the auto-load `$effect` left `toggleSort()` unable to
  re-query. It now calls `runSearch()` (which guards the empty-query case). Removed the
  now-dead `handlePageChange`. Locked with a source-structure guard test.
- **#2 Doc contradiction (fixed):** `find-transaction-requirements.md` claimed the backend
  "must not clamp an invoice search by date." Corrected — the backend *does* apply a 30-day
  default window; the frontend achieves date-independence by widening `start_date` to the
  epoch (`FindTransaction.svelte` `buildFilters()`). No backend change required.
- **#3 Banner over-report (fixed):** Manager "N new transactions" banner now also clears on
  the manual **Refresh** button (previously only on **View**). It intentionally still
  persists across *filter changes*, since a new sale may not match the new filter and should
  stay flagged (see note in chat).
- **#4 Result truncation (by design, kept):** pagination removed; fuzzy matches beyond
  `pageSize` (20) are unreachable and show a "refine search" hint. Acceptable for the
  proof-of-purchase (exact-invoice) flow. Open option: raise `pageSize` or re-add pagination
  for fuzzy browsing.
- **Verification after fixes:** `svelte-check` 0 errors; frontend guard tests 37/37 pass
  (FindTransaction suite now 8/8). Backend `TestLookup*` unchanged and passing.

---

## Implementation Plan (component-level)

### 1. `web/src/modules/sales/components/FindTransaction.svelte` — refactor to search-only
- **Remove default load.** Delete the `$effect` (currently ~L90-95) that auto-runs `load()`
  on mount/filter change, and drop any mount-time `getSalesLookup` call. Keep
  `onMount(store.loadPaymentMethods())`.
- **Replace `<TransactionFilters … />` with a single search box.** Reuse `SearchBar` from
  `$shared/ui` (verify it exposes `onsubmit`/Enter; else wrap `Input` + `Button`):
  `<SearchBar bind:value={searchQuery} placeholder={labels.searchByInvoiceNumber}
  onsubmit={runSearch} />`.
- **`buildFilters()`**: when `searchQuery` is non-empty, widen `startDate`/`endDate` to a
  broad range (e.g. `2000-01-01` … today) so invoice lookup is **date-independent** — no
  backend change required.
- **`runSearch()`**: `page = 0; await load();` (load = `getSalesLookup(buildFilters())`).
- **Result rendering:** keep `data` list, simplified to minimal rows
  (`invoice_number`, date, cashier, total) clickable → `openLookupDetail(id)`. If
  `data.length === 1`, optionally auto-open the lookup drawer.
- **Drop `Pagination`** for this tab (results expected tiny); if `total > pageSize` show a
  "refine your search" note instead.
- Remove now-unused `dateRange`, `paymentMethods`, `minTotal/maxTotal` state (keep
  `sortBy='created_at'`, `sortDir='desc'` for rare multi-result cases).
- Keep `TransactionDrawer` (`mode="lookup"`, `canReprint`), `copyInvoice`, and the
  `lookupRedactedNotice` line.

### 2. `web/src/modules/sales/components/TransactionsPage.svelte` — refresh + banner
New imports: `useWebSocket` from `$shared/api/websocket` (already imports `goto`,
`useRBAC`, `formatDateTimeInJakarta` via jakartaTime). `RefreshCw` from `lucide-svelte`.

New state:
```ts
let lastUpdated = $state<Date | null>(null);
let refreshing  = $state(false);
let newTxnCount = $state(0);
let newTxnSince = $state<Date | null>(null);
```

New functions:
```ts
function fmtTime(d: Date | null): string {
  if (!d) return '—';
  return formatDateTimeInJakarta(d.toISOString()).slice(11, 16) + ' WIB'; // HH:MM WIB
}
function refresh() {
  refreshing = true;
  store.page = 0;                         // decision #3: surface newest first
  store.load(store.currentFilters).finally(() => {
    refreshing = false;
    lastUpdated = new Date();            // decision #4: Jakarta-stamped on load/refresh
  });
}
function viewNew() { refresh(); newTxnCount = 0; newTxnSince = null; }
```
- Stamp `lastUpdated` on every successful fetch (covers filter/pagination too) via
  `$effect(() => { store.salesData; lastUpdated = new Date(); });`.
- **Toolbar** — insert between `TransactionFilters` and `TransactionTable`, inside the
  `{#if activeTab === 'mine'}` block (covers both cashier "My Transactions" and the
  manager single all-sales view):
  ```svelte
  <div class="flex items-center justify-between px-1 py-2">
    <span class="text-xs text-text-muted">{store.total} {labels.transactions} · {labels.updated} {fmtTime(lastUpdated)}</span>
    <Button variant="ghost" size="icon" onclick={refresh} disabled={refreshing} aria-label={labels.refresh}>
      <RefreshCw size={16} class={refreshing ? 'animate-spin' : ''} />
    </Button>
  </div>
  ```
- **Manager banner** — above the toolbar, gated by `canAccessAll` (the all-sales view):
  ```svelte
  {#if canAccessAll && newTxnCount > 0}
    <div class="banner">🟢 {newTxnCount} {labels.newTransactionsSince} {fmtTime(newTxnSince)}
      <Button variant="secondary" size="sm" onclick={viewNew}>{labels.view}</Button>
    </div>
  {/if}
  ```
- **WebSocket subscription** (in `onMount`, alongside existing logic; clean up on destroy):
  ```ts
  const ws = useWebSocket();
  const unsubWs = ws.on('sale_created', () => {
    if (!canAccessAll) return;            // own-scope tab must NOT count store-wide sales
    newTxnCount += 1;
    if (!newTxnSince) newTxnSince = new Date();
  });
  ```
  Note: `NotificationBell` still owns the global toast; this is a separate, view-local
  listener (multiple subscribers are allowed).

### 3. `web/src/shared/i18n` — add labels
`updated`, `newTransactionsSince`, `refresh`, `transactions`, `refineSearch`, `view`.

### 4. Backend (only if client widening is rejected)
If instead we want the server to relax the date clamp: in the `/api/sales/lookup`
handler/repo (`internal/sale`), skip `start_date`/`end_date` when `search` is non-empty.
Add a Go test: lookup by invoice returns a sale outside the default 30-day window.
(Preferred path is the client-side widening in §1 — no backend change.)

### Tests
**Frontend** (`web/src/modules/sales/components/__tests__/`, vitest):
- `FindTransaction.svelte.test.ts`
  - mount → `getSalesLookup` **not** called (no default load).
  - type invoice + submit → `getSalesLookup` called with `search=invoice` and widened date range.
  - single result → drawer opens in `lookup` mode on row click (or auto-opens).
  - no result → empty state shown.
- `TransactionsPage.svelte.test.ts`
  - click refresh → `store.load` called and `store.page === 0`.
  - `sale_created` while `canAccessAll` → banner count increments; while cashier (not
    `canAccessAll`) → count stays 0 (banner hidden).
  - click banner [View] → `refresh()` called, count reset to 0.
  - "Updated HH:MM WIB" reflects `lastUpdated` in Jakarta time.
- `NotificationBell.svelte.test.ts` (extend): `sale_created` pushes a notification whose
  `navigateTo` contains `?txn=<id>`.

**Backend** (Go, only if §4 taken): `go test -p 1 -count=1 ./internal/sale/...` for the
lookup date-relax behaviour.

### Verification commands
- `cd web && npx svelte-check` → 0 errors.
- `cd web && npm run build`.
- Targeted: `cd web && npx vitest run src/modules/sales/components/__tests__/`.
- Backend (if touched): `go test -p 1 -count=1 ./internal/sale/...`.
