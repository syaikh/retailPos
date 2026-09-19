# Consignment: Return All Pending Feature

## Overview

Add a "Return All Pending" button that auto-populates the return form with all open pending returns, allowing users to process them in one batch instead of manually linking each one.

## Business Context

### Problem

When a supplier has multiple pending returns (e.g., damaged goods from several visits), the user must:
1. Open the return modal
2. Add a line for each pending return
3. Manually select the pending return from a dropdown for each line

This is tedious for arrangements with many pending returns.

### Solution

Add a "Return All Pending" button that:
1. Opens the return modal
2. Auto-creates one line per open pending return
3. Pre-fills product, qty, and reason from the pending return
4. Links each line to its pending return

The user can then review, adjust quantities, or remove lines before submitting.

## UI Design

### Button Placement

Next to the existing "Record Return" button in the Returns page header:

```
┌─────────────────────────────────────────────────────────────┐
│ Returns (Hand-back to Supplier)                             │
│                          [Return All Pending] [Record Return]│
├─────────────────────────────────────────────────────────────┤
│ ⚠ 3 pending returns are not yet returned — resolve them...  │
└─────────────────────────────────────────────────────────────┘
```

### Pre-populated Modal (Return All Pending)

When "Return All Pending" is clicked, pre-filled lines use a **compact read-only layout** with blue-tinted borders to distinguish them from manual lines:

```
┌─────────────────────────────────────────────────────────────┐
│ Record Return                                            [×]│
├─────────────────────────────────────────────────────────────┤
│ ℹ 3 pending returns are not yet returned                    │
│                                                             │
│ Item lines                                    [Add Line]    │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Widget A (WA-001)                                       │ │
│ │ Reason: Damaged                                         │ │
│ │                                    Qty: [5]        [🗑] │ │
│ │ Notes: [___________________________________________]   │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Widget B (WB-002)                                       │ │
│ │ Reason: Expired                                         │ │
│ │                                    Qty: [3]        [🗑] │ │
│ │ Notes: [___________________________________________]   │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ Overall Notes                                               │
│ [_______________________________________________]          │
├─────────────────────────────────────────────────────────────┤
│                                              [Cancel] [Save]│
└─────────────────────────────────────────────────────────────┘
```

### Manual Lines (Record Return / Add Line)

Manual lines keep the full form with product selector, reason dropdown, and pending return link:

```
┌─────────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Product: [Select product ▾]  Qty: [1]  Reason: [Other] │ │
│ │ Link to pending return: [— No link —]                  │ │
│ │ Notes: [___________________________________________]   │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Implementation

### Changes

| File | Change |
|------|--------|
| `ReturnPage.svelte` | Add `returnAllPending()` function, add button, `fromPending` flag on Line interface, compact modal layout for pre-filled lines |
| `en.ts` | Add `consignmentReturnAllPending` label |
| `id.ts` | Add `consignmentReturnAllPending` label |

### Logic

```typescript
interface Line {
  product_id?: number;
  qty: number;
  reason: string;
  pending_return_id?: number;
  notes: string;
  fromPending?: boolean;  // ← true when pre-filled from a pending return
}

function returnAllPending() {
  lines = openPending.map((pr) => ({
    product_id: pr.product_id,
    qty: pr.qty,
    reason: pr.reason,
    pending_return_id: pr.id,
    notes: "",
    fromPending: true,
  }));
  returnNotes = "";
  showModal = true;
}
```

### Dual Layout

The modal renders two distinct layouts based on `line.fromPending`:

| Layout | When | Fields |
|--------|------|--------|
| **Compact** | `fromPending === true` | Product name (read-only), reason (read-only), qty (editable), notes (editable), delete |
| **Full form** | `fromPending === false` | Product selector, qty, reason dropdown, pending return link, notes, delete |

Compact lines use `border-primary/20 bg-primary/5` styling to visually distinguish them from manual lines.

### Button Visibility

The "Return All Pending" button only appears when there are open pending returns (`openPending.length > 0`).

## i18n Labels

### English

```
consignmentReturnAllPending: "Return All Pending"
```

### Indonesian

```
consignmentReturnAllPending: "Kembalikan Semua Pending"
```
