package inventory

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

type ReportAdapter struct{}

func NewReportAdapter() *ReportAdapter {
	return &ReportAdapter{}
}

func (ReportAdapter) GetLowStockCount(ctx context.Context, db shared.DBPool, threshold int, storeID *int) (count int64, err error) {
	query := `SELECT COUNT(*) FROM product_stock WHERE quantity <= $1`
	args := []interface{}{threshold}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	err = db.QueryRow(ctx, query, args...).Scan(&count)
	return
}
