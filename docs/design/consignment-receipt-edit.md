# Consignment Receipt Edit Feature

## Problem

Users may mistakenly input incorrect quantities or prices when recording a consignment receipt. Currently, receipts are immutable — there's no way to fix errors without creating workarounds. This creates operational friction and potential data accuracy issues.

## Architecture Decisions

### Q1: Who should have permission to update receipts?

**Current permission model:**
| Role | `consignment.view` | `consignment.create` | `consignment.update` | `consignment.settle` | `consignment.pay` |
|------|:--:|:--:|:--:|:--:|:--:|
| Superadmin | ✅ | ✅ | ✅ | ✅ | ✅ |
| Admin | ✅ | ✅ | ✅ | ✅ | ✅ |
| Manager | ✅ | ✅ | ✅ | ✅ | ❌ |
| Cashier | ✅ | ❌ | ❌ | ❌ | ❌ |

**Decision:** Reuse `consignment.update` for receipt editing.

**Rationale:**
- `consignment.update` is defined as "Ubah Terms Konsinyasi" — currently only for editing arrangement terms
- Expanding its scope to include receipt editing is a natural extension
- Managers already have this permission, which makes sense (they can create AND fix receipts)
- No new permission needed — keeps the permission model simple

**Alternative considered:** Create `consignment.edit_receipt` — rejected as over-engineering for v1.

---

### Q2: How does the stock model work and how does receipt editing fit?

**Exclusive Ownership Model (BR-03/BR-18):**
A product is EITHER store-owned OR consignment-owned — never both. Enforced by `UNIQUE (product_id)` on `consignment_stock` table and BR-02 validation at receipt creation.

```
Store-owned product:
  product_stock: 50 units (sellable)
  consignment_stock: NO ROW

Consignment-owned product:
  product_stock: 30 units (sellable)
  consignment_stock: 30 units (owned by Supplier X)
```

**Two Ledgers — Different Purposes:**

| Ledger | Purpose | Updated By |
|--------|---------|------------|
| `product_stock` (global row) | What POS can sell | `ConsignmentAdjuster.ApplyConsignmentDelta()` |
| `consignment_stock` | Who owns it (for settlement) | `UpsertConsignmentStock()` |

For consignment products, both ledgers always have the **same quantity**, but `consignment_stock` has additional metadata (supplier_id, arrangement_id, available_qty vs pending_return_qty) needed for settlement calculations.

**Atomicity Guarantee:**
Both updates run inside the same `pgx.Tx` (Unit of Work pattern from `ADR_Cross_Module_Transaction_Strategy`). If either fails, both roll back.

**Decision:** REUSE `ConsignmentAdjuster.ApplyConsignmentDelta()` — do NOT introduce a new stock adjustment process.

**Current receipt creation flow:**
```
Receipt created → UpsertConsignmentStock(+qty) AND ApplyConsignmentDelta(+qty) [same tx]
```

**Receipt editing flow (same pattern):**
```
Receipt edited → Calculate delta (new - old) → UpsertConsignmentStock(delta) AND ApplyConsignmentDelta(delta) [same tx]
```

**Rationale:**
- `ConsignmentAdjuster` already handles:
  - Locking `product_stock` row (`FOR UPDATE`)
  - Applying signed delta to global stock
  - Writing to `inventory_movements` ledger
  - Clamping at zero (prevents negative stock)
- The adjuster is battle-tested and correct
- No need to reinvent the wheel

---

### Q3: How should price change be handled?

**Key insight from code analysis:**

| When | Price Source | Used For |
|------|-------------|----------|
| Receipt creation | `term.Price` (snapshotted) | Stock valuation, audit trail |
| POS Sale | `item.UnitPrice` (actual sale price) | Settlement calculation |
| Settlement | `consignment_sale_items.unit_price` | Supplier payout |

**The receipt price is NOT used for settlement** — settlement uses the actual POS sale price.

**Decision:**
- **Quantity editing:** Always allowed (if no downstream activity) — stock delta recalculation is straightforward
- **Price editing:** Block if any item from this receipt has been sold
  - Rationale: Changing price after sale creates audit inconsistency
  - The receipt price is for audit trail / stock valuation, not settlement
  - If user needs to adjust pricing for sold items, they should use settlement adjustments (separate feature)

**Editable fields (revised):**
| Field | Can Edit | Condition |
|-------|----------|-----------|
| Accepted qty | ✅ | No downstream activity |
| Notes | ✅ | Always |
| Price per unit | ✅ | Only if NO sales exist for this receipt |
| Store share type/value | ✅ | Only if NO sales exist for this receipt |
| Receipt number | ❌ | Immutable |
| Receipt date | ❌ | Immutable (prevents backdating) |
| Supplier | ❌ | Immutable |
| Received by | ❌ | Immutable |

---

### Q4: Is there an existing process we can reuse?

**Decision:** Yes — reuse the **inventory module's stock adjustment process** from Product Master Data.

**Existing pattern in `internal/inventory/handler.go:AdjustStock`:**
```go
// POST /inventory/adjust
// Input: { product_id, quantity_change, notes }
// Flow:
// 1. Validate: quantity_change != 0, notes required
// 2. AdjustStockTx() — applies delta to product_stock
// 3. AuditLogTx() — writes to audit_logs
// 4. Publishes StockAdjusted event
```

**Existing pattern in `web/src/modules/inventory/components/StockAdjustModal.svelte`:**
```
┌─────────────────────────────────────────────┐
│  Adjust Stock                          [X]  │
├─────────────────────────────────────────────┤
│  Product: Indomie Goreng                    │
│  Current Stock: 150                         │
│                                             │
│  Quantity Change *                          │
│  [________] (positive to add, negative to   │
│              reduce)                        │
│                                             │
│  Notes *                                    │
│  [________________________________________] │
│  Reason for adjustment                      │
│                                             │
│              [Cancel]  [Adjust Stock]       │
└─────────────────────────────────────────────┘
```

**Reuse for receipt editing:**
| Inventory Module | Receipt Edit | Purpose |
|-----------------|--------------|---------|
| `POST /inventory/adjust` | `PUT /consignment/receipts/:id` | Update endpoint |
| `quantity_change` (signed delta) | `accepted_qty delta` | Stock delta calculation |
| `notes` (mandatory) | `reason` (mandatory) | Justification |
| `AdjustStockTx()` | `ApplyConsignmentDelta()` | Stock adjustment |
| `AuditLogTx()` | `InsertReceiptEdit()` | Audit trail |
| `StockAdjustModal` | Edit mode in detail modal | UI pattern |

**Key insight:** The inventory module already has a stock adjustment process with audit trail. We can reuse this exact pattern:
1. Calculate delta (new_qty - old_qty)
2. Call `ApplyConsignmentDelta(delta)` — same as receipt creation
3. Write audit entry — same pattern as `AuditLogTx`

**Also reuse from stock opname module:**
- `isEditableStatus()` pattern — guardrail function
- `CountRecord` pattern — immutable audit trail with sequence numbers

---

### Q5: What about inventory page adjustments on consignment products?

**Problem identified:** Currently, the inventory module's `POST /inventory/adjust` has NO guardrail against adjusting consignment products. If a manager adjusts a consignment product via the inventory page:
- `product_stock` changes ✅
- `consignment_stock` unchanged ❌
- **Inconsistency created** — settlement data wrong

**Root cause:** `checkProductStore()` only validates store ownership, not consignment ownership.

**Decision:** Add inventory guardrail as **prerequisite task** (Phase 0) before receipt editing.

**Solution — New Port + Check:**
```go
// internal/consignment/ports.go — new port
type ConsignmentOwnerChecker interface {
    IsConsignmentOwned(ctx context.Context, productID int) (bool, error)
}

// internal/inventory/repository.go — add check in AdjustStockTx
func (r *Repository) AdjustStockTx(...) error {
    if r.consignmentOwner != nil {
        owned, _ := r.consignmentOwner.IsConsignmentOwned(ctx, productID)
        if owned {
            return ErrConsignmentProduct  // "Consignment products cannot be adjusted via inventory"
        }
    }
    // ... existing logic
}
```

**Also add frontend guardrail:**
- Show "Konsinyasi" badge on consignment products in inventory page
- Disable adjust button for consignment products
- Show tooltip: "Stok konsinyasi hanya bisa diubah melalui modul Konsinyasi"

---

## Business Context

### Why not Void Receipt?
- Void creates extra documents that clutter the system
- Operators think "I typed 50, should be 55, let me fix it" — not "let me void and re-create"
- Single correct document is cleaner than void + correction pair

### Why not just add notes?
- Doesn't fix the root problem (wrong quantities/prices)
- Stock levels remain incorrect
- Settlement calculations remain incorrect

## Design: Edit Receipt with Guardrails

### Core Principle
Edit in place with safeguards. The system must:
1. Prevent edits that would break downstream data
2. Maintain a complete, immutable audit trail
3. Require justification for every edit
4. Restrict who can edit and when

### Business Rules

#### Editable Fields
| Field | Can Edit | Condition |
|-------|----------|-----------|
| Accepted qty | ✅ | No downstream activity (sold/returned/settled) |
| Price per unit | ✅ | Only if NO sales exist for this receipt |
| Store share type/value | ✅ | Only if NO sales exist for this receipt |
| Notes | ✅ | Always editable (low risk) |
| Receipt number | ❌ | Immutable document identifier |
| Receipt date | ❌ | Prevents backdating fraud |
| Supplier | ❌ | Immutable per receipt |
| Received by | ❌ | Immutable per receipt |

**Price edit rationale:** The receipt price is snapshotted from `term.Price` at creation time and used for audit trail / stock valuation. Settlement uses the actual POS sale price (`consignment_sale_items.unit_price`), NOT the receipt price. Changing receipt price after sales would create audit inconsistency.

#### Edit Restrictions

**Block edit if:**
1. Any quantity from this receipt has been sold (partial or full)
2. Any quantity has been pending returned or returned
3. Settlement period is closed (receipt older than 7 days OR financial month closed)
4. Receipt is already settled

**Allow edit if:**
- All quantities are still in stock (no downstream activity)
- Within edit window (7 days from receipt date)
- User has `consignment.update` permission

#### Stock Recalculation

**Reuses existing `ConsignmentAdjuster.ApplyConsignmentDelta()` pattern from receipt creation.**

**Atomicity guarantee:** Both ledgers updated in same `pgx.Tx` — all-or-nothing.

```
Original receipt: accepted_qty = 50, price = 10000
Stock impact: +50 units (via UpsertConsignmentStock(+50) AND ApplyConsignmentDelta(+50))

Edit to: accepted_qty = 55, price = 10000
Delta: +5 units
Stock impact: +55 units (via UpsertConsignmentStock(+5) AND ApplyConsignmentDelta(+5))

Edit to: accepted_qty = 45, price = 10000
Delta: -5 units
Stock impact: +45 units (via UpsertConsignmentStock(-5) AND ApplyConsignmentDelta(-5))
```

**Block if delta would cause negative stock:**
- Cannot reduce accepted_qty below what's been sold/returned
- The `ConsignmentAdjuster` already clamps at zero, but we should validate before applying

**Also update `consignment_stock` ownership ledger:**
- `UpsertConsignmentStock(delta)` — same function used in receipt creation

### Fraud Prevention

#### Layer 1: Access Control
- Only users with `consignment.update` permission can edit
- Logs which user made the edit (immutable audit record)

#### Layer 2: Immutable Audit Trail
New table `consignment_receipt_edits` (append-only):
```sql
CREATE TABLE consignment_receipt_edits (
    id SERIAL PRIMARY KEY,
    receipt_id INTEGER NOT NULL REFERENCES consignment_receipts(id),
    edited_by INTEGER NOT NULL REFERENCES users(id),
    edited_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    field TEXT NOT NULL,           -- 'accepted_qty', 'price', 'notes', etc.
    old_value TEXT NOT NULL,       -- JSON string of previous value
    new_value TEXT NOT NULL,       -- JSON string of new value
    reason TEXT NOT NULL,          -- Mandatory justification
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Append-only trigger (same pattern as audit_logs)
CREATE TRIGGER trg_consignment_receipt_edits_immutable
    BEFORE UPDATE OR DELETE ON consignment_receipt_edits
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();
```

**This table is NEVER deletable, NEVER editable** — even by admins.

#### Layer 3: Business Rules
- Cannot edit after settlement period closes (7 days)
- Cannot edit after financial month close
- Cannot edit if any item quantity has been sold/returned
- Requires mandatory `reason` field on every edit

#### Layer 4: Approval Workflow (v2 - Deferred)
- Edits below threshold (qty delta ≤ 5, price change ≤ 5%) → auto-approve
- Edits above threshold → require supervisor approval
- Approval logged in same audit table

#### Layer 5: Monitoring (v2 - Deferred)
- Weekly report: "All receipt edits this week"
- Alert on suspicious patterns (same user editing frequently, large deltas)

### API Design

#### Endpoint
```
PUT /consignment/receipts/:id
```

#### Request Body
```json
{
  "items": [
    {
      "id": 1,
      "accepted_qty": 55,
      "price": 10000,
      "notes": "Corrected quantity"
    }
  ],
  "notes": "Corrected total receipt notes",
  "reason": "Operator typo: 50 should be 55"
}
```

#### Response
```json
{
  "data": {
    "id": 1,
    "receipt_number": "RCP-2026-00042",
    "items": [...],
    "updated_at": "2026-09-07T10:30:00Z"
  }
}
```

#### Error Responses
- `400 Bad Request` — Edit not allowed (downstream activity detected)
- `400 Bad Request` — Consignment product cannot be adjusted via inventory (Phase 0)
- `403 Forbidden` — User lacks `consignment.update` permission
- `409 Conflict` — Receipt outside edit window (7 days)
- `422 Unprocessable Entity` — Validation error (negative stock, missing reason)

### Frontend Design

#### Edit Button
- Show "Edit" button in detail modal footer (next to "Close")
- Only visible if user has `consignment.update` permission
- Disabled/hidden if receipt is outside edit window or has downstream activity

#### Edit Mode
When user clicks "Edit":
1. Detail modal switches to edit mode
2. Accepted qty, price, notes become editable inputs
3. Real-time calculation shows: "Stock delta: +5 units"
4. Mandatory "Reason" textarea appears
5. "Save Changes" and "Cancel" buttons replace "Close"

#### Edit Mode Mockup
```
┌─────────────────────────────────────────────────────────────────────────────┐
│  RCP-2026-00042 — Edit Receipt                                    [X]    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  RECEIPT NO.        DATE                RECEIVED BY                         │
│  ─────────────      ─────────────────   ──────────────                      │
│  RCP-2026-00042     5 Sep 2026 14:30    operator_1                          │
│                                                                             │
│  ┌─────────────────────┬──────────┬────────────┬────────────┬──────────┐   │
│  │  PRODUCT            │ ACCEPTED │  PRICE/UNIT│  NEW TOTAL │  DELTA   │   │
│  ├─────────────────────┼──────────┼────────────┼────────────┼──────────┤   │
│  │  Indomie Goreng     │ [55]     │ [Rp 2.500] │ Rp 137.500 │  +5 qty  │   │
│  │  INDO-001           │          │            │            │          │   │
│  ├─────────────────────┼──────────┼────────────┼────────────┼──────────┤   │
│  │  Teh Botol Sosro    │ [100]    │ [Rp 3.000] │ Rp 300.000 │  0      │   │
│  │  SOSRO-002          │          │            │            │          │   │
│  └─────────────────────┴──────────┴────────────┴────────────┴──────────┘   │
│                                                                             │
│  Reason *: [Operator typo: 50 should be 55________________________]        │
│                                                                             │
│  Notes: [Corrected total receipt notes________________________________]     │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                              [Cancel]  [Save Changes]      │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Downstream Activity Check
When user clicks "Edit", system checks:
```
IF (any quantity from this receipt has been sold)
   OR (any quantity is pending return)
   OR (receipt older than 7 days)
   OR (receipt is settled)
THEN
   Show toast: "Cannot edit: receipt has downstream activity"
   Show info badge: "Settled" or "Has sales" or "Outside edit window"
ELSE
   Enable edit mode
```

## Implementation Plan

### Phase 0: Inventory Guardrail (Prerequisite)

**Goal:** Prevent inventory module from adjusting consignment products (which would create inconsistency between `product_stock` and `consignment_stock`).

**Files to change:**
| File | Change |
|------|--------|
| `internal/consignment/ports.go` | Add `ConsignmentOwnerChecker` interface |
| `internal/consignment/service.go` | Implement `IsConsignmentOwned()` |
| `internal/inventory/repository.go` | Add `consignmentOwner` field + check in `AdjustStockTx` |
| `internal/inventory/handler.go` | Wire `ConsignmentOwnerChecker` via constructor |
| `cmd/server/main.go` | Wire consignment service as `ConsignmentOwnerChecker` |
| `web/src/modules/inventory/components/StockAdjustModal.svelte` | Disable adjust for consignment products |
| `web/src/shared/i18n/en.ts`, `id.ts` | Add error message labels |

**Backend changes:**
1. Add `ConsignmentOwnerChecker` port in `internal/consignment/ports.go`
2. Implement `IsConsignmentOwned()` in `internal/consignment/service.go`
3. Add `consignmentOwner` field to inventory `Repository`
4. Add check in `AdjustStockTx()`: if product is consignment-owned, return `ErrConsignmentProduct`
5. Wire in `cmd/server/main.go`

**Frontend changes:**
1. In `StockAdjustModal.svelte`: disable adjust button if product is consignment-owned
2. Show "Konsinyasi" badge on consignment products
3. Add tooltip explaining why adjust is disabled

**Testing:**
- [ ] Adjust consignment product via API → 400 error
- [ ] Adjust store-owned product via API → success
- [ ] Adjust consignment product via UI → button disabled
- [ ] Adjust store-owned product via UI → success

---

### Phase 1: Database Migration

**File:** `database/migrations/042_consignment_receipt_edit.sql`

```sql
-- Consignment receipt edit audit trail (append-only)
CREATE TABLE IF NOT EXISTS consignment_receipt_edits (
    id SERIAL,
    receipt_id INTEGER NOT NULL,
    edited_by INTEGER NOT NULL,
    edited_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    field TEXT NOT NULL,
    old_value TEXT NOT NULL,
    new_value TEXT NOT NULL,
    reason TEXT NOT NULL,
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    CONSTRAINT consignment_receipt_edits_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_receipt_edits_receipt_id_fkey FOREIGN KEY (receipt_id)
        REFERENCES consignment_receipts(id) ON DELETE CASCADE,
    CONSTRAINT consignment_receipt_edits_edited_by_fkey FOREIGN KEY (edited_by)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_receipt
    ON consignment_receipt_edits (receipt_id);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_edited_by
    ON consignment_receipt_edits (edited_by);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_edited_at
    ON consignment_receipt_edits (edited_at);

-- Append-only trigger
CREATE OR REPLACE FUNCTION prevent_consignment_receipt_edit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'consignment_receipt_edits is append-only: modifications are not permitted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_consignment_receipt_edits_immutable
    BEFORE UPDATE OR DELETE ON consignment_receipt_edits
    FOR EACH ROW
    EXECUTE FUNCTION prevent_consignment_receipt_edit_modification();

COMMENT ON TABLE consignment_receipt_edits IS 'Immutable audit trail for consignment receipt edits';
```

### Phase 2: Backend — Repository Layer

**File:** `internal/consignment/repository.go`

**Reuses existing functions:**
- `UpsertConsignmentStock()` — already used in receipt creation
- `GetTermByProduct()` — already used to fetch price/share terms

**Add new methods:**
1. `GetReceiptForEdit(ctx, receiptID) (*Receipt, error)` — Fetch receipt with items for editing
2. `CheckDownstreamActivity(ctx, receiptID) (*DownstreamCheck, error)` — Check if receipt has been sold/returned/settled
3. `UpdateReceiptItem(ctx, item) error` — Update a single receipt item
4. `UpdateConsignmentStock(ctx, productID, delta) error` — Update ownership ledger (wrapper around existing `UpsertConsignmentStock`)
5. `InsertReceiptEdit(ctx, edit) error` — Write to `consignment_receipt_edits`

#### Downstream Check Logic
```sql
-- Check if any quantity from this receipt has been sold
SELECT EXISTS (
    SELECT 1 FROM sale_items si
    JOIN consignment_receipt_items cri ON cri.product_id = si.product_id
    JOIN consignment_receipts cr ON cr.id = cri.consignment_receipt_id
    WHERE cr.id = $1
    AND si.created_at > cr.created_at
) AS has_sales;

-- Check if any quantity is pending return
SELECT EXISTS (
    SELECT 1 FROM consignment_pending_returns cpr
    JOIN consignment_receipt_items cri ON cri.product_id = cpr.product_id
    JOIN consignment_receipts cr ON cr.id = cri.consignment_receipt_id
    WHERE cr.id = $1
) AS has_pending_returns;

-- Check if receipt is settled
SELECT EXISTS (
    SELECT 1 FROM consignment_settlement_items csi
    WHERE csi.consignment_receipt_id = $1
) AS is_settled;
```

### Phase 3: Backend — Service Layer

**File:** `internal/consignment/service.go`

**Reuses existing functions (from receipt creation + inventory module):**
- `ConsignmentAdjuster.ApplyConsignmentDelta()` — stock delta calculations (same as receipt creation)
- `UpsertConsignmentStock()` — ownership ledger updates (same as receipt creation)
- `TouchVisit()` — arrangement timestamp updates (same as receipt creation)
- `auditSvc.CreateAuditLogTx()` — audit trail pattern (from inventory `AdjustStock`)

**Add new method (follows inventory `AdjustStock` pattern):**
```go
func (s *Service) EditReceipt(ctx context.Context, receiptID int, input EditReceiptInput, userID int, ipAddress string) (*Receipt, error)
```

**Logic (follows inventory `AdjustStock` + stock opname `SaveCount` patterns):**
```go
1. GetReceiptForEdit(receiptID)           // Fetch current state
2. isEditableForEdit(receipt)             // Guardrail: 7-day window + no downstream activity
3. BeginTx()                              // Start transaction for atomicity
4. For each edited item:
   a. Calculate delta (new_qty - old_qty)
   b. Validate delta (no negative stock)
   c. UpdateReceiptItem(item)             // Update in DB
   d. UpsertConsignmentStock(delta)       // Update ownership ledger
   e. ApplyConsignmentDelta(delta)        // Update global stock (same tx)
5. InsertReceiptEdit(audit_entry)         // Write audit trail (like stock opname CountRecord)
6. TouchVisit(arrangementID)              // Update arrangement timestamp
7. Commit()                               // All-or-nothing (atomicity guarantee)
8. Return updated receipt
```

**Input validation (follows inventory `AdjustStock` pattern):**
```go
type EditReceiptInput struct {
    Items []EditReceiptItemInput `json:"items" binding:"required"`
    Notes *string                `json:"notes"`
    Reason string                `json:"reason" binding:"required"`  // mandatory, like inventory notes
}

type EditReceiptItemInput struct {
    ID           int    `json:"id" binding:"required"`
    AcceptedQty  int    `json:"accepted_qty" binding:"required,min=1"`
    Price        int    `json:"price" binding:"required,min=0"`
    Notes        *string `json:"notes"`
}
```

### Phase 4: Backend — Handler Layer

**File:** `internal/consignment/handler.go`

Add route:
```go
r.PUT("/consignment/receipts/:id", auth, perm(permissions.ConsignmentUpdate), h.EditReceipt)
```

Handler logic:
1. Parse request body
2. Get user ID from context
3. Get IP address from request
4. Call `svc.EditReceipt()`
5. Return updated receipt or error

### Phase 5: Frontend — Service Layer

**File:** `web/src/modules/consignment/services/consignment-service.ts`

Add function:
```typescript
export async function editReceipt(
  receiptId: number,
  input: {
    items: { id: number; accepted_qty: number; price: number; notes?: string }[];
    notes?: string;
    reason: string;
  }
): Promise<Receipt> {
  const res = await api.put(`/consignment/receipts/${receiptId}`, input);
  return res.data.data;
}
```

### Phase 6: Frontend — Component Layer

**File:** `web/src/modules/consignment/components/ReceiptEntry.svelte`

**Reuses existing patterns:**
- `StockAdjustModal` pattern from `web/src/modules/inventory/components/StockAdjustModal.svelte`
- Same validation: mandatory reason, quantity validation
- Same UI structure: current value display, input fields, submit/cancel buttons

#### State additions (follows `StockAdjustModal` pattern)
```svelte
let editing = $state(false);
let editItems = $state<{ id: number; accepted_qty: number; price: number; notes: string }[]>([]);
let editNotes = $state('');
let editReason = $state('');  // mandatory, like inventory notes
let savingEdit = $state(false);
let canEdit = $state(false);
let editBlockedReason = $state('');
```

#### Functions (follows `StockAdjustModal` pattern)
1. `startEdit()` — Initialize edit state from detailReceipt
2. `cancelEdit()` — Reset edit state
3. `saveEdit()` — Call API, refresh data
4. `checkEditability()` — Check permissions + downstream activity (like `isEditableStatus`)

#### Edit Mode UI (follows `StockAdjustModal` pattern)
- Toggle between view mode and edit mode in detail modal
- Show editable inputs for accepted_qty, price, notes
- Show real-time delta calculation (like "Current Stock" display in `StockAdjustModal`)
- Show mandatory reason textarea (like "Notes" in `StockAdjustModal`)
- Show "Save Changes" and "Cancel" buttons (same as `StockAdjustModal`)

### Phase 7: i18n Labels

**Files:** `web/src/shared/i18n/en.ts`, `web/src/shared/i18n/id.ts`

Add labels:
```typescript
consignmentEditReceipt: 'Edit Receipt' / 'Edit Penerimaan'
consignmentEditReason: 'Reason for edit' / 'Alasan edit'
consignmentEditReasonRequired: 'Reason is required' / 'Alasan wajib diisi'
consignmentEditBlocked: 'Cannot edit receipt' / 'Tidak dapat mengedit penerimaan'
consignmentEditBlockedSales: 'Has downstream sales' / 'Memiliki penjualan'
consignmentEditBlockedSettled: 'Already settled' / 'Sudah diselesaikan'
consignmentEditBlockedExpired: 'Outside 7-day edit window' / 'Di luar jendela edit 7 hari'
consignmentStockDelta: 'Stock delta' / 'Selisih stok'
consignmentEditSaved: 'Receipt updated successfully' / 'Penerimaan berhasil diperbarui'
consignmentEditFailed: 'Failed to update receipt' / 'Gagal memperbarui penerimaan'
```

## Files to Modify

### Phase 0: Inventory Guardrail
| File | Change |
|------|--------|
| `internal/consignment/ports.go` | Add `ConsignmentOwnerChecker` interface |
| `internal/consignment/service.go` | Implement `IsConsignmentOwned()` |
| `internal/inventory/repository.go` | Add `consignmentOwner` field + check in `AdjustStockTx` |
| `internal/inventory/handler.go` | Wire `ConsignmentOwnerChecker` via constructor |
| `cmd/server/main.go` | Wire consignment service as `ConsignmentOwnerChecker` |
| `web/src/modules/inventory/components/StockAdjustModal.svelte` | Disable adjust for consignment products |
| `web/src/shared/i18n/en.ts`, `id.ts` | Add error message labels |

### Phase 1-7: Receipt Edit Feature
| File | Change |
|------|--------|
| `database/migrations/042_consignment_receipt_edit.sql` | New migration: `consignment_receipt_edits` table + append-only trigger |
| `internal/consignment/repository.go` | Add `GetReceiptForEdit`, `CheckDownstreamActivity`, `UpdateReceipt`, `InsertReceiptEdit` |
| `internal/consignment/service.go` | Add `EditReceipt` method with validation + stock delta logic |
| `internal/consignment/handler.go` | Add `PUT /consignment/receipts/:id` route + handler |
| `internal/consignment/domain.go` | Add `EditReceiptInput`, `DownstreamCheck` types |
| `web/src/modules/consignment/services/consignment-service.ts` | Add `editReceipt` function |
| `web/src/modules/consignment/components/ReceiptEntry.svelte` | Add edit mode UI + logic |
| `web/src/shared/i18n/en.ts` | Add edit-related labels |
| `web/src/shared/i18n/id.ts` | Add edit-related labels |

## Design Decisions

1. **Edit in place (not void):** Simpler UX for operators, single correct document
2. **7-day edit window:** Balances flexibility with data integrity
3. **Mandatory reason:** Every edit must have justification for audit trail
4. **Append-only audit table:** Immutable record of all changes, even if receipt is edited multiple times
5. **Downstream activity check:** Prevents edits that would break stock/sales/settlement data
6. **Stock delta calculation:** Automatic recalculation ensures stock levels remain accurate
7. **Permission-based:** Only users with `consignment.update` can edit
8. **Atomicity via pgx.Tx:** Both `product_stock` and `consignment_stock` updated in same transaction (ADR_Cross_Module_Transaction_Strategy)
9. **Exclusive ownership model:** A product is EITHER store-owned OR consignment-owned — never both (BR-03/BR-18)
10. **Inventory guardrail (Phase 0):** Prevent inventory module from adjusting consignment products to maintain ledger consistency

## Implementation Priority

### v1 (Current)
- [ ] **Phase 0: Inventory guardrail** (prerequisite — prevents inconsistency)
- [x] Receipt detail view (read-only)
- [ ] Edit receipt with guardrails
- [ ] Audit trail table
- [ ] Downstream activity check
- [ ] Stock delta calculation

### v2 (Deferred)
- [ ] Approval workflow for large edits
- [ ] Pattern monitoring + alerts
- [ ] Edit history view in UI
- [ ] Bulk edit (multiple receipts)

## Testing Checklist

### Phase 0: Inventory Guardrail
- [ ] Adjust consignment product via API → 400 error
- [ ] Adjust store-owned product via API → success
- [ ] Adjust consignment product via UI → button disabled
- [ ] Adjust store-owned product via UI → success

### Receipt Edit Feature
- [ ] Edit receipt within 7-day window → success
- [ ] Edit receipt after 7 days → blocked
- [ ] Edit receipt with downstream sales → blocked (qty)
- [ ] Edit receipt with downstream sales → blocked (price)
- [ ] Edit receipt with pending returns → blocked
- [ ] Edit receipt after settlement → blocked
- [ ] Edit without `consignment.update` permission → 403
- [ ] Edit without reason → validation error
- [ ] Stock delta calculated correctly (increase)
- [ ] Stock delta calculated correctly (decrease)
- [ ] Audit trail written correctly
- [ ] Receipt number/date/supplier locked
- [ ] Multiple edits on same receipt → all logged
- [ ] Edit notes only → no stock impact
- [ ] Edit qty without sales → allowed
- [ ] Edit price without sales → allowed
- [ ] Edit qty with sales → blocked
- [ ] Edit price with sales → blocked
- [ ] Edit qty that would cause negative stock → blocked
- [ ] Edit qty that reduces but stays positive → allowed (with delta -N)
- [ ] Verify inventory_movements ledger entry written correctly
- [ ] Verify consignment_stock ownership ledger updated correctly
