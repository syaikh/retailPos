package salereturn

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// Consumer-side ports of the salereturn module. Each one is implemented by
// the module that single-owns the data it needs and wired by internal/wiring
// through a fail-fast setter (ADR_Modular_Monolith_Module_Boundaries,
// ADR_Cross_Module_Transaction_Strategy). Types shared with the providers
// live in internal/shared so no domain module imports another.

// SaleReader loads the original sale as needed by the return flow: header,
// lines and payments, read ON THE CALLER'S TRANSACTION so the sale cannot
// change between validation and posting. Implemented by internal/sale.
type SaleReader interface {
	SaleForReturn(ctx context.Context, tx pgx.Tx, saleID int) (*shared.ReturnSale, error)
}

// ActiveShiftFinder returns the caller's currently open shift id, or 0 when
// there is none. Implemented by internal/shift.
type ActiveShiftFinder interface {
	ActiveShiftID(ctx context.Context, userID int) (int, error)
}

// ShiftTotalAdjuster applies one completed return's effect to a shift's
// running totals, guarded by the shift being open. Implemented by
// internal/shift (shift.TotalUpdater).
type ShiftTotalAdjuster interface {
	ApplyReturnAdjustment(ctx context.Context, tx pgx.Tx, adj shared.ShiftReturnAdjustment) error
}

// StockChecker reads a product's sellable (global) stock so the service can
// decide replacement vs refund per line. Implemented by internal/inventory.
type StockChecker interface {
	AvailableStock(ctx context.Context, db shared.DBPool, productID int) (int, error)
}

// StockDeducer deducts replacement stock on the caller's transaction.
// Implemented by internal/inventory (inventory.StockDeducer, structural match
// with sale's port so the same provider instance is wired for both consumers).
type StockDeducer interface {
	DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error
}

// ConsignmentGuard reports which of a sale's products were sold as
// consignment-owned goods; the service rejects a return touching any of them
// in Phase 1. Implemented by internal/consignment.
type ConsignmentGuard interface {
	ConsignmentProductIDs(ctx context.Context, tx pgx.Tx, saleID int) ([]int, error)
}

// SettingsProvider reads the return policy app_settings (return.window_days,
// return.approval_threshold, return.approval_timeout_minutes). Implemented by
// internal/appsettings.
type SettingsProvider interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
}
