package salereturn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
)

// Policy keys in app_settings (defaults seeded by migration 067, PRD §5).
const (
	settingWindowDays        = "return.window_days"
	settingApprovalThreshold = "return.approval_threshold"
	settingApprovalTimeout   = "return.approval_timeout_minutes"

	defaultWindowDays        = 1
	defaultApprovalThreshold = 500000
	defaultApprovalTimeout   = 5
)

// Service implements the Sales Return use cases. Every mutation is a Unit of
// Work (ADR_Cross_Module_Transaction_Strategy): validation, stock movement,
// shift adjustment, return rows and the audit log commit atomically; the
// sale.returned.v1 event is published only after commit.
type Service struct {
	repo        *Repository
	bus         shared.EventBus
	sale        SaleReader
	activeShift ActiveShiftFinder
	adjuster    ShiftTotalAdjuster
	stock       StockChecker
	deductor    StockDeducer
	consign     ConsignmentGuard
	settings    SettingsProvider
	audit       *audit.Service
}

func NewService(repo *Repository, bus shared.EventBus) *Service {
	return &Service{repo: repo, bus: bus}
}

// Set* wiring. The composition root MUST set every port before a request
// touches the service; a missing port fails fast at call time with an
// explicit error rather than a nil-pointer panic.

func (s *Service) SetSaleReader(p SaleReader)                 { s.sale = p }
func (s *Service) SetActiveShiftFinder(p ActiveShiftFinder)   { s.activeShift = p }
func (s *Service) SetShiftTotalAdjuster(p ShiftTotalAdjuster) { s.adjuster = p }
func (s *Service) SetStockChecker(p StockChecker)             { s.stock = p }
func (s *Service) SetStockDeducer(p StockDeducer)             { s.deductor = p }
func (s *Service) SetConsignmentGuard(p ConsignmentGuard)     { s.consign = p }
func (s *Service) SetSettingsProvider(p SettingsProvider)     { s.settings = p }
func (s *Service) SetAuditService(a *audit.Service)           { s.audit = a }

func (s *Service) wired(name string, isSet bool) error {
	if isSet {
		return nil
	}
	return fmt.Errorf("salereturn: %s port is not wired", name)
}

// InTx runs fn inside a transaction, committing on success and rolling back
// on error (mirror of sale.service.InTx).
func (s *Service) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// policy is the effective return policy for one request.
type policy struct {
	windowDays              int
	approvalThreshold       int
	approvalTimeoutDuration time.Duration
}

func (s *Service) loadPolicy(ctx context.Context) (policy, error) {
	pol := policy{
		windowDays:              defaultWindowDays,
		approvalThreshold:       defaultApprovalThreshold,
		approvalTimeoutDuration: defaultApprovalTimeout * time.Minute,
	}
	if err := s.wired("SettingsProvider", s.settings != nil); err != nil {
		return pol, err
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingWindowDays, settingApprovalThreshold, settingApprovalTimeout})
	if err != nil {
		// A settings outage must not take returns down: fall back to the
		// seeded defaults and log.
		slog.Error("salereturn: failed to read policy settings, using defaults", "error", err)
		return pol, nil
	}
	if v, err := strconv.Atoi(values[settingWindowDays]); err == nil && v > 0 {
		pol.windowDays = v
	}
	if v, err := strconv.Atoi(values[settingApprovalThreshold]); err == nil && v >= 0 {
		pol.approvalThreshold = v
	}
	if v, err := strconv.Atoi(values[settingApprovalTimeout]); err == nil && v > 0 {
		pol.approvalTimeoutDuration = time.Duration(v) * time.Minute
	}
	return pol, nil
}

// plannedLine is one validated return line on its way to being posted.
type plannedLine struct {
	saleItem shared.ReturnSaleItem
	request  ItemRequest
	qty      int

	// lineRefund values (tax-inclusive refund of qty units).
	subtotal int
	dpp      int
	tax      int

	// replaceQty > 0 means the customer takes same-product stock instead of
	// money for this line; equal to qty (a line is replaced fully or refunded
	// fully, never split).
	replaceQty int
}

func (l *plannedLine) isRefund() bool { return l.replaceQty <= 0 }

// ProcessReturn processes a customer return against a completed sale
// (POST /sales/:id/return). The flow: validate the sale and the request, then
// either complete immediately (stock decided, refund split, shift adjusted)
// or park the return as pending_approval when the refund exceeds the
// approval threshold. Concurrent returns of the same sale are serialized by a
// transaction-scoped advisory lock so the over-return guard cannot race.
func (s *Service) ProcessReturn(ctx context.Context, actor Actor, storeID *int, saleID int, req ProcessRequest) (*Return, error) {
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}
	for _, port := range []struct {
		name  string
		isSet bool
	}{
		{"SaleReader", s.sale != nil},
		{"ActiveShiftFinder", s.activeShift != nil},
		{"ShiftTotalAdjuster", s.adjuster != nil},
		{"StockChecker", s.stock != nil},
		{"StockDeducer", s.deductor != nil},
		{"ConsignmentGuard", s.consign != nil},
	} {
		if err := s.wired(port.name, port.isSet); err != nil {
			return nil, err
		}
	}

	pol, err := s.loadPolicy(ctx)
	if err != nil {
		return nil, err
	}

	var out *Return
	invoiceNumber := ""
	err = s.InTx(ctx, func(tx pgx.Tx) error {
		out = nil

		// Serialize concurrent returns of the same sale: the over-return
		// guard and the stock decision both read rows a competing return
		// could change.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sale_return'), $1)`, saleID); err != nil {
			return fmt.Errorf("failed to lock sale for return: %w", err)
		}

		sale, err := s.sale.SaleForReturn(ctx, tx, saleID)
		if err != nil {
			return err
		}
		if sale == nil {
			return ErrSaleNotFound
		}
		invoiceNumber = sale.InvoiceNumber
		if err := validateSaleForReturn(sale, storeID, pol); err != nil {
			return err
		}

		// A return pays out of a drawer: without the cashier's own open shift
		// there is nowhere to take the money from (review decision — the PRD's
		// "skip the adjustment and accept drift" alternative was rejected).
		shiftID, err := s.activeShift.ActiveShiftID(ctx, actor.ID)
		if err != nil {
			return fmt.Errorf("failed to find active shift: %w", err)
		}
		if shiftID == 0 {
			return ErrShiftNotOpen
		}

		lines, err := s.planLines(ctx, tx, sale, req, 0)
		if err != nil {
			return err
		}

		if err := s.rejectConsignment(ctx, tx, sale, lines); err != nil {
			return err
		}

		// Decide replacement vs refund from CURRENT stock. On the immediate
		// path the decision also deducts; on the pending path it is only a
		// proposal that approval re-runs.
		immediate := true
		if err := s.proposeOutcomes(ctx, tx, lines); err != nil {
			return err
		}
		if totalRefundOf(lines) > pol.approvalThreshold {
			immediate = false
		}

		ret := &Return{
			OriginalSaleID: sale.ID,
			CashierID:      actor.ID,
			StoreID:        sale.StoreID,
			CustomerID:     sale.CustomerID,
			ShiftID:        shiftID,
			Reason:         req.Reason,
			Condition:      req.Condition,
			Notes:          req.Notes,
			ChangeDue:      sale.ChangeDue,
		}
		ret.ReturnNumber, err = s.repo.NextReturnNumber(ctx, tx)
		if err != nil {
			return err
		}

		if immediate {
			ret.Status = ReturnStatusCompleted
			if err := s.deduceReplacementStock(ctx, tx, lines); err != nil {
				return err
			}
			// The deduction above can demote a replacement line to refund
			// (lost the conditional decrement to a concurrent sale), and the
			// grown refund may now exceed the threshold the pre-deduction
			// check passed us under. Completing anyway would bypass the
			// approval flow; parking pending would double-deduct at approval
			// (no restore port). Roll back and let the cashier retry: the
			// retry re-plans the demoted line as a refund and goes pending
			// (or completes immediately if stock recovered).
			if totalRefundOf(lines) > pol.approvalThreshold {
				return ErrStockChanged
			}
			s.applyLineFigures(ret, lines)
			if err := s.repo.InsertReturn(ctx, tx, ret); err != nil {
				return err
			}
			// InsertItems fills the row IDs back into the slice; keep it so
			// the 201 response and the sale.returned.v1 ItemCount are not
			// empty (payments are already surfaced — items must match).
			ret.Items = returnItemsOf(lines)
			if err := s.repo.InsertItems(ctx, tx, ret.ID, ret.Items); err != nil {
				return err
			}
			alloc := allocateRefund(sale.Payments, sale.ChangeDue, ret.TotalRefundAmount)
			if err := s.insertRefundPayments(ctx, tx, ret, alloc); err != nil {
				return err
			}
			if err := s.applyShiftAdjustment(ctx, tx, ret, alloc); err != nil {
				return err
			}
		} else {
			ret.Status = ReturnStatusPendingApproval
			expiresAt := time.Now().Add(pol.approvalTimeoutDuration)
			ret.ExpiresAt = &expiresAt
			s.applyLineFigures(ret, lines)
			if err := s.repo.InsertReturn(ctx, tx, ret); err != nil {
				return err
			}
			ret.Items = returnItemsOf(lines)
			if err := s.repo.InsertItems(ctx, tx, ret.ID, ret.Items); err != nil {
				return err
			}
			reason := req.Reason
			if err := s.repo.InsertApproval(ctx, tx, ret.ID, actor.ID, &reason, expiresAt); err != nil {
				return err
			}
		}

		if err := s.auditTx(ctx, tx, actor, "return", ret, "Memproses retur penjualan"); err != nil {
			return err
		}
		out = ret
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out != nil && out.Status == ReturnStatusPendingApproval {
		s.publishApprovalRequested(ctx, out, actor, invoiceNumber)
	}
	s.publishReturned(ctx, out)
	return out, nil
}

// ApproveReturn completes a pending return (POST
// /sales/returns/:returnId/approve). The sale, the over-return guard and the
// stock/outcome decision are all re-run inside the approval transaction —
// the proposal written at creation time is advisory only. An elapsed approval
// window flips the return to expired in the SAME committed transaction while
// still returning ErrApprovalExpired to the caller.
func (s *Service) ApproveReturn(ctx context.Context, actor Actor, storeID *int, returnID int, note *string) (*Return, error) {
	for _, port := range []struct {
		name  string
		isSet bool
	}{
		{"SaleReader", s.sale != nil},
		{"ShiftTotalAdjuster", s.adjuster != nil},
		{"StockChecker", s.stock != nil},
		{"StockDeducer", s.deductor != nil},
		{"ConsignmentGuard", s.consign != nil},
	} {
		if err := s.wired(port.name, port.isSet); err != nil {
			return nil, err
		}
	}

	var out *Return
	var expiredReturn *Return
	expired := false
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		out, expired, expiredReturn = nil, false, nil

		ret, err := s.repo.GetReturnForUpdate(ctx, tx, returnID)
		if err != nil {
			return err
		}
		if storeID != nil && ret.StoreID != *storeID {
			return ErrReturnNotFound
		}
		if ret.Status != ReturnStatusPendingApproval {
			return ErrReturnNotPending
		}
		// Serialize with ProcessReturn and other approvals of the same sale
		// (same lock key/order as ProcessReturn: advisory lock is acquired
		// while this TX holds only the return's own row lock, and creation
		// never row-locks existing returns — no cycle is possible). Without
		// it, an approval racing its own expiry boundary could complete while
		// a concurrent creation treats it as an expired (non-consuming)
		// return, letting the sale's quantity be returned twice.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sale_return'), $1)`, ret.OriginalSaleID); err != nil {
			return fmt.Errorf("failed to lock sale for return: %w", err)
		}
		if ret.Approval != nil && time.Now().After(ret.Approval.ExpiresAt) {
			if err := s.repo.ExpireReturn(ctx, tx, ret.ID); err != nil {
				return err
			}
			if err := s.auditTx(ctx, tx, actor, "return", ret, "Retur kedaluwarsa karena melewati batas waktu persetujuan"); err != nil {
				return err
			}
			expired = true
			expiredReturn = ret
			return nil
		}

		sale, err := s.sale.SaleForReturn(ctx, tx, ret.OriginalSaleID)
		if err != nil {
			return err
		}
		if sale == nil {
			return ErrSaleNotFound
		}
		if sale.Status != "completed" {
			return ErrSaleNotCompleted
		}
		if storeID != nil && sale.StoreID != *storeID {
			return ErrStoreMismatch
		}

		lines, err := s.planLinesFromItems(sale, ret)
		if err != nil {
			return err
		}
		// Exclude the return's own proposal rows: they are about to be
		// replaced, and other returns may have consumed the sale's returnable
		// quantity since this one was created.
		returned, err := s.repo.ReturnedQtyBySale(ctx, tx, sale.ID, ret.ID)
		if err != nil {
			return err
		}
		for i := range lines {
			l := &lines[i]
			if returned[l.saleItem.ID]+l.qty > l.saleItem.Quantity {
				return fmt.Errorf("%w: %s", ErrQtyExceedsSold, l.saleItem.ProductName)
			}
		}

		if err := s.rejectConsignment(ctx, tx, sale, lines); err != nil {
			return err
		}

		if err := s.proposeOutcomes(ctx, tx, lines); err != nil {
			return err
		}
		if err := s.deduceReplacementStock(ctx, tx, lines); err != nil {
			return err
		}

		if err := s.repo.ClearItems(ctx, tx, ret.ID); err != nil {
			return err
		}
		// Replace the stale proposal rows loaded by GetReturnForUpdate with
		// the re-validated ones, and keep them on the in-memory return so the
		// approval response reflects the committed outcome (bug 4).
		ret.Items = returnItemsOf(lines)
		if err := s.repo.InsertItems(ctx, tx, ret.ID, ret.Items); err != nil {
			return err
		}
		s.applyLineFigures(ret, lines)
		alloc := allocateRefund(sale.Payments, sale.ChangeDue, ret.TotalRefundAmount)
		if err := s.insertRefundPayments(ctx, tx, ret, alloc); err != nil {
			return err
		}
		if err := s.applyShiftAdjustment(ctx, tx, ret, alloc); err != nil {
			return err
		}
		ret.Status = ReturnStatusCompleted
		if err := s.repo.MarkCompleted(ctx, tx, ret); err != nil {
			return err
		}
		if err := s.repo.ApproveApproval(ctx, tx, ret.ID, actor.ID, note); err != nil {
			return err
		}
		if err := s.auditTx(ctx, tx, actor, "return_approve", ret, "Menyetujui retur penjualan"); err != nil {
			return err
		}
		out = ret
		return nil
	})
	if err != nil {
		return nil, err
	}
	if expired {
		// The return flipped to expired in the committed transaction; the
		// caller still gets ErrApprovalExpired, but the queue must learn the
		// request is over. ret was captured before the rollback-free path
		// returned, so the id/number/store are the committed values.
		if expiredReturn != nil {
			expiredReturn.Status = ReturnStatusExpired
			s.publishApprovalResolved(ctx, expiredReturn)
		}
		return nil, ErrApprovalExpired
	}
	s.publishApprovalResolved(ctx, out)
	s.publishReturned(ctx, out)
	return out, nil
}

// RejectReturn cancels a pending return (POST /sales/returns/:returnId/reject).
// The cancelled rows stop consuming the sale's returnable quantity, so the
// customer can be served a fresh return afterwards.
func (s *Service) RejectReturn(ctx context.Context, actor Actor, storeID *int, returnID int, reason string) (*Return, error) {
	if reason == "" {
		return nil, fmt.Errorf("%w: rejection reason is required", ErrInvalidReturn)
	}

	var out *Return
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		out = nil
		ret, err := s.repo.GetReturnForUpdate(ctx, tx, returnID)
		if err != nil {
			return err
		}
		if storeID != nil && ret.StoreID != *storeID {
			return ErrReturnNotFound
		}
		if ret.Status != ReturnStatusPendingApproval {
			return ErrReturnNotPending
		}
		if err := s.repo.RejectApproval(ctx, tx, ret.ID, actor.ID, reason); err != nil {
			return err
		}
		ret.Status = ReturnStatusCancelled
		if err := s.auditTx(ctx, tx, actor, "return_reject", ret, "Menolak retur penjualan"); err != nil {
			return err
		}
		out = ret
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishApprovalResolved(ctx, out)
	return out, nil
}

// ListReturns lists a store's returns (optionally one sale or status).
func (s *Service) ListReturns(ctx context.Context, storeID *int, saleID *int, status *string, limit, offset int) ([]Return, int, error) {
	return s.repo.ListReturns(ctx, storeID, saleID, status, limit, offset)
}

// ListPending lists actionable pending approvals.
func (s *Service) ListPending(ctx context.Context, storeID *int, limit, offset int) ([]Return, int, error) {
	return s.repo.ListPending(ctx, storeID, limit, offset)
}

// GetReturn loads one return's full detail.
func (s *Service) GetReturn(ctx context.Context, storeID *int, id int) (*Return, error) {
	return s.repo.GetReturn(ctx, storeID, id)
}

// ==================== INTERNALS ====================

func (s *Service) validateRequest(req ProcessRequest) error {
	if req.Reason == "" {
		return fmt.Errorf("%w: reason is required", ErrInvalidReturn)
	}
	if req.Condition != ConditionDamaged {
		// PRD §10 locks the condition to 'damaged'; a functional unit would be
		// sellable stock and Phase 1 has no inspection workflow for it.
		return ErrConditionUnsupported
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("%w: no items", ErrInvalidReturn)
	}
	return nil
}

// validateSaleForReturn applies the eligibility rules of PRD §12.1: the sale
// must exist, be completed, belong to the caller's store and be inside the
// return window.
func validateSaleForReturn(sale *shared.ReturnSale, storeID *int, pol policy) error {
	if sale.Status != "completed" {
		return ErrSaleNotCompleted
	}
	if storeID != nil && sale.StoreID != *storeID {
		return ErrStoreMismatch
	}
	if time.Since(sale.CreatedAt) > time.Duration(pol.windowDays)*24*time.Hour {
		return ErrReturnWindowExpired
	}
	return nil
}

// planLines validates the requested quantities against the sale's lines and
// the already-returned quantities, and pre-computes each line's refund.
func (s *Service) planLines(ctx context.Context, tx pgx.Tx, sale *shared.ReturnSale, req ProcessRequest, excludeReturnID int) ([]plannedLine, error) {
	returned, err := s.repo.ReturnedQtyBySale(ctx, tx, sale.ID, excludeReturnID)
	if err != nil {
		return nil, err
	}
	return buildPlan(sale, req, returned)
}

// planLinesFromItems rebuilds a plan from a stored return's proposal rows,
// reusing the same validation against the sale.
func (s *Service) planLinesFromItems(sale *shared.ReturnSale, ret *Return) ([]plannedLine, error) {
	req := ProcessRequest{Reason: ret.Reason, Condition: ret.Condition, Notes: ret.Notes}
	for _, it := range ret.Items {
		req.Items = append(req.Items, ItemRequest{SaleItemID: it.SaleItemID, Quantity: it.Quantity, Notes: it.Notes})
	}
	return buildPlan(sale, req, nil)
}

func buildPlan(sale *shared.ReturnSale, req ProcessRequest, returned map[int]int) ([]plannedLine, error) {
	byID := make(map[int]shared.ReturnSaleItem, len(sale.Items))
	for _, it := range sale.Items {
		byID[it.ID] = it
	}
	seen := make(map[int]bool, len(req.Items))
	out := make([]plannedLine, 0, len(req.Items))
	for _, r := range req.Items {
		if r.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity must be positive for sale item %d", ErrInvalidReturn, r.SaleItemID)
		}
		if seen[r.SaleItemID] {
			return nil, fmt.Errorf("%w: duplicate sale item %d", ErrInvalidReturn, r.SaleItemID)
		}
		seen[r.SaleItemID] = true
		si, ok := byID[r.SaleItemID]
		if !ok {
			return nil, fmt.Errorf("%w: sale item %d does not belong to this sale", ErrSaleItemNotFound, r.SaleItemID)
		}
		if returned[si.ID]+r.Quantity > si.Quantity {
			return nil, fmt.Errorf("%w: %s", ErrQtyExceedsSold, si.ProductName)
		}
		subtotal, dpp, tax := lineRefund(r.Quantity, si.UnitPrice, si.TaxRate)
		out = append(out, plannedLine{
			saleItem: si,
			request:  r,
			qty:      r.Quantity,
			subtotal: subtotal,
			dpp:      dpp,
			tax:      tax,
		})
	}
	return out, nil
}

// rejectConsignment rejects the whole return when any requested line is a
// consignment-owned sale item (Phase 1 has no consignment restock; review
// decision).
func (s *Service) rejectConsignment(ctx context.Context, tx pgx.Tx, sale *shared.ReturnSale, lines []plannedLine) error {
	consignIDs, err := s.consign.ConsignmentProductIDs(ctx, tx, sale.ID)
	if err != nil {
		return fmt.Errorf("failed to check consignment items: %w", err)
	}
	if len(consignIDs) == 0 {
		return nil
	}
	consignSet := make(map[int]bool, len(consignIDs))
	for _, id := range consignIDs {
		consignSet[id] = true
	}
	for _, l := range lines {
		if consignSet[l.saleItem.ProductID] {
			return fmt.Errorf("%w: %s", ErrConsignmentNotSupported, l.saleItem.ProductName)
		}
	}
	return nil
}

// proposeOutcomes marks each line as replacement candidate when sellable
// stock is available (PRD §4.5 step 3). It only reads stock.
func (s *Service) proposeOutcomes(ctx context.Context, tx pgx.Tx, lines []plannedLine) error {
	for i := range lines {
		l := &lines[i]
		stock, err := s.stock.AvailableStock(ctx, tx, l.saleItem.ProductID)
		if err != nil {
			return fmt.Errorf("failed to check stock for product %d: %w", l.saleItem.ProductID, err)
		}
		if stock > 0 {
			l.replaceQty = l.qty
		} else {
			l.replaceQty = 0
		}
	}
	return nil
}

// deduceReplacementStock deducts each replacement line atomically. A
// concurrent sale can win the stock between the proposal and the deduction;
// when the conditional decrement fails, the line gracefully degrades to a
// refund instead of failing the whole return (PRD §4.5: refund when the item
// is the last one).
func (s *Service) deduceReplacementStock(ctx context.Context, tx pgx.Tx, lines []plannedLine) error {
	for i := range lines {
		l := &lines[i]
		if l.replaceQty <= 0 {
			continue
		}
		err := s.deductor.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: l.saleItem.ProductID, Quantity: l.replaceQty}})
		if errors.Is(err, shared.ErrInsufficientStock) {
			l.replaceQty = 0
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to deduct replacement stock for product %d: %w", l.saleItem.ProductID, err)
		}
	}
	return nil
}

// totalRefundOf sums the refunded (non-replacement) lines.
func totalRefundOf(lines []plannedLine) int {
	total := 0
	for i := range lines {
		if lines[i].isRefund() {
			total += lines[i].subtotal
		}
	}
	return total
}

// applyLineFigures recomputes the header's financial columns from the plan:
// subtotal/tax_reversal/total_refund count refund-outcome lines only (a
// replacement moves goods, not money); discount and restocking_fee stay 0
// (PRD §7: no restocking fee in Phase 1, item discounts are baked into
// unit_price).
func (s *Service) applyLineFigures(ret *Return, lines []plannedLine) {
	ret.Subtotal = 0
	ret.TaxReversal = 0
	ret.TotalRefundAmount = 0
	ret.Discount = 0
	ret.RestockingFee = 0
	anyRefund := false
	for i := range lines {
		if !lines[i].isRefund() {
			continue
		}
		anyRefund = true
		ret.Subtotal += lines[i].subtotal
		ret.TaxReversal += lines[i].tax
	}
	ret.TotalRefundAmount = ret.Subtotal
	if anyRefund {
		ret.ReturnType = ReturnTypeRefund
	} else {
		ret.ReturnType = ReturnTypeReplacement
	}
}

func returnItemsOf(lines []plannedLine) []ReturnItem {
	items := make([]ReturnItem, 0, len(lines))
	for _, l := range lines {
		it := ReturnItem{
			SaleItemID:  l.saleItem.ID,
			ProductID:   l.saleItem.ProductID,
			ProductName: l.saleItem.ProductName,
			Quantity:    l.qty,
			UnitPrice:   l.saleItem.UnitPrice,
			Subtotal:    l.subtotal,
			DppAmount:   l.dpp,
			TaxAmount:   l.tax,
			TaxRate:     l.saleItem.TaxRate,
			Condition:   ConditionDamaged,
			Outcome:     OutcomeWrittenOff,
			Notes:       l.request.Notes,
		}
		if l.replaceQty > 0 {
			it.Replacements = []Replacement{{ProductID: l.saleItem.ProductID, Quantity: l.replaceQty}}
		}
		items = append(items, it)
	}
	return items
}

func (s *Service) insertRefundPayments(ctx context.Context, tx pgx.Tx, ret *Return, alloc []refundAllocation) error {
	payments := make([]ReturnPayment, 0, len(alloc))
	for _, a := range alloc {
		payments = append(payments, ReturnPayment{
			PaymentMethodID:   a.PaymentMethodID,
			PaymentMethodCode: a.PaymentMethodCode,
			Amount:            a.Amount,
		})
	}
	if len(payments) == 0 {
		ret.Payments = nil
		return nil
	}
	if err := s.repo.InsertPayments(ctx, tx, ret.ID, payments); err != nil {
		return err
	}
	ret.Payments = payments
	return nil
}

// applyShiftAdjustment nets the refund into the processing shift's running
// totals. The shift module guards that the shift is still open; a shift that
// closed concurrently rejects the return (ErrShiftNotOpen) instead of
// silently corrupting closed totals.
func (s *Service) applyShiftAdjustment(ctx context.Context, tx pgx.Tx, ret *Return, alloc []refundAllocation) error {
	if ret.TotalRefundAmount <= 0 {
		return nil
	}
	cash, nonCash := cashAndNonCash(alloc)
	err := s.adjuster.ApplyReturnAdjustment(ctx, tx, shared.ShiftReturnAdjustment{
		ShiftID:       ret.ShiftID,
		CashRefund:    cash,
		NonCashRefund: nonCash,
		TotalRefund:   ret.TotalRefundAmount,
	})
	if errors.Is(err, shared.ErrShiftNotOpen) {
		return ErrShiftNotOpen
	}
	if err != nil {
		return fmt.Errorf("failed to adjust shift totals: %w", err)
	}
	return nil
}

// MarkCompleted writes the final header financials + status (used by the
// approval path; the creation path inserts final values directly).
func (r *Repository) MarkCompleted(ctx context.Context, tx pgx.Tx, ret *Return) error {
	tag, err := tx.Exec(ctx, `
		UPDATE sale_returns
		SET return_type = $2, subtotal = $3, discount = $4, restocking_fee = $5,
		    tax_reversal = $6, total_refund_amount = $7, status = 'completed',
		    change_due = $8, updated_at = now()
		WHERE id = $1
	`, ret.ID, ret.ReturnType, ret.Subtotal, ret.Discount, ret.RestockingFee,
		ret.TaxReversal, ret.TotalRefundAmount, ret.ChangeDue)
	if err != nil {
		return fmt.Errorf("failed to mark return completed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReturnNotFound
	}
	ret.Status = ReturnStatusCompleted
	return nil
}

func (s *Service) auditTx(ctx context.Context, tx pgx.Tx, actor Actor, action string, ret *Return, description string) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.CreateAuditLogTx(ctx, tx, &audit.Log{
		UserID:      &actor.ID,
		StoreID:     actor.StoreID,
		Username:    actor.Username,
		Role:        actor.Role,
		Action:      action,
		EntityType:  "sale_return",
		EntityID:    &ret.ID,
		Description: description,
		NewValues: shared.ToJSONMap(map[string]interface{}{
			"return_number":       ret.ReturnNumber,
			"original_sale_id":    ret.OriginalSaleID,
			"total_refund_amount": ret.TotalRefundAmount,
			"return_type":         ret.ReturnType,
			"status":              ret.Status,
			"shift_id":            ret.ShiftID,
		}),
		IPAddress: actor.IPAddress,
		UserAgent: actor.UserAgent,
	})
}

// publishReturned emits sale.returned.v1 after a successful commit (never
// inside the Unit of Work).
func (s *Service) publishReturned(ctx context.Context, ret *Return) {
	if s.bus == nil || ret == nil || ret.Status != ReturnStatusCompleted {
		return
	}
	_ = s.bus.Publish(ctx, events.TopicSaleReturned, &events.SaleReturned{
		ID:                ret.ID,
		ReturnNumber:      ret.ReturnNumber,
		OriginalSaleID:    ret.OriginalSaleID,
		StoreID:           ret.StoreID,
		ReturnType:        ret.ReturnType,
		TotalRefundAmount: ret.TotalRefundAmount,
		ItemCount:         len(ret.Items),
	})
}

// publishApprovalRequested emits sale.return.approval_requested.v1 after a
// return is parked pending approval, so a manager's open dashboard sees the
// request without polling. Published after commit, never inside the Unit of
// Work; invoice and cashier name come from the caller's snapshot (the invoice
// number is the sale's, and the cashier is the actor who created the return).
func (s *Service) publishApprovalRequested(ctx context.Context, ret *Return, actor Actor, invoiceNumber string) {
	if s.bus == nil || ret == nil || ret.Status != ReturnStatusPendingApproval {
		return
	}
	expiresAt := ""
	if ret.ExpiresAt != nil {
		expiresAt = ret.ExpiresAt.UTC().Format(time.RFC3339)
	}
	_ = s.bus.Publish(ctx, events.TopicReturnApprovalRequested, &events.ReturnApprovalRequested{
		ReturnID:          ret.ID,
		ReturnNumber:      ret.ReturnNumber,
		OriginalSaleID:    ret.OriginalSaleID,
		InvoiceNumber:     invoiceNumber,
		CashierID:         actor.ID,
		CashierName:       actor.Username,
		StoreID:           ret.StoreID,
		TotalRefundAmount: ret.TotalRefundAmount,
		ItemCount:         len(ret.Items),
		ExpiresAt:         expiresAt,
	})
}

// publishApprovalResolved emits sale.return.approval_resolved.v1 after a
// pending return reaches a terminal status through the approval queue
// (approved, rejected or expired). Published after commit, never inside the
// Unit of Work. Only pending approvals are announced, so a return that
// completed immediately (ProcessReturn) never emits a spurious resolution.
func (s *Service) publishApprovalResolved(ctx context.Context, ret *Return) {
	if s.bus == nil || ret == nil {
		return
	}
	switch ret.Status {
	case ReturnStatusCompleted, ReturnStatusCancelled, ReturnStatusExpired:
	default:
		return
	}
	_ = s.bus.Publish(ctx, events.TopicReturnApprovalResolved, &events.ReturnApprovalResolved{
		ReturnID:     ret.ID,
		ReturnNumber: ret.ReturnNumber,
		StoreID:      ret.StoreID,
		Status:       ret.Status,
	})
}

// ShiftReturnTotalsProvider implements the shift module's consumer-side port
// (shift.ReturnTotalsProvider): shift needs a shift's aggregate completed
// refunds to net returns out of its stored/live running totals (close-time
// netting, live fallback). salereturn owns the data, so the implementation
// lives here; internal/wiring passes it to shift via
// shift.Repository.SetReturnTotalsProvider (structural typing — no import of
// internal/shift).
type ShiftReturnTotalsProvider struct {
	Repo *Repository
}

// ReturnTotals aggregates on the pool.
func (p ShiftReturnTotalsProvider) ReturnTotals(ctx context.Context, db shared.DBPool, shiftID int) (shared.ShiftReturnTotals, error) {
	return p.Repo.ReturnTotalsForShift(ctx, db, shiftID)
}

// ReturnTotalsInTx aggregates inside the caller's transaction (shift close).
func (p ShiftReturnTotalsProvider) ReturnTotalsInTx(ctx context.Context, tx pgx.Tx, shiftID int) (shared.ShiftReturnTotals, error) {
	return p.Repo.ReturnTotalsForShift(ctx, tx, shiftID)
}

// ReportRefundStatsProvider implements the report module's consumer-side port
// (report.RefundStatsProvider): the report read model needs the completed
// refunds paid out in a window to net them out of gross completed-sale revenue.
// salereturn owns the data, so the implementation lives here; internal/wiring
// passes it to report via report.Repository.SetRefundStatsProvider (structural
// typing — no import of internal/report).
type ReportRefundStatsProvider struct {
	Repo *Repository
}

// GetCompletedRefundTotal aggregates the completed refunds in the window.
func (p ReportRefundStatsProvider) GetCompletedRefundTotal(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) (int, error) {
	return p.Repo.RefundTotalForWindow(ctx, db, start, end, storeID)
}
