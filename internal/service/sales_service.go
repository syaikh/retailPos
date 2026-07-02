package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	model "retailPos/internal/model"
	"retailPos/internal/repo"
	"retailPos/internal/ws"
)

type SalesService struct {
	db          *sql.DB
	productRepo *repo.ProductRepo
	userRepo    *repo.UserRepo
	hub         *ws.Hub
}

func NewSalesService(db *sql.DB, productRepo *repo.ProductRepo, userRepo *repo.UserRepo, hub *ws.Hub) *SalesService {
	return &SalesService{
		db:          db,
		productRepo: productRepo,
		userRepo:    userRepo,
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

	// 1. Get cashier's store_id
	cashier, err := s.userRepo.GetByID(sale.CashierID)
	if err != nil {
		fmt.Printf("CreateSale error: cashier not found: %v\n", err)
		return errors.New("cashier not found")
	}

	// Set store_id on sale if cashier has one
	if cashier != nil && cashier.StoreID != nil {
		sale.StoreID = cashier.StoreID
	}

	// 2. Insert Sale record with store_id
	querySale := `INSERT INTO sales (total_amount, payment_method, cashier_id, store_id) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	err = tx.QueryRowContext(ctx, querySale, sale.TotalAmount, sale.PaymentMethod, sale.CashierID, sale.StoreID).Scan(&sale.ID, &sale.CreatedAt)
	if err != nil {
		fmt.Printf("CreateSale error: failed to insert sale: %v\n", err)
		return err
	}

	// Generate transaction code after we have the ID
	sale.TransactionCode = fmt.Sprintf("TRX-%06d", sale.ID)

	// 3. Insert Sale Items and Update Stock
	for i := range sale.Items {
		item := &sale.Items[i]
		item.SaleID = sale.ID

		// Fetch snapshot product name
		p, err := s.productRepo.GetByID(item.ProductID, sale.StoreID)
		if err != nil || p == nil {
			fmt.Printf("CreateSale error: product %d not found in store %v: %v\n", item.ProductID, sale.StoreID, err)
			return errors.New("product not found during snapshot")
		}
		item.ProductName = p.Name

		// Record item with snapshot
		queryItem := `INSERT INTO sale_items (sale_id, product_id, product_name, quantity, price_at_sale) VALUES ($1, $2, $3, $4, $5) RETURNING id`
		err = tx.QueryRowContext(ctx, queryItem, item.SaleID, item.ProductID, item.ProductName, item.Quantity, item.PriceAtSale).Scan(&item.ID)
		if err != nil {
			fmt.Printf("CreateSale error: failed to insert sale item: %v\n", err)
			return err
		}

		// Update stock
		queryStock := `UPDATE products SET stock = stock - $1, updated_at = NOW() WHERE id = $2 AND stock >= $1`
		res, err := tx.ExecContext(ctx, queryStock, item.Quantity, item.ProductID)
		if err != nil {
			fmt.Printf("CreateSale error: failed to update stock for product %d: %v\n", item.ProductID, err)
			return err
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			fmt.Printf("CreateSale error: insufficient stock for product %d\n", item.ProductID)
			return errors.New("insufficient stock or product not found")
		}
	}

	if err := tx.Commit(); err != nil {
		fmt.Printf("CreateSale error: failed to commit transaction: %v\n", err)
		return err
	}

	// 4. Broadcast events to the store
	var storeIDStr string
	if sale.StoreID != nil {
		storeIDStr = fmt.Sprintf("%d", *sale.StoreID)
	}
	_ = s.hub.Broadcast(storeIDStr, "sale.created", sale)
	for _, item := range sale.Items {
		_ = s.hub.Broadcast(storeIDStr, "stock.updated", map[string]any{
			"product_id": item.ProductID,
		})
	}

	return nil
}
