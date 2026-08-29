package consignment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Repository struct {
	db                  shared.DBPool
	stockAdjuster       StockAdjuster
	supplierStore       SupplierStore
	productMetaProvider ProductMetaProvider
	usernameProvider    UsernameProvider
	paymentMethods      PaymentMethodProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetStockAdjuster(p StockAdjuster) {
	r.stockAdjuster = p
}

func (r *Repository) SetSupplierStore(p SupplierStore) {
	r.supplierStore = p
}

func (r *Repository) SetProductMetaProvider(p ProductMetaProvider) {
	r.productMetaProvider = p
}

func (r *Repository) SetUsernameProvider(p UsernameProvider) {
	r.usernameProvider = p
}

func (r *Repository) SetPaymentMethods(p PaymentMethodProvider) {
	r.paymentMethods = p
}

func (r *Repository) stockAdjusterOrPanic() StockAdjuster {
	if r.stockAdjuster == nil {
		panic("consignment.Repository: StockAdjuster not wired (SetStockAdjuster)")
	}
	return r.stockAdjuster
}

func (r *Repository) supplierStoreOrPanic() SupplierStore {
	if r.supplierStore == nil {
		panic("consignment.Repository: SupplierStore not wired (SetSupplierStore)")
	}
	return r.supplierStore
}

func (r *Repository) paymentMethodsOrPanic() PaymentMethodProvider {
	if r.paymentMethods == nil {
		panic("consignment.Repository: PaymentMethodProvider not wired (SetPaymentMethods)")
	}
	return r.paymentMethods
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func nextDocumentNumber(ctx context.Context, db queryer, seq string, prefix string) (string, error) {
	var seqValue int
	err := db.QueryRow(ctx, fmt.Sprintf(`SELECT nextval('%s')`, seq)).Scan(&seqValue)
	if err != nil {
		return "", fmt.Errorf("failed to get next %s number: %w", prefix, err)
	}
	year := time.Now().In(shared.JakartaLocation()).Year()
	return fmt.Sprintf("%s-%d-%06d", prefix, year, seqValue), nil
}

func (r *Repository) NextReceiptNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	return nextDocumentNumber(ctx, tx, "consignment_receipt_seq", "CR")
}

func (r *Repository) NextReturnNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	return nextDocumentNumber(ctx, tx, "consignment_return_seq", "RT")
}

func (r *Repository) NextSettlementNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	return nextDocumentNumber(ctx, tx, "consignment_settlement_seq", "CS")
}

func (r *Repository) NextPayoutNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	return nextDocumentNumber(ctx, tx, "consignment_payout_seq", "CP")
}

// --- Arrangements ---

func (r *Repository) GetArrangementByID(ctx context.Context, q queryer, id int) (*Arrangement, error) {
	var a Arrangement
	var lastVisitAt, endedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx, `
		SELECT a.id, a.supplier_id, COALESCE(s.name,''), a.store_id, COALESCE(st.name,''), a.status,
		       a.last_visit_at, a.ended_at, a.created_by, a.created_at, a.updated_at
		FROM consignment_arrangements a
		LEFT JOIN suppliers s ON s.id = a.supplier_id
		LEFT JOIN stores st ON st.id = a.store_id
		WHERE a.id = $1
	`, id).Scan(&a.ID, &a.SupplierID, &a.SupplierName, &a.StoreID, &a.StoreName, &a.Status,
		&lastVisitAt, &endedAt, &a.CreatedBy, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrConsignmentNotFound
		}
		return nil, err
	}
	a.LastVisitAt = nullTimePtr(lastVisitAt)
	a.EndedAt = nullTimePtr(endedAt)
	a.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	a.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &a, nil
}

// GetActiveArrangement returns the active arrangement for a supplier/store, or
// nil when none exists. Ended arrangements are excluded so a new arrangement
// can be opened after the previous one ends (BR-07).
func (r *Repository) GetActiveArrangement(ctx context.Context, q queryer, supplierID, storeID int) (*Arrangement, error) {
	a, err := r.GetArrangementByIDQuery(ctx, q, `
		WHERE a.supplier_id = $1 AND a.store_id = $2 AND a.status = 'active'
	`, supplierID, storeID)
	if err != nil && errors.Is(err, ErrConsignmentNotFound) {
		return nil, nil
	}
	return a, err
}

func (r *Repository) GetArrangementByIDQuery(ctx context.Context, q queryer, where string, args ...any) (*Arrangement, error) {
	var a Arrangement
	var lastVisitAt, endedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx, `
		SELECT a.id, a.supplier_id, COALESCE(s.name,''), a.store_id, COALESCE(st.name,''), a.status,
		       a.last_visit_at, a.ended_at, a.created_by, a.created_at, a.updated_at
		FROM consignment_arrangements a
		LEFT JOIN suppliers s ON s.id = a.supplier_id
		LEFT JOIN stores st ON st.id = a.store_id
		`+where+`
	`, args...).Scan(&a.ID, &a.SupplierID, &a.SupplierName, &a.StoreID, &a.StoreName, &a.Status,
		&lastVisitAt, &endedAt, &a.CreatedBy, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrConsignmentNotFound
		}
		return nil, err
	}
	a.LastVisitAt = nullTimePtr(lastVisitAt)
	a.EndedAt = nullTimePtr(endedAt)
	a.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	a.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &a, nil
}

// ListArrangements returns arrangements for a store (all stores when storeID
// is nil). Status is computed lazily: a row still 'active' whose last visit is
// older than 2 weeks is reported as 'ended' (lazy-ended, never auto-persisted).
func (r *Repository) ListArrangements(ctx context.Context, q queryer, storeID *int, limit, offset int, search, status string) ([]Arrangement, int, error) {
	effStatus := `CASE WHEN a.status = 'active' AND (a.last_visit_at IS NULL OR a.last_visit_at < NOW() - INTERVAL '14 days') THEN 'ended' ELSE a.status END`

	var conds []string
	var args []any
	if storeID != nil {
		args = append(args, *storeID)
		conds = append(conds, fmt.Sprintf("a.store_id = $%d", len(args)))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		conds = append(conds, fmt.Sprintf("(s.name ILIKE $%d OR a.id::text = $%d)", len(args), len(args)))
	}
	if status != "" {
		args = append(args, status)
		conds = append(conds, fmt.Sprintf("%s = $%d", effStatus, len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM consignment_arrangements a LEFT JOIN suppliers s ON s.id = a.supplier_id` + where
	if err := q.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT a.id, a.supplier_id, COALESCE(s.name,''), a.store_id, COALESCE(st.name,''), ` + effStatus + `,
		       a.last_visit_at, a.ended_at, a.created_by, a.created_at, a.updated_at
		FROM consignment_arrangements a
		LEFT JOIN suppliers s ON s.id = a.supplier_id
		LEFT JOIN stores st ON st.id = a.store_id
	` + where + ` ORDER BY a.created_at DESC`
	if limit > 0 {
		args = append(args, limit, offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []Arrangement
	for rows.Next() {
		var a Arrangement
		var lastVisitAt, endedAt sql.NullTime
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&a.ID, &a.SupplierID, &a.SupplierName, &a.StoreID, &a.StoreName, &a.Status,
			&lastVisitAt, &endedAt, &a.CreatedBy, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		a.LastVisitAt = nullTimePtr(lastVisitAt)
		a.EndedAt = nullTimePtr(endedAt)
		a.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		a.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (r *Repository) InsertArrangement(ctx context.Context, tx pgx.Tx, a *Arrangement) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_arrangements (supplier_id, store_id, status, last_visit_at, created_by)
		VALUES ($1, $2, $3, now(), $4)
		RETURNING id, created_at
	`, a.SupplierID, a.StoreID, StatusActive, a.CreatedBy).Scan(&a.ID, &createdAt)
	if err != nil {
		return err
	}
	a.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	a.Status = StatusActive
	return nil
}

func (r *Repository) TouchVisit(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, `
		UPDATE consignment_arrangements
		SET last_visit_at = now(), status = 'active', ended_at = NULL,
		    updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

func (r *Repository) EndArrangement(ctx context.Context, tx pgx.Tx, id int) error {
	var endedAt time.Time
	err := tx.QueryRow(ctx, `
		UPDATE consignment_arrangements
		SET status = 'ended', ended_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING ended_at
	`, id).Scan(&endedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrConsignmentNotFound
		}
		return err
	}
	return nil
}

// --- Terms ---

func (r *Repository) ListTerms(ctx context.Context, q queryer, arrangementID int) ([]Term, error) {
	rows, err := q.Query(ctx, `
		SELECT t.id, t.arrangement_id, t.product_id, COALESCE(p.sku,''), COALESCE(p.name,''),
		       t.price, t.store_share_type, t.store_share_value, t.effective_from, t.created_by, t.created_at
		FROM consignment_terms t
		JOIN products p ON p.id = t.product_id
		WHERE t.arrangement_id = $1
		ORDER BY t.id ASC
	`, arrangementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Term
	for rows.Next() {
		var t Term
		var effectiveFrom, createdAt time.Time
		if err := rows.Scan(&t.ID, &t.ArrangementID, &t.ProductID, &t.ProductSKU, &t.ProductName,
			&t.Price, &t.StoreShareType, &t.StoreShareValue, &effectiveFrom, &t.CreatedBy, &createdAt); err != nil {
			return nil, err
		}
		t.EffectiveFrom = effectiveFrom.In(shared.JakartaLocation()).Format(time.RFC3339)
		t.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *Repository) GetTermByProduct(ctx context.Context, q queryer, arrangementID, productID int) (*Term, error) {
	var t Term
	var effectiveFrom, createdAt time.Time
	err := q.QueryRow(ctx, `
		SELECT t.id, t.arrangement_id, t.product_id, COALESCE(p.sku,''), COALESCE(p.name,''),
		       t.price, t.store_share_type, t.store_share_value, t.effective_from, t.created_by, t.created_at
		FROM consignment_terms t
		JOIN products p ON p.id = t.product_id
		WHERE t.arrangement_id = $1 AND t.product_id = $2
	`, arrangementID, productID).Scan(&t.ID, &t.ArrangementID, &t.ProductID, &t.ProductSKU, &t.ProductName,
		&t.Price, &t.StoreShareType, &t.StoreShareValue, &effectiveFrom, &t.CreatedBy, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	t.EffectiveFrom = effectiveFrom.In(shared.JakartaLocation()).Format(time.RFC3339)
	t.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &t, nil
}

func (r *Repository) InsertTerm(ctx context.Context, tx pgx.Tx, t *Term) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_terms (arrangement_id, product_id, price, store_share_type, store_share_value, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, t.ArrangementID, t.ProductID, t.Price, t.StoreShareType, t.StoreShareValue, t.CreatedBy).
		Scan(&t.ID, &createdAt)
	if err != nil {
		return err
	}
	t.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) ReplaceTerms(ctx context.Context, tx pgx.Tx, arrangementID int, terms []Term) error {
	if _, err := tx.Exec(ctx, `DELETE FROM consignment_terms WHERE arrangement_id = $1`, arrangementID); err != nil {
		return err
	}
	for i := range terms {
		terms[i].ArrangementID = arrangementID
		if err := r.InsertTerm(ctx, tx, &terms[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- Consignment stock ledger (consignment-owned) ---

// GetConsignmentStock returns the ownership ledger row for a product, or nil
// when the product is not consignment-owned.
func (r *Repository) GetConsignmentStock(ctx context.Context, q queryer, productID int) (*StockRow, error) {
	var s StockRow
	var updatedAt sql.NullTime
	err := q.QueryRow(ctx, `
		SELECT cs.product_id, cs.supplier_id, COALESCE(sup.name,''), cs.arrangement_id, cs.store_id,
		       cs.available_qty, cs.pending_return_qty, cs.updated_at
		FROM consignment_stock cs
		JOIN suppliers sup ON sup.id = cs.supplier_id
		WHERE cs.product_id = $1
	`, productID).Scan(&s.ProductID, &s.SupplierID, &s.SupplierName, &s.ArrangementID, &s.StoreID,
		&s.AvailableQty, &s.PendingReturnQty, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s.UpdatedAt = nullTimePtr(updatedAt)
	return &s, nil
}

// LockConsignmentStock locks the ledger row FOR UPDATE so concurrent checkout
// and receipt writes serialize. Returns nil when the product has no row.
func (r *Repository) LockConsignmentStock(ctx context.Context, tx pgx.Tx, productID int) (*StockRow, error) {
	var s StockRow
	var updatedAt sql.NullTime
	err := tx.QueryRow(ctx, `
		SELECT cs.product_id, cs.supplier_id, COALESCE(sup.name,''), cs.arrangement_id, cs.store_id,
		       cs.available_qty, cs.pending_return_qty, cs.updated_at
		FROM consignment_stock cs
		JOIN suppliers sup ON sup.id = cs.supplier_id
		WHERE cs.product_id = $1
		FOR UPDATE
	`, productID).Scan(&s.ProductID, &s.SupplierID, &s.SupplierName, &s.ArrangementID, &s.StoreID,
		&s.AvailableQty, &s.PendingReturnQty, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s.UpdatedAt = nullTimePtr(updatedAt)
	return &s, nil
}

// UpsertConsignmentStock adds delta to available_qty for the owning row,
// creating it (supplier_id, arrangement_id, store_id) when missing. On conflict
// the owner columns are refreshed to the incoming supplier/arrangement so a
// fully-released row (available=0, pending_return=0) taken over by a new
// supplier (BR-03 re-ownership) is not left with stale ownership. The SQL
// guard rejects a negative available_qty. It returns the new available qty.
// Callers MUST have already run resolveOwnership so the owner refresh can only
// happen for a released/same-supplier row.
func (r *Repository) UpsertConsignmentStock(ctx context.Context, tx pgx.Tx, productID, supplierID, arrangementID, storeID, delta int) (int, error) {
	var available int
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_stock (product_id, supplier_id, arrangement_id, store_id, available_qty, pending_return_qty)
		VALUES ($1, $2, $3, $4, GREATEST($5, 0), 0)
		ON CONFLICT (product_id) DO UPDATE
		SET supplier_id = EXCLUDED.supplier_id,
		    arrangement_id = EXCLUDED.arrangement_id,
		    store_id = EXCLUDED.store_id,
		    available_qty = consignment_stock.available_qty + $5,
		    updated_at = now()
		RETURNING available_qty
	`, productID, supplierID, arrangementID, storeID, delta).Scan(&available)
	if err != nil {
		return 0, err
	}
	if available < 0 {
		return 0, ErrInsufficientConsignmentStock
	}
	return available, nil
}

// MoveToPendingReturn moves qty from available to pending_return (BR-25).
func (r *Repository) MoveToPendingReturn(ctx context.Context, tx pgx.Tx, productID, qty int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE consignment_stock
		SET available_qty = available_qty - $2,
		    pending_return_qty = pending_return_qty + $2,
		    updated_at = now()
		WHERE product_id = $1
	`, productID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConsignmentNotFound
	}
	return nil
}

// ReduceAvailable decreases available_qty by qty, guarding against negatives.
// It is used for returns from available stock (non-pending-return path) and by
// checkout (through the sale-facing port).
func (r *Repository) ReduceAvailable(ctx context.Context, tx pgx.Tx, productID, qty int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE consignment_stock
		SET available_qty = available_qty - $2, updated_at = now()
		WHERE product_id = $1 AND available_qty >= $2
	`, productID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientConsignmentStock
	}
	return nil
}

// ResolvePendingReturn moves qty from pending_return back to available.
func (r *Repository) ResolvePendingReturn(ctx context.Context, tx pgx.Tx, productID, qty int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE consignment_stock
		SET pending_return_qty = pending_return_qty - $2, updated_at = now()
		WHERE product_id = $1 AND pending_return_qty >= $2
	`, productID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientConsignmentStock
	}
	return nil
}

// ReleaseOwnership deletes the ledger row once both available and pending
// return hit 0, freeing the SKU for a future supplier.
func (r *Repository) ReleaseOwnership(ctx context.Context, tx pgx.Tx, productID int) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM consignment_stock
		WHERE product_id = $1 AND available_qty = 0 AND pending_return_qty = 0
	`, productID)
	return err
}

// ListConsignmentStock returns the ownership ledger for a store (all stores
// when storeID is nil), optionally filtered by supplier.
func (r *Repository) ListConsignmentStock(ctx context.Context, q queryer, supplierID *int, storeID *int) ([]StockRow, error) {
	query := `
		SELECT cs.product_id, COALESCE(p.sku,''), COALESCE(p.name,''), cs.supplier_id, COALESCE(sup.name,''),
		       cs.arrangement_id, cs.store_id, cs.available_qty, cs.pending_return_qty, cs.updated_at
		FROM consignment_stock cs
		JOIN suppliers sup ON sup.id = cs.supplier_id
		JOIN products p ON p.id = cs.product_id
	`
	var conds []string
	var args []any
	if supplierID != nil {
		conds = append(conds, fmt.Sprintf("cs.supplier_id = $%d", len(args)+1))
		args = append(args, *supplierID)
	}
	if storeID != nil {
		conds = append(conds, fmt.Sprintf("cs.store_id = $%d", len(args)+1))
		args = append(args, *storeID)
	}
	if len(conds) > 0 {
		query += " WHERE " + joinConds(conds)
	}
	query += " ORDER BY p.name ASC"
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StockRow
	for rows.Next() {
		var s StockRow
		var updatedAt sql.NullTime
		if err := rows.Scan(&s.ProductID, &s.ProductSKU, &s.ProductName, &s.SupplierID, &s.SupplierName,
			&s.ArrangementID, &s.StoreID, &s.AvailableQty, &s.PendingReturnQty, &updatedAt); err != nil {
			return nil, err
		}
		s.UpdatedAt = nullTimePtr(updatedAt)
		result = append(result, s)
	}
	return result, rows.Err()
}

// --- Receipts ---

func (r *Repository) InsertReceipt(ctx context.Context, tx pgx.Tx, rec *Receipt) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_receipts (receipt_number, supplier_id, store_id, arrangement_id, received_by, received_at, notes)
		VALUES ($1, $2, $3, $4, $5, now(), $6)
		RETURNING id, created_at
	`, rec.ReceiptNumber, rec.SupplierID, rec.StoreID, rec.ArrangementID, rec.ReceivedBy, rec.Notes).
		Scan(&rec.ID, &createdAt)
	if err != nil {
		return err
	}
	rec.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	rec.ReceivedAt = rec.CreatedAt
	return nil
}

func (r *Repository) InsertReceiptItem(ctx context.Context, tx pgx.Tx, item *ReceiptItem) error {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_receipt_items (consignment_receipt_id, product_id, accepted_qty, price, store_share_type, store_share_value, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, item.ConsignmentReceiptID, item.ProductID, item.AcceptedQty, item.Price, item.StoreShareType, item.StoreShareValue, item.Notes).Scan(&id)
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

func (r *Repository) GetReceiptByID(ctx context.Context, q queryer, id int) (*Receipt, error) {
	var rec Receipt
	var receivedAt, createdAt time.Time
	err := q.QueryRow(ctx, `
		SELECT r.id, r.receipt_number, r.supplier_id, COALESCE(s.name,''), r.store_id, r.arrangement_id,
		       r.received_by, COALESCE(u.username,''), r.received_at, COALESCE(r.notes,''), r.created_at
		FROM consignment_receipts r
		JOIN suppliers s ON s.id = r.supplier_id
		JOIN users u ON u.id = r.received_by
		WHERE r.id = $1
	`, id).Scan(&rec.ID, &rec.ReceiptNumber, &rec.SupplierID, &rec.SupplierName, &rec.StoreID, &rec.ArrangementID,
		&rec.ReceivedBy, &rec.ReceivedByUsername, &receivedAt, &rec.Notes, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrConsignmentNotFound
		}
		return nil, err
	}
	rec.ReceivedAt = receivedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	rec.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	items, err := r.getReceiptItems(ctx, q, id)
	if err != nil {
		return nil, err
	}
	rec.Items = items
	return &rec, nil
}

func (r *Repository) getReceiptItems(ctx context.Context, q queryer, receiptID int) ([]ReceiptItem, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.consignment_receipt_id, i.product_id, COALESCE(p.sku,''), COALESCE(p.name,''),
		       i.accepted_qty, i.price, i.store_share_type, i.store_share_value, COALESCE(i.notes,'')
		FROM consignment_receipt_items i
		JOIN products p ON p.id = i.product_id
		WHERE i.consignment_receipt_id = $1
		ORDER BY i.id ASC
	`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ReceiptItem
	for rows.Next() {
		var it ReceiptItem
		if err := rows.Scan(&it.ID, &it.ConsignmentReceiptID, &it.ProductID, &it.ProductSKU, &it.ProductName,
			&it.AcceptedQty, &it.Price, &it.StoreShareType, &it.StoreShareValue, &it.Notes); err != nil {
			return nil, err
		}
		result = append(result, it)
	}
	return result, rows.Err()
}

func (r *Repository) ListReceipts(ctx context.Context, q queryer, supplierID, storeID int) ([]Receipt, error) {
	rows, err := q.Query(ctx, `
		SELECT r.id, r.receipt_number, r.supplier_id, COALESCE(s.name,''), r.store_id, r.arrangement_id,
		       r.received_by, COALESCE(u.username,''), r.received_at, COALESCE(r.notes,''), r.created_at
		FROM consignment_receipts r
		JOIN suppliers s ON s.id = r.supplier_id
		JOIN users u ON u.id = r.received_by
		WHERE r.supplier_id = $1 AND r.store_id = $2
		ORDER BY r.created_at DESC
	`, supplierID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Receipt
	for rows.Next() {
		var rec Receipt
		var receivedAt, createdAt time.Time
		if err := rows.Scan(&rec.ID, &rec.ReceiptNumber, &rec.SupplierID, &rec.SupplierName, &rec.StoreID, &rec.ArrangementID,
			&rec.ReceivedBy, &rec.ReceivedByUsername, &receivedAt, &rec.Notes, &createdAt); err != nil {
			return nil, err
		}
		rec.ReceivedAt = receivedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		rec.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, rec)
	}
	return result, rows.Err()
}

// --- Pending returns ---

func (r *Repository) GetPendingReturnByID(ctx context.Context, q queryer, id int) (*PendingReturn, error) {
	var pr PendingReturn
	var returnedAt, createdAt sql.NullTime
	err := q.QueryRow(ctx, `
		SELECT pr.id, pr.supplier_id, pr.product_id, COALESCE(p.sku,''), COALESCE(p.name,''),
		       pr.arrangement_id, pr.store_id, pr.qty, pr.reason, COALESCE(pr.notes,''), pr.status,
		       pr.returned_at, pr.created_by, pr.created_at
		FROM consignment_pending_returns pr
		JOIN products p ON p.id = pr.product_id
		WHERE pr.id = $1
	`, id).Scan(&pr.ID, &pr.SupplierID, &pr.ProductID, &pr.ProductSKU, &pr.ProductName,
		&pr.ArrangementID, &pr.StoreID, &pr.Qty, &pr.Reason, &pr.Notes, &pr.Status,
		&returnedAt, &pr.CreatedBy, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPendingReturnNotFound
		}
		return nil, err
	}
	pr.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	if returnedAt.Valid {
		rt := returnedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		pr.ReturnedAt = &rt
	}
	return &pr, nil
}

func (r *Repository) MarkPendingReturnReturned(ctx context.Context, tx pgx.Tx, id int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE consignment_pending_returns
		SET status = 'returned', returned_at = now()
		WHERE id = $1 AND status = 'open'
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPendingReturnNotFound
	}
	return nil
}

// ReducePendingReturnQty decrements an open pending return's remaining qty
// without closing it. Used when a formal return only partially resolves a
// pending return (AC-C25), so the leftover quantity stays open and can be
// returned later instead of being orphaned by a premature 'returned' status.
func (r *Repository) ReducePendingReturnQty(ctx context.Context, tx pgx.Tx, id, qty int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE consignment_pending_returns
		SET qty = qty - $2
		WHERE id = $1 AND status = 'open' AND qty >= $2
	`, id, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPendingReturnNotFound
	}
	return nil
}

func (r *Repository) InsertPendingReturn(ctx context.Context, tx pgx.Tx, pr *PendingReturn) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_pending_returns (supplier_id, product_id, arrangement_id, store_id, qty, reason, notes, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'open', $8)
		RETURNING id, created_at
	`, pr.SupplierID, pr.ProductID, pr.ArrangementID, pr.StoreID, pr.Qty, pr.Reason, pr.Notes, pr.CreatedBy).
		Scan(&pr.ID, &createdAt)
	if err != nil {
		return err
	}
	pr.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	pr.Status = PendingReturnOpen
	return nil
}

func (r *Repository) ListOpenPendingReturns(ctx context.Context, q queryer, supplierID, storeID int) ([]PendingReturn, error) {
	rows, err := q.Query(ctx, `
		SELECT pr.id, pr.supplier_id, pr.product_id, COALESCE(p.sku,''), COALESCE(p.name,''),
		       pr.arrangement_id, pr.store_id, pr.qty, pr.reason, COALESCE(pr.notes,''), pr.status,
		       pr.returned_at, pr.created_by, pr.created_at
		FROM consignment_pending_returns pr
		JOIN products p ON p.id = pr.product_id
		WHERE pr.supplier_id = $1 AND pr.store_id = $2 AND pr.status = 'open'
		ORDER BY pr.created_at ASC
	`, supplierID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PendingReturn
	for rows.Next() {
		var pr PendingReturn
		var returnedAt, createdAt sql.NullTime
		if err := rows.Scan(&pr.ID, &pr.SupplierID, &pr.ProductID, &pr.ProductSKU, &pr.ProductName,
			&pr.ArrangementID, &pr.StoreID, &pr.Qty, &pr.Reason, &pr.Notes, &pr.Status,
			&returnedAt, &pr.CreatedBy, &createdAt); err != nil {
			return nil, err
		}
		pr.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		if returnedAt.Valid {
			rt := returnedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
			pr.ReturnedAt = &rt
		}
		result = append(result, pr)
	}
	return result, rows.Err()
}

// --- Returns ---

func (r *Repository) InsertReturn(ctx context.Context, tx pgx.Tx, ret *Return) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_returns (return_number, supplier_id, store_id, arrangement_id, returned_by, returned_at, notes)
		VALUES ($1, $2, $3, $4, $5, now(), $6)
		RETURNING id, created_at
	`, ret.ReturnNumber, ret.SupplierID, ret.StoreID, ret.ArrangementID, ret.ReturnedBy, ret.Notes).
		Scan(&ret.ID, &createdAt)
	if err != nil {
		return err
	}
	ret.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	ret.ReturnedAt = ret.CreatedAt
	return nil
}

func (r *Repository) InsertReturnItem(ctx context.Context, tx pgx.Tx, item *ReturnItem) error {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_return_items (consignment_return_id, product_id, qty, reason, pending_return_id, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, item.ConsignmentReturnID, item.ProductID, item.Qty, item.Reason, item.PendingReturnID, item.Notes).Scan(&id)
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

func (r *Repository) GetReturnByID(ctx context.Context, q queryer, id int) (*Return, error) {
	var ret Return
	var returnedAt, createdAt time.Time
	err := q.QueryRow(ctx, `
		SELECT rt.id, rt.return_number, rt.supplier_id, COALESCE(s.name,''), rt.store_id, rt.arrangement_id,
		       rt.returned_by, COALESCE(u.username,''), rt.returned_at, COALESCE(rt.notes,''), rt.created_at
		FROM consignment_returns rt
		JOIN suppliers s ON s.id = rt.supplier_id
		JOIN users u ON u.id = rt.returned_by
		WHERE rt.id = $1
	`, id).Scan(&ret.ID, &ret.ReturnNumber, &ret.SupplierID, &ret.SupplierName, &ret.StoreID, &ret.ArrangementID,
		&ret.ReturnedBy, &ret.ReturnedByUsername, &returnedAt, &ret.Notes, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrReturnNotFound
		}
		return nil, err
	}
	ret.ReturnedAt = returnedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	ret.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	items, err := r.getReturnItems(ctx, q, id)
	if err != nil {
		return nil, err
	}
	ret.Items = items
	return &ret, nil
}

func (r *Repository) getReturnItems(ctx context.Context, q queryer, returnID int) ([]ReturnItem, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.consignment_return_id, i.product_id, COALESCE(p.sku,''), COALESCE(p.name,''),
		       i.qty, i.reason, i.pending_return_id, COALESCE(i.notes,'')
		FROM consignment_return_items i
		JOIN products p ON p.id = i.product_id
		WHERE i.consignment_return_id = $1
		ORDER BY i.id ASC
	`, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ReturnItem
	for rows.Next() {
		var it ReturnItem
		var pendingReturnID sql.NullInt64
		if err := rows.Scan(&it.ID, &it.ConsignmentReturnID, &it.ProductID, &it.ProductSKU, &it.ProductName,
			&it.Qty, &it.Reason, &pendingReturnID, &it.Notes); err != nil {
			return nil, err
		}
		if pendingReturnID.Valid {
			id := int(pendingReturnID.Int64)
			it.PendingReturnID = &id
		}
		result = append(result, it)
	}
	return result, rows.Err()
}

func (r *Repository) ListReturns(ctx context.Context, q queryer, supplierID, storeID int) ([]Return, error) {
	rows, err := q.Query(ctx, `
		SELECT rt.id, rt.return_number, rt.supplier_id, COALESCE(s.name,''), rt.store_id, rt.arrangement_id,
		       rt.returned_by, COALESCE(u.username,''), rt.returned_at, COALESCE(rt.notes,''), rt.created_at
		FROM consignment_returns rt
		JOIN suppliers s ON s.id = rt.supplier_id
		JOIN users u ON u.id = rt.returned_by
		WHERE rt.supplier_id = $1 AND rt.store_id = $2
		ORDER BY rt.created_at DESC
	`, supplierID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Return
	for rows.Next() {
		var ret Return
		var returnedAt, createdAt time.Time
		if err := rows.Scan(&ret.ID, &ret.ReturnNumber, &ret.SupplierID, &ret.SupplierName, &ret.StoreID, &ret.ArrangementID,
			&ret.ReturnedBy, &ret.ReturnedByUsername, &returnedAt, &ret.Notes, &createdAt); err != nil {
			return nil, err
		}
		ret.ReturnedAt = returnedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		ret.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, ret)
	}
	return result, rows.Err()
}

// --- Consignment sale items (checkout writes) ---

// InsertConsignmentSaleItem persists one checkout-time consignment sale record.
// The consignment module is the single-writer of consignment_sale_items; sale
// writes through the sale-facing ConsignmentCheckout port inside the checkout
// Unit of Work so the sale and its consignment lines commit atomically.
func (r *Repository) InsertConsignmentSaleItem(ctx context.Context, tx pgx.Tx, rec shared.ConsignmentSaleRecord) error {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_sale_items
			(sale_id, invoice_number, product_id, supplier_id, arrangement_id, store_id,
			 quantity, unit_price, subtotal, store_share_type, store_share_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, rec.SaleID, rec.InvoiceNumber, rec.ProductID, rec.SupplierID, rec.ArrangementID, rec.StoreID,
		rec.Quantity, rec.UnitPrice, rec.Subtotal, rec.StoreShareType, rec.StoreShareValue).Scan(&id)
	if err != nil {
		return err
	}
	return nil
}

// --- Unsettled consignment sale items (settlement reads) ---

// ListUnsettledSaleItems returns consignment sale items of a supplier/store
// that are not yet covered by a settlement, newest first (BR-24).
func (r *Repository) ListUnsettledSaleItems(ctx context.Context, q queryer, supplierID, storeID int) ([]SaleItemRecord, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.sale_id, i.invoice_number, i.product_id, COALESCE(p.name,''),
		       i.quantity, i.unit_price, i.subtotal, i.store_share_type, i.store_share_value, i.created_at
		FROM consignment_sale_items i
		JOIN products p ON p.id = i.product_id
		WHERE i.supplier_id = $1 AND i.store_id = $2 AND i.settlement_id IS NULL
		ORDER BY i.created_at DESC
	`, supplierID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSaleItemRecords(rows)
}

// ListSaleItemsByIDs returns consignment sale items by their ids, used to
// re-read records inside the settlement Unit of Work.
func (r *Repository) ListSaleItemsByIDs(ctx context.Context, tx pgx.Tx, ids []int) ([]SaleItemRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT i.id, i.sale_id, i.invoice_number, i.product_id, COALESCE(p.name,''),
		       i.quantity, i.unit_price, i.subtotal, i.store_share_type, i.store_share_value, i.created_at
		FROM consignment_sale_items i
		JOIN products p ON p.id = i.product_id
		WHERE i.id = ANY($1)
		ORDER BY i.id ASC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSaleItemRecords(rows)
}

func (r *Repository) scanSaleItemRecords(rows pgx.Rows) ([]SaleItemRecord, error) {
	var result []SaleItemRecord
	for rows.Next() {
		var it SaleItemRecord
		var createdAt time.Time
		if err := rows.Scan(&it.ID, &it.SaleID, &it.InvoiceNumber, &it.ProductID, &it.ProductName,
			&it.Quantity, &it.UnitPrice, &it.Subtotal, &it.StoreShareType, &it.StoreShareValue, &createdAt); err != nil {
			return nil, err
		}
		it.StoreShareAmount = computeStoreShare(it.UnitPrice, it.Quantity, it.StoreShareType, it.StoreShareValue)
		it.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, it)
	}
	return result, rows.Err()
}

// --- Settlements & payouts ---

func (r *Repository) InsertSettlement(ctx context.Context, tx pgx.Tx, s *Settlement) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_settlements (settlement_number, supplier_id, store_id, total_sale_value, total_store_share, total_payable, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending_payment', $7)
		RETURNING id, created_at
	`, s.SettlementNumber, s.SupplierID, s.StoreID, s.TotalSaleValue, s.TotalStoreShare, s.TotalPayable, s.CreatedBy).
		Scan(&s.ID, &createdAt)
	if err != nil {
		return err
	}
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.Status = SettlementPendingPayment
	return nil
}

func (r *Repository) InsertSettlementItem(ctx context.Context, tx pgx.Tx, item *SettlementItem) error {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_settlement_items (consignment_settlement_id, consignment_sale_item_id, product_id, quantity, unit_price, subtotal, store_share)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, item.ConsignmentSettlementID, item.ConsignmentSaleItemID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal, item.StoreShare).Scan(&id)
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

// MarkSaleItemsSettled links all ids to the settlement, closing the "unsettled"
// window atomically.
func (r *Repository) MarkSaleItemsSettled(ctx context.Context, tx pgx.Tx, ids []int, settlementID int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE consignment_sale_items
		SET settlement_id = $2
		WHERE id = ANY($1)
	`, ids, settlementID)
	return err
}

func (r *Repository) GetSettlementByID(ctx context.Context, q queryer, id int) (*Settlement, error) {
	var s Settlement
	var paidAt, createdAt sql.NullTime
	err := q.QueryRow(ctx, `
		SELECT st.id, st.settlement_number, st.supplier_id, COALESCE(sup.name,''), st.store_id,
		       st.total_sale_value, st.total_store_share, st.total_payable, st.status, st.created_by, st.created_at, st.paid_at
		FROM consignment_settlements st
		JOIN suppliers sup ON sup.id = st.supplier_id
		WHERE st.id = $1
	`, id).Scan(&s.ID, &s.SettlementNumber, &s.SupplierID, &s.SupplierName, &s.StoreID,
		&s.TotalSaleValue, &s.TotalStoreShare, &s.TotalPayable, &s.Status, &s.CreatedBy, &createdAt, &paidAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSettlementNotFound
		}
		return nil, err
	}
	s.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	if paidAt.Valid {
		pt := paidAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.PaidAt = &pt
	}
	items, err := r.getSettlementItems(ctx, q, id)
	if err != nil {
		return nil, err
	}
	s.Items = items
	payouts, err := r.getPayoutsBySettlement(ctx, q, id)
	if err != nil {
		return nil, err
	}
	s.Payouts = payouts
	return &s, nil
}

// GetSettlementByIDQuery returns a settlement header WITHOUT items/payouts,
// for in-transaction checks that only need the totals/status.
func (r *Repository) GetSettlementByIDQuery(ctx context.Context, q queryer, id int) (*Settlement, error) {
	var s Settlement
	var paidAt, createdAt sql.NullTime
	err := q.QueryRow(ctx, `
		SELECT st.id, st.settlement_number, st.supplier_id, COALESCE(sup.name,''), st.store_id,
		       st.total_sale_value, st.total_store_share, st.total_payable, st.status, st.created_by, st.created_at, st.paid_at
		FROM consignment_settlements st
		JOIN suppliers sup ON sup.id = st.supplier_id
		WHERE st.id = $1
	`, id).Scan(&s.ID, &s.SettlementNumber, &s.SupplierID, &s.SupplierName, &s.StoreID,
		&s.TotalSaleValue, &s.TotalStoreShare, &s.TotalPayable, &s.Status, &s.CreatedBy, &createdAt, &paidAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSettlementNotFound
		}
		return nil, err
	}
	s.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	if paidAt.Valid {
		pt := paidAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.PaidAt = &pt
	}
	return &s, nil
}

func (r *Repository) getSettlementItems(ctx context.Context, q queryer, settlementID int) ([]SettlementItem, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.consignment_settlement_id, i.consignment_sale_item_id, i.product_id, COALESCE(p.name,''),
		       i.quantity, i.unit_price, i.subtotal, i.store_share
		FROM consignment_settlement_items i
		JOIN products p ON p.id = i.product_id
		WHERE i.consignment_settlement_id = $1
		ORDER BY i.id ASC
	`, settlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SettlementItem
	for rows.Next() {
		var it SettlementItem
		if err := rows.Scan(&it.ID, &it.ConsignmentSettlementID, &it.ConsignmentSaleItemID, &it.ProductID, &it.ProductName,
			&it.Quantity, &it.UnitPrice, &it.Subtotal, &it.StoreShare); err != nil {
			return nil, err
		}
		result = append(result, it)
	}
	return result, rows.Err()
}

func (r *Repository) ListSettlements(ctx context.Context, q queryer, supplierID, storeID int) ([]Settlement, error) {
	rows, err := q.Query(ctx, `
		SELECT st.id, st.settlement_number, st.supplier_id, COALESCE(sup.name,''), st.store_id,
		       st.total_sale_value, st.total_store_share, st.total_payable, st.status, st.created_by, st.created_at, st.paid_at
		FROM consignment_settlements st
		JOIN suppliers sup ON sup.id = st.supplier_id
		WHERE st.supplier_id = $1 AND st.store_id = $2
		ORDER BY st.created_at DESC
	`, supplierID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Settlement
	for rows.Next() {
		var s Settlement
		var paidAt, createdAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.SettlementNumber, &s.SupplierID, &s.SupplierName, &s.StoreID,
			&s.TotalSaleValue, &s.TotalStoreShare, &s.TotalPayable, &s.Status, &s.CreatedBy, &createdAt, &paidAt); err != nil {
			return nil, err
		}
		s.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		if paidAt.Valid {
			pt := paidAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
			s.PaidAt = &pt
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *Repository) InsertPayout(ctx context.Context, tx pgx.Tx, p *Payout) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO consignment_payouts (payout_number, settlement_id, payment_method_id, amount, reference_number, paid_by, paid_at, notes)
		VALUES ($1, $2, $3, $4, $5, $6, now(), $7)
		RETURNING id, created_at
	`, p.PayoutNumber, p.SettlementID, p.PaymentMethodID, p.Amount, p.ReferenceNumber, p.PaidBy, p.Notes).
		Scan(&p.ID, &createdAt)
	if err != nil {
		return err
	}
	p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	p.PaidAt = p.CreatedAt
	return nil
}

func (r *Repository) getPayoutsBySettlement(ctx context.Context, q queryer, settlementID int) ([]Payout, error) {
	rows, err := q.Query(ctx, `
		SELECT p.id, p.payout_number, p.settlement_id, p.payment_method_id, COALESCE(pm.code,''), COALESCE(pm.name,''),
		       p.amount, COALESCE(p.reference_number,''), p.paid_by, COALESCE(u.username,''), p.paid_at, COALESCE(p.notes,''), p.created_at
		FROM consignment_payouts p
		JOIN payment_methods pm ON pm.id = p.payment_method_id
		JOIN users u ON u.id = p.paid_by
		WHERE p.settlement_id = $1
		ORDER BY p.id ASC
	`, settlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Payout
	for rows.Next() {
		var p Payout
		var paidAt, createdAt time.Time
		if err := rows.Scan(&p.ID, &p.PayoutNumber, &p.SettlementID, &p.PaymentMethodID, &p.PaymentMethodCode, &p.PaymentMethodName,
			&p.Amount, &p.ReferenceNumber, &p.PaidBy, &p.PaidByUsername, &paidAt, &p.Notes, &createdAt); err != nil {
			return nil, err
		}
		p.PaidAt = paidAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, p)
	}
	return result, rows.Err()
}

// MarkSettlementPaid flips the settlement to 'paid' (final, no more payouts).
func (r *Repository) MarkSettlementPaid(ctx context.Context, tx pgx.Tx, id int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE consignment_settlements
		SET status = 'paid', paid_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'pending_payment'
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSettlementAlreadyPaid
	}
	return nil
}

func (r *Repository) SumPayoutsBySettlement(ctx context.Context, tx pgx.Tx, settlementID int) (int, error) {
	var total int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM consignment_payouts
		WHERE settlement_id = $1
	`, settlementID).Scan(&total)
	return total, err
}

// --- Shared helpers ---

func nullTimePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	v := t.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &v
}

func isStaleVisit(lastVisitAt *string) bool {
	if lastVisitAt == nil {
		return true
	}
	t, err := time.Parse(time.RFC3339, *lastVisitAt)
	if err != nil {
		return true
	}
	return time.Since(t) > 14*24*time.Hour
}

func joinConds(conds []string) string {
	out := ""
	for i, c := range conds {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}

// computeStoreShare returns the store share for one consignment sale LINE
// (already multiplied by quantity). Percentage shares apply to the line's total
// sale value (unit price × qty); fixed_amount shares are per unit, so they are
// multiplied by qty too. This keeps the line's store share consistent with its
// Subtotal (unitPrice × qty) so TotalPayable = TotalSaleValue − TotalStoreShare
// is exact (PRD §10.2/§10.3, AC-C29).
func computeStoreShare(unitPrice, quantity int, shareType string, shareValue float64) int {
	if shareType == ShareTypePercentage {
		return int(float64(unitPrice) * float64(quantity) * shareValue / 100.0)
	}
	return int(shareValue) * quantity
}