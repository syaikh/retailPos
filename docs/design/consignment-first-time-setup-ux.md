# Consignment First-Time Setup UX

> **Status:** Implemented

## Problem

When a user opens a freshly created arrangement, the UI lands on the **Receipts** tab by default. If the user tries to record a receipt, they get a yellow warning per product: "Product has no terms yet — add them in the Terms tab." This is confusing — the user must set terms first, but the UI doesn't make that obvious.

## Solution

Three changes to make the terms-first requirement clear:

1. **Smart default tab** — new arrangement (no terms) lands on Terms tab; existing arrangement (has terms) lands on Receipts tab.
2. **Warning banner** — when terms are empty, show a persistent banner above the tab bar: "Set pricing terms before receiving goods."
3. **Keep receipt modal warning** — the per-product warning in `ReceiptEntry.svelte` stays as a fallback edge case (e.g., new product added to arrangement without a term).

## Files to Change

### 1. `web/src/modules/consignment/components/ArrangementsPage.svelte`

**Change A — Smart default tab (line 125)**

```js
// Before
function openArrangement(a: Arrangement) {
  activeArrangement = a;
  activeTab = 'receipt';
}

// After
function openArrangement(a: Arrangement) {
  activeArrangement = a;
  activeTab = (a.terms?.length ?? 0) > 0 ? 'receipt' : 'terms';
}
```

**Change B — Add `AlertTriangle` import (line 6)**

Add `AlertTriangle` to the existing lucide-svelte import:

```js
// Before
import { Plus, ClipboardList, Truck, RotateCcw, Wallet, ArrowLeft } from 'lucide-svelte';

// After
import { Plus, ClipboardList, Truck, RotateCcw, Wallet, ArrowLeft, AlertTriangle } from 'lucide-svelte';
```

**Change C — Warning banner above tabs (between line 168 and line 169)**

Insert a conditional banner between the arrangement header and the tab bar:

```svelte
{#if (activeArrangement.terms?.length ?? 0) === 0}
  <div class="rounded-xl border border-warning/40 bg-warning-subtle/20 p-4 flex items-start gap-3">
    <AlertTriangle size={18} class="text-warning shrink-0 mt-0.5" />
    <div>
      <p class="text-sm font-semibold text-text-primary">{labels.consignmentTermsRequiredBanner}</p>
      <p class="text-xs text-text-muted mt-1">{labels.consignmentTermsRequiredHint}</p>
    </div>
  </div>
{/if}
```

This uses the same styling pattern as `PricingRulesPage.svelte:1005` and `RolesPage.svelte:496`.

### 2. `web/src/shared/i18n/en.ts`

Add two new labels after `consignmentNoTermsSubtitle` (line 1799):

```ts
consignmentTermsRequiredBanner: 'Set pricing terms before receiving goods',
consignmentTermsRequiredHint: 'Each product needs a price and store share before you can record receipts.',
```

### 3. `web/src/shared/i18n/id.ts`

Add two new labels after `consignmentNoTermsSubtitle` (line 1799):

```ts
consignmentTermsRequiredBanner: 'Tentukan terms harga sebelum menerima barang',
consignmentTermsRequiredHint: 'Setiap produk memerlukan harga dan hak toko sebelum Anda bisa mencatat penerimaan.',
```

## What Does NOT Change

- **Tab order** — stays Receipts, Terms, Pending Returns, Returns, Settlement, Stock
- **`ReceiptEntry.svelte`** — the per-product yellow warning stays as fallback
- **`TermsEditor.svelte`** — no `$effect` sync; the `onMount` → `load()` handles initial sync, and `submitAdd()` sets `terms = saved` directly after save
- **Backend** — no API changes; `Arrangement` type already carries `terms[]`

## Visual Result

**New arrangement (no terms):**
```
┌──────────────────────────────────────────────────────┐
│ ← Back   Toko Kopi Maju                             │
│           ● Active   Last visit: —                   │
├──────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────────────────────┐ │
│ │ ⚠  Set pricing terms before receiving goods      │ │
│ │    Each product needs a price and store share     │ │
│ │    before you can record receipts.                │ │
│ └──────────────────────────────────────────────────┘ │
│ [Receipts]  [Terms●]  [Pending Returns]  ...        │
│ ═══════════                                         │
│ Terms content (empty state with Add Term button)    │
```

**Mature arrangement (has terms):**
```
┌──────────────────────────────────────────────────────┐
│ ← Back   Toko Kopi Maju                             │
│           ● Active   Last visit: 2026-09-03          │
├──────────────────────────────────────────────────────┤
│ [Receipts●]  [Terms]  [Pending Returns]  ...        │
│  ═══════════                                        │
│ Receipt history                                     │
```

No banner, lands on Receipts — exactly like current behavior.

## Verification

1. `go build ./...` — backend compiles
2. `cd web && npm run build` — frontend compiles
3. Manual test: create new arrangement → should land on Terms tab with banner
4. Manual test: add a term → banner disappears
5. Manual test: close and reopen arrangement → should land on Receipts tab (no banner)
