# Consignment: Manage Terms Modal

## Overview

The "Add Term" modal is renamed to "Manage Terms" and evolves into a comprehensive term management interface. Users can view, edit, delete, and add consignment terms — all in one modal with per-product pricing.

## Business Context

### Problem

1. **No edit/delete for individual terms** — If a product's price or share needs changing, there is no way to do it. The only mechanism is full replacement via `SetTerms`, but the UI always copies all existing terms unchanged.

2. **Shared pricing only** — All products in a batch must share the same price, share type, and share value. In practice, products have different prices.

3. **Validation gaps** — Price=0 passes frontend validation but fails backend. Share value fields allow values outside valid ranges.

4. **Silent full replacement** — Users may not realize that saving replaces all terms, not just adds new ones.

### Solution

Evolve the modal from "Add Term" to "Manage Terms":

- Load existing terms into an editable table
- Allow per-product pricing (each product can have different price/share)
- Allow deletion of individual terms
- Add confirmation before save
- Fix validation gaps

## Architecture

### No Backend Changes

The existing `SetTerms` endpoint (`PUT /consignment/arrangements/:id/terms`) performs full replacement: deletes all existing terms and inserts the new array. The frontend builds the complete term list (existing + new - deleted) and sends it as a single payload.

### Data Flow

```
Modal Open
  → Initialize editableTerms from arrangement.terms (deep copy)
  → User can edit any row, delete rows, or add new products

Product Selected from Dropdown
  → Add row to editableTerms with default pricing
  → Mark as "new" for visual distinction

Row Deleted
  → Remove from editableTerms

Save
  → Build payload from editableTerms: [{product_id, price, store_share_type, store_share_value}, ...]
  → Call SetTerms(arrangementId, payload)
  → Server deletes all old terms, inserts new array
```

## UI Design

### Modal Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ Manage Terms                                                 [×]│
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Add Products                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ [Search products... ▼]                                      │ │
│  │   ☐ Widget A (WA-001)                                       │ │
│  │   ☐ Widget B (WB-002)                                       │ │
│  │   ────────────────────────────                              │ │
│  │   + Create 1 new product                                    │ │
│  │   + Create multiple products                                │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Terms (3)                                                       │
│  ┌──────────────────────┬─────────┬──────────┬────────┬──────┐  │
│  │ Product              │ Price   │ Type     │ Share  │      │  │
│  ├──────────────────────┼─────────┼──────────┼────────┼──────┤  │
│  │ Widget A (WA-001)    │ [10000] │ [%   ▼] │ [20]   │ [🗑] │  │
│  │ Widget B (WB-002)    │ [15000] │ [Rp  ▼] │ [5000] │ [🗑] │  │
│  │ ★ New (NP-003)       │ [20000] │ [%   ▼] │ [25]   │ [🗑] │  │
│  └──────────────────────┴─────────┴──────────┴────────┴──────┘  │
│                                                                 │
│  Empty state (when no terms):                                    │
│  "Select products from the dropdown above to add terms."        │
│                                                                 │
│  ────────────────────────────────────────────────────────────── │
│                                                                 │
│  Default pricing for new products                                │
│  Price: [0]  Type: [% ▼]  Value: [20]                          │
│  ℹ Applied to newly added products. Existing terms keep their   │
│    original pricing unless edited above.                        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                              [Cancel] [Save]    │
└─────────────────────────────────────────────────────────────────┘
```

### Visual Distinction

- **Existing terms**: Normal row styling, no badge
- **New terms**: Subtle left border accent (primary color) or "New" badge
- **Deleted terms**: Removed from table immediately (no undo within modal)

## Validation Rules

### Frontend

| Field | Rule | Error |
|-------|------|-------|
| Price | `>= 1` | "Price must be at least Rp 1" |
| Share value (percentage) | `1–99` | "Percentage must be 1–99" |
| Share value (fixed) | `1` to `price - 1` | "Amount must be between Rp 1 and Rp {price - 1}" |
| At least one term | `editableTerms.length > 0` | "Add at least one product" |

### Backend (unchanged)

| Rule | Error |
|------|-------|
| `price > 0` | "price must be greater than zero" |
| `share_value > 0` | "share value must be greater than zero" |
| Percentage: `share_value < 100` | "percentage must be less than 100" |
| Fixed: `share_value < price` | "fixed amount must be less than price" |
| No duplicate `product_id` | "duplicate product" |

## i18n Labels

### English

```
consignmentManageTerms: "Manage Terms"
consignmentManageTermsDescription: "Edit existing terms, remove products, or add new ones."
consignmentTermsCount: "{count} terms"
consignmentDefaultPricing: "Default pricing for new products"
consignmentDefaultPricingHint: "Applied to newly added products. Existing terms keep their original pricing unless edited above."
consignmentConfirmReplace: "This will replace all {count} terms. Continue?"
consignmentRemoveTerm: "Remove"
consignmentNoTermsHint: "Select products from the dropdown above to add terms."
```

### Indonesian

```
consignmentManageTerms: "Kelola Term"
consignmentManageTermsDescription: "Edit term yang ada, hapus produk, atau tambah yang baru."
consignmentTermsCount: "{count} term"
consignmentDefaultPricing: "Harga default untuk produk baru"
consignmentDefaultPricingHint: "Diterapkan ke produk yang baru ditambahkan. Term yang ada mempertahankan harga asli kecuali diedit di atas."
consignmentConfirmReplace: "Ini akan mengganti semua {count} term. Lanjutkan?"
consignmentRemoveTerm: "Hapus"
consignmentNoTermsHint: "Pilih produk dari dropdown di atas untuk menambahkan term."
```

## Files

| File | Change |
|------|--------|
| `web/src/modules/consignment/components/TermsEditor.svelte` | Major rewrite — modal logic, per-product table, edit/delete |
| `web/src/shared/i18n/en.ts` | New labels |
| `web/src/shared/i18n/id.ts` | New labels (Indonesian) |

## Implementation Steps

1. Add i18n labels
2. Rewrite `TermsEditor.svelte`:
   - Rename button and modal title
   - Load existing terms into editable state on modal open
   - Build per-product pricing table
   - Wire multi-select dropdown to add rows
   - Add delete button per row
   - Add default pricing section
   - Add confirmation dialog before save
   - Fix validation gaps
3. Verify the implementation
