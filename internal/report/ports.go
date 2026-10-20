package report

import (
	"context"
	"time"

	"retail-pos-system/internal/shared"
)

// SaleStatsProvider provides sales aggregations needed by the report module.
// Owner: internal/sale.
//
// These ports exist so that report/repository.go never queries the sales,
// products, product_stock, or sale_items tables directly. When sale/product/inventory
// are extracted to microservices, only the adapter implementation in the owning
// module needs to change; report/ports.go and report/repository.go are untouched.
// The reporting materialized views (mv_*) are report's own read model and are
// queried directly by report/repository.go — they need no port.
type SaleStatsProvider interface {
	// GetCompletedSalesStats returns the GROSS completed-sale revenue and order
	// count in [start, end). It deliberately does not net Sales Returns: sale
	// owns the sales table only, and sale_returns belongs to internal/salereturn.
	// The report module nets the refund side itself via RefundStatsProvider, so
	// the two owners stay behind their own ports.
	GetCompletedSalesStats(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) (revenue int, orders int, err error)
	GetActiveCustomerCount(ctx context.Context, db shared.DBPool, storeID *int) (count int64, err error)
	GetWeeklySales(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.WeeklyReportItem, error)
	GetMonthlySales(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.MonthlyReportItem, error)
	GetPricingBreakdown(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.PricingBreakdownItem, error)
}

// RefundStatsProvider provides the completed-refund money paid out in a window,
// which the report module subtracts from gross sale revenue. Owner:
// internal/salereturn.
//
// Returns are not sales rows, so sale cannot net them from its own tables;
// internal/salereturn owns sale_returns/sale_return_payments and provides the
// production implementation (structural typing — wiring passes
// salereturn.ReportRefundStatsProvider here, no import of internal/salereturn).
//
// Only status = 'completed' returns count, and only total_refund_amount is
// subtracted: a pure replacement return pays nothing out. The underlying
// netting must stay in step with the reporting materialized views
// (068_report_return_netting.sql) and the shift close-time netting
// (salereturn.Repository.ReturnTotalsForShift).
type RefundStatsProvider interface {
	// GetCompletedRefundTotal returns the total refund paid out in [start, end)
	// for the optional store. Returns are bucketed by their own created_at in
	// Asia/Jakarta, exactly as the sales side is.
	GetCompletedRefundTotal(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) (refundTotal int, err error)
}

// ProductStatsProvider provides product statistics needed by the report module.
// Owner: internal/product.
type ProductStatsProvider interface {
	GetActiveProductCount(ctx context.Context, db shared.DBPool, storeID *int) (count int64, err error)
}

// StockStatsProvider provides stock statistics needed by the report module.
// Owner: internal/inventory.
type StockStatsProvider interface {
	GetLowStockCount(ctx context.Context, db shared.DBPool, threshold int, storeID *int) (count int64, err error)
}
