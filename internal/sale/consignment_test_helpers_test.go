package sale

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// noopConsignmentCheckout is a test stand-in for the consignment checkout port.
// It treats every product as store-owned (returns no consignment records), so
// tests that exercise the sale flow without consignment wiring continue to run
// against product_stock only. It mirrors how tests wire the real-but-inert
// inventory.StockDeducer{} and shift.TotalUpdater{}.
type noopConsignmentCheckout struct{}

func (noopConsignmentCheckout) ResolveAndDeductConsignment(ctx context.Context, tx pgx.Tx, items []shared.ConsignmentCheckoutItem) ([]shared.ConsignmentSaleRecord, error) {
	return nil, nil
}

func (noopConsignmentCheckout) RecordConsignmentSaleItems(ctx context.Context, tx pgx.Tx, saleID int, records []shared.ConsignmentSaleRecord) error {
	return nil
}