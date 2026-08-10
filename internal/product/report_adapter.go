package product

import (
	"context"

	"retail-pos-system/internal/shared"
)

type reportAdapter struct{}

func NewReportAdapter() *reportAdapter {
	return &reportAdapter{}
}

func (reportAdapter) GetActiveProductCount(ctx context.Context, db shared.DBPool, storeID *int) (count int64, err error) {
	query := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	args := []interface{}{}
	if storeID != nil {
		query += ` AND store_id = $1`
		args = append(args, *storeID)
	}
	err = db.QueryRow(ctx, query, args...).Scan(&count)
	return
}
