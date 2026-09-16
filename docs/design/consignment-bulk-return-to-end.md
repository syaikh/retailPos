# Consignment Bulk Return to End Arrangement

> **Status:** Implemented

## Problem

When a user tries to end a consignment arrangement with remaining stock, the backend returns a 409 Conflict (`CNS-201: all consignment stock must be returned before ending`). The user gets a generic toast error and must manually:

1. Go to Pending Returns tab → create pending return for each product
2. Go to Returns tab → record return for each pending return
3. Repeat until all stock is returned
4. Try ending again

This is tedious and error-prone. Users need a one-click solution to return all remaining stock.

## Solution (Two Parts)

### Part 1: Fix Return Tab Product Dropdown (Bug Fix)

**Problem:** The Return tab's Record Return modal shows ALL active products in the system (~hundreds), but only a handful are consignment items for this supplier. Users can accidentally select wrong products.

**Fix:** Replace global product list with stock-scoped list, filtered to `available_qty > 0` for this arrangement. Show stock count in dropdown label.

### Part 2: Bulk Return All Stock (New Feature)

When the user tries to end with remaining stock:
1. Show a guidance banner with a "Return All Remaining Stock" button
2. Click opens a confirmation modal showing all products to be returned
3. On confirm → single `createReturn` call with all items → user can try ending again

## Mockups

### Banner (shown after 409 error on End Arrangement)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠ All consignment stock must be returned before ending this arrangement.  │
│  Please return the remaining stock below.                   [Dismiss]      │
└─────────────────────────────────────────────────────────────────────────────┘
```

- Amber background, border, icon
- Dismiss button to hide banner
- Auto-hides when arrangement is successfully ended

### Bulk Return Confirmation Modal

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Return All Remaining Stock                                           [X]  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  The following products will be returned to the supplier:                   │
│                                                                             │
│  ┌─────────────────────────────────┬─────────────────────────────────────┐  │
│  │  PRODUCT                        │  QTY TO RETURN                      │  │
│  ├─────────────────────────────────┼─────────────────────────────────────┤  │
│  │  Indomie Goreng (INDO-001)      │  5                                  │  │
│  │  Teh Botol Sosro (SOSRO-002)    │  3                                  │  │
│  │  Pop Mie (NISSIN-003)           │  2                                  │  │
│  ├─────────────────────────────────┼─────────────────────────────────────┤  │
│  │  Total: 3 products              │  10 items                           │  │
│  └─────────────────────────────────┴─────────────────────────────────────┘  │
│                                                                             │
│  Reason: Arrangement termination                                            │
│                                                                             │
│  ⚠ This action cannot be undone.                                           │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                         [Cancel] [Return]   │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Design points:**
- Read-only product table — user sees exactly what will be returned
- Reason is hardcoded as "Arrangement termination" — no dropdown needed
- Total row shows product count + item count
- Warning text emphasizes irreversibility
- Danger-styled confirm button

## How the Existing API Supports This

The `POST /consignment/returns` endpoint already supports multi-item returns:

```json
{
  "arrangement_id": 42,
  "notes": "Bulk return — ending arrangement",
  "items": [
    { "product_id": 101, "qty": 2, "reason": "termination", "pending_return_id": 7 },
    { "product_id": 101, "qty": 5, "reason": "termination" },
    { "product_id": 202, "qty": 3, "reason": "termination" }
  ]
}
```

**Key detail:** A single product can have two entries — one linked to a pending return (reduces `pending_return_qty`), one free (reduces `available_qty`). Both run atomically in one transaction.

**No new backend endpoint needed.** The existing `CreateReturn` service method handles:
- Multiple items in one transaction
- Pending return linking (Path A) and free returns (Path B)
- Stock adjustment via `ApplyConsignmentDelta`
- Row-level locking via `SELECT ... FOR UPDATE`

## Files to Change

### 1. Backend: Add `termination` to valid return reasons

**File:** `internal/consignment/service.go`

In `CreateReturn`, the reason validation currently accepts: `damaged`, `expired`, `customer_return`, `other`. Add `termination`.

```go
// Before (service.go ~line 1010)
validReasons := map[string]bool{
    "damaged": true, "expired": true, "customer_return": true, "other": true,
}

// After
validReasons := map[string]bool{
    "damaged": true, "expired": true, "customer_return": true,
    "termination": true, "other": true,
}
```

### 2. Frontend: Add `termination` reason constant and label

**File:** `web/src/modules/consignment/types/index.ts`

```typescript
// Add constant
export const RETURN_REASON_TERMINATION = "termination";

// Add to RETURN_REASON_LABELS
export const RETURN_REASON_LABELS: Record<string, keyof Labels> = {
  // ... existing ...
  [RETURN_REASON_TERMINATION]: "returnReasonTermination",
};

// Add to RETURN_REASONS array
export const RETURN_REASONS = [
  RETURN_REASON_DAMAGED,
  RETURN_REASON_EXPIRED,
  RETURN_REASON_CUSTOMER_RETURN,
  RETURN_REASON_TERMINATION,  // ← new
  RETURN_REASON_OTHER,
];
```

**File:** `web/src/shared/i18n/en.ts`

```typescript
returnReasonTermination: "Arrangement termination",
```

**File:** `web/src/shared/i18n/id.ts`

```typescript
returnReasonTermination: "Pengakhiran kesepakatan",
```

### 3. Frontend: Add `bulkReturnAllStock()` service function

**File:** `web/src/modules/consignment/services/consignment-service.ts`

```typescript
export async function bulkReturnAllStock(
  arrangementId: number,
  supplierId: number,
): Promise<ConsignmentReturn> {
  const [stock, pendingReturns] = await Promise.all([
    listStock(supplierId),
    listPendingReturns(supplierId),
  ]);

  const openPending = pendingReturns.filter(
    (pr) => pr.status === "open" && pr.arrangement_id === arrangementId,
  );

  const items: ReturnItemPayload[] = [];

  for (const row of stock) {
    if (row.arrangement_id !== arrangementId) continue;
    const totalQty = row.available_qty + row.pending_return_qty;
    if (totalQty <= 0) continue;

    // Find open pending returns for this product
    const productPending = openPending.filter(
      (pr) => pr.product_id === row.product_id,
    );

    if (productPending.length > 0) {
      // Path A: Resolve pending returns first
      for (const pr of productPending) {
        items.push({
          product_id: row.product_id,
          qty: pr.qty,
          reason: "termination",
          pending_return_id: pr.id,
        });
      }
    }

    // Path B: Return remaining available stock
    if (row.available_qty > 0) {
      items.push({
        product_id: row.product_id,
        qty: row.available_qty,
        reason: "termination",
      });
    }
  }

  if (items.length === 0) {
    throw new Error("No stock to return");
  }

  return createReturn({
    arrangement_id: arrangementId,
    notes: "Bulk return — ending arrangement",
    items,
  });
}
```

### 4. Frontend: Bulk return modal in ArrangementsPage

**File:** `web/src/modules/consignment/components/ArrangementsPage.svelte`

**State additions:**

```typescript
let showBulkReturnModal = $state(false);
let bulkReturnStock = $state<StockRow[]>([]);
let bulkReturning = $state(false);
```

**Load stock when 409 error occurs:**

In `confirmEndArrangement()` catch block, also load stock for the modal:

```typescript
if (msg.includes("stock must be returned")) {
  showEndModal = false;
  showReturnBanner = true;
  activeTab = "return";
  // Load stock for bulk return modal
  try {
    bulkReturnStock = await listStock(activeArrangement.supplier_id);
  } catch {
    bulkReturnStock = [];
  }
}
```

**Banner update — add "Return All" button:**

```svelte
{#if showReturnBanner}
  <div class="rounded-xl border border-amber-300 bg-amber-50 p-4 flex items-center justify-between">
    <div class="flex items-center gap-2 text-amber-800 text-sm">
      <AlertTriangle class="w-4 h-4 shrink-0" />
      <span>{t("consignmentReturnStockToEnd")}</span>
    </div>
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={() => (showBulkReturnModal = true)}>
        {labels.consignmentReturnAllStock}
      </Button>
      <Button variant="ghost" size="sm" onclick={() => (showReturnBanner = false)}>
        {labels.dismiss}
      </Button>
    </div>
  </div>
{/if}
```

**Bulk return confirmation modal:**

```svelte
<Modal bind:open={showBulkReturnModal} title={labels.consignmentReturnAllStock} size="md">
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">
      {labels.consignmentBulkReturnDescription}
    </p>

    {#if filteredStock.length > 0}
      <div class="rounded-lg border border-border-default overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-muted/50">
            <tr class="text-left text-xs uppercase tracking-wider text-text-secondary">
              <th class="px-4 py-2">{labels.consignmentProduct}</th>
              <th class="px-4 py-2 text-right">{labels.consignmentQty}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredStock as row (row.product_id)}
              <tr class="border-t border-border/40">
                <td class="px-4 py-2">
                  <span class="font-medium">{row.product_name}</span>
                  {#if row.product_sku}
                    <span class="text-text-muted ml-1">({row.product_sku})</span>
                  {/if}
                </td>
                <td class="px-4 py-2 text-right">{row.available_qty}</td>
              </tr>
            {/each}
          </tbody>
          <tfoot class="bg-muted/30 border-t border-border/50">
            <tr class="text-sm font-medium">
              <td class="px-4 py-2">
                {t("consignmentBulkReturnProductCount", { count: filteredStock.length })}
              </td>
              <td class="px-4 py-2 text-right">
                {t("consignmentBulkReturnTotalQty", { count: totalReturnQty })}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    {:else}
      <p class="text-sm text-text-muted">{labels.consignmentNoStockToReturn}</p>
    {/if}

    <div class="rounded-lg bg-muted/30 p-3">
      <p class="text-sm text-text-secondary">
        <span class="font-medium">{labels.consignmentReason}:</span>
        {labels.returnReasonTermination}
      </p>
    </div>

    <p class="text-xs text-danger">{labels.consignmentBulkReturnWarning}</p>
  </div>

  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showBulkReturnModal = false)}>
        {labels.cancel}
      </Button>
      <Button
        variant="danger"
        onclick={confirmBulkReturn}
        disabled={bulkReturning || filteredStock.length === 0}
      >
        {bulkReturning ? labels.saving : labels.consignmentReturnAllStock}
      </Button>
    </div>
  {/snippet}
</Modal>
```

**Bulk return submit function:**

```typescript
async function confirmBulkReturn() {
  if (!activeArrangement) return;
  bulkReturning = true;
  try {
    await bulkReturnAllStock(activeArrangement.id, activeArrangement.supplier_id);
    toast.success(labels.consignmentBulkReturnSuccess);
    showBulkReturnModal = false;
    showReturnBanner = false;
    await refreshArrangement();
  } catch (e: unknown) {
    toast.error(getApiErrorMessage(e, labels.consignmentBulkReturnError));
  } finally {
    bulkReturning = false;
  }
}
```

**Derived for filtered stock:**

```typescript
const filteredStock = $derived(
  bulkReturnStock.filter(
    (s) => s.arrangement_id === activeArrangement?.id && s.available_qty > 0,
  ),
);

const totalReturnQty = $derived(
  filteredStock.reduce((sum, s) => sum + s.available_qty, 0),
);
```

### 5. New i18n labels

**File:** `web/src/shared/i18n/en.ts`

```typescript
consignmentReturnAllStock: "Return All Remaining Stock",
consignmentBulkReturnDescription:
  "The following products will be returned to the supplier. This action cannot be undone.",
consignmentBulkReturnProductCount: "{count} products",
consignmentBulkReturnTotalQty: "{count} items total",
consignmentNoStockToReturn: "No remaining stock to return.",
consignmentBulkReturnWarning:
  "This will create a return document for all remaining consignment stock. The arrangement can be ended after this.",
consignmentBulkReturnSuccess: "All remaining stock returned successfully",
consignmentBulkReturnError: "Failed to return stock",
```

**File:** `web/src/shared/i18n/id.ts`

```typescript
consignmentReturnAllStock: "Kembalikan Semua Stok Tersisa",
consignmentBulkReturnDescription:
  "Produk berikut akan dikembalikan ke pemasok. Tindakan ini tidak dapat dibatalkan.",
consignmentBulkReturnProductCount: "{count} produk",
consignmentBulkReturnTotalQty: "{count} item total",
consignmentNoStockToReturn: "Tidak ada stok tersisa untuk dikembalikan.",
consignmentBulkReturnWarning:
  "Ini akan membuat dokumen retur untuk semua stok konsinyasi yang tersisa. Kesepakatan dapat diakhiri setelah ini.",
consignmentBulkReturnSuccess: "Semua stok tersisa berhasil dikembalikan",
consignmentBulkReturnError: "Gagal mengembalikan stok",
```

## Summary of All File Changes

| File | Change | Type |
|------|--------|------|
| `internal/consignment/service.go` | Add `"termination"` to valid reasons | Bug fix |
| `web/src/modules/consignment/types/index.ts` | Add `RETURN_REASON_TERMINATION` + label mapping | Feature |
| `web/src/shared/i18n/en.ts` | Add `termination` label + 8 bulk return labels | Feature |
| `web/src/shared/i18n/id.ts` | Add `termination` label + 8 bulk return labels | Feature |
| `web/src/modules/consignment/services/consignment-service.ts` | Add `bulkReturnAllStock()` function | Feature |
| `web/src/modules/consignment/components/ReturnPage.svelte` | Replace global products with stock-scoped dropdown + qty validation | Bug fix |
| `web/src/modules/consignment/components/ArrangementsPage.svelte` | Add 409 redirect + banner + bulk return modal | Feature |

## Implementation Order

1. Backend: Add `"termination"` reason to `service.go`
2. Frontend types: Add `RETURN_REASON_TERMINATION` to `types/index.ts`
3. Frontend i18n: Add labels to `en.ts` and `id.ts`
4. Frontend service: Add `bulkReturnAllStock()` to `consignment-service.ts`
5. Frontend UI: Update `ReturnPage.svelte` (stock-scoped dropdown)
6. Frontend UI: Update `ArrangementsPage.svelte` (redirect + banner + modal)
7. Verify: Prettier + svelte-check
