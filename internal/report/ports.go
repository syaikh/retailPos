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
type SaleStatsProvider interface {
	GetCompletedSalesStats(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) (revenue int, orders int, err error)
	GetAllCompletedSalesStats(ctx context.Context, db shared.DBPool, storeID *int) (revenue int, orders int, err error)
	GetActiveCustomerCount(ctx context.Context, db shared.DBPool, storeID *int) (count int64, err error)
	GetWeeklySales(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.WeeklyReportItem, error)
	GetMonthlySales(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.MonthlyReportItem, error)
	GetPricingBreakdown(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.PricingBreakdownItem, error)
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
