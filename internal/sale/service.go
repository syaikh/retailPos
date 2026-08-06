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

var ErrInsufficientStock = shared.ErrInsufficientStock
var ErrPriceMismatch = errors.New("price mismatch: client-submitted price does not match server price")
var ErrSaleNotFound = errors.New("sale not found")
var ErrParkedSaleNotRecalled = errors.New("parked sale not in recalled state")

type ProductPriceGetter interface {
	GetProductPrice(ctx context.Context, productID int) (int, error)
}

type ProductBatchPriceGetter interface {
	GetProductPrices(ctx context.Context, productIDs []int) (map[int]int, error)
}

type service struct {
	repo        *Repository
	eventBus    shared.EventBus
	priceStore  ProductPriceGetter
	resolver    PriceResolver
	stockStore  StockDeducer
	shiftStore  ShiftTotalUpdater
	cartConfig  CartConfig
}

func NewService(repo *Repository, eventBus shared.EventBus) Service {
	return &service{
		repo:     repo,
		eventBus: eventBus,
	}
}

func (s *service) SetPriceStore(ps ProductPriceGetter) {
	s.priceStore = ps
}

func (s *service) SetPriceResolver(r PriceResolver) {
	s.resolver = r
}

func (s *service) SetStockDeducer(sd StockDeducer) {
	s.stockStore = sd
}

func (s *service) SetShiftTotalUpdater(st ShiftTotalUpdater) {
	s.shiftStore = st
}

// publishSaleCreated publishes the cross-module sale.created event as a DTO so
// downstream modules (report, websocket) never need to import this package.
func (s *service) publishSaleCreated(ctx context.Context, sale *Sale) {
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

func (s *service) processSaleItems(ctx context.Context, tx pgx.Tx, sale *Sale, items []Item) error {
	if err := s.finalizeSaleItems(ctx, tx, sale, items); err != nil {
		return err
	}
	return nil
}

// validateCheckoutItems validates the internal consistency of submitted sale items.
// It does NOT re-resolve prices — prices are treated as immutable snapshots.
func validateCheckoutItems(items []Item) error {
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

// toStockDeductItems reduces sale items to the minimal deduction contract.
func toStockDeductItems(items []Item) []shared.StockDeductItem {
	result := make([]shared.StockDeductItem, len(items))
	for i, item := range items {
		result[i] = shared.StockDeductItem{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	return result
}

func (s *service) validatePayments(ctx context.Context, totalAmount int, payments []CreatePaymentRequest) ([]Payment, error) {
	if len(payments) == 0 {
		return nil, ErrZeroPaymentAmount
	}
	if len(payments) > MaxPaymentsPerSale {
		return nil, ErrMaxPaymentsExceeded
	}

	var totalPaid int
	result := make([]Payment, 0, len(payments))
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
		result = append(result, Payment{
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

func (s *service) CreateSale(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
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

func (s *service) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	return s.repo.GetSaleByID(ctx, id, storeID)
}

func (s *service) ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
	return s.repo.GetAllSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, cashierID)
}

func (s *service) GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error) {
	return s.repo.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}

func (s *service) StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
	return s.repo.StreamSalesExportCSV(ctx, w, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}

func (s *service) GetNextInvoiceNumber(ctx context.Context) (string, error) {
	return s.repo.GetNextInvoiceNumber(ctx)
}

func (s *service) GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *service) GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error) {
	return s.repo.GetPaymentMethodByCode(ctx, code)
}

func (s *service) ParkSale(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int) error {
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

func (s *service) RecallSale(ctx context.Context, saleID int) (*Sale, error) {
	return s.repo.RecallSale(ctx, saleID)
}

func (s *service) CancelParkedSale(ctx context.Context, saleID int) error {
	return s.repo.CancelParkedSale(ctx, saleID)
}

func (s *service) ListParkedSales(ctx context.Context, cashierID int) ([]Sale, error) {
	return s.repo.GetParkedSales(ctx, cashierID)
}

func (s *service) GetParkedSaleByID(ctx context.Context, saleID int, cashierID int) (*Sale, error) {
	return s.repo.GetParkedSaleByID(ctx, saleID, cashierID)
}

func (s *service) CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
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

func (s *service) finalizeSaleCreation(ctx context.Context, tx pgx.Tx, sale *Sale, items []Item, payments []Payment) error {
	if err := s.repo.CreateSalePayments(ctx, tx, sale.ID, payments); err != nil {
		return err
	}
	if sale.ShiftID != nil {
		if s.shiftStore == nil {
			return errors.New("sale service: shift store not wired; call SetShiftTotalUpdater")
		}
		if err := s.shiftStore.UpdateShiftTotals(ctx, tx, shiftContribution(*sale.ShiftID, sale.TotalAmount, payments)); err != nil {
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

// shiftContribution computes the share of a completed sale accumulated onto its
// shift's running totals. The cash/non-cash split is derived from the sale's own
// payments; the shift module only accumulates what it is handed.
func shiftContribution(shiftID, totalAmount int, payments []Payment) shared.ShiftSaleContribution {
	c := shared.ShiftSaleContribution{ShiftID: shiftID, TotalAmount: totalAmount}
	for _, p := range payments {
		if strings.EqualFold(p.PaymentMethodCode, "CASH") {
			c.CashSales += p.Amount
		} else {
			c.NonCashSales += p.Amount
		}
	}
	return c
}
