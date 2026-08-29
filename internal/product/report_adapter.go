package product

import (
	"context"

	"retail-pos-system/internal/shared"
)

type ReportAdapter struct{}

func NewReportAdapter() *ReportAdapter {
	return &ReportAdapter{}
}

func (ReportAdapter) GetActiveProductCount(ctx context.Context, db shared.DBPool, storeID *int) (count int64, err error) {
	query := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	args := []interface{}{}
	if storeID != nil {
		query += ` AND store_id = $1`
		args = append(args, *storeID)
	}
	err = db.QueryRow(ctx, query, args...).Scan(&count)
	return
}
