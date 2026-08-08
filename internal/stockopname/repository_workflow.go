package stockopname

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

// --- scopes ---

// LoadSnapshotProductsByIDs returns the general-stock snapshot for the given
// active product ids (used for scoped cycle counts). When ids is empty all
// active products are returned.
// LoadSnapshotProductsByIDs returns the general-stock snapshot for the given
// active product ids (used for scoped cycle counts). When ids is empty all
// active products are returned. The catalog rows come from the product-owned
// ProductCatalogProvider and the stock quantities from the inventory-owned
// StockSnapshotProvider; the global snapshot inner-joins product_stock, so
// products without a global stock row are excluded.
func (r *Repository) LoadSnapshotProductsByIDs(ctx context.Context, db shared.DBPool, ids []int) ([]SessionItem, error) {
	catalog, err := r.snapshotCatalog(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	catalogIDs := make([]int, 0, len(catalog))
	for _, p := range catalog {
		catalogIDs = append(catalogIDs, p.ProductID)
	}
	quantities, err := r.snapshotQuantities(ctx, db, catalogIDs, nil)
	if err != nil {
		return nil, err
	}
	return r.buildSnapshotItems(ctx, db, catalog, quantities, false)
}

// ResolveScopeName returns a human-readable name for a scope reference, or an
// empty string when the referenced record does not exist or the scope type is
// not one of the supported referensi/katalog scopes. The read is routed
// through the ScopeNameResolver port to each owner module.
func (r *Repository) ResolveScopeName(ctx context.Context, db shared.DBPool, scopeType string, scopeID int64) (string, error) {
	if scopeID <= 0 {
		return "", nil
	}
	names, err := r.scopeNames(ctx, db, []ScopeRef{{ScopeType: scopeType, ScopeID: scopeID}})
	if err != nil {
		return "", err
	}
	return names[ScopeRef{ScopeType: scopeType, ScopeID: scopeID}], nil
}

// ScopeProductIDs returns the product universe covered by a scope. The read
// is routed to the owner modules: product-scoped scopes
// (store/category/brand/supplier/product/manual) through the product-owned
// ProductScopeProvider, stock-scoped scopes (warehouse/location) through the
// inventory-owned StockSnapshotProvider.
func (r *Repository) ScopeProductIDs(ctx context.Context, db shared.DBPool, scope Scope) ([]int, error) {
	return r.scopeProductIDs(ctx, db, scope)
}

// InsertSessionScopes persists the scope list of a session.
func (r *Repository) InsertSessionScopes(ctx context.Context, tx pgx.Tx, sessionID int, scopes []SessionScope) error {
	if len(scopes) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(scopes))
	for i, sc := range scopes {
		rows[i] = []interface{}{sessionID, sc.ScopeType, sc.ScopeID, sc.ScopeName}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"stock_opname_session_scopes"},
		[]string{"stock_opname_id", "scope_type", "scope_id", "scope_name"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert session scopes: %w", err)
	}
	return nil
}

// LoadSessionScopes returns the scope list of a session.
func (r *Repository) LoadSessionScopes(ctx context.Context, q queryer, sessionID int) ([]SessionScope, error) {
	rows, err := q.Query(ctx, `
		SELECT id, stock_opname_id, scope_type, scope_id, COALESCE(scope_name, '')
		FROM stock_opname_session_scopes
		WHERE stock_opname_id = $1
		ORDER BY id ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session scopes: %w", err)
	}
	defer rows.Close()
	var out []SessionScope
	for rows.Next() {
		var sc SessionScope
		if err := rows.Scan(&sc.ID, &sc.StockOpnameID, &sc.ScopeType, &sc.ScopeID, &sc.ScopeName); err != nil {
			return nil, fmt.Errorf("failed to scan session scope: %w", err)
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// --- overlap detection ---

// AcquireCreateLock serialises concurrent session creation so the overlap
// check is race-free (single active scope guard replaced by per-SKU overlap).
func (r *Repository) AcquireCreateLock(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('stock_opname_create'))`)
	if err != nil {
		return fmt.Errorf("failed to acquire stock opname create lock: %w", err)
	}
	return nil
}

// ListActiveSessions returns sessions still in progress, with their scopes
// populated, so callers can enforce the per-SKU overlap rule.
func (r *Repository) ListActiveSessions(ctx context.Context, q queryer) ([]Session, error) {
	rows, err := q.Query(ctx, `
		SELECT id, session_number, status, store_id
		FROM stock_opnames
		WHERE status IN ('draft','open','counting','needs_recount','verification','approved')
		  AND deleted_at IS NULL
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var s Session
		var storeID sql.NullInt64
		if err := rows.Scan(&s.ID, &s.SessionNumber, &s.Status, &storeID); err != nil {
			return nil, fmt.Errorf("failed to scan active session: %w", err)
		}
		if storeID.Valid {
			v := int(storeID.Int64)
			s.StoreID = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetProductSKUs returns a product id -> sku map for the given ids. The read
// is routed through the ProductCatalogProvider port owned by internal/product.
func (r *Repository) GetProductSKUs(ctx context.Context, productIDs []int) (map[int]string, error) {
	metas, err := r.productMetas(ctx, r.db, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load product skus: %w", err)
	}
	out := make(map[int]string, len(metas))
	for id, m := range metas {
		out[id] = m.SKU
	}
	return out, nil
}

// --- workflow transitions ---

func (r *Repository) MarkSessionOpened(ctx context.Context, tx pgx.Tx, id, userID int) error {
	return r.guardedTransition(ctx, tx, `
		UPDATE stock_opnames SET status = 'open', opened_by = $2, opened_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'draft' AND deleted_at IS NULL`, id, userID)
}

func (r *Repository) MarkSessionVerified(ctx context.Context, tx pgx.Tx, id, userID int) error {
	return r.guardedTransition(ctx, tx, `
		UPDATE stock_opnames SET status = 'approved', verified_by = $2, verified_at = NOW(),
		       approved_by = $2, approved_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'verification' AND deleted_at IS NULL`, id, userID)
}

func (r *Repository) MarkSessionPosted(ctx context.Context, tx pgx.Tx, id, userID int) error {
	return r.guardedTransition(ctx, tx, `
		UPDATE stock_opnames SET status = 'posted', posted_by = $2, posted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'approved' AND deleted_at IS NULL`, id, userID)
}

func (r *Repository) MarkSessionClosed(ctx context.Context, tx pgx.Tx, id, userID int) error {
	return r.guardedTransition(ctx, tx, `
		UPDATE stock_opnames SET status = 'closed', closed_by = $2, closed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'posted' AND deleted_at IS NULL`, id, userID)
}

func (r *Repository) MarkSessionTotals(ctx context.Context, tx pgx.Tx, id int, totalDiff, totalAdj float64) error {
	_, err := tx.Exec(ctx, `
		UPDATE stock_opnames SET total_difference = $2, total_adjustment = $3, updated_at = NOW()
		WHERE id = $1`, id, totalDiff, totalAdj)
	if err != nil {
		return fmt.Errorf("failed to update session totals: %w", err)
	}
	return nil
}

func (r *Repository) guardedTransition(ctx context.Context, tx pgx.Tx, sql string, id, userID int) error {
	tag, err := tx.Exec(ctx, sql, id, userID)
	if err != nil {
		return fmt.Errorf("failed to transition session status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidState
	}
	return nil
}

// InsertRecountRequest records why a supervisor requested a recount.
func (r *Repository) InsertRecountRequest(ctx context.Context, sessionID, userID int, reason string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO stock_opname_recount_requests (stock_opname_id, requested_by, reason)
		VALUES ($1, $2, NULLIF($3, ''))
	`, sessionID, userID, reason)
	if err != nil {
		return fmt.Errorf("failed to insert recount request: %w", err)
	}
	return nil
}

// --- adjustment ledger ---

// GetNextAdjustmentNumber returns the next adjustment document number.
func (r *Repository) GetNextAdjustmentNumber(ctx context.Context) (string, error) {
	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('ia_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next adjustment number: %w", err)
	}
	year := time.Now().In(shared.JakartaLocation()).Year()
	return fmt.Sprintf("IA-%d-%06d", year, seq), nil
}

func (r *Repository) InsertAdjustment(ctx context.Context, tx pgx.Tx, adj *Adjustment) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO inventory_adjustments (adjustment_number, session_id, status, notes, created_by)
		VALUES ($1, $2, 'posted', NULLIF($3, ''), $4)
		RETURNING id, created_at
	`, adj.AdjustmentNumber, adj.SessionID, adj.Notes, adj.CreatedBy).Scan(&adj.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("failed to insert inventory adjustment: %w", err)
	}
	adj.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) InsertAdjustmentItems(ctx context.Context, tx pgx.Tx, adjustmentID int, items []AdjustmentItem) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(items))
	for i, it := range items {
		rows[i] = []interface{}{
			adjustmentID, it.ProductID, it.WarehouseID, it.StoreID,
			it.ExpectedQty, it.PhysicalQty, it.DifferenceQty, it.AdjustmentQty,
			it.UnitCost, it.LineTotal, it.Reason,
		}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"inventory_adjustment_items"},
		[]string{"adjustment_id", "product_id", "warehouse_id", "store_id",
			"expected_qty", "physical_qty", "difference_qty", "adjustment_qty",
			"unit_cost", "line_total", "reason"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert adjustment items: %w", err)
	}
	return nil
}

// GetAdjustmentBySession returns the posted adjustment document for a session.
func (r *Repository) GetAdjustmentBySession(ctx context.Context, sessionID int) (*Adjustment, error) {
	adj, err := r.getAdjustment(ctx, r.db, `WHERE a.session_id = $1`, sessionID)
	if err != nil {
		return nil, err
	}
	return adj, nil
}

// GetAdjustment returns an adjustment document by id with items.
func (r *Repository) GetAdjustment(ctx context.Context, id int) (*Adjustment, error) {
	adj, err := r.getAdjustment(ctx, r.db, `WHERE a.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return adj, nil
}

func (r *Repository) getAdjustment(ctx context.Context, db shared.DBPool, where string, arg interface{}) (*Adjustment, error) {
	var adj Adjustment
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		SELECT a.id, a.adjustment_number, a.session_id, COALESCE(o.session_number,''), a.status,
		       COALESCE(a.notes,''), COALESCE(a.created_by,0), a.created_at
		FROM inventory_adjustments a
		LEFT JOIN stock_opnames o ON o.id = a.session_id
		`+where, arg).
		Scan(&adj.ID, &adj.AdjustmentNumber, &adj.SessionID, &adj.SessionNumber, &adj.Status,
			&adj.Notes, &adj.CreatedBy, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdjustmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load adjustment: %w", err)
	}
	adj.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	if adj.CreatedBy != 0 {
		usernames, err := r.usernamesByIDs(ctx, []int{adj.CreatedBy})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve adjustment creator: %w", err)
		}
		adj.CreatedByName = usernames[adj.CreatedBy]
	}

	items, err := r.getAdjustmentItems(ctx, db, adj.ID)
	if err != nil {
		return nil, err
	}
	adj.Items = items
	adj.TotalDifference = 0
	adj.TotalAdjustment = 0
	for _, it := range items {
		adj.TotalDifference += it.DifferenceQty
		adj.TotalAdjustment += it.AdjustmentQty
	}
	return &adj, nil
}

func (r *Repository) getAdjustmentItems(ctx context.Context, db shared.DBPool, adjustmentID int) ([]AdjustmentItem, error) {
	rows, err := db.Query(ctx, `
		SELECT i.id, i.adjustment_id, i.product_id,
		       i.warehouse_id, i.store_id, i.expected_qty, i.physical_qty, i.difference_qty,
		       i.adjustment_qty, i.unit_cost, i.line_total, COALESCE(i.reason,'')
		FROM inventory_adjustment_items i
		WHERE i.adjustment_id = $1
		ORDER BY i.id ASC
	`, adjustmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load adjustment items: %w", err)
	}
	defer rows.Close()
	var out []AdjustmentItem
	productIDs := make([]int, 0)
	for rows.Next() {
		var it AdjustmentItem
		var warehouseID, storeID sql.NullInt64
		if err := rows.Scan(&it.ID, &it.AdjustmentID, &it.ProductID,
			&warehouseID, &storeID, &it.ExpectedQty, &it.PhysicalQty, &it.DifferenceQty,
			&it.AdjustmentQty, &it.UnitCost, &it.LineTotal, &it.Reason); err != nil {
			return nil, fmt.Errorf("failed to scan adjustment item: %w", err)
		}
		if warehouseID.Valid {
			v := int(warehouseID.Int64)
			it.WarehouseID = &v
		}
		if storeID.Valid {
			v := int(storeID.Int64)
			it.StoreID = &v
		}
		out = append(out, it)
		productIDs = append(productIDs, it.ProductID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	metas, err := r.productMetas(ctx, db, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve adjustment item products: %w", err)
	}
	for i := range out {
		if m, ok := metas[out[i].ProductID]; ok {
			out[i].ProductName = m.Name
			out[i].SKU = m.SKU
		}
	}
	return out, nil
}

// ListAdjustments returns paginated adjustment documents.
func (r *Repository) ListAdjustments(ctx context.Context, limit, offset int, status, search string) ([]Adjustment, int, error) {
	var where []string
	var args []interface{}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("a.status = $%d", len(args)))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf("(LOWER(a.adjustment_number) LIKE $%d OR LOWER(COALESCE(o.session_number,'')) LIKE $%d)", len(args), len(args)))
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM inventory_adjustments a
		LEFT JOIN stock_opnames o ON o.id = a.session_id
		`+whereSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count adjustments: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.adjustment_number, a.session_id, COALESCE(o.session_number,''), a.status,
		       COALESCE(a.notes,''), COALESCE(a.created_by,0), a.created_at,
		       COALESCE(SUM(i.difference_qty),0), COALESCE(SUM(i.adjustment_qty),0)
		FROM inventory_adjustments a
		LEFT JOIN stock_opnames o ON o.id = a.session_id
		LEFT JOIN inventory_adjustment_items i ON i.adjustment_id = a.id
		`+whereSQL+`
		GROUP BY a.id, o.session_number
		ORDER BY a.created_at DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)-1)+` OFFSET $`+fmt.Sprintf("%d", len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list adjustments: %w", err)
	}
	defer rows.Close()

	var out []Adjustment
	var userIDs []int
	for rows.Next() {
		var adj Adjustment
		var createdAt time.Time
		if err := rows.Scan(&adj.ID, &adj.AdjustmentNumber, &adj.SessionID, &adj.SessionNumber, &adj.Status,
			&adj.Notes, &adj.CreatedBy, &createdAt, &adj.TotalDifference, &adj.TotalAdjustment); err != nil {
			return nil, 0, fmt.Errorf("failed to scan adjustment: %w", err)
		}
		adj.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		out = append(out, adj)
		if adj.CreatedBy != 0 {
			userIDs = append(userIDs, adj.CreatedBy)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	usernames, err := r.usernamesByIDs(ctx, userIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to resolve adjustment creators: %w", err)
	}
	for i := range out {
		out[i].CreatedByName = usernames[out[i].CreatedBy]
	}
	return out, total, nil
}
