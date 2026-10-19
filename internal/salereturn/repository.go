package salereturn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// Repository owns all access to the salereturn tables (strict module
// ownership, internal/archtest TestModuleSQLTableOwnership). Mutation methods
// take pgx.Tx (the Unit of Work); read-only list methods run on the pool.
// shared.DBPool parameters accept either, because pgx.Tx satisfies it.
type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// BeginTx starts a Unit of Work for the service.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// NextReturnNumber mints a RET-<year>-<seq> number from return_seq, mirroring
// the sale module's invoice numbering.
func (r *Repository) NextReturnNumber(ctx context.Context, q shared.DBPool) (string, error) {
	now := time.Now().In(shared.JakartaLocation())
	var seq int
	if err := q.QueryRow(ctx, `SELECT nextval('return_seq')`).Scan(&seq); err != nil {
		return "", fmt.Errorf("failed to get next return sequence: %w", err)
	}
	return fmt.Sprintf("RET-%d-%06d", now.Year(), seq), nil
}

// InsertReturn inserts the header and returns the generated id.
func (r *Repository) InsertReturn(ctx context.Context, tx pgx.Tx, ret *Return) error {
	err := tx.QueryRow(ctx, `
		INSERT INTO sale_returns (
			return_number, original_sale_id, cashier_id, store_id, customer_id,
			shift_id, return_type, reason, condition, notes, subtotal, discount,
			restocking_fee, tax_reversal, total_refund_amount, status, change_due,
			expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id, created_at, updated_at
	`,
		ret.ReturnNumber, ret.OriginalSaleID, ret.CashierID, ret.StoreID,
		ret.CustomerID, ret.ShiftID, ret.ReturnType, ret.Reason, ret.Condition,
		ret.Notes, ret.Subtotal, ret.Discount, ret.RestockingFee, ret.TaxReversal,
		ret.TotalRefundAmount, ret.Status, ret.ChangeDue, ret.ExpiresAt,
	).Scan(&ret.ID, &ret.CreatedAt, &ret.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert sale return: %w", err)
	}
	return nil
}

// InsertItems inserts the return's lines and their replacement rows. On the
// approval path this REPLACES the proposal rows written at creation time
// (ClearItems first).
func (r *Repository) InsertItems(ctx context.Context, tx pgx.Tx, returnID int, items []ReturnItem) error {
	for i := range items {
		it := &items[i]
		err := tx.QueryRow(ctx, `
			INSERT INTO sale_return_items (
				sale_return_id, sale_item_id, product_id, product_name, quantity,
				unit_price, subtotal, dpp_amount, tax_amount, tax_rate, condition,
				outcome, notes
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			RETURNING id
		`,
			returnID, it.SaleItemID, it.ProductID, it.ProductName, it.Quantity,
			it.UnitPrice, it.Subtotal, it.DppAmount, it.TaxAmount, it.TaxRate,
			it.Condition, it.Outcome, it.Notes,
		).Scan(&it.ID)
		if err != nil {
			return fmt.Errorf("failed to insert sale return item: %w", err)
		}
		for j := range it.Replacements {
			rep := &it.Replacements[j]
			rep.ItemID = it.ID
			err := tx.QueryRow(ctx, `
				INSERT INTO sale_return_replacements (sale_return_id, sale_return_item_id, product_id, quantity)
				VALUES ($1,$2,$3,$4)
				RETURNING id
			`, returnID, it.ID, rep.ProductID, rep.Quantity).Scan(&rep.ID)
			if err != nil {
				return fmt.Errorf("failed to insert sale return replacement: %w", err)
			}
		}
	}
	return nil
}

// ClearItems deletes a return's lines (cascading its replacement rows) so an
// approved return can be re-written from the re-validated proposal.
func (r *Repository) ClearItems(ctx context.Context, tx pgx.Tx, returnID int) error {
	_, err := tx.Exec(ctx, `DELETE FROM sale_return_items WHERE sale_return_id = $1`, returnID)
	if err != nil {
		return fmt.Errorf("failed to clear sale return items: %w", err)
	}
	return nil
}

// InsertPayments writes the refund split rows (refund-outcome lines only).
func (r *Repository) InsertPayments(ctx context.Context, tx pgx.Tx, returnID int, payments []ReturnPayment) error {
	for i := range payments {
		p := &payments[i]
		err := tx.QueryRow(ctx, `
			INSERT INTO sale_return_payments (sale_return_id, payment_method_id, payment_method_code, amount)
			VALUES ($1,$2,$3,$4)
			RETURNING id
		`, returnID, p.PaymentMethodID, p.PaymentMethodCode, p.Amount).Scan(&p.ID)
		if err != nil {
			return fmt.Errorf("failed to insert sale return payment: %w", err)
		}
	}
	return nil
}

// InsertApproval records the pending approval row for an
// approval-requiring return.
func (r *Repository) InsertApproval(ctx context.Context, tx pgx.Tx, returnID, requestedBy int, reason *string, expiresAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO return_approvals (sale_return_id, requested_by, status, reason, expires_at)
		VALUES ($1,$2,'pending',$3,$4)
	`, returnID, requestedBy, reason, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert return approval: %w", err)
	}
	return nil
}

// ApproveApproval flips the pending approval row to approved.
func (r *Repository) ApproveApproval(ctx context.Context, tx pgx.Tx, returnID, approverID int, note *string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE return_approvals
		SET status = 'approved', approved_by = $2, reason = COALESCE($3, reason), updated_at = now()
		WHERE sale_return_id = $1 AND status = 'pending'
	`, returnID, approverID, note)
	if err != nil {
		return fmt.Errorf("failed to approve return approval: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: no pending approval row", ErrReturnNotPending)
	}
	return nil
}

// RejectApproval marks the approval rejected and the return cancelled.
func (r *Repository) RejectApproval(ctx context.Context, tx pgx.Tx, returnID, approverID int, rejectedReason string) error {
	_, err := tx.Exec(ctx, `
		UPDATE return_approvals
		SET status = 'rejected', approved_by = $2, rejected_reason = $3, updated_at = now()
		WHERE sale_return_id = $1 AND status = 'pending'
	`, returnID, approverID, rejectedReason)
	if err != nil {
		return fmt.Errorf("failed to reject return approval: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE sale_returns SET status = 'cancelled', updated_at = now() WHERE id = $1
	`, returnID)
	if err != nil {
		return fmt.Errorf("failed to cancel return: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReturnNotFound
	}
	return nil
}

// ExpireReturn lazily cancels a return whose approval window elapsed (PRD
// §12.3 #16). It is only ever called from a committed Unit of Work so the
// flip survives the ErrApprovalExpired returned to the caller.
func (r *Repository) ExpireReturn(ctx context.Context, tx pgx.Tx, returnID int) error {
	_, err := tx.Exec(ctx, `
		UPDATE return_approvals
		SET status = 'expired', updated_at = now()
		WHERE sale_return_id = $1 AND status = 'pending'
	`, returnID)
	if err != nil {
		return fmt.Errorf("failed to expire return approval: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE sale_returns SET status = 'expired', updated_at = now() WHERE id = $1
	`, returnID)
	if err != nil {
		return fmt.Errorf("failed to expire return: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReturnNotFound
	}
	return nil
}

const returnHeaderColumns = `
	return_number, original_sale_id, cashier_id, store_id, customer_id, shift_id,
	return_type, reason, condition, notes, subtotal, discount, restocking_fee,
	tax_reversal, total_refund_amount, status, change_due, created_at, updated_at,
	expires_at
`

func scanReturnHeader(row pgx.Row) (*Return, error) {
	var ret Return
	err := row.Scan(
		&ret.ReturnNumber, &ret.OriginalSaleID, &ret.CashierID, &ret.StoreID,
		&ret.CustomerID, &ret.ShiftID, &ret.ReturnType, &ret.Reason,
		&ret.Condition, &ret.Notes, &ret.Subtotal, &ret.Discount,
		&ret.RestockingFee, &ret.TaxReversal, &ret.TotalRefundAmount,
		&ret.Status, &ret.ChangeDue, &ret.CreatedAt, &ret.UpdatedAt, &ret.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// GetReturnForUpdate locks a return header for the approval/rejection Unit of
// Work and loads its approval row.
func (r *Repository) GetReturnForUpdate(ctx context.Context, tx pgx.Tx, id int) (*Return, error) {
	ret, err := scanReturnHeader(tx.QueryRow(ctx, `
		SELECT `+returnHeaderColumns+`
		FROM sale_returns
		WHERE id = $1
		FOR UPDATE
	`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReturnNotFound
		}
		return nil, fmt.Errorf("failed to get sale return: %w", err)
	}
	ret.ID = id
	if err := r.loadDetail(ctx, tx, ret); err != nil {
		return nil, err
	}
	approval, err := r.getApproval(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	ret.Approval = approval
	return ret, nil
}

func (r *Repository) getApproval(ctx context.Context, q shared.DBPool, returnID int) (*Approval, error) {
	var a Approval
	err := q.QueryRow(ctx, `
		SELECT id, requested_by, approved_by, status, reason, rejected_reason, expires_at
		FROM return_approvals
		WHERE sale_return_id = $1
	`, returnID).Scan(&a.ID, &a.RequestedBy, &a.ApprovedBy, &a.Status, &a.Reason, &a.RejectedReason, &a.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get return approval: %w", err)
	}
	return &a, nil
}

// GetReturn loads one return with lines, replacements, payments and approval,
// scoped to a store (nil = unscoped, superadmin).
func (r *Repository) GetReturn(ctx context.Context, storeID *int, id int) (*Return, error) {
	ret, err := scanReturnHeader(r.db.QueryRow(ctx, `
		SELECT `+returnHeaderColumns+`
		FROM sale_returns
		WHERE id = $1 AND ($2::int IS NULL OR store_id = $2)
	`, id, storeID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReturnNotFound
		}
		return nil, fmt.Errorf("failed to get sale return: %w", err)
	}
	ret.ID = id
	if err := r.loadDetail(ctx, r.db, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// loadDetail fills lines, replacements, payments and approval — on the pool
// for reads, or on the caller's transaction for the approval Unit of Work.
func (r *Repository) loadDetail(ctx context.Context, q shared.DBPool, ret *Return) error {
	rows, err := q.Query(ctx, `
		SELECT id, sale_item_id, product_id, product_name, quantity, unit_price,
		       subtotal, dpp_amount, tax_amount, tax_rate, condition, outcome, notes
		FROM sale_return_items
		WHERE sale_return_id = $1
		ORDER BY id
	`, ret.ID)
	if err != nil {
		return fmt.Errorf("failed to list return items: %w", err)
	}
	defer rows.Close()
	items := make([]ReturnItem, 0)
	for rows.Next() {
		var it ReturnItem
		if err := rows.Scan(&it.ID, &it.SaleItemID, &it.ProductID, &it.ProductName,
			&it.Quantity, &it.UnitPrice, &it.Subtotal, &it.DppAmount, &it.TaxAmount,
			&it.TaxRate, &it.Condition, &it.Outcome, &it.Notes); err != nil {
			return fmt.Errorf("failed to scan return item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate return items: %w", err)
	}

	if len(items) > 0 {
		repRows, err := q.Query(ctx, `
			SELECT id, sale_return_item_id, product_id, quantity
			FROM sale_return_replacements
			WHERE sale_return_id = $1
			ORDER BY id
		`, ret.ID)
		if err != nil {
			return fmt.Errorf("failed to list return replacements: %w", err)
		}
		defer repRows.Close()
		byItem := make(map[int]int, len(items))
		for i := range items {
			byItem[items[i].ID] = i
		}
		for repRows.Next() {
			var rep Replacement
			if err := repRows.Scan(&rep.ID, &rep.ItemID, &rep.ProductID, &rep.Quantity); err != nil {
				return fmt.Errorf("failed to scan return replacement: %w", err)
			}
			if i, ok := byItem[rep.ItemID]; ok {
				items[i].Replacements = append(items[i].Replacements, rep)
			}
		}
		if err := repRows.Err(); err != nil {
			return fmt.Errorf("failed to iterate return replacements: %w", err)
		}
	}
	ret.Items = items

	payRows, err := q.Query(ctx, `
		SELECT id, payment_method_id, payment_method_code, amount
		FROM sale_return_payments
		WHERE sale_return_id = $1
		ORDER BY id
	`, ret.ID)
	if err != nil {
		return fmt.Errorf("failed to list return payments: %w", err)
	}
	defer payRows.Close()
	payments := make([]ReturnPayment, 0)
	for payRows.Next() {
		var p ReturnPayment
		if err := payRows.Scan(&p.ID, &p.PaymentMethodID, &p.PaymentMethodCode, &p.Amount); err != nil {
			return fmt.Errorf("failed to scan return payment: %w", err)
		}
		payments = append(payments, p)
	}
	if err := payRows.Err(); err != nil {
		return fmt.Errorf("failed to iterate return payments: %w", err)
	}
	ret.Payments = payments

	approval, err := r.getApproval(ctx, q, ret.ID)
	if err != nil {
		return err
	}
	ret.Approval = approval
	return nil
}

// ListReturns lists return headers, optionally narrowed to one original sale
// or one status. Totals come from the same filtered query.
func (r *Repository) ListReturns(ctx context.Context, storeID *int, saleID *int, status *string, limit, offset int) ([]Return, int, error) {
	where := `WHERE ($1::int IS NULL OR store_id = $1)
	          AND ($2::int IS NULL OR original_sale_id = $2)
	          AND ($3::text IS NULL OR status = $3)`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sale_returns `+where, storeID, saleID, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count returns: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, `+returnHeaderColumns+`
		FROM sale_returns `+where+`
		ORDER BY created_at DESC, id DESC
		LIMIT $4 OFFSET $5
	`, storeID, saleID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list returns: %w", err)
	}
	defer rows.Close()

	out := make([]Return, 0)
	for rows.Next() {
		var ret Return
		if err := rows.Scan(&ret.ID,
			&ret.ReturnNumber, &ret.OriginalSaleID, &ret.CashierID, &ret.StoreID,
			&ret.CustomerID, &ret.ShiftID, &ret.ReturnType, &ret.Reason,
			&ret.Condition, &ret.Notes, &ret.Subtotal, &ret.Discount,
			&ret.RestockingFee, &ret.TaxReversal, &ret.TotalRefundAmount,
			&ret.Status, &ret.ChangeDue, &ret.CreatedAt, &ret.UpdatedAt, &ret.ExpiresAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan return: %w", err)
		}
		out = append(out, ret)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate returns: %w", err)
	}
	return out, total, nil
}

// ListPending lists actionable pending approvals (the expired-but-uncollected
// rows are filtered out; a late approval attempt flips them via
// ExpireReturn). Oldest first: first requested, first reviewed.
func (r *Repository) ListPending(ctx context.Context, storeID *int, limit, offset int) ([]Return, int, error) {
	where := `WHERE status = 'pending_approval'
	          AND ($1::int IS NULL OR store_id = $1)
	          AND (expires_at IS NULL OR expires_at > now())`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sale_returns `+where, storeID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count pending returns: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, `+returnHeaderColumns+`
		FROM sale_returns `+where+`
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`, storeID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pending returns: %w", err)
	}
	defer rows.Close()

	out := make([]Return, 0)
	for rows.Next() {
		var ret Return
		if err := rows.Scan(&ret.ID,
			&ret.ReturnNumber, &ret.OriginalSaleID, &ret.CashierID, &ret.StoreID,
			&ret.CustomerID, &ret.ShiftID, &ret.ReturnType, &ret.Reason,
			&ret.Condition, &ret.Notes, &ret.Subtotal, &ret.Discount,
			&ret.RestockingFee, &ret.TaxReversal, &ret.TotalRefundAmount,
			&ret.Status, &ret.ChangeDue, &ret.CreatedAt, &ret.UpdatedAt, &ret.ExpiresAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan pending return: %w", err)
		}
		out = append(out, ret)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate pending returns: %w", err)
	}
	return out, total, nil
}

// ReturnedQtyBySale sums the already-returned quantity per sale_item of a
// sale, for the over-return guard (PRD §12.2 #14). Cancelled returns and
// expired returns do not consume returnable quantity — both the pendings
// whose window has elapsed and the rows already flipped to 'expired' (the
// customer may re-return after an expiry); excludeReturnID removes the return
// being approved so its own proposal rows are not counted against themselves
// (0 at creation time — no rows exist yet).
func (r *Repository) ReturnedQtyBySale(ctx context.Context, q shared.DBPool, saleID, excludeReturnID int) (map[int]int, error) {
	rows, err := q.Query(ctx, `
		SELECT sri.sale_item_id, SUM(sri.quantity)
		FROM sale_return_items sri
		JOIN sale_returns sr ON sr.id = sri.sale_return_id
		WHERE sr.original_sale_id = $1
		  AND sr.id <> $2
		  AND sr.status NOT IN ('cancelled', 'expired')
		  AND NOT (sr.status = 'pending_approval' AND sr.expires_at IS NOT NULL AND sr.expires_at < now())
		GROUP BY sri.sale_item_id
	`, saleID, excludeReturnID)
	if err != nil {
		return nil, fmt.Errorf("failed to sum returned quantities: %w", err)
	}
	defer rows.Close()

	out := make(map[int]int)
	for rows.Next() {
		var itemID, qty int
		if err := rows.Scan(&itemID, &qty); err != nil {
			return nil, fmt.Errorf("failed to scan returned quantity: %w", err)
		}
		out[itemID] = qty
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate returned quantities: %w", err)
	}
	return out, nil
}

// ReturnTotalsForShift aggregates the completed refunds paid out by one
// shift: the cash drawer payout (cash payment rows) and the total refund.
// It is the query behind the shift module's close-time netting
// (ShiftReturnTotalsProvider). Pending approvals are excluded — the money has
// not left the drawer yet.
func (r *Repository) ReturnTotalsForShift(ctx context.Context, q shared.DBPool, shiftID int) (shared.ShiftReturnTotals, error) {
	var totals shared.ShiftReturnTotals
	err := q.QueryRow(ctx, `
		WITH cash AS (
			SELECT COALESCE(SUM(sp.amount) FILTER (WHERE LOWER(sp.payment_method_code) = 'cash'), 0) AS amount
			FROM sale_return_payments sp
			JOIN sale_returns sr ON sr.id = sp.sale_return_id
			WHERE sr.shift_id = $1 AND sr.status = 'completed'
		), total AS (
			SELECT COALESCE(SUM(sr.total_refund_amount), 0) AS amount
			FROM sale_returns sr
			WHERE sr.shift_id = $1 AND sr.status = 'completed'
		)
		SELECT cash.amount, total.amount - cash.amount, total.amount
		FROM cash, total
	`, shiftID).Scan(&totals.CashRefund, &totals.NonCashRefund, &totals.TotalRefund)
	if err != nil {
		return shared.ShiftReturnTotals{}, fmt.Errorf("failed to aggregate shift return totals: %w", err)
	}
	return totals, nil
}
