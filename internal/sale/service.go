package sale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/shared"
)

var ErrInsufficientStock = errors.New("insufficient stock")
var ErrPriceMismatch = errors.New("price mismatch: client-submitted price does not match server price")
var ErrSaleNotFound = errors.New("sale not found")
var ErrParkedSaleNotRecalled = errors.New("parked sale not in recalled state")

type ProductPriceGetter interface {
	GetProductPrice(ctx context.Context, productID int) (int, error)
}

type ProductBatchPriceGetter interface {
	GetProductPrices(ctx context.Context, productIDs []int) (map[int]int, error)
}

type Service struct {
	repo       *Repository
	eventBus   shared.EventBus
	priceStore ProductPriceGetter
	resolver   pricing.PriceResolver
}

func NewService(repo *Repository, eventBus shared.EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) SetPriceStore(ps ProductPriceGetter) {
	s.priceStore = ps
}

func (s *Service) SetPriceResolver(r pricing.PriceResolver) {
	s.resolver = r
}

func (s *Service) processSaleItems(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem) error {
	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("invalid quantity %d for product %d", item.Quantity, item.ProductID)
		}
	}

	productIDs := make([]int, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	rows, err := tx.Query(ctx, `SELECT product_id, COALESCE(quantity, 0) FROM product_stock WHERE product_id = ANY($1) AND warehouse_id IS NULL AND store_id IS NULL FOR UPDATE`, productIDs)
	if err != nil {
		return fmt.Errorf("batch check stock: %w", err)
	}
	stockMap := make(map[int]int, len(items))
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan stock: %w", err)
		}
		stockMap[pid] = qty
	}
	rows.Close()

	for _, item := range items {
		stock, ok := stockMap[item.ProductID]
		if !ok {
			return fmt.Errorf("stock record not found for product %d", item.ProductID)
		}
		if stock < item.Quantity {
			return ErrInsufficientStock
		}
	}

	stockPIDs := make([]int, len(items))
	stockQtys := make([]int, len(items))
	for i, item := range items {
		stockPIDs[i] = item.ProductID
		stockQtys[i] = item.Quantity
	}
	_, err = tx.Exec(ctx, `UPDATE product_stock SET quantity = quantity - v.qty
		FROM (SELECT unnest($1::int[]) AS product_id, unnest($2::int[]) AS qty) v
		WHERE product_stock.product_id = v.product_id AND warehouse_id IS NULL AND store_id IS NULL`, stockPIDs, stockQtys)
	if err != nil {
		return fmt.Errorf("batch deduct stock: %w", err)
	}

	if s.resolver != nil {
		resolveItems := make([]pricing.ResolveItem, len(items))
		for i, item := range items {
			resolveItems[i] = pricing.ResolveItem{ProductID: item.ProductID, Quantity: item.Quantity}
		}

		resolved, err := s.resolver.ResolveBatch(ctx, resolveItems)
		if err != nil {
			return fmt.Errorf("resolve prices: %w", err)
		}

		sale.Subtotal = 0
		for i := range items {
			r := resolved[i]

			if clientUnitPrice := items[i].UnitPrice; clientUnitPrice != r.UnitPrice {
				slog.Warn("price mismatch", "product_id", items[i].ProductID, "server_price", r.UnitPrice, "client_price", clientUnitPrice)
			}

			items[i].UnitPrice = r.UnitPrice
			items[i].Subtotal = r.UnitPrice * items[i].Quantity
			items[i].OriginalPrice = &r.OriginalPrice

			if r.Rule != nil {
				ruleID := r.Rule.ID
				ruleName := r.Rule.Name
				ruleType := string(r.Rule.PricingType)
				pt := string(r.PricingType)
				items[i].PricingRuleID = &ruleID
				items[i].PricingRuleName = &ruleName
				items[i].PricingRuleType = &ruleType
				items[i].PricingType = &pt
			} else {
				pt := string(pricing.PricingTypeDefault)
				items[i].PricingType = &pt
			}

			sale.Subtotal += items[i].Subtotal
		}
	} else if s.priceStore != nil {
		productIDs := make([]int, len(items))
		for i, item := range items {
			productIDs[i] = item.ProductID
		}

		var prices map[int]int
		if batchGetter, ok := s.priceStore.(ProductBatchPriceGetter); ok {
			prices, err = batchGetter.GetProductPrices(ctx, productIDs)
			if err != nil {
				return fmt.Errorf("batch lookup prices: %w", err)
			}
		} else {
			prices = make(map[int]int, len(productIDs))
			for _, pid := range productIDs {
				p, err := s.priceStore.GetProductPrice(ctx, pid)
				if err != nil {
					return fmt.Errorf("lookup price for product %d: %w", pid, err)
				}
				prices[pid] = p
			}
		}

		sale.Subtotal = 0
		for i, item := range items {
			serverPrice, ok := prices[item.ProductID]
			if !ok {
				return fmt.Errorf("price not found for product %d", item.ProductID)
			}

			if clientUnitPrice := item.UnitPrice; clientUnitPrice != serverPrice {
				slog.Warn("price mismatch", "product_id", item.ProductID, "server_price", serverPrice, "client_price", clientUnitPrice)
			}

			items[i].UnitPrice = serverPrice
			items[i].Subtotal = serverPrice * item.Quantity
			sale.Subtotal += items[i].Subtotal
		}
	}

	sale.TotalAmount = sale.Subtotal - sale.Discount
	if sale.TotalAmount < 0 {
		sale.TotalAmount = 0
	}

	return nil
}

func (s *Service) CreateSale(ctx context.Context, sale *Sale, items []SaleItem) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.processSaleItems(ctx, tx, sale, items); err != nil {
		return err
	}

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return err
	}

	if sale.ShiftID != nil {
		if err := s.repo.UpdateShiftTotals(ctx, tx, *sale.ShiftID, sale.TotalAmount, sale.PaymentMethod); err != nil {
			return fmt.Errorf("update shift totals: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	sale.Items = items
	_ = s.eventBus.Publish(ctx, "sale.created", sale)

	return nil
}

func (s *Service) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	return s.repo.GetSaleByID(ctx, id, storeID)
}

func (s *Service) ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
	return s.repo.GetAllSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, cashierID)
}

func (s *Service) GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
	return s.repo.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}

func (s *Service) StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
	return s.repo.StreamSalesExportCSV(ctx, w, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}

func (s *Service) GetNextInvoiceNumber(ctx context.Context) (string, error) {
	return s.repo.GetNextInvoiceNumber(ctx)
}

func (s *Service) GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *Service) GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error) {
	return s.repo.GetPaymentMethodByCode(ctx, code)
}

func (s *Service) ParkSale(ctx context.Context, sale *Sale, items []SaleItem, recalledSaleID *int) error {
	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("invalid quantity %d for product %d", item.Quantity, item.ProductID)
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if recalledSaleID != nil {
		_, err = tx.Exec(ctx, `
			UPDATE sales SET status = 'cancelled', updated_at = NOW()
			WHERE id = $1 AND status = 'recalled'
		`, *recalledSaleID)
		if err != nil {
			return fmt.Errorf("cancel previous recalled sale: %w", err)
		}
	}

	sale.Status = "parked"
	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	sale.Items = items
	return nil
}

func (s *Service) RecallSale(ctx context.Context, saleID int) (*Sale, error) {
	return s.repo.RecallSale(ctx, saleID)
}

func (s *Service) CancelParkedSale(ctx context.Context, saleID int) error {
	return s.repo.CancelParkedSale(ctx, saleID)
}

func (s *Service) ListParkedSales(ctx context.Context, cashierID int) ([]Sale, error) {
	return s.repo.GetParkedSales(ctx, cashierID)
}

func (s *Service) GetParkedSaleByID(ctx context.Context, saleID int, cashierID int) (*Sale, error) {
	return s.repo.GetParkedSaleByID(ctx, saleID, cashierID)
}

func (s *Service) CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []SaleItem, parkedSaleID *int) error {
	if parkedSaleID == nil {
		return s.CreateSale(ctx, sale, items)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the parked row to prevent concurrent checkouts
	var parkedStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM sales WHERE id = $1 AND status = 'recalled' FOR UPDATE
	`, *parkedSaleID).Scan(&parkedStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrParkedSaleNotRecalled
		}
		return fmt.Errorf("lock parked sale: %w", err)
	}

	if err := s.processSaleItems(ctx, tx, sale, items); err != nil {
		return err
	}

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return err
	}

	// Cancel the parked sale
	if err := s.repo.ConsumeParkedSale(ctx, tx, *parkedSaleID); err != nil {
		return fmt.Errorf("consume parked sale: %w", err)
	}

	if sale.ShiftID != nil {
		if err := s.repo.UpdateShiftTotals(ctx, tx, *sale.ShiftID, sale.TotalAmount, sale.PaymentMethod); err != nil {
			return fmt.Errorf("update shift totals: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	sale.Items = items
	_ = s.eventBus.Publish(ctx, "sale.created", sale)

	return nil
}
