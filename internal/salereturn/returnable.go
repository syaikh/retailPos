package salereturn

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Ineligible reason codes reported by GetReturnable. They mirror the guards
// ProcessReturn enforces (status, window, shift, over-return) but travel as
// data instead of errors, so the UI can disable the "Return Items" button
// with an explanation instead of failing after the form is filled (PRD
// §10.1: the entry point disables the button on non-completed sales, an
// expired window, or a fully returned sale).
const (
	IneligibleNotCompleted  = "not_completed"
	IneligibleWindowExpired = "window_expired"
	IneligibleShiftNotOpen  = "shift_not_open"
	IneligibleFullyReturned = "fully_returned"
)

// ReturnableSale is the read-only pre-flight snapshot behind
// GET /sales/:id/returnable: enough to render the return modal (PRD §10.2)
// without guessing — server-authoritative eligibility, the policy values the
// UI cannot read from /api/settings, and one row per original sale line with
// how much of it can still be returned. Read-only; ProcessReturn re-runs
// every guard inside its own transaction, so this snapshot is advisory only.
type ReturnableSale struct {
	SaleID            int              `json:"sale_id"`
	InvoiceNumber     string           `json:"invoice_number"`
	Status            string           `json:"status"`
	CreatedAt         time.Time        `json:"created_at"`
	TotalAmount       int              `json:"total_amount"`
	Eligible          bool             `json:"eligible"`
	IneligibleReason  string           `json:"ineligible_reason,omitempty"`
	WindowDays        int              `json:"window_days"`
	ApprovalThreshold int              `json:"approval_threshold"`
	Items             []ReturnableItem `json:"items"`
}

// ReturnableItem is one original sale line as the return modal sees it.
// ReturnedQuantity counts units already consumed by non-cancelled returns
// (ReturnedQtyBySale excludes cancelled/expired rows); StockAvailable feeds
// the replacement-vs-refund indicator, which stays advisory because approval
// re-decides from current stock. Consignment marks lines Phase 1 cannot
// return, so the UI can refuse to select them instead of erroring at submit.
type ReturnableItem struct {
	SaleItemID         int     `json:"sale_item_id"`
	ProductID          int     `json:"product_id"`
	ProductName        string  `json:"product_name"`
	Quantity           int     `json:"quantity"`
	ReturnedQuantity   int     `json:"returned_quantity"`
	ReturnableQuantity int     `json:"returnable_quantity"`
	UnitPrice          int     `json:"unit_price"`
	Subtotal           int     `json:"subtotal"`
	TaxRate            float64 `json:"tax_rate"`
	StockAvailable     int     `json:"stock_available"`
	Consignment        bool    `json:"consignment"`
}

// GetReturnable builds the snapshot the return modal renders (GET
// /sales/:id/returnable, gated by sale.return). Only "no such sale" and a
// cross-store sale remain errors (404 — do not leak sale existence); every
// policy violation becomes Eligible=false plus IneligibleReason so the entry
// point can render a disabled button with a reason.
func (s *Service) GetReturnable(ctx context.Context, actor Actor, storeID *int, saleID int) (*ReturnableSale, error) {
	for _, port := range []struct {
		name  string
		isSet bool
	}{
		{"SaleReader", s.sale != nil},
		{"ActiveShiftFinder", s.activeShift != nil},
		{"StockChecker", s.stock != nil},
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

	var out *ReturnableSale
	err = s.InTx(ctx, func(tx pgx.Tx) error {
		out = nil

		sale, err := s.sale.SaleForReturn(ctx, tx, saleID)
		if err != nil {
			return err
		}
		if sale == nil {
			return ErrSaleNotFound
		}
		if storeID != nil && sale.StoreID != *storeID {
			return ErrStoreMismatch
		}

		// Same guard order as validateSaleForReturn, as reason codes.
		reason := ""
		switch {
		case sale.Status != "completed":
			reason = IneligibleNotCompleted
		case time.Since(sale.CreatedAt) > time.Duration(pol.windowDays)*24*time.Hour:
			reason = IneligibleWindowExpired
		}

		// A return pays out of a drawer: without an open shift there is
		// nowhere to take the money from (same guard as ProcessReturn).
		if reason == "" {
			shiftID, err := s.activeShift.ActiveShiftID(ctx, actor.ID)
			if err != nil {
				return fmt.Errorf("failed to find active shift: %w", err)
			}
			if shiftID == 0 {
				reason = IneligibleShiftNotOpen
			}
		}

		returned, err := s.repo.ReturnedQtyBySale(ctx, tx, sale.ID, 0)
		if err != nil {
			return err
		}
		consignIDs, err := s.consign.ConsignmentProductIDs(ctx, tx, sale.ID)
		if err != nil {
			return fmt.Errorf("failed to check consignment items: %w", err)
		}
		consignSet := make(map[int]bool, len(consignIDs))
		for _, id := range consignIDs {
			consignSet[id] = true
		}

		items := make([]ReturnableItem, 0, len(sale.Items))
		openLines := 0
		for _, it := range sale.Items {
			returnedQty := returned[it.ID]
			if returnedQty < 0 {
				returnedQty = 0
			}
			returnable := it.Quantity - returnedQty
			if returnable < 0 {
				returnable = 0
			}
			// Stock only matters for a line that can still be returned.
			stock := 0
			if returnable > 0 {
				stock, err = s.stock.AvailableStock(ctx, tx, it.ProductID)
				if err != nil {
					return fmt.Errorf("failed to check stock: %w", err)
				}
				openLines++
			}
			items = append(items, ReturnableItem{
				SaleItemID:         it.ID,
				ProductID:          it.ProductID,
				ProductName:        it.ProductName,
				Quantity:           it.Quantity,
				ReturnedQuantity:   returnedQty,
				ReturnableQuantity: returnable,
				UnitPrice:          it.UnitPrice,
				Subtotal:           it.Subtotal,
				TaxRate:            it.TaxRate,
				StockAvailable:     stock,
				Consignment:        consignSet[it.ProductID],
			})
		}
		if reason == "" && openLines == 0 {
			reason = IneligibleFullyReturned
		}

		out = &ReturnableSale{
			SaleID:            sale.ID,
			InvoiceNumber:     sale.InvoiceNumber,
			Status:            sale.Status,
			CreatedAt:         sale.CreatedAt,
			TotalAmount:       sale.TotalAmount,
			Eligible:          reason == "",
			IneligibleReason:  reason,
			WindowDays:        pol.windowDays,
			ApprovalThreshold: pol.approvalThreshold,
			Items:             items,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
