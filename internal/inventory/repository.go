package inventory

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Warning: failed to load Asia/Jakarta timezone: %v. Falling back to UTC.", err)
		jakartaLoc = time.UTC
	}
}

func mustLoadJakarta() *time.Location {
	if jakartaLoc == nil {
		return time.UTC
	}
	return jakartaLoc
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error) {
	var ps ProductStock
	var storeID, warehouseID sql.NullInt64
	var reorderPoint, reorderQuantity sql.NullInt64
	var lastRestockedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, warehouse_id, store_id, quantity, reorder_point, reorder_quantity, last_restocked_at, created_at, updated_at
		FROM product_stock
		WHERE product_id = $1
		ORDER BY id ASC LIMIT 1
	`, productID).Scan(&ps.ID, &ps.ProductID, &warehouseID, &storeID, &ps.Quantity, &reorderPoint, &reorderQuantity, &lastRestockedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		ps.WarehouseID = &v
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		ps.StoreID = &v
	}
	if reorderPoint.Valid {
		v := int(reorderPoint.Int64)
		ps.ReorderPoint = v
	}
	if reorderQuantity.Valid {
		v := int(reorderQuantity.Int64)
		ps.ReorderQuantity = v
	}
	if lastRestockedAt.Valid {
		ps.LastRestockedAt = lastRestockedAt.Time.Format(time.RFC3339)
	}
	ps.CreatedAt = createdAt.Format(time.RFC3339)
	ps.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &ps, nil
}

func (r *Repository) AdjustStock(ctx context.Context, productID int, quantityChange int, userID *int, notes string) error {
	if quantityChange == 0 {
		return fmt.Errorf("quantity change must not be zero")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var currentStock int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(quantity, 0) FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
		FOR UPDATE
	`, productID).Scan(&currentStock)
	if err != nil {
		if err == pgx.ErrNoRows {
			currentStock = 0
		} else {
			return fmt.Errorf("failed to load product stock: %w", err)
		}
	}

	newStock := currentStock + quantityChange
	if newStock < 0 {
		return fmt.Errorf("insufficient stock: current %d, requested %d", currentStock, quantityChange)
	}

	cmd, err := tx.Exec(ctx, `
		UPDATE product_stock
		SET quantity = $2, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
	`, productID, newStock)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at)
			VALUES ($1, $2, NOW())
		`, productID, newStock)
		if err != nil {
			return fmt.Errorf("failed to insert stock: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes)
		VALUES ($1, $2, $3, NULL, NULL, $4, $5)
	`, productID, quantityChange, "adjustment", userID, notes)
	if err != nil {
		return fmt.Errorf("failed to record inventory movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit stock adjustment: %w", err)
	}

	return nil
}
