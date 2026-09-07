# Consignment Receipt Detail View

## Problem

The receipt list in `ReceiptEntry.svelte` shows a summary table (receipt #, date, item count, total value) but there's no way to see individual receipt items (product name/SKU, accepted qty, price, store share, notes). Users need to inspect receipt details.

## Mockup

### Current: Receipt List (clickable rows)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Receipt History                                    [+ Record Receipt]      │
├──────────────────┬──────────────────────┬──────────┬───────────────────────┤
│  RECEIPT NO.     │  DATE                │  ITEMS   │  TOTAL VALUE          │
├──────────────────┼──────────────────────┼──────────┼───────────────────────┤
│  RCP-2026-00042  │  5 Sep 2026 14:30    │  3 items │  Rp 1.250.000         │  ← clickable
│  RCP-2026-00038  │  3 Sep 2026 10:15    │  2 items │  Rp 875.000           │  ← clickable
│  RCP-2026-00031  │  1 Sep 2026 09:00    │  5 items │  Rp 2.100.000         │  ← clickable
└──────────────────┴──────────────────────┴──────────┴───────────────────────┘
      ↑ cursor: pointer, hover: bg-surface-subtle/50
```

### Detail Modal (on row click)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  RCP-2026-00042                                                    [X]    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  RECEIPT NO.        DATE                RECEIVED BY      TOTAL VALUE        │
│  ─────────────      ─────────────────   ──────────────   ──────────────     │
│  RCP-2026-00042     5 Sep 2026 14:30    operator_1       Rp 1.250.000      │
│                                                                             │
│  Notes: Consignment delivery for September                                   │
│                                                                             │
│  ┌─────────────────────┬──────────┬────────────┬────────────┬──────────┬──┐ │
│  │  PRODUCT            │ ACCEPTED │  PRICE/UNIT│  TOTAL     │ STORE    │  │ │
│  │                     │          │            │  VALUE     │ SHARE    │  │ │
│  ├─────────────────────┼──────────┼────────────┼────────────┼──────────┼──┤ │
│  │  Indomie Goreng     │     50   │  Rp 2.500  │  Rp 125.000│  30%     │- │ │
│  │  INDO-001           │          │            │            │          │   │ │
│  ├─────────────────────┼──────────┼────────────┼────────────┼──────────┼──┤ │
│  │  Teh Botol Sosro    │    100   │  Rp 3.000  │  Rp 300.000│  25%     │- │ │
│  │  SOSRO-002          │          │            │            │          │   │ │
│  ├─────────────────────┼──────────┼────────────┼────────────┼──────────┼──┤ │
│  │  Pop Mie            │     75   │  Rp 10.000 │  Rp 750.000│  Rp 2.500│- │ │
│  │  NISSIN-003         │          │            │            │  (fixed) │   │ │
│  ├─────────────────────┼──────────┼────────────┼────────────┼──────────┼──┤ │
│  │                     │          │            │  1.250.000 │          │   │ │
│  └─────────────────────┴──────────┴────────────┴────────────┴──────────┴──┘ │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                              [Close]        │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key design points:**
- Header grid: 4 columns (Receipt No, Date, Received By, Total Value)
- Notes shown below header if present
- Items table: Product (name + SKU on two lines), Accepted qty, Price/Unit, Total Value, Store Share, Notes
- Store Share column: shows percentage (e.g. "30%") or fixed amount (e.g. "Rp 2.500")
- Footer: single "Close" button (read-only modal)
- Total row at bottom of items table

## Current State

- **Backend:** `GET /consignment/receipts/:id` returns full receipt + items + hydrated names ✅
- **Frontend service:** `getReceipt(id)` defined at `consignment-service.ts:59` but never called ✅
- **Frontend UI:** No detail view — list rows have no click handler ❌

## Plan

### Step 1: Add receipt detail modal state and loading

**File:** `web/src/modules/consignment/components/ReceiptEntry.svelte`

Add state variables for the detail modal:

```svelte
let showDetailModal = $state(false);
let detailReceipt = $state<Receipt | null>(null);
let loadingDetail = $state(false);
```

### Step 2: Add `openDetail` function

Import `getReceipt` from the service (already exists, never used):

```svelte
import { createReceipt, listReceipts, getReceipt } from '../services/consignment-service';
```

Add function to fetch and display receipt detail:

```svelte
async function openDetail(receiptId: number) {
  loadingDetail = true;
  showDetailModal = true;
  try {
    detailReceipt = await getReceipt(receiptId);
  } catch (e: any) {
    toast.error(e?.response?.data?.error || e.message || labels.consignmentLoadError);
    showDetailModal = false;
  } finally {
    loadingDetail = false;
  }
}
```

### Step 3: Make receipt rows clickable

**File:** `web/src/modules/consignment/components/ReceiptEntry.svelte` (lines 157-165)

Add `cursor-pointer` and `onclick` to each receipt row, plus a visual indicator:

```svelte
<tr
  class="border-b border-border/40 cursor-pointer hover:bg-surface-subtle/50 transition-colors"
  onclick={() => openDetail(r.id)}
  role="button"
  tabindex="0"
  onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') openDetail(r.id); }}
>
```

### Step 4: Add detail modal template

Add a new `<Modal>` after the existing entry modal (around line 243):

```svelte
<Modal bind:open={showDetailModal} title={detailReceipt?.receipt_number || labels.consignmentReceiptDetail} size="lg">
  {#snippet children()}
    {#if loadingDetail}
      <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
    {:else if detailReceipt}
      <div class="space-y-4">
        <!-- Receipt header info -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div class="text-text-secondary text-xs uppercase tracking-wider">{labels.consignmentReceiptNo}</div>
            <div class="font-medium text-text-primary">{detailReceipt.receipt_number}</div>
          </div>
          <div>
            <div class="text-text-secondary text-xs uppercase tracking-wider">{labels.consignmentDate}</div>
            <div class="font-medium text-text-primary">{formatDateTime(detailReceipt.received_at)}</div>
          </div>
          <div>
            <div class="text-text-secondary text-xs uppercase tracking-wider">{labels.consignmentReceivedBy}</div>
            <div class="font-medium text-text-primary">{detailReceipt.received_by_username || '-'}</div>
          </div>
          <div>
            <div class="text-text-secondary text-xs uppercase tracking-wider">{labels.consignmentTotalValue}</div>
            <div class="font-medium text-text-primary">
              {formatCurrency((detailReceipt.items || []).reduce((s, i) => s + i.accepted_qty * i.price, 0))}
            </div>
          </div>
        </div>

        {#if detailReceipt.notes}
          <div class="text-sm">
            <span class="text-text-secondary">{labels.notes}:</span>
            <span class="text-text-primary ml-1">{detailReceipt.notes}</span>
          </div>
        {/if}

        <!-- Items table -->
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
                <th class="px-4 py-3">{labels.consignmentProduct}</th>
                <th class="px-4 py-3 text-right">{labels.consignmentAccepted}</th>
                <th class="px-4 py-3 text-right">{labels.consignmentPricePerUnit}</th>
                <th class="px-4 py-3 text-right">{labels.consignmentTotalValue}</th>
                <th class="px-4 py-3">{labels.consignmentStoreShare}</th>
                <th class="px-4 py-3">{labels.notes}</th>
              </tr>
            </thead>
            <tbody>
              {#each detailReceipt.items || [] as item}
                <tr class="border-b border-border/40">
                  <td class="px-4 py-3">
                    <div class="font-medium text-text-primary">{item.product_name || `Product #${item.product_id}`}</div>
                    <div class="text-xs text-text-secondary">{item.product_sku || ''}</div>
                  </td>
                  <td class="px-4 py-3 text-right text-text-primary">{item.accepted_qty}</td>
                  <td class="px-4 py-3 text-right text-text-primary">{formatCurrency(item.price)}</td>
                  <td class="px-4 py-3 text-right text-text-primary">{formatCurrency(item.accepted_qty * item.price)}</td>
                  <td class="px-4 py-3 text-text-secondary">
                    {item.store_share_type === 'percentage'
                      ? `${item.store_share_value}%`
                      : formatCurrency(item.store_share_value)}
                  </td>
                  <td class="px-4 py-3 text-text-secondary">{item.notes || '-'}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end">
      <Button variant="secondary" onclick={() => (showDetailModal = false)}>{labels.close}</Button>
    </div>
  {/snippet}
</Modal>
```

### Step 5: Add missing i18n labels

**File:** `web/src/shared/i18n/en.ts` (or the equivalent labels file)

Add any missing labels:
- `consignmentReceiptDetail` → "Receipt Detail"
- `consignmentReceivedBy` → "Received By"
- `consignmentPricePerUnit` → "Price/Unit"
- `consignmentStoreShare` → "Store Share"
- `consignmentLoadError` → "Failed to load receipt details"
- `close` → "Close" (may already exist)

Check which labels already exist before adding. Existing labels from the codebase:
- `consignmentReceiptNo`, `consignmentDate`, `consignmentTotalValue`, `consignmentProduct`, `consignmentAccepted`, `notes` — likely already defined

### Step 6: Verify and test

1. **Type check:** `cd web && npx svelte-check`
2. **Build:** `cd web && npm run build`
3. **Manual test:** Navigate to consignment → open arrangement → Receipts tab → click a receipt row → verify detail modal shows items

## Files to modify

| File | Change |
|------|--------|
| `web/src/modules/consignment/components/ReceiptEntry.svelte` | Steps 1-4: detail modal state, openDetail function, clickable rows, modal template |
| `web/src/shared/i18n/en.ts` (or labels file) | Step 5: add missing labels |

## Design decisions

- **Modal (not separate page):** Consistent with all other consignment views (terms, pending returns, settlements all use modals). No new routes needed.
- **Click on row (not "View" button):** Saves space, consistent with the compact table. Cursor pointer + hover feedback makes it discoverable.
- **Read-only modal:** Receipts are immutable once created, so no edit functionality needed.
- **Store share display:** Shows percentage or fixed amount based on `store_share_type`.
- **`getReceipt(id)` already exists:** No backend changes needed — the API and service function are ready.
