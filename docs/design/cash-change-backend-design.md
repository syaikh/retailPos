# C1 — Cash Change (Over-Tender) Backend Design

**Feature:** Allow cash customers to tender more than the bill and receive change.
**Status:** ✅ Implemented & Verified (2026-08-23). Depends on and extends `docs/design/payment-checkout-redesign.md` (Phase 2).
**Scope:** `internal/sale` (domain, validation, repository, presenter, handlers) + one DB migration + frontend receipt/summary wiring.

---

## 1. Problem (recap)

`internal/sale/service.go:247` currently enforces **exact payment only**:

```go
if totalPaid != totalAmount {
    return nil, fmt.Errorf("%w: paid %d, expected %d", ErrPaymentTotalMismatch, totalPaid, totalAmount)
}
```

There is **no `change_due` field** on `Sale` or `Payment`, no DB column, and the receipt hardcodes `changeDue: 0`. A customer handing Rp 100.000 for an Rp 85.000 bill cannot be processed — the sale must be keyed to the exact total, leaving **no audit trail of tendered vs. change** and breaking cash-drawer reconciliation.

---

## 2. Goals / Non-Goals

**Goals**
- Accept a CASH tender `>=` the amount owed and compute `change_due = cashTendered − total`.
- Persist `change_due` on the sale and return it in the API response + receipt.
- Keep single-CASH rule (the CASH line carries the tendered amount; change is derived).
- Reject over-tender on **non-cash** methods (no way to return change for QRIS/card).

**Non-Goals (this doc)**
- Refunds / cancellations / partial payments / layaway (separate features).
- Multi-currency.
- Changing the single-CASH-per-sale rule (kept intentionally).

---

## 3. Business Rules

| Scenario | Behavior |
|----------|----------|
| `totalPaid < totalAmount` | Reject `ErrPaymentTotalMismatch` (under-payment) |
| `totalPaid == totalAmount` | Accept, `change_due = 0` |
| `totalPaid > totalAmount` **and** excess is on CASH | Accept, `change_due = totalPaid − totalAmount` |
| `totalPaid > totalAmount` **and** excess is on a non-cash method | Reject `ErrPaymentOverTenderNonCash` |
| CASH appears more than once | Reject `ErrMultipleCashPayments` (unchanged) |

Derivation of "excess on CASH":
```
nonCashTotal = Σ non-cash amounts
cashTotal    = Σ cash amounts (≤ 1 row)
if totalPaid > total:
    // Change can only be returned from physical cash. If the non-cash tender
    // alone already exceeds the bill, the overage sits entirely on a non-cash
    // method and cannot be refunded.
    if nonCashTotal > total → reject
    else change = totalPaid − total
```
This is correct because change can only be given from physical cash.

---

## 4. Data Model Changes

### 4.1 Migration — `database/migrations/033_cash_change.sql`
```sql
-- Must be applied BEFORE the binary that reads/writes change_due (deployment ordering:
-- migrations run prior to deploy, per AGENTS.md).
ALTER TABLE sales ADD COLUMN IF NOT EXISTS change_due integer NOT NULL DEFAULT 0;
```
> Numbering follows the existing `001..032` sequence. If a consolidated `000_squash.sql` is regenerated, add the column there too.

### 4.2 Domain — `internal/sale/domain.go` (`Sale` struct, ~line 39)
```go
type Sale struct {
    ID            int       `json:"id"`
    InvoiceNumber string    `json:"invoice_number"`
    // ... existing fields ...
    TotalAmount   int       `json:"total_amount"`
    ChangeDue     int       `json:"change_due,omitempty"` // NEW
    PaymentMethod string    `json:"payment_method"`
    // ...
}
```
`presentSale` (`presenter.go:22`) returns `*Sale` directly, so `change_due` flows into the API response automatically (including the cost-stripped `saleWithoutCost` embedding).

### 4.3 New error — `internal/sale/domain.go` (errors block, ~line 19)
```go
ErrPaymentOverTenderNonCash = errors.New("overpayment is only allowed on cash tender")
```

---

## 5. Validation Changes — `internal/sale/service.go` `validatePayments` (~line 186)

Change the signature to also return `change int`, track cash vs. non-cash totals, and relax the final check.

```go
func (s *service) validatePayments(ctx context.Context, totalAmount int, payments []CreatePaymentRequest) ([]Payment, int, error) {
    if len(payments) == 0 {
        return nil, 0, ErrZeroPaymentAmount
    }
    if len(payments) > MaxPaymentsPerSale {
        return nil, 0, ErrMaxPaymentsExceeded
    }
    allMethods, err := s.repo.GetAllPaymentMethods(ctx)
    if err != nil {
        return nil, 0, err
    }
    methodsByCode := make(map[string]*PaymentMethod, len(allMethods))
    for i := range allMethods {
        methodsByCode[strings.ToUpper(allMethods[i].Code)] = &allMethods[i]
    }

    var totalPaid, cashTotal, nonCashTotal int
    result := make([]Payment, 0, len(payments))
    seenMethods := make(map[string]bool)
    cashCount := 0

    for _, p := range payments {
        if p.Amount <= 0 {
            return nil, 0, ErrZeroPaymentAmount
        }
        pm, ok := methodsByCode[strings.ToUpper(p.PaymentMethodCode)]
        if !ok {
            return nil, 0, ErrInvalidPaymentMethod
        }
        if !pm.IsActive {
            return nil, 0, ErrPaymentMethodInactive
        }
        methodUpper := strings.ToUpper(p.PaymentMethodCode)
        if strings.EqualFold(p.PaymentMethodCode, "CASH") {
            cashCount++
            if cashCount > 1 {
                return nil, 0, ErrMultipleCashPayments
            }
            cashTotal += p.Amount
        } else {
            if seenMethods[methodUpper] {
                return nil, 0, ErrDuplicatePaymentMethod
            }
            nonCashTotal += p.Amount
        }
        seenMethods[methodUpper] = true
        if pm.RequiresReference && strings.TrimSpace(p.ReferenceNumber) == "" {
            return nil, 0, ErrPaymentReferenceRequired
        }
        totalPaid += p.Amount
        result = append(result, Payment{
            PaymentMethodID:   pm.ID,
            PaymentMethodCode: pm.Code,
            Amount:            p.Amount,
            ReferenceNumber:   p.ReferenceNumber,
        })
    }

    if totalPaid < totalAmount {
        return nil, 0, fmt.Errorf("%w: paid %d, expected %d", ErrPaymentTotalMismatch, totalPaid, totalAmount)
    }

    change := 0
    if totalPaid > totalAmount {
        if nonCashTotal > totalAmount {
            return nil, 0, ErrPaymentOverTenderNonCash
        }
        change = totalPaid - totalAmount
    }

    return result, change, nil
}
```

### 5.1 Update call sites
- `cart_service.go:519` — `checkoutCart`:
  ```go
  validatedPayments, change, err := s.validatePayments(ctx, sale.TotalAmount, payments)
  if err != nil { return nil, err }
  sale.ChangeDue = change
  ```
- `service.go:265` — `CreateSale` (direct sale creation):
  ```go
  validatedPayments, change, err := s.validatePayments(ctx, sale.TotalAmount, payments)
  if err != nil { return err }
  sale.ChangeDue = change
  ```

### 5.2 Error mapping (3 handlers → 400)
Add `ErrPaymentOverTenderNonCash` alongside the existing payment errors in:
- `internal/sale/cart_handler.go:86` (`cartError`)
- `internal/sale/handler.go:380` and `:1254` (direct sale paths)

---

## 6. Persistence Changes — `internal/sale/repository.go`

### 6.1 `CreateSale` INSERT (~line 120)
```go
INSERT INTO sales (invoice_number, cashier_id, store_id, customer_id, shift_id, subtotal, discount, tax, total_amount, payment_method, status, hold_note, change_due)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id, created_at, updated_at
```
pass `sale.ChangeDue` as the 13th arg.

### 6.2 `GetSaleByID` SELECT/SCAN (~lines 196–237)
Add `s.change_due` to the SELECT column list and `&sale.ChangeDue` to the `Scan` call (alongside the other sale fields). No change to the `payments` JSON aggregation is needed — change is a sale-level attribute.

> `sale_payments` is intentionally **not** changed: change is derived from the CASH line at sale level, not stored per payment.

---

## 7. Frontend Integration (cross-ref `payment-checkout-redesign.md` Phase 2)

- **`CheckoutModal.svelte`** — `canComplete` must allow `remainingBalance <= 0` when the excess is on a CASH row; show a **Change** preview (the "OVER-TENDER" summary variant) instead of the over-by error.
- **`PosPage.svelte:452`** & **`TransactionDrawer.svelte:103`** — replace hardcoded `changeDue: 0` with the sale's `change_due`.
- **`ReceiptPrintOverlay.svelte`** — render a `CHANGE` line when `changeDue > 0` (the type already supports `change_due`).

---

## 8. Shift Reconciliation Consideration (follow-up)

`checkoutCart` calls `shiftStore.UpdateShiftTotals(... shiftContribution(shiftID, cashierID, total, payments))`. Today the cash portion is treated as cash received. Once change exists, **physical cash in the drawer = cashTendered − changeDue**. ✅ **Resolved (2026-08-23):** `shiftContribution` now takes `changeDue` and subtracts it from `CashSales` (cash-only, since change is always returned from physical cash). Previously the full cash tendered was credited, overstating expected cash and producing false shortfalls at shift close for every over-tender sale (e.g., Rp 8.000 tendered / Rp 5.000 bill reported a false Rp 3.000 shortfall). Covered by `TestShiftContributionChangeDue`.

---

## 9. Edge Cases

- **Exact split (cash + card, sums to total):** `change = 0`, accepted (unchanged behavior).
- **Cash over, plus card:** `nonCashTotal ≤ total` → `change = totalPaid − total`, accepted.
- **Card over only:** `nonCashTotal > total` → rejected `ErrPaymentOverTenderNonCash`.
- **Multiple non-cash, exact:** allowed (duplicate rule already enforced per method).
- **Zero/negative amounts:** still rejected by `ErrZeroPaymentAmount` (unchanged).

---

## 10. Testing

- **Unit** (`internal/sale`): over-tender on CASH returns `change` and persists; non-cash over-tender returns `ErrPaymentOverTenderNonCash`; exact unchanged; under-payment still `ErrPaymentTotalMismatch`.
- **E2E** (`tests/e2e/payment-validation.spec.ts`): add CS-D11 (cash over-tender → 201, `change_due` present), CS-D12 (non-cash over-tender → 400), and a receipt assertion that `changeDue > 0` renders.
- **Backend cmd/dummy / test DB:** migration `033` auto-applied by the test framework's `schema_migrations` tracker.

---

## 11. Rollout / Deployment Ordering

Per `AGENTS.md`, apply migration **before** deploying the binary that reads/writes `change_due`. Add `033_cash_change.sql` to the migration set; the server will fail to insert `change_due` until the column exists, so ordering is enforced by the existing migration-before-deploy contract.

---

## 12. Implementation Status (2026-08-23) — ✅ DONE & VERIFIED

**Implemented (matches §4–§7):**
- Migration `033_cash_change.sql` — `ALTER TABLE sales ADD COLUMN change_due integer NOT NULL DEFAULT 0` (applied to `retail_pos`).
- `Sale.ChangeDue int` (`json:"change_due,omitempty"`) + `ErrPaymentOverTenderNonCash = errors.New("overpayment is only allowed on cash tender")` in `internal/sale/domain.go`.
- `validatePayments` returns `([]Payment, int, error)`; tracks `cashTotal`/`nonCashTotal`; corrected over-tender rule: reject `ErrPaymentOverTenderNonCash` when `nonCashTotal > totalAmount` (change can only come from physical cash). See §3.
- Call sites set `sale.ChangeDue = change`: `cart_service.go:519` (`checkoutCart`), `service.go` `CreateSale`, `service.go` `CreateSaleWithParkedSale`.
- `repository.go` `CreateSale` INSERT (`$13` arg) and `GetSaleByID` SELECT/SCAN persist & read `sale.change_due`.
- Error mapping added in `cart_handler.go` (`cartError`) and `handler.go` (×2) → HTTP 400.
- Frontend: `CheckoutModal.svelte` (over-tender allowance `remainingBalance < 0 && nonCashTotal <= totalAmount` + Change preview), `PosPage.svelte:452` & `TransactionDrawer.svelte:103` (`change_due ?? 0`), `ReceiptPrintOverlay.svelte` (CHANGE line when `changeDue > 0`).

**Verification (all green):**
- Go unit `internal/sale/...` — pass, incl. `TestSaleService_CashChangeOverTender` (5 cases) + updated `TestSaleService_CreateSaleTotalAmountClamp`.
- Frontend `vitest run src/modules/pos src/modules/sales` — 255 pass.
- E2E API `tests/e2e/payment-validation.spec.ts` — **13/13**, incl. CS-D1b (cash over-tender 201), CS-D11 (cash over-tender → `change_due` persisted via GET), CS-D12 (non-cash over-tender → 400, error contains 'cash').
- E2E UI `tests/e2e/print-receipt.spec.ts` — **4/4**, incl. new **"over-tender on cash renders a CHANGE line on the thermal receipt (C1)"**.

**Testing note (environment):** The dev frontend renders in `id` (Indonesian) locale by default; the `localStorage['pos.locale']='en'` test init-script does **not** force English post-login. `print-receipt.spec.ts` selectors were therefore made locale-neutral (`getByRole('dialog')`, `button:has-text("Enter")`, product-row double-click to add, `/Change|Uang Kembali/`, `/Print|Cetak/`, `/Invoice|Faktur/`, `/Time|Waktu/`, shop-name visible+non-empty).

**Follow-up (non-blocking, §8):** ✅ Resolved — `shiftContribution(shiftID, cashierID, totalAmount, changeDue, payments)` subtracts `change_due` from `CashSales`; covered by `TestShiftContributionChangeDue` (4 cases: exact, cash over-tender, mixed-with-change, mixed exact).

**Open (minor, not fixed):** Checkout UX — `CheckoutModal` prefills CASH = total and `addAllocation` blocks a second method when `remainingBalance <= 0`, so a CASH+QRIS split requires first reducing the prefilled cash (clicking QRIS silently no-ops). Consider allowing `addAllocation` to still add a non-cash method. Behavioral only; no correctness impact.
