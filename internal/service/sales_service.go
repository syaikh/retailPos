package service

import (
	"context"
	"database/sql"
	"errors"
	model "retailPos/internal/model"
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

	// Validate payment method
	if sale.PaymentMethod != "cash" && sale.PaymentMethod != "card" {
		return errors.New("invalid payment method: must be 'cash' or 'card'")
	}

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

		// Fetch snapshot product name
		p, err := s.productRepo.GetByID(item.ProductID)
		if err != nil || p == nil {
			return errors.New("product not found during snapshot")
		}
		item.ProductName = p.Name

		// Record item with snapshot
		queryItem := `INSERT INTO sale_items (sale_id, product_id, product_name, quantity, price_at_sale) VALUES ($1, $2, $3, $4, $5) RETURNING id`
		err = tx.QueryRowContext(ctx, queryItem, item.SaleID, item.ProductID, item.ProductName, item.Quantity, item.PriceAtSale).Scan(&item.ID)
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
	_ = s.hub.Broadcast("", "sale.created", sale)
	for _, item := range sale.Items {
		_ = s.hub.Broadcast("", "stock.updated", map[string]any{
			"product_id": item.ProductID,
		})
	}

	return nil
}
