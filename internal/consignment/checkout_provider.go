package consignment

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// CheckoutProvider is the consignment-owned implementation of the sale module's
// ConsignmentCheckout port (structural typing — no import of internal/sale
// needed). It resolves which checkout lines are consignment-owned, deducts
// their quantity from the consignment_stock ledger, and records the
// checkout-time consignment sale lines. internal/consignment is the canonical
// single-writer of consignment_stock and consignment_sale_items.
type CheckoutProvider struct {
	repo *Repository
}

func NewCheckoutProvider(repo *Repository) *CheckoutProvider {
	return &CheckoutProvider{repo: repo}
}

// ResolveAndDeductConsignment runs inside the checkout Unit of Work. Each line
// is locked FOR UPDATE; consignment-owned lines are checked against the
// available quantity and deducted from the ownership ledger (consignment_stock),
// and their current term's store-share is snapshotted into the returned record.
// Lines NOT consignment-owned are skipped (nil ledger row) and are still
// deducted from product_stock by the caller. Note: consignment_stock tracks
// ownership for settlement only; product_stock is the SELLABLE total (Model A),
// so the caller must also deduct consignment-owned lines from product_stock
// (their quantity was added there at receipt). The unit price used is the
// ACTUAL sale unit price from the pricing engine (BR-15/AC-C11), so the store
// share is computed against real sale value (PRD §10.2/§10.3).
func (p *CheckoutProvider) ResolveAndDeductConsignment(ctx context.Context, tx pgx.Tx, items []shared.ConsignmentCheckoutItem) ([]shared.ConsignmentSaleRecord, error) {
	var records []shared.ConsignmentSaleRecord
	for _, item := range items {
		row, err := p.repo.LockConsignmentStock(ctx, tx, item.ProductID)
		if err != nil {
			return nil, err
		}
		if row == nil {
			// Store-owned: caller handles this line via product_stock.
			continue
		}
		if row.AvailableQty < item.Quantity {
			return nil, ErrInsufficientConsignmentStock
		}
		if err := p.repo.ReduceAvailable(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return nil, err
		}
		term, err := p.repo.GetTermByProduct(ctx, tx, row.ArrangementID, item.ProductID)
		if err != nil {
			return nil, err
		}
		records = append(records, shared.ConsignmentSaleRecord{
			ProductID:       item.ProductID,
			SupplierID:      row.SupplierID,
			ArrangementID:   row.ArrangementID,
			StoreID:         row.StoreID,
			Quantity:        item.Quantity,
			UnitPrice:       item.UnitPrice,
			Subtotal:        item.UnitPrice * item.Quantity,
			StoreShareType:  term.StoreShareType,
			StoreShareValue: term.StoreShareValue,
		})
	}
	return records, nil
}

// RecordConsignmentSaleItems persists the checkout-time records to
// consignment_sale_items, linked to the just-created sale. Runs on the caller's
// tx so the sale and its consignment lines commit atomically.
func (p *CheckoutProvider) RecordConsignmentSaleItems(ctx context.Context, tx pgx.Tx, saleID int, records []shared.ConsignmentSaleRecord) error {
	for _, rec := range records {
		rec.SaleID = saleID
		if err := p.repo.InsertConsignmentSaleItem(ctx, tx, rec); err != nil {
			return err
		}
	}
	return nil
}
