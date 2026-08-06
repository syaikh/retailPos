package sale

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// ==================== CART SESSIONS ====================

// AtomicGetOrCreateOpenCart atomically returns the open cart for a cashier, creating one if absent.
func (r *Repository) AtomicGetOrCreateOpenCart(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
	var c CartSession
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		WITH existing AS (
			SELECT id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
			FROM cart_sessions
			WHERE cashier_id = $1 AND status = 'open'
		),
		new_cart AS (
			INSERT INTO cart_sessions (cashier_id, store_id, shift_id, customer_id)
			SELECT $1, $2, $3, $4
			WHERE NOT EXISTS (SELECT 1 FROM existing)
			RETURNING id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
		)
		SELECT id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
		FROM existing
		UNION ALL
		SELECT id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
		FROM new_cart
	`, cashierID, storeID, shiftID, customerID).Scan(
		&c.ID, &c.CashierID, &c.StoreID, &c.ShiftID, &c.CustomerID,
		&c.Status, &c.Subtotal, &c.Discount, &c.Tax, &c.TotalAmount,
		&c.ExpiredAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("atomic get or create open cart: %w", err)
	}
	c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &c, nil
}

func (r *Repository) CreateCartSession(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
	var cart CartSession
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO cart_sessions (cashier_id, store_id, shift_id, customer_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
	`, cashierID, storeID, shiftID, customerID).Scan(
		&cart.ID, &cart.CashierID, &cart.StoreID, &cart.ShiftID, &cart.CustomerID,
		&cart.Status, &cart.Subtotal, &cart.Discount, &cart.Tax, &cart.TotalAmount,
		&cart.ExpiredAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert cart session: %w", err)
	}
	cart.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	cart.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &cart, nil
}

func (r *Repository) GetCartSessionByID(ctx context.Context, cartID int) (*CartSession, error) {
	cart, err := r.scanCartSession(ctx, `
		SELECT id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
		FROM cart_sessions WHERE id = $1
	`, cartID)
	if err != nil {
		return nil, err
	}
	items, err := r.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, err
	}
	cart.Items = items
	return cart, nil
}

func (r *Repository) GetOpenCartByCashier(ctx context.Context, cashierID int) (*CartSession, error) {
	cart, err := r.scanCartSession(ctx, `
		SELECT id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
		FROM cart_sessions WHERE cashier_id = $1 AND status = 'open'
	`, cashierID)
	if err != nil {
		return nil, err
	}
	items, err := r.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	cart.Items = items
	return cart, nil
}

func (r *Repository) ListHeldCarts(ctx context.Context, cashierID int) ([]CartSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, cashier_id, store_id, shift_id, customer_id, status, subtotal, discount, tax, total_amount, expired_at, created_at, updated_at
		FROM cart_sessions WHERE cashier_id = $1 AND status = 'held'
		ORDER BY created_at DESC
	`, cashierID)
	if err != nil {
		return nil, fmt.Errorf("list held carts: %w", err)
	}
	defer rows.Close()

	var carts []CartSession
	var cartIDs []int
	for rows.Next() {
		var c CartSession
		var createdAt, updatedAt time.Time
		var expiredAt sql.NullTime
		var storeIDVal, shiftIDVal, customerIDVal sql.NullInt64
		if err := rows.Scan(&c.ID, &c.CashierID, &storeIDVal, &shiftIDVal, &customerIDVal,
			&c.Status, &c.Subtotal, &c.Discount, &c.Tax, &c.TotalAmount,
			&expiredAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan held cart: %w", err)
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			c.StoreID = &v
		}
		if shiftIDVal.Valid {
			v := int(shiftIDVal.Int64)
			c.ShiftID = &v
		}
		if customerIDVal.Valid {
			v := int(customerIDVal.Int64)
			c.CustomerID = &v
		}
		if expiredAt.Valid {
			s := expiredAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
			c.ExpiredAt = &s
		}
		c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		carts = append(carts, c)
		cartIDs = append(cartIDs, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(cartIDs) > 0 {
		cartMap := make(map[int]*CartSession, len(carts))
		for i := range carts {
			cartMap[carts[i].ID] = &carts[i]
		}
		itemRows, err := r.db.Query(ctx, `
			SELECT id, cart_session_id, product_id, product_name, quantity, unit_price, original_price, discount,
			       pricing_rule_id, pricing_rule_name, pricing_rule_type, pricing_type, cost, tax_class_id, tax_rate,
			       snapshot_created_at, subtotal, dpp_amount, tax_amount
			FROM cart_items WHERE cart_session_id = ANY($1)
			ORDER BY cart_session_id, id
		`, cartIDs)
		if err != nil {
			return nil, fmt.Errorf("load held cart items: %w", err)
		}
		defer itemRows.Close()
		for itemRows.Next() {
			item, err := scanCartItem(itemRows)
			if err != nil {
				return nil, err
			}
			if c, ok := cartMap[item.CartSessionID]; ok {
				c.Items = append(c.Items, *item)
			}
		}
	}

	return carts, nil
}

func (r *Repository) scanCartSession(ctx context.Context, query string, args ...interface{}) (*CartSession, error) {
	var c CartSession
	var createdAt, updatedAt time.Time
	var expiredAt sql.NullTime
	var storeIDVal, shiftIDVal, customerIDVal sql.NullInt64

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&c.ID, &c.CashierID, &storeIDVal, &shiftIDVal, &customerIDVal,
		&c.Status, &c.Subtotal, &c.Discount, &c.Tax, &c.TotalAmount,
		&expiredAt, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, err
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		c.StoreID = &v
	}
	if shiftIDVal.Valid {
		v := int(shiftIDVal.Int64)
		c.ShiftID = &v
	}
	if customerIDVal.Valid {
		v := int(customerIDVal.Int64)
		c.CustomerID = &v
	}
	if expiredAt.Valid {
		s := expiredAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		c.ExpiredAt = &s
	}
	c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &c, nil
}

// LockCartSession locks the cart row for update and returns its status and expiry.
func (r *Repository) LockCartSession(ctx context.Context, tx pgx.Tx, cartID int) (status string, expiredAt *time.Time, err error) {
	var expired sql.NullTime
	err = tx.QueryRow(ctx, `
		SELECT status, expired_at FROM cart_sessions WHERE id = $1 FOR UPDATE
	`, cartID).Scan(&status, &expired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrCartNotFound
		}
		return "", nil, err
	}
	if expired.Valid {
		expiredAt = &expired.Time
	}
	return status, expiredAt, nil
}

func (r *Repository) UpdateCartStatus(ctx context.Context, tx pgx.Tx, cartID int, status string, expiredAt *time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE cart_sessions SET status = $1, expired_at = $2, updated_at = NOW() WHERE id = $3
	`, status, expiredAt, cartID)
	if err != nil {
		return fmt.Errorf("update cart status: %w", err)
	}
	return nil
}

func (r *Repository) UpdateCartTotals(ctx context.Context, tx pgx.Tx, cartID, subtotal, discount, tax, totalAmount int) error {
	_, err := tx.Exec(ctx, `
		UPDATE cart_sessions SET subtotal = $1, discount = $2, tax = $3, total_amount = $4, updated_at = NOW() WHERE id = $5
	`, subtotal, discount, tax, totalAmount, cartID)
	if err != nil {
		return fmt.Errorf("update cart totals: %w", err)
	}
	return nil
}

func (r *Repository) UpdateCartCustomer(ctx context.Context, tx pgx.Tx, cartID int, customerID *int) error {
	_, err := tx.Exec(ctx, `
		UPDATE cart_sessions SET customer_id = $1, updated_at = NOW() WHERE id = $2
	`, customerID, cartID)
	if err != nil {
		return fmt.Errorf("update cart customer: %w", err)
	}
	return nil
}

// ==================== CART ITEMS ====================

func (r *Repository) InsertCartItem(ctx context.Context, tx pgx.Tx, item *CartItem) error {
	var createdAt, updatedAt, snapshotCreatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO cart_items (cart_session_id, product_id, product_name, quantity, unit_price, original_price, discount,
		       pricing_rule_id, pricing_rule_name, pricing_rule_type, pricing_type, cost, tax_class_id, tax_rate,
		       snapshot_created_at, subtotal, dpp_amount, tax_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, snapshot_created_at, created_at, updated_at
	`, item.CartSessionID, item.ProductID, item.ProductName, item.Quantity, item.UnitPrice, item.OriginalPrice,
		item.Discount, item.PricingRuleID, item.PricingRuleName, item.PricingRuleType, item.Type,
		item.Cost, item.TaxClassID, item.TaxRate, item.SnapshotCreatedAt, item.Subtotal, item.DPPAmount, item.TaxAmount).
		Scan(&item.ID, &snapshotCreatedAt, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert cart item: %w", err)
	}
	item.SnapshotCreatedAt = snapshotCreatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) GetCartItems(ctx context.Context, cartID int) ([]CartItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, cart_session_id, product_id, product_name, quantity, unit_price, original_price, discount,
		       pricing_rule_id, pricing_rule_name, pricing_rule_type, pricing_type, cost, tax_class_id, tax_rate,
		       snapshot_created_at, subtotal, dpp_amount, tax_amount
		FROM cart_items WHERE cart_session_id = $1
		ORDER BY id
	`, cartID)
	if err != nil {
		return nil, fmt.Errorf("get cart items: %w", err)
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		item, err := scanCartItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanCartItem(row pgx.Row) (*CartItem, error) {
	var item CartItem
	var snapshotCreatedAt time.Time
	var taxClassID sql.NullInt64
	var taxRate sql.NullFloat64
	var pricingRuleID sql.NullInt64
	var pricingRuleName, pricingRuleType, pricingType sql.NullString

	err := row.Scan(
		&item.ID, &item.CartSessionID, &item.ProductID, &item.ProductName,
		&item.Quantity, &item.UnitPrice, &item.OriginalPrice, &item.Discount,
		&pricingRuleID, &pricingRuleName, &pricingRuleType, &pricingType,
		&item.Cost, &taxClassID, &taxRate,
		&snapshotCreatedAt, &item.Subtotal, &item.DPPAmount, &item.TaxAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("scan cart item: %w", err)
	}
	if pricingRuleID.Valid {
		v := int(pricingRuleID.Int64)
		item.PricingRuleID = &v
	}
	if pricingRuleName.Valid {
		s := pricingRuleName.String
		item.PricingRuleName = &s
	}
	if pricingRuleType.Valid {
		s := pricingRuleType.String
		item.PricingRuleType = &s
	}
	if pricingType.Valid {
		s := pricingType.String
		item.Type = &s
	}
	if taxClassID.Valid {
		v := int(taxClassID.Int64)
		item.TaxClassID = &v
	}
	if taxRate.Valid {
		v := taxRate.Float64
		item.TaxRate = &v
	}
	item.SnapshotCreatedAt = snapshotCreatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &item, nil
}

func (r *Repository) UpdateCartItemQuantity(ctx context.Context, tx pgx.Tx, cartID, itemID, quantity, subtotal, dppAmount, taxAmount int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE cart_items SET quantity = $1, subtotal = $2, dpp_amount = $3, tax_amount = $4, updated_at = NOW()
		WHERE id = $5 AND cart_session_id = $6
	`, quantity, subtotal, dppAmount, taxAmount, itemID, cartID)
	if err != nil {
		return fmt.Errorf("update cart item quantity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCartItemNotFound
	}
	return nil
}

func (r *Repository) DeleteCartItem(ctx context.Context, tx pgx.Tx, cartID, itemID int) error {
	tag, err := tx.Exec(ctx, `
		DELETE FROM cart_items WHERE id = $1 AND cart_session_id = $2
	`, itemID, cartID)
	if err != nil {
		return fmt.Errorf("delete cart item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCartItemNotFound
	}
	return nil
}

// ==================== CART CHECKOUT ====================

// LoadCartItemsForCheckout loads all items of a cart for checkout.
func (r *Repository) LoadCartItemsForCheckout(ctx context.Context, tx pgx.Tx, cartID int) ([]CartItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, cart_session_id, product_id, product_name, quantity, unit_price, original_price, discount,
		       pricing_rule_id, pricing_rule_name, pricing_rule_type, pricing_type, cost, tax_class_id, tax_rate,
		       snapshot_created_at, subtotal, dpp_amount, tax_amount
		FROM cart_items WHERE cart_session_id = $1
		ORDER BY id
	`, cartID)
	if err != nil {
		return nil, fmt.Errorf("load cart items for checkout: %w", err)
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		item, err := scanCartItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
