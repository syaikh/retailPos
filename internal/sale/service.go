package sale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/events"
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
	resolver   PriceResolver
	cartConfig CartConfig
}

func NewService(repo *Repository, eventBus shared.EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) SetPriceStore(ps ProductPriceGetter) {
	s.priceStore = ps
}

func (s *Service) SetPriceResolver(r PriceResolver) {
	s.resolver = r
}

// publishSaleCreated publishes the cross-module sale.created event as a DTO so
// downstream modules (report, websocket) never need to import this package.
func (s *Service) publishSaleCreated(ctx context.Context, sale *Sale) {
	itemCount := 0
	if sale.Items != nil {
		itemCount = len(sale.Items)
	}
	_ = s.eventBus.Publish(ctx, events.TopicSaleCreated, &events.SaleCreated{
		ID:            sale.ID,
		InvoiceNumber: sale.InvoiceNumber,
		StoreID:       sale.StoreID,
		TotalAmount:   sale.TotalAmount,
		ItemCount:     itemCount,
	})
}

func (s *Service) processSaleItems(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem) error {
	if err := s.finalizeSaleItems(ctx, tx, sale, items); err != nil {
		return err
	}
	return nil
}

// validateCheckoutItems validates the internal consistency of submitted sale items.
// It does NOT re-resolve prices — prices are treated as immutable snapshots.
func validateCheckoutItems(items []SaleItem) error {
	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("invalid quantity %d for product %d", item.Quantity, item.ProductID)
		}
		if item.UnitPrice < 0 {
			return fmt.Errorf("invalid unit price %d for product %d", item.UnitPrice, item.ProductID)
		}
		if item.Subtotal != item.UnitPrice*item.Quantity {
			return fmt.Errorf("price mismatch: subtotal %d does not match unit price %d * quantity %d for product %d",
				item.Subtotal, item.UnitPrice, item.Quantity, item.ProductID)
		}
	}
	return nil
}

// deductStock checks and deducts stock for the given items.
func deductStock(ctx context.Context, tx pgx.Tx, items []SaleItem) error {
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

	return nil
}

func (s *Service) validatePayments(ctx context.Context, totalAmount int, payments []CreatePaymentRequest) ([]SalePayment, error) {
	if len(payments) == 0 {
		return nil, ErrZeroPaymentAmount
	}
	if len(payments) > MaxPaymentsPerSale {
		return nil, ErrMaxPaymentsExceeded
	}

	var totalPaid int
	result := make([]SalePayment, 0, len(payments))
	seenMethods := make(map[string]bool)
	cashCount := 0

	for _, p := range payments {
		if p.Amount <= 0 {
			return nil, ErrZeroPaymentAmount
		}

		pm, err := s.repo.GetPaymentMethodByCode(ctx, p.PaymentMethodCode)
		if err != nil {
			return nil, ErrInvalidPaymentMethod
		}
		if !pm.IsActive {
			return nil, ErrPaymentMethodInactive
		}

		methodUpper := strings.ToUpper(p.PaymentMethodCode)
		if strings.EqualFold(p.PaymentMethodCode, "CASH") {
			cashCount++
			if cashCount > 1 {
				return nil, ErrMultipleCashPayments
			}
		} else {
			if seenMethods[methodUpper] {
				return nil, ErrDuplicatePaymentMethod
			}
		}
		seenMethods[methodUpper] = true

		if pm.RequiresReference && strings.TrimSpace(p.ReferenceNumber) == "" {
			return nil, ErrPaymentReferenceRequired
		}

		totalPaid += p.Amount
		result = append(result, SalePayment{
			PaymentMethodID:   pm.ID,
			PaymentMethodCode: pm.Code,
			Amount:            p.Amount,
			ReferenceNumber:   p.ReferenceNumber,
		})
	}

	if totalPaid != totalAmount {
		return nil, fmt.Errorf("%w: paid %d, expected %d", ErrPaymentTotalMismatch, totalPaid, totalAmount)
	}

	return result, nil
}

func (s *Service) CreateSale(ctx context.Context, sale *Sale, items []SaleItem, payments []CreatePaymentRequest) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.processSaleItems(ctx, tx, sale, items); err != nil {
		return err
	}

	validatedPayments, err := s.validatePayments(ctx, sale.TotalAmount, payments)
	if err != nil {
		return err
	}

	codes := make([]string, len(validatedPayments))
	for i, p := range validatedPayments {
		codes[i] = p.PaymentMethodCode
	}
	sale.PaymentMethod = strings.Join(codes, ",")
	sale.Payments = validatedPayments

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return err
	}

	if err := s.finalizeSaleCreation(ctx, tx, sale, items, validatedPayments); err != nil {
		return err
	}

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

func (s *Service) CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []SaleItem, parkedSaleID *int, payments []CreatePaymentRequest) error {
	if parkedSaleID == nil {
		return s.CreateSale(ctx, sale, items, payments)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	validatedPayments, err := s.validatePayments(ctx, sale.TotalAmount, payments)
	if err != nil {
		return err
	}

	codes := make([]string, len(validatedPayments))
	for i, p := range validatedPayments {
		codes[i] = p.PaymentMethodCode
	}
	sale.PaymentMethod = strings.Join(codes, ",")
	sale.Payments = validatedPayments

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return err
	}

	if parkedSaleID != nil {
		if err := s.repo.ConsumeParkedSale(ctx, tx, *parkedSaleID); err != nil {
			return err
		}
	}

	if err := s.finalizeSaleCreation(ctx, tx, sale, items, validatedPayments); err != nil {
		return err
	}

	return nil
}

func (s *Service) finalizeSaleCreation(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem, payments []SalePayment) error {
	if err := s.repo.CreateSalePayments(ctx, tx, sale.ID, payments); err != nil {
		return err
	}
	if sale.ShiftID != nil {
		if err := s.repo.UpdateShiftTotals(ctx, tx, *sale.ShiftID, sale.TotalAmount, payments); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	sale.Items = items
	s.publishSaleCreated(ctx, sale)
	return nil
}
