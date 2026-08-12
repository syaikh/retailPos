package sale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/shared"
)

var ErrInsufficientStock = shared.ErrInsufficientStock
var ErrPriceMismatch = errors.New("price mismatch: client-submitted price does not match server price")
var ErrSaleNotFound = errors.New("sale not found")
var ErrParkedSaleNotRecalled = errors.New("parked sale not in recalled state")
var ErrPermissionDenied = errors.New("permission denied")
var ErrCheckoutProductNotFound = errors.New("checkout product not found")

type ProductPriceGetter interface {
	GetProductPrice(ctx context.Context, productID int) (int, error)
}

type ProductBatchPriceGetter interface {
	GetProductPrices(ctx context.Context, productIDs []int) (map[int]int, error)
}

type Repo interface {
	AtomicGetOrCreateOpenCart(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error)
	BeginTx(ctx context.Context) (pgx.Tx, error)
	CancelParkedSale(ctx context.Context, saleID int, ownerID, storeID *int) error
	ConsumeParkedSale(ctx context.Context, tx pgx.Tx, parkedSaleID int, ownerID, storeID *int) error
	CreateSale(ctx context.Context, tx pgx.Tx, sale *Sale, items []Item) error
	CreateSalePayments(ctx context.Context, tx pgx.Tx, saleID int, payments []Payment) error
	DeleteCartItem(ctx context.Context, tx pgx.Tx, cartID, itemID int) error
	GetAllActive(ctx context.Context) ([]PaymentMethod, error)
	GetAllSales(ctx context.Context, limit, offset int, search string, sortBy, sortDir, startDate, endDate string, storeID *int, paymentMethods string, minTotal, maxTotal, cashierID *int) ([]Sale, int, error)
	GetCartItems(ctx context.Context, cartID int) ([]CartItem, error)
	GetCartSessionByID(ctx context.Context, cartID int) (*CartSession, error)
	GetNextInvoiceNumber(ctx context.Context) (string, error)
	GetOpenCartByCashier(ctx context.Context, cashierID int) (*CartSession, error)
	GetParkedSaleByID(ctx context.Context, id int, ownerID, storeID *int) (*Sale, error)
	GetParkedSales(ctx context.Context, ownerID, storeID *int) ([]Sale, error)
	GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error)
	GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error)
	GetSalesForExport(ctx context.Context, search, startDate, endDate string, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error)
	InsertCartItem(ctx context.Context, tx pgx.Tx, item *CartItem) error
	ListHeldCarts(ctx context.Context, cashierID int) ([]CartSession, error)
	LoadCartItemsForCheckout(ctx context.Context, tx pgx.Tx, cartID int) ([]CartItem, error)
	LockCartSession(ctx context.Context, tx pgx.Tx, cartID int) (status string, expiredAt *time.Time, err error)
	RecallSale(ctx context.Context, saleID int, ownerID, storeID *int) (*Sale, error)
	StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error
	UpdateCartCustomer(ctx context.Context, tx pgx.Tx, cartID int, customerID *int) error
	UpdateCartItemQuantity(ctx context.Context, tx pgx.Tx, cartID, itemID, quantity, subtotal, dppAmount, taxAmount int) error
	UpdateCartStatus(ctx context.Context, tx pgx.Tx, cartID int, status string, expiredAt *time.Time) error
	UpdateCartTotals(ctx context.Context, tx pgx.Tx, cartID, subtotal, discount, tax, totalAmount int) error
}

type service struct {
	repo       Repo
	eventBus   shared.EventBus
	priceStore ProductPriceGetter
	resolver   PriceResolver
	stockStore StockDeducer
	shiftStore ShiftTotalUpdater
	cartConfig CartConfig
}

func NewService(repo Repo, eventBus shared.EventBus) Service {
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

// ResolveCheckoutPrices re-resolves server-authoritative unit prices (including
// engine-computed rule discounts) for a direct checkout. The pricing engine is
// the single source of truth for sale prices; the client's submitted prices are
// never trusted. Requiring a wired resolver fails fast at runtime, matching the
// stock/shift ports, instead of silently accepting client prices.
func (s *service) ResolveCheckoutPrices(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error) {
	if s.resolver == nil {
		return nil, errors.New("sale service: price resolver not wired; call SetPriceResolver")
	}
	snaps, err := s.resolver.ResolveSnapshotsBatch(ctx, items)
	if err != nil {
		if errors.Is(err, pricing.ErrProductNotFound) {
			return nil, fmt.Errorf("%w: %w", ErrCheckoutProductNotFound, err)
		}
		return nil, err
	}
	return snaps, nil
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

// toStockDeductItems reduces sale items to the minimal deduction contract,
// aggregating quantities per product so a duplicated line item can never cause
// overselling. The receipt/ledger item rows are unaffected; only the deduction
// list is deduplicated (P2-1 D2).
func toStockDeductItems(items []Item) []shared.StockDeductItem {
	byProduct := make(map[int]int, len(items))
	for _, item := range items {
		byProduct[item.ProductID] += item.Quantity
	}
	result := make([]shared.StockDeductItem, 0, len(byProduct))
	for productID, quantity := range byProduct {
		result = append(result, shared.StockDeductItem{ProductID: productID, Quantity: quantity})
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

func (s *service) ParkSale(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int, caller Caller) error {
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

	// Re-parking cancels the previously recalled sale. Cashiers may only cancel
	// their own recalled sale (owner-scoped); managers/elevated may cancel any
	// within their store, mirroring RecallSale/CancelParkedSale (P2-6 D4).
	if recalledSaleID != nil {
		args := []interface{}{*recalledSaleID}
		query := `
			UPDATE sales SET status = 'cancelled', updated_at = NOW()
			WHERE id = $1 AND status = 'recalled'
		`
		if ownerID := caller.ownerScope(); ownerID != nil {
			args = append(args, *ownerID)
			query += fmt.Sprintf(` AND cashier_id = $%d`, len(args))
		}
		if storeID := caller.storeScope(); storeID != nil {
			args = append(args, *storeID)
			query += fmt.Sprintf(` AND store_id = $%d`, len(args))
		}
		_, err = tx.Exec(ctx, query, args...)
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

// RecallSale marks a parked sale as recalled. Cashiers are restricted to their
// own sales (non-owner renders ErrSaleNotFound); managers and elevated roles
// may recall any cashier's parked sale (P2-6 D4).
func (s *service) RecallSale(ctx context.Context, saleID int, caller Caller) (*Sale, error) {
	return s.repo.RecallSale(ctx, saleID, caller.ownerScope(), caller.storeScope())
}

// CancelParkedSale voids a parked/recalled sale. Managers are denied outright
// (recall-only); cashiers are restricted to their own sales; elevated roles may
// cancel any (P2-6 D4).
func (s *service) CancelParkedSale(ctx context.Context, saleID int, caller Caller) error {
	if caller.IsManager() {
		return ErrPermissionDenied
	}
	return s.repo.CancelParkedSale(ctx, saleID, caller.ownerScope(), caller.storeScope())
}

func (s *service) ListParkedSales(ctx context.Context, caller Caller) ([]Sale, error) {
	return s.repo.GetParkedSales(ctx, caller.ownerScope(), caller.storeScope())
}

func (s *service) GetParkedSaleByID(ctx context.Context, saleID int, caller Caller) (*Sale, error) {
	return s.repo.GetParkedSaleByID(ctx, saleID, caller.ownerScope(), caller.storeScope())
}

// CreateSaleWithParkedSale creates a completed sale, optionally consuming a
// previously recalled parked sale. Managers have no blanket sale.create: a
// manager-initiated sale without a parked_sale_id is rejected (defense in depth
// with the SalePark-gated dedicated completion route). Cashiers consume only
// their own recalled sales (P2-6 D4).
func (s *service) CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest, caller Caller) error {
	if parkedSaleID == nil {
		if caller.IsManager() {
			return ErrPermissionDenied
		}
		return s.CreateSale(ctx, sale, items, payments)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var parkedStatus string
	args := []interface{}{*parkedSaleID}
	lockQuery := `SELECT status FROM sales WHERE id = $1 AND status = 'recalled'`
	if storeID := caller.storeScope(); storeID != nil {
		args = append(args, *storeID)
		lockQuery += fmt.Sprintf(` AND store_id = $%d`, len(args))
	}
	lockQuery += ` FOR UPDATE`
	err = tx.QueryRow(ctx, lockQuery, args...).Scan(&parkedStatus)
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
		if err := s.repo.ConsumeParkedSale(ctx, tx, *parkedSaleID, caller.ownerScope(), caller.storeScope()); err != nil {
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
