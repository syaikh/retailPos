package service

import (
	"context"
	"database/sql"
	"errors"
	"retailPos/internal/model"
	"retailPos/internal/repo"
	"retailPos/internal/ws"
)

type SalesService struct {
	db          *sql.DB
	productRepo *repo.ProductRepo
	hub         *ws.Hub
}

func NewSalesService(db *sql.DB, productRepo *repo.ProductRepo, hub *ws.Hub) *SalesService {
	return &SalesService{
		db:          db,
		productRepo: productRepo,
		hub:         hub,
	}
}

func (s *SalesService) CreateSale(ctx context.Context, sale *model.Sale) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insert Sale record
	querySale := `INSERT INTO sales (total_amount, payment_method, cashier_id) VALUES ($1, $2, $3) RETURNING id, created_at`
	err = tx.QueryRowContext(ctx, querySale, sale.TotalAmount, sale.PaymentMethod, sale.CashierID).Scan(&sale.ID, &sale.CreatedAt)
	if err != nil {
		return err
	}

	// 2. Insert Sale Items and Update Stock
	for i := range sale.Items {
		item := &sale.Items[i]
		item.SaleID = sale.ID

		// Record item
		queryItem := `INSERT INTO sale_items (sale_id, product_id, quantity, price_at_sale) VALUES ($1, $2, $3, $4) RETURNING id`
		err = tx.QueryRowContext(ctx, queryItem, item.SaleID, item.ProductID, item.Quantity, item.PriceAtSale).Scan(&item.ID)
		if err != nil {
			return err
		}

		// Update stock
		queryStock := `UPDATE products SET stock = stock - $1, updated_at = NOW() WHERE id = $2 AND stock >= $1`
		res, err := tx.ExecContext(ctx, queryStock, item.Quantity, item.ProductID)
		if err != nil {
			return err
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return errors.New("insufficient stock or product not found")
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 3. Broadcast events
	s.hub.Broadcast("sale_created", sale)
	for _, item := range sale.Items {
		s.hub.Broadcast("stock_updated", map[string]interface{}{
			"product_id": item.ProductID,
		})
	}

	return nil
}
