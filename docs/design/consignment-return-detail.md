# Consignment: Return Detail View

## Overview

Add a clickable return row that opens a detail modal showing the return header and line items.

## UI Design

### Return Row (Updated)

Return rows become clickable with a hover indicator:

```
┌─────────────────────────────────────────────────────────────┐
│ Returns (Hand-back to Supplier)                             │
│                          [Return All Pending] [Record Return]│
├─────────────────────────────────────────────────────────────┤
│ No. Retur    Date          Items  Qty   Notes               │
│─────────────────────────────────────────────────────────────│
│ RT-001       2026-09-18    3      11    ← clickable row    │
│ RT-002       2026-09-17    1      5     ← clickable row    │
└─────────────────────────────────────────────────────────────┘
```

### Detail Modal

Clicking a return row opens a read-only detail modal:

```
┌─────────────────────────────────────────────────────────────┐
│ RT-001                                                  [×]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ Return No.        Date              Returned By             │
│ RT-001            2026-09-18 14:30  admin                   │
│                                                             │
│ Notes                                                        │
│ Customer returned damaged goods                              │
│                                                             │
│ Line Items                                                   │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Product          Qty  Reason    Pending #  Notes        │ │
│ │─────────────────────────────────────────────────────────│ │
│ │ Widget A (WA-001) 5  Damaged    PR-3       —            │ │
│ │ Widget B (WB-002) 3  Expired    PR-4       —            │ │
│ │ Widget C (WC-003) 3  Damaged    —          Bulk return  │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│                                              [Close]        │
└─────────────────────────────────────────────────────────────┘
```

## Implementation

### Changes

| File | Change |
|------|--------|
| `ReturnPage.svelte` | Add `selectedReturn`, `showDetailModal`, `loadDetail()`, click handler on rows, detail modal |
| `en.ts` | Add `consignmentReturnDetail`, `consignmentReturnedBy` labels |
| `id.ts` | Add `consignmentReturnDetail`, `consignmentReturnedBy` labels |

### Logic

```typescript
let selectedReturn = $state<ConsignmentReturn | null>(null);
let showDetailModal = $state(false);
let loadingDetail = $state(false);

async function loadDetail(id: number) {
  loadingDetail = true;
  try {
    selectedReturn = await getReturn(id);
    showDetailModal = true;
  } catch {
    toast.error("Failed to load return details");
  } finally {
    loadingDetail = false;
  }
}
```

### Row Click

```svelte
<tr
  class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
  onclick={() => loadDetail(r.id)}
>
```

## i18n Labels

### English

```
consignmentReturnDetail: "Return Detail"
consignmentReturnedBy: "Returned By"
```

### Indonesian

```
consignmentReturnDetail: "Detail Retur"
consignmentReturnedBy: "Dikembalikan Oleh"
```
