# Receipts List: Product Filter

**Date:** 2026-09-22
**Status:** Implemented

## Problem

Receipts history tab shows all receipts for a supplier with no way to filter by product. When there are many receipts, finding receipts containing a specific product is tedious.

## Solution

Add a product search input with a clear button to the receipts list. When the search text exactly matches a term product label, the backend `product_id` filter is used (DB-level). Client-side text matching serves as a fallback for partial matches.

## UI

```
┌──────────────────────────────────────────────────────────┐
│ Receipt History    [T-Shirt... ✕]    [Record Receipt]    │
├──────┬──────────┬──────────┬─────────────────────────────┤
│ No.  │ Date     │ Items    │ Total                       │
├──────┼──────────┼──────────┼─────────────────────────────┤
│ R-001│ 20/09/26 │ 3 items  │ Rp 150.000                  │
│ R-003│ 15/09/26 │ 2 items  │ Rp 80.000                   │
└──────┴──────────┴──────────┴─────────────────────────────┘
```

- **Search bar**: Type product name or SKU to filter (uses `SearchBar` component)
- **Clear button** (✕): Resets filter, shows all receipts
- **Empty state**: Shows different message when filter returns no results
- **Pagination**: Updates total count based on filtered results

## Implementation

### Backend (`product_id` query param)

| File | Change |
|------|--------|
| `internal/consignment/handler.go:279` | Parse optional `product_id` query param |
| `internal/consignment/service.go:736` | Add `productID *int` param, normalize 0 → nil |
| `internal/consignment/repository.go:747` | Add dynamic `EXISTS` subquery on `consignment_receipt_items` |

SQL condition:
```sql
EXISTS (SELECT 1 FROM consignment_receipt_items cri
        WHERE cri.consignment_receipt_id = r.id
        AND cri.product_id = $N)
```

### Frontend (wired backend + client-side fallback)

| File | Change |
|------|--------|
| `web/src/modules/consignment/services/consignment-service.ts:127` | `listReceipts(supplierId, productId?)` passes `product_id` to API |
| `web/src/modules/consignment/components/ReceiptEntry.svelte` | `filterProductId` state, `$effect` resolves ID from search text and triggers reload |
| `web/src/shared/i18n/en.ts` | +3 labels: `consignmentFilterByProduct`, `consignmentNoMatchingReceipts`, `consignmentNoMatchingReceiptsSubtitle` |
| `web/src/shared/i18n/id.ts` | Same 3 labels in Indonesian |

### Filter Logic

**Backend filter** (DB-level, when search text exactly matches a term product):
```typescript
$effect(() => {
  const q = filterProductSearch.trim().toLowerCase();
  if (!q) {
    filterProductId = undefined;
  } else {
    const match = productOptions.find(
      (opt) => opt.label.toLowerCase() === q,
    );
    filterProductId = match?.value;
  }
});

$effect(() => {
  void filterProductId;
  load(); // passes filterProductId to listReceipts API
});
```

**Client-side filter** (fallback for partial text matches):
```typescript
const filteredReceipts = $derived.by(() => {
  if (!filterProductSearch.trim()) return receipts;
  const q = filterProductSearch.toLowerCase();
  return receipts.filter((r) =>
    (r.items || []).some(
      (item) =>
        (item.product_name && item.product_name.toLowerCase().includes(q)) ||
        (item.product_sku && item.product_sku.toLowerCase().includes(q)),
    ),
  );
});
```

- Backend filter: exact match on term product label → `product_id` passed to API
- Client-side filter: partial text match on `product_name` / `product_sku` in loaded receipts
- Pagination resets to page 0 on filter change
- `filteredReceipts` feeds into `pagedReceipts` and `Pagination` total

## Why Both Filters?

- **Backend**: efficient for large datasets (DB-level filtering)
- **Client-side**: instant feedback for partial text matches, no extra API call
- The two complement each other: backend handles exact matches, client-side handles fuzzy search
