package sale

import (
	"context"
	"errors"
	"fmt"

	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/shared"
)

var ErrInsufficientStock = errors.New("insufficient stock")
var ErrPriceMismatch = errors.New("price mismatch: client-submitted price does not match server price")
var ErrSaleNotFound = errors.New("sale not found")

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

func (s *Service) CreateSale(ctx context.Context, sale *Sale, items []SaleItem) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

			clientUnitPrice := items[i].UnitPrice
			if clientUnitPrice != r.UnitPrice {
				return fmt.Errorf("%w: product %d, server=%d, client=%d", ErrPriceMismatch, items[i].ProductID, r.UnitPrice, clientUnitPrice)
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
				pt := string(pricing.PricingTypeNormal)
				items[i].PricingType = &pt
			}

			sale.Subtotal += items[i].Subtotal
		}
		sale.TotalAmount = sale.Subtotal - sale.Discount
		if sale.TotalAmount < 0 {
			sale.TotalAmount = 0
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
			clientUnitPrice := item.UnitPrice
			if clientUnitPrice != serverPrice {
				return fmt.Errorf("%w: product %d, server=%d, client=%d", ErrPriceMismatch, item.ProductID, serverPrice, clientUnitPrice)
			}
			items[i].UnitPrice = serverPrice
			items[i].Subtotal = serverPrice * item.Quantity
			sale.Subtotal += items[i].Subtotal
		}
		sale.TotalAmount = sale.Subtotal - sale.Discount
		if sale.TotalAmount < 0 {
			sale.TotalAmount = 0
		}
	}

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return err
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

func (s *Service) ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
	return s.repo.GetAllSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal)
}

func (s *Service) GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
	return s.repo.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
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
