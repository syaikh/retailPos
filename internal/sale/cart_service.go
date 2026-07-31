package sale

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/shared"
)

// defaultCartHoldTTLHours is used when no TTL is configured via CART_HOLD_TTL_HOURS.
const defaultCartHoldTTLHours = 24

// CartConfig holds tunables for cart sessions.
type CartConfig struct {
	HoldTTLHours int
}

// SetCartConfig configures cart session behavior (e.g. hold TTL).
func (s *Service) SetCartConfig(cfg CartConfig) {
	s.cartConfig = cfg
}

// ==================== UC-01: CREATE / GET OPEN CART ====================

// ensureCartOwned verifies that the cart belongs to the given cashier.
func ensureCartOwned(cart *CartSession, cashierID int) error {
	if cart == nil || cart.CashierID != cashierID {
		return ErrCartNotOwned
	}
	return nil
}

// CreateOrGetOpenCart returns the open cart for the cashier, or creates one if none exists.
func (s *Service) CreateOrGetOpenCart(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
	cart, err := s.repo.AtomicGetOrCreateOpenCart(ctx, cashierID, storeID, shiftID, customerID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	cart.Items = items
	return cart, nil
}

// GetOpenCart returns the open cart for the cashier (no auto-create).
func (s *Service) GetOpenCart(ctx context.Context, cashierID int) (*CartSession, error) {
	cart, err := s.repo.GetOpenCartByCashier(ctx, cashierID)
	if err != nil {
		return nil, err
	}
	return cart, nil
}

// GetCartByID returns a cart session with its items, ensuring it belongs to the cashier.
func (s *Service) GetCartByID(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}
	return cart, nil
}

// ListHeldCarts returns all held cart sessions for a cashier (with items).
func (s *Service) ListHeldCarts(ctx context.Context, cashierID int) ([]CartSession, error) {
	return s.repo.ListHeldCarts(ctx, cashierID)
}

// UpdateCartCustomer sets the customer on an open cart, ensuring ownership.
func (s *Service) UpdateCartCustomer(ctx context.Context, cartID int, customerID *int, cashierID int) (*CartSession, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, _, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status != "open" {
		return nil, ErrCartNotOpen
	}

	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateCartCustomer(ctx, tx, cartID, customerID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetCartSessionByID(ctx, cartID)
}

// ==================== UC-02: ADD CART ITEM ====================

// AddCartItem resolves a server-side price snapshot and adds the item to the open cart.
func (s *Service) AddCartItem(ctx context.Context, cartID int, productID, quantity int, customerGroupID *int, cashierID int) (*CartSession, error) {
	if quantity <= 0 {
		return nil, ErrCartItemQuantity
	}
	if s.resolver == nil {
		return nil, errors.New("price resolver is not configured")
	}

	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, _, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status != "open" {
		return nil, ErrCartNotOpen
	}

	snapshots, err := s.resolver.ResolveSnapshotsBatch(ctx, []pricing.ResolveItem{{
		ProductID:       productID,
		Quantity:        quantity,
		CustomerGroupID: customerGroupID,
	}})
	if err != nil {
		return nil, fmt.Errorf("resolve price snapshot: %w", err)
	}
	snap := snapshots[0]

	item := &CartItem{
		CartSessionID:     cartID,
		ProductID:         snap.ProductID,
		ProductName:       snap.ProductName,
		Quantity:          quantity,
		UnitPrice:         snap.UnitPrice,
		OriginalPrice:     snap.OriginalPrice,
		Discount:          snap.Discount,
		PricingRuleID:     nil,
		PricingRuleName:   nil,
		PricingRuleType:   nil,
		PricingType:       stringPtr(string(snap.PricingType)),
		Cost:              snap.Cost,
		TaxClassID:        snap.TaxClassID,
		TaxRate:           snap.TaxRate,
		SnapshotCreatedAt: snap.SnapshotAt.In(shared.JakartaLocation()).Format(time.RFC3339),
	}
	if snap.Rule != nil {
		ruleID := snap.Rule.ID
		ruleName := snap.Rule.Name
		ruleType := string(snap.Rule.PricingType)
		item.PricingRuleID = &ruleID
		item.PricingRuleName = &ruleName
		item.PricingRuleType = &ruleType
	}
	item.Subtotal, item.DPPAmount, item.TaxAmount = computeLineTotals(quantity, snap.UnitPrice, snap.TaxRate)

	if err := s.repo.InsertCartItem(ctx, tx, item); err != nil {
		return nil, err
	}

	if err := s.recalculateCartTotals(ctx, tx, cartID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetCartSessionByID(ctx, cartID)
}

// ==================== UC-03: UPDATE QUANTITY ====================

// UpdateCartItemQuantity changes the quantity of an existing cart item.
// The unit price snapshot is preserved; only line totals are recomputed.
func (s *Service) UpdateCartItemQuantity(ctx context.Context, cartID, itemID, quantity int, cashierID int) (*CartSession, error) {
	if quantity <= 0 {
		return nil, ErrCartItemQuantity
	}

	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, _, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status != "open" {
		return nil, ErrCartNotOpen
	}

	items, err := s.repo.LoadCartItemsForCheckout(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	var target *CartItem
	for i := range items {
		if items[i].ID == itemID {
			target = &items[i]
			break
		}
	}
	if target == nil {
		return nil, ErrCartItemNotFound
	}

	subtotal, dpp, tax := computeLineTotals(quantity, target.UnitPrice, target.TaxRate)
	if err := s.repo.UpdateCartItemQuantity(ctx, tx, cartID, itemID, quantity, subtotal, dpp, tax); err != nil {
		return nil, err
	}

	if err := s.recalculateCartTotalsFromItems(ctx, tx, cartID, items); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetCartSessionByID(ctx, cartID)
}

// ==================== UC-04: REMOVE CART ITEM ====================

// RemoveCartItem removes an item from the open cart.
func (s *Service) RemoveCartItem(ctx context.Context, cartID, itemID int, cashierID int) (*CartSession, error) {
	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, _, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status != "open" {
		return nil, ErrCartNotOpen
	}

	if err := s.repo.DeleteCartItem(ctx, tx, cartID, itemID); err != nil {
		return nil, err
	}

	if err := s.recalculateCartTotals(ctx, tx, cartID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetCartSessionByID(ctx, cartID)
}

// ==================== UC-05: HOLD CART ====================

// HoldCart parks the open cart, setting an expiry based on the configured TTL.
func (s *Service) HoldCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
	ttlHours := s.cartConfig.HoldTTLHours
	if ttlHours <= 0 {
		ttlHours = defaultCartHoldTTLHours
	}
	expiredAt := time.Now().In(shared.JakartaLocation()).Add(time.Duration(ttlHours) * time.Hour)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, _, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status != "open" {
		return nil, ErrCartNotOpen
	}

	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateCartStatus(ctx, tx, cartID, "held", &expiredAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetCartSessionByID(ctx, cartID)
}

// ==================== UC-06: RESUME CART ====================

// ResumeCart re-opens a held cart if it has not expired.
func (s *Service) ResumeCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, expiredAt, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status != "held" {
		return nil, ErrCartNotOpen
	}
	if expiredAt != nil && expiredAt.Before(time.Now()) {
		return nil, ErrCartExpired
	}

	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateCartStatus(ctx, tx, cartID, "open", nil); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetCartSessionByID(ctx, cartID)
}

// ==================== UC-07: CHECKOUT CART ====================

// CheckoutCart converts the cart into a completed sale using the stored snapshots,
// deducts stock, records payments, and updates the shift totals atomically.
func (s *Service) CheckoutCart(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, expiredAt, err := s.repo.LockCartSession(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if status == "checked_out" || status == "cancelled" {
		return nil, ErrCartAlreadyCheckedOut
	}
	if status != "open" && status != "held" {
		return nil, ErrCartNotOpen
	}
	if expiredAt != nil && expiredAt.Before(time.Now()) {
		return nil, ErrCartExpired
	}

	cart, err := s.repo.GetCartSessionByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if err := ensureCartOwned(cart, cashierID); err != nil {
		return nil, err
	}
	if len(cart.Items) == 0 {
		return nil, errors.New("cart is empty")
	}

	// Build sale items from the immutable snapshots.
	items := make([]SaleItem, 0, len(cart.Items))
	var subtotal, dppTotal, taxTotal int
	for _, ci := range cart.Items {
		if ci.UnitPrice != ci.Subtotal/ci.Quantity {
			return nil, fmt.Errorf("%w: product %d", ErrPriceMismatch, ci.ProductID)
		}
		items = append(items, ci.ToSaleItem())
		subtotal += ci.Subtotal
		dppTotal += ci.DPPAmount
		taxTotal += ci.TaxAmount
	}

	invoiceNumber, err := s.repo.GetNextInvoiceNumber(ctx)
	if err != nil {
		return nil, err
	}

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cart.CashierID,
		ShiftID:       cart.ShiftID,
		CustomerID:    cart.CustomerID,
		StoreID:       cart.StoreID,
		Subtotal:      subtotal,
		Discount:      cart.Discount,
		Tax:           taxTotal,
		TotalAmount:   subtotal - cart.Discount,
		Status:        "completed",
	}
	if sale.TotalAmount < 0 {
		sale.TotalAmount = 0
	}

	if err := s.finalizeSaleItems(ctx, tx, sale, items); err != nil {
		return nil, err
	}

	validatedPayments, err := s.validatePayments(ctx, sale.TotalAmount, payments)
	if err != nil {
		return nil, err
	}

	codes := make([]string, len(validatedPayments))
	for i, p := range validatedPayments {
		codes[i] = p.PaymentMethodCode
	}
	sale.PaymentMethod = strings.Join(codes, ",")
	sale.Payments = validatedPayments

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		return nil, err
	}

	if err := s.repo.CreateSalePayments(ctx, tx, sale.ID, validatedPayments); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateCartStatus(ctx, tx, cartID, "checked_out", nil); err != nil {
		return nil, err
	}

	if sale.ShiftID != nil {
		if err := s.repo.UpdateShiftTotals(ctx, tx, *sale.ShiftID, sale.TotalAmount, validatedPayments); err != nil {
			return nil, fmt.Errorf("update shift totals: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	sale.Items = items
	_ = s.eventBus.Publish(ctx, "sale.created", sale)
	_ = s.eventBus.Publish(ctx, "cart.checked_out", &CartCheckedOutEvent{
		CartID:   cartID,
		SaleID:   sale.ID,
		CashierID: cart.CashierID,
	})

	return sale, nil
}

// finalizeSaleItems validates checkout items and deducts stock, then normalizes the sale total.
// This helper is shared by CreateSale/CreateSaleWithParkedSale and CheckoutCart so the
// validation/stock-deduction order cannot diverge between code paths.
func (s *Service) finalizeSaleItems(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem) error {
	if err := validateCheckoutItems(items); err != nil {
		return err
	}
	if err := deductStock(ctx, tx, items); err != nil {
		return err
	}

	sale.TotalAmount = sale.Subtotal - sale.Discount
	if sale.TotalAmount < 0 {
		sale.TotalAmount = 0
	}

	return nil
}

// ==================== INTERNAL HELPERS ====================

// recalculateCartTotals recomputes cart subtotal, tax, and total from its items.
func (s *Service) recalculateCartTotals(ctx context.Context, tx pgx.Tx, cartID int) error {
	items, err := s.repo.LoadCartItemsForCheckout(ctx, tx, cartID)
	if err != nil {
		return err
	}
	return s.recalculateCartTotalsFromItems(ctx, tx, cartID, items)
}

// recalculateCartTotalsFromItems computes totals from already-loaded items, avoiding
// a second bulk read when the caller already has the item list in hand.
func (s *Service) recalculateCartTotalsFromItems(ctx context.Context, tx pgx.Tx, cartID int, items []CartItem) error {
	var subtotal, tax int
	for _, item := range items {
		subtotal += item.Subtotal
		tax += item.TaxAmount
	}
	discount := 0
	total := subtotal - discount
	if total < 0 {
		total = 0
	}
	return s.repo.UpdateCartTotals(ctx, tx, cartID, subtotal, discount, tax, total)
}

// computeLineTotals computes subtotal, DPP, and tax for a single line.
func computeLineTotals(quantity, unitPrice int, taxRate *float64) (subtotal, dpp, tax int) {
	subtotal = quantity * unitPrice
	if taxRate == nil || *taxRate <= 0 {
		return subtotal, subtotal, 0
	}
	dpp = int(math.Round(float64(subtotal) * 100.0 / (100.0 + *taxRate)))
	tax = subtotal - dpp
	return subtotal, dpp, tax
}

func stringPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }
