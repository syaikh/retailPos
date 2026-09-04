# Implementation Plan: Sales Return (Retur Penjualan)

| Field | Value |
|-------|-------|
| Status | **Planned** |
| Date | 2026-09-04 |
| PRD | `docs/prd/PRD-Sales-Return.md` |
| Estimated Effort | 10-12 days |

---

## Overview

This document provides a step-by-step implementation plan for the Sales Return feature, broken into 4 phases with specific files, functions, and migration steps.

---

## Phase 1: Database & Backend Core (5-6 days)

### Step 1.1: Database Migration (Day 1)

**File:** `database/migrations/040_sales_return.sql`

Create migration with:
1. `sale_returns` table (with `return_type` column: 'replacement' or 'refund')
2. `sale_return_items` table (with `outcome` column: 'replacement' or 'refund')
3. `sale_return_replacements` table (tracks replacement items)
4. `sale_return_payments` table (only for refund outcomes)
5. `return_approvals` table
6. `return_seq` sequence
7. Indexes on all FK columns + `outcome` column
8. Permission inserts (`sale.return`, `sale.return.approve`)
9. Role permission grants
10. `app_settings` inserts for `return.window_days`, `return.approval_threshold`, `return.approval_timeout_minutes`

**Verify:** Run migration against dev DB, confirm tables created.

---

### Step 1.2: Domain Layer (Day 1)

**File:** `internal/salereturn/domain.go`

Define:

```go
// Entities
type SaleReturn struct { ... }
type SaleReturnItem struct { ... }
type SaleReturnReplacement struct { ... }
type SaleReturnPayment struct { ... }
type ReturnApproval struct { ... }

// Request DTOs
type ProcessReturnRequest struct { ... }
type ApproveReturnRequest struct { ... }
type RejectReturnRequest struct { ... }

// Domain errors
var (
    ErrReturnWindowExpired      = errors.New("return window has expired")
    ErrReturnQtyExceedsOriginal = errors.New("return quantity exceeds original purchase")
    ErrSaleNotCompleted         = errors.New("only completed sales can be returned")
    ErrSaleAlreadyFullyReturned = errors.New("sale has already been fully returned")
    ErrItemNotEligible          = errors.New("item is not eligible for return")
    ErrApprovalRequired         = errors.New("manager approval required for this return")
    ErrApprovalExpired          = errors.New("approval request has expired")
    ErrShiftNotOpen             = errors.New("shift is not open for return processing")
    ErrInsufficientReturnQty    = errors.New("insufficient returnable quantity remaining")
    ErrInsufficientStock        = errors.New("insufficient stock for replacement")
    ErrReplacementNotAvailable  = errors.New("replacement not available")
)

// Constants
const (
    ReturnReasonDamaged   = "damaged"
    ReturnReasonDefective = "defective"
    ReturnReasonWrongItem = "wrong_item"
    ReturnReasonOther     = "other"

    ConditionResellable = "resellable"
    ConditionDamaged    = "damaged"
    ConditionDefective  = "defective"

    ReturnStatusCompleted  = "completed"
    ReturnStatusPending    = "pending_approval"
    ReturnStatusCancelled  = "cancelled"

    ApprovalStatusPending  = "pending"
    ApprovalStatusApproved = "approved"
    ApprovalStatusRejected = "rejected"
    ApprovalStatusExpired  = "expired"

    RefundMethodOriginal = "original"

    // Return outcome constants
    OutcomeReplacement = "replacement"
    OutcomeRefund      = "refund"
    
    // Return type constants
    ReturnTypeReplacement = "replacement"
    ReturnTypeRefund      = "refund"
)
```

---

### Step 1.3: Ports (Day 1)

**File:** `internal/salereturn/ports.go`

```go
type StockRestorer interface {
    RestoreStock(ctx context.Context, tx pgx.Tx, items []shared.StockRestoreItem) error
}

type StockDeducer interface {
    DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error
}

type StockChecker interface {
    GetAvailableStock(ctx context.Context, productID int) (int, error)
}

type SaleLookup interface {
    GetSaleByID(ctx context.Context, id int) (*sale.Sale, error)
}

type ConsignmentReturner interface {
    RestoreConsignmentStock(ctx context.Context, tx pgx.Tx, items []shared.ConsignmentReturnItem) error
}
```

**File:** `internal/shared/stock.go` — Add `StockRestoreItem`:

```go
type StockRestoreItem struct {
    ProductID int
    Quantity  int
}
```

---

### Step 1.4: Repository Layer (Days 1-2)

**File:** `internal/salereturn/repository.go`

Methods:

| Method | SQL |
|--------|-----|
| `CreateReturn(ctx, tx, return)` | INSERT sale_returns |
| `CreateReturnItems(ctx, tx, items)` | Batch INSERT sale_return_items |
| `CreateReturnReplacements(ctx, tx, replacements)` | Batch INSERT sale_return_replacements |
| `CreateReturnPayments(ctx, tx, payments)` | Batch INSERT sale_return_payments |
| `GetReturnByID(ctx, id)` | SELECT with JOIN to sale_returns + items + replacements + payments |
| `GetReturnsBySaleID(ctx, saleID)` | SELECT list |
| `GetReturnedQtyBySaleItemID(ctx, saleItemID)` | SUM(quantity) from sale_return_items |
| `CreateApproval(ctx, tx, approval)` | INSERT return_approvals |
| `GetApprovalByID(ctx, id)` | SELECT |
| `UpdateApprovalStatus(ctx, tx, id, status, approvedBy)` | UPDATE |
| `GetPendingApprovals(ctx, storeID)` | SELECT WHERE status='pending' AND expires_at > NOW() |
| `ExpireStaleApprovals(ctx)` | UPDATE WHERE status='pending' AND expires_at < NOW() |

---

### Step 1.5: Service Layer (Days 2-3)

**File:** `internal/salereturn/service.go`

Core method: `ProcessReturn(ctx, req, caller) (*SaleReturn, error)`

```
1. Fetch original sale via SaleLookup port
2. Validate: sale status == 'completed'
3. Validate: return window (now - sale.created_at ≤ return.window_days)
4. Validate: sale belongs to caller's store (store scoping)

5. For EACH return item in request:
   a. Fetch original sale_item by sale_item_id
   b. Query: SUM(quantity) from sale_return_items WHERE sale_item_id = X
      → already_returned = result
   c. Validate: requested_qty ≤ (original_qty - already_returned)
      → If exceeded: return error "Hanya {remaining} unit yang tersisa untuk diretur"
   d. Validate: item condition == 'damaged' (Phase 1 restriction)
   e. Check product status:
      - IF product is archived/discontinued → skip stock restoration for this item
      - IF product is active → proceed with stock check
   f. Check stock availability for replacement decision (with FOR UPDATE lock):
      - Lock product_stock row via SELECT FOR UPDATE
      - available = product_stock.quantity
      - IF available >= requested_qty → outcome = 'replacement'
      - ELSE → outcome = 'refund' (last stock)
   g. Calculate per-line amounts:
      - For replacement: refund_amount = 0 (no money changes hands)
      - For refund: 
        - subtotal = requested_qty × unit_price (from original sale_item snapshot)
        - tax = requested_qty × (original_tax_amount / original_quantity)
        - refund_amount = subtotal + tax

6. Determine overall return type:
   - IF all items have outcome='replacement' → return_type = 'replacement'
   - IF any item has outcome='refund' → return_type = 'refund'

7. For refund returns only: Calculate split payment refund (proportional):
   For each original payment:
     - ratio = payment.amount / original_sale.total_amount
     - refund_portion = ratio × total_refund_amount
     → Creates sale_return_payments records

8. If return requires manager approval (refund amount > threshold):
   a. Create pending approval record
   b. Publish ApprovalRequested event (WebSocket)
   c. Return pending status
9. If auto-approve OR already approved:
   a. Begin TX
   b. IF replacement items exist:
      - RestoreStock (returned items) — via StockRestorer port (skip if product archived)
      - DeductStock (replacement items) — via StockDeducer port
      - RestoreConsignmentStock (if applicable)
   c. IF refund items exist:
      - RestoreStock (returned items) — via StockRestorer port (skip if product archived)
      - RestoreConsignmentStock (if applicable)
   d. INSERT sale_returns (return_type, total_refund_amount)
   e. INSERT sale_return_items (per-line with outcome)
   f. IF replacement: INSERT sale_return_replacements
   g. IF refund: INSERT sale_return_payments (proportional split)
   h. IF refund AND shift is open: UpdateShiftTotals (negative contribution)
   i. IF refund AND shift is closed: Log adjustment in audit trail (skip shift update)
   j. IF registered customer: Update customer total_spent and loyalty_points
   k. Audit log
   l. Publish SaleReturned event
   m. COMMIT TX
```

#### Return Outcome Decision — Detail

```
For each item:
  available_stock = StockChecker.GetAvailableStock(product_id)
  
  IF available_stock >= requested_qty:
    outcome = 'replacement'
    → Customer gets new item, no money changes hands
  ELSE:
    outcome = 'refund'
    → Customer gets money back, no replacement

Overall return type:
  IF ALL items are replacement → return_type = 'replacement'
  IF ANY item is refund → return_type = 'refund' (partial refund possible)
```

#### Partial Return — Validation Detail

```
Input: request.items = [{sale_item_id: 101, qty: 2}]

Step 4a: SELECT * FROM sale_items WHERE id = 101
  → result: {quantity: 5, unit_price: 3500, tax_amount: 1750}

Step 4b: SELECT COALESCE(SUM(quantity), 0) FROM sale_return_items WHERE sale_item_id = 101
  → result: 0 (first return)

Step 4c: 2 ≤ (5 - 0) = 5 ✓

Step 4e: available_stock = 12 → outcome = 'replacement'

Step 4f: refund_amount = 0 (replacement)
```

#### Split Payment Refund — Calculation Detail (Refund Only)

```
Original sale total: Rp51,150
Original payments:
  Cash: Rp30,000 (ratio = 30000/51150 = 0.5865)
  QRIS: Rp21,150 (ratio = 21150/51150 = 0.4135)

Total refund: Rp7,700

Cash refund:  0.5865 × 7700 = Rp4,516
QRIS refund:  0.4135 × 7700 = Rp3,184
Sum check:    4516 + 3184 = Rp7,700 ✓
```

Additional methods:

| Method | Description |
|--------|-------------|
| `ApproveReturn(ctx, approvalID, managerID)` | Validate approval → call `ProcessReturn` with approved flag |
| `RejectReturn(ctx, approvalID, managerID, reason)` | Update approval status → audit log |
| `GetReturnByID(ctx, returnID)` | Fetch return detail |
| `GetReturnsBySaleID(ctx, saleID)` | List returns for a sale |
| `GetPendingApprovals(ctx, caller)` | List pending approvals for manager's store |
| `CheckReturnEligibility(ctx, saleID)` | Pre-check: can this sale be returned? |
| `GetReturnableQty(ctx, saleItemID)` | Returns remaining returnable qty for a sale item |
| `CheckReplacementAvailability(ctx, productID, qty)` | Returns whether replacement is possible |

---

### Step 1.6: Handler Layer (Day 3)

**File:** `internal/salereturn/handler.go`

| Route | Method | Permission | Handler |
|-------|--------|------------|---------|
| `/api/sales/:id/return` | POST | `sale.return` | `ProcessReturn` |
| `/api/sales/:id/returns` | GET | `sale.view` | `ListReturns` |
| `/api/sales/returns/:returnId` | GET | `sale.view` | `GetReturn` |
| `/api/sales/returns/:returnId/approve` | POST | `sale.return.approve` | `ApproveReturn` |
| `/api/sales/returns/:returnId/reject` | POST | `sale.return.approve` | `RejectReturn` |
| `/api/sales/returns/pending` | GET | `sale.return.approve` | `ListPendingApprovals` |

---

### Step 1.7: Inventory StockRestorer (Day 3)

**File:** `internal/inventory/stock_restorer.go`

```go
type StockRestorer struct{}

func (StockRestorer) RestoreStock(ctx context.Context, tx pgx.Tx, items []shared.StockRestoreItem) error {
    for _, item := range items {
        // 1. SELECT FOR UPDATE on product_stock (lock row)
        // 2. UPDATE product_stock SET quantity = quantity + $1 WHERE product_id = $2
        // 3. INSERT inventory_movements (type='sale_return', quantity=+$1)
    }
    return nil
}
```

**Note:** For replacement returns, the service also calls the existing `StockDeducer.DeductStock()` method to deduct the replacement item's stock. No new code needed for that — reuse existing inventory module.

**Verify:** Unit tests pass for stock restoration.

---

### Step 1.8: Composition Root Wiring (Day 4)

**File:** `cmd/server/main.go`

1. Import `internal/salereturn`
2. Create `StockRestorer` instance
3. Wire into salereturn service: `salereturnService.SetStockRestorer(restorer)`
4. Register routes under `/api/sales/returns`
5. Add permissions to `permissions.All()` in `internal/permissions/permissions.go`

**File:** `internal/permissions/permissions.go`

Add constants:
```go
SaleReturn       Code = "sale.return"
SaleReturnApprove Code = "sale.return.approve"
```

Update `All()` function to include new codes.

---

### Step 1.9: Tests (Days 4-6)

**Backend tests:**

| File | Tests |
|------|-------|
| `internal/salereturn/service_test.go` | ProcessReturn success (replacement), ProcessReturn success (refund), window expired, over-return, shift closed (skip adjustment), approval required, approve success, reject success, approval expired, stock unavailable → refund, **discontinued product (skip stock restore)**, **consignment-owned item (dual ledger restore)**, **registered customer (total_spent update)**, **cross-store rejection** |
| `internal/salereturn/handler_test.go` | All endpoints: success, permission denied, validation errors |
| `internal/salereturn/repository_test.go` | CRUD operations, returned qty calculation, replacement CRUD |
| `internal/inventory/stock_restorer_test.go` | RestoreStock success, concurrent access |

**Verify:** `go test ./internal/salereturn/... ./internal/inventory/...` passes.

---

## Phase 2: Frontend (3-4 days)

### Step 2.1: TypeScript Types (Day 7)

**File:** `web/src/modules/salereturn/types/index.ts`

```typescript
export interface SaleReturn { ... }
export interface SaleReturnItem { ... }
export interface SaleReturnReplacement { ... }
export interface SaleReturnPayment { ... }
export interface ReturnApproval { ... }
export interface ProcessReturnRequest { ... }
```

---

### Step 2.2: API Service (Day 7)

**File:** `web/src/modules/salereturn/services/salereturn-service.ts`

Functions:
- `processReturn(saleId, request)` — POST
- `getReturnsBySaleId(saleId)` — GET
- `getReturnById(returnId)` — GET
- `approveReturn(returnId)` — POST
- `rejectReturn(returnId, reason)` — POST
- `getPendingApprovals()` — GET

---

### Step 2.3: Return Modal Components (Days 7-8)

**File:** `web/src/modules/salereturn/components/ReturnModal.svelte`

- Item selection with checkbox and quantity input
- Return reason dropdown
- Condition dropdown (locked to "damaged" in Phase 1)
- **Stock availability indicator** per item (shows replacement vs refund)
- Notes textarea
- Financial summary (shows replacement items + refund items separately)
- **Replacement summary**: "Penggantian: X item (Rp0, tanpa uang)"
- **Refund summary**: "Pengembalian: X item (RpX,XXX)"
- Refund method display (only for refund items)
- Submit button (disabled if validation fails)
- Loading state during API call

**File:** `web/src/modules/salereturn/components/ReturnItemRow.svelte`

- Product name, original qty, return qty input (max = original - already returned)
- Unit price display
- **Stock indicator**: "Stok: 12" (replacement) or "Stok: 0" (refund)
- Subtotal calculation (only for refund items)

**File:** `web/src/modules/salereturn/components/ReturnSummary.svelte`

- **Replacement count**: X items
- **Refund count**: Y items
- Subtotal returned (refund items only)
- Tax reversed (refund items only)
- Total refund amount (refund items only)
- Refund method (refund items only)

---

### Step 2.4: Integration with Find Transaction (Day 8)

**File:** `web/src/modules/pos/components/FindTransaction.svelte` (or `TransactionDrawer.svelte`)

- Add "Return Items" button to sale detail drawer
- Button visible only if user has `sale.return` permission
- Button disabled if: sale not completed, window expired, fully returned
- Click opens `ReturnModal`

---

### Step 2.5: Manager Approval Components (Days 8-9)

**File:** `web/src/modules/salereturn/components/ApprovalNotification.svelte`

- WebSocket listener for `return_approval_requested` events
- Toast notification: "Permintaan retur dari Budi — Rp750.000 — INV-2026-000123"
- Click opens ApprovalModal

**File:** `web/src/modules/salereturn/components/ApprovalModal.svelte`

- Display: cashier name, invoice number, items, reason, condition, total
- **Show replacement items** (Rp0, no money)
- **Show refund items** (with refund amount)
- Approve button (green)
- Reject button (red) with reason input
- Loading state during API call

---

### Step 2.6: i18n Strings (Day 9)

**File:** `web/src/shared/i18n/en.ts` and `web/src/shared/i18n/id.ts`

Add keys:
- `returnItems`, `returnReason`, `returnCondition`, `returnSummary`
- `returnDamaged`, `returnDefective`, `returnWrongItem`, `returnOther`
- `returnProcess`, `returnApprove`, `returnReject`
- `returnWindowExpired`, `returnQtyExceeds`, `returnApprovalRequired`
- `returnNotificationTitle`, `returnNotificationBody`

---

### Step 2.7: Frontend Tests (Day 9-10)

| File | Tests |
|------|-------|
| `ReturnModal.svelte.test.ts` | Item selection, qty validation, reason selection, submit |
| `ApprovalModal.svelte.test.ts` | Approve, reject, loading states |
| `salereturn-service.test.ts` | API call mocking |

**Verify:** `cd web && npx vitest run` passes.

---

## Phase 3: WebSocket & Real-time (1-2 days)

### Step 3.1: Approval Event Publishing (Day 10)

**File:** `internal/salereturn/service.go`

After creating pending approval:
```go
s.eventBus.Publish("return_approval_requested", ApprovalRequestedPayload{
    ApprovalID:    approval.ID,
    ReturnID:      saleReturn.ID,
    StoreID:       caller.StoreID,
    CashierName:   caller.Username,
    InvoiceNumber: sale.InvoiceNumber,
    TotalAmount:   totalAmount,
    ExpiresAt:     approval.ExpiresAt,
})
```

---

### Step 3.2: Approval Response Handling (Day 10)

**File:** `internal/salereturn/handler.go`

After approval/rejection:
```go
s.eventBus.Publish("return_approval_resolved", ApprovalResolvedPayload{
    ApprovalID: approvalID,
    Status:     "approved", // or "rejected"
    StoreID:    storeID,
})
```

---

### Step 3.3: Frontend WebSocket Listener (Day 10)

**File:** `web/src/modules/salereturn/stores/salereturn-store.svelte.ts`

- Subscribe to `return_approval_requested` → show notification
- Subscribe to `return_approval_resolved` → update UI

---

### Step 3.4: Approval Expiry Timer (Day 11)

**File:** `internal/salereturn/service.go`

Background goroutine or cron:
```go
// Run every minute
func (s *service) ExpireStaleApprovals(ctx context.Context) {
    // UPDATE return_approvals SET status='expired'
    // WHERE status='pending' AND expires_at < NOW()
    // For each expired: audit log + publish event
}
```

---

## Phase 4: Polish & Edge Cases (1-2 days)

### Step 4.1: Shift Integration (Day 11)

- Verify shift total reversal works correctly
- Test: return during open shift → totals decrease
- Test: return after shift closed → rejected

---

### Step 4.2: Consignment Integration (Day 11)

- Test: return consignment-owned item → both `product_stock` and `consignment_stock` restored
- Verify settlement unaffected (returned items excluded)

---

### Step 4.3: Reporting Adjustments (Day 12)

- Verify `SaleReturned` event consumed by reporting module
- Dashboard shows return count, replacement count, and refund total
- Sales reports deduct refund returns from gross sales (replacement returns have no financial impact)
- Shift summary: only refund amounts deducted from shift totals

---

### Step 4.4: Receipt Reprint (Day 12)

- Generate return receipt with "RETUR" header
- Link to original invoice number
- **For replacement**: Show "Penggantian" with replacement item details (no refund amount)
- **For refund**: Show "Pengembalian" with refund amount and method
- Include: returned items, reason, condition

---

### Step 4.5: E2E Tests (Day 12)

**File:** `tests/e2e/sales-return.spec.ts`

| Test | Description |
|------|-------------|
| `TestE2E_SalesReturn_Replacement` | Cashier processes replacement (stock available) |
| `TestE2E_SalesReturn_Refund` | Cashier processes refund (last stock) |
| `TestE2E_SalesReturn_WindowExpired` | Return rejected after window |
| `TestE2E_SalesReturn_ManagerApproval` | High-value return → manager approves |
| `TestE2E_SalesReturn_ApprovalExpired` | Approval times out |

---

## File Checklist

### New Files

| File | Description |
|------|-------------|
| `database/migrations/040_sales_return.sql` | Database migration |
| `internal/salereturn/domain.go` | Domain entities + errors |
| `internal/salereturn/repository.go` | Database operations |
| `internal/salereturn/service.go` | Business logic |
| `internal/salereturn/handler.go` | HTTP handlers |
| `internal/salereturn/ports.go` | Port interfaces |
| `internal/salereturn/presenter.go` | JSON formatting |
| `internal/salereturn/service_test.go` | Service tests |
| `internal/salereturn/handler_test.go` | Handler tests |
| `internal/salereturn/repository_test.go` | Repository tests |
| `internal/inventory/stock_restorer.go` | Stock restoration implementation |
| `internal/inventory/stock_restorer_test.go` | Stock restorer tests |
| `web/src/modules/salereturn/types/index.ts` | TypeScript types |
| `web/src/modules/salereturn/services/salereturn-service.ts` | API service |
| `web/src/modules/salereturn/components/ReturnModal.svelte` | Return modal |
| `web/src/modules/salereturn/components/ReturnItemRow.svelte` | Item row component |
| `web/src/modules/salereturn/components/ReturnSummary.svelte` | Summary component |
| `web/src/modules/salereturn/components/ApprovalNotification.svelte` | Approval toast |
| `web/src/modules/salereturn/components/ApprovalModal.svelte` | Approval modal |
| `web/src/modules/salereturn/stores/salereturn-store.svelte.ts` | State management |
| `web/src/modules/salereturn/components/__tests__/ReturnModal.svelte.test.ts` | Modal tests |
| `web/src/modules/salereturn/components/__tests__/ApprovalModal.svelte.test.ts` | Approval tests |
| `tests/e2e/sales-return.spec.ts` | E2E tests |

### Modified Files

| File | Change |
|------|--------|
| `internal/permissions/permissions.go` | Add `SaleReturn`, `SaleReturnApprove` codes |
| `internal/shared/stock.go` | Add `StockRestoreItem` struct |
| `cmd/server/main.go` | Wire salereturn routes + ports |
| `docs/roadmap/upcoming-features.md` | Update return policy status |
| `web/src/shared/i18n/en.ts` | Add return-related strings |
| `web/src/shared/i18n/id.ts` | Add return-related strings |
| `web/src/modules/pos/components/FindTransaction.svelte` | Add "Return Items" button |

---

## Dependency Graph

```
Step 1.1 (Migration)
    │
    ├──► Step 1.2 (Domain)
    │       │
    │       ├──► Step 1.3 (Ports)
    │       │
    │       └──► Step 1.4 (Repository)
    │               │
    │               └──► Step 1.5 (Service)
    │                       │
    │                       ├──► Step 1.6 (Handler)
    │                       │
    │                       └──► Step 1.7 (StockRestorer)
    │                               │
    │                               └──► Step 1.8 (Wiring)
    │                                       │
    │                                       └──► Step 1.9 (Backend Tests)
    │
    └──► Step 2.1-2.7 (Frontend, parallel with 1.4-1.9)
            │
            └──► Step 3.1-3.4 (WebSocket)
                    │
                    └──► Step 4.1-4.5 (Polish + E2E)
```

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Stock race condition (concurrent return + sale) | `SELECT FOR UPDATE` on product_stock row before stock operations |
| Approval timeout edge case | Background expiry job runs every minute |
| Consignment stock double-restore | Validate consignment ownership before restore; check settlement status |
| Shift closed during return | Skip shift adjustment; log adjustment in audit trail |
| Manager not available for approval | 5-minute expiry + audit trail |
| **Replacement stock unavailable** | Check stock before processing; fallback to refund |
| **Partial return with mixed outcomes** | Track each item's outcome separately; overall return type = 'refund' if any refund |
| **Discontinued product return** | Allow return; skip stock restoration if product archived/discontinued |
| **Cross-store return attempt** | Reject with store scoping validation at service layer |
| **Registered customer total_spent drift** | Decrement total_spent and loyalty_points atomically in same TX |
| **Consignment settlement already paid** | Reject return if consignment_sale_items has linked settlement |
| **Pricing rule expired/deleted** | Use snapshotted unit_price from sale_items (immutable) |
| **Tax class deleted** | Use snapshotted tax_rate from sale_items (immutable) |

---

## Success Criteria

| Criterion | How Verified |
|-----------|--------------|
| **Replacement when stock available** | Unit + E2E test |
| **Refund when last stock** | Unit + E2E test |
| Return within window succeeds | Unit + E2E test |
| Return outside window rejected | Unit test |
| Stock restored correctly (replacement) | Unit test + manual verification |
| Stock deducted correctly (replacement) | Unit test + manual verification |
| Stock restored correctly (refund) | Unit test + manual verification |
| Shift totals reversed (refund only, open shift) | Unit test |
| Shift adjustment skipped (closed shift) | Unit test |
| Manager approval flow works | E2E test |
| Approval timeout works | E2E test |
| Audit log created for every action | Unit test |
| Permission gating works | Handler test |
| Store scoping enforced | Handler test |
| **Discontinued product return (skip stock restore)** | Unit test |
| **Consignment-owned item return (dual ledger)** | Unit test |
| **Registered customer total_spent update** | Unit test |
| **Cross-store return rejected** | Handler test |
