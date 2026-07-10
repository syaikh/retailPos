package sale

import (
	"context"
	"errors"
	"fmt"
)

var ErrInsufficientStock = errors.New("insufficient stock")
var ErrPriceMismatch = errors.New("price mismatch: client-submitted price does not match server price")
var ErrSaleNotFound = errors.New("sale not found")

type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}

type ProductPriceGetter interface {
	GetProductPrice(ctx context.Context, productID int) (int, error)
}

type Service struct {
	repo       *Repository
	eventBus   EventBus
	priceStore ProductPriceGetter
}

func NewService(repo *Repository, eventBus EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) SetPriceStore(ps ProductPriceGetter) {
	s.priceStore = ps
}

func (s *Service) CreateSale(ctx context.Context, sale *Sale, items []SaleItem) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("invalid quantity %d for product %d", item.Quantity, item.ProductID)
		}

		var stock int
		err := tx.QueryRow(ctx, `SELECT COALESCE(quantity, 0) FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, item.ProductID).Scan(&stock)
		if err != nil {
			return fmt.Errorf("check stock for product %d: %w", item.ProductID, err)
		}
		if stock < item.Quantity {
			return ErrInsufficientStock
		}

		if s.priceStore != nil {
			serverPrice, err := s.priceStore.GetProductPrice(ctx, item.ProductID)
			if err != nil {
				return fmt.Errorf("lookup price for product %d: %w", item.ProductID, err)
			}
			clientUnitPrice := item.UnitPrice
			if clientUnitPrice != serverPrice {
				return fmt.Errorf("%w: product %d, server=%d, client=%d", ErrPriceMismatch, item.ProductID, serverPrice, clientUnitPrice)
			}
			items[i].UnitPrice = serverPrice
			items[i].Subtotal = serverPrice * item.Quantity
		}
	}

	if s.priceStore != nil {
		sale.Subtotal = 0
		for _, item := range items {
			sale.Subtotal += item.Subtotal
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

	_ = s.eventBus.Publish(ctx, "sale.created", sale)

	return nil
}

func (s *Service) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	return s.repo.GetSaleByID(ctx, id, storeID)
}

func (s *Service) ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
	return s.repo.GetAllSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal)
}

func (s *Service) GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int) ([]SaleExportRow, error) {
	return s.repo.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal)
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
