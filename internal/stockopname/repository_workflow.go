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
func (r *Repository) LoadSnapshotProductsByIDs(ctx context.Context, q queryer, ids []int) ([]SessionItem, error) {
	query := `
		SELECT ps.product_id, p.name, p.sku, COALESCE(p.barcode, ''), COALESCE(u.name, 'pcs'), COALESCE(ps.quantity, 0)
		FROM product_stock ps
		JOIN products p ON p.id = ps.product_id
		LEFT JOIN units_of_measure u ON u.id = p.unit_of_measure_id
		WHERE ps.warehouse_id IS NULL AND ps.store_id IS NULL
		  AND p.status = 'active' AND p.deleted_at IS NULL`
	args := []interface{}{}
	if len(ids) > 0 {
		args = append(args, ids)
		query += ` AND p.id = ANY($1::int[])`
	}
	query += ` ORDER BY p.name ASC`
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot products: %w", err)
	}
	defer rows.Close()
	var items []SessionItem
	for rows.Next() {
		var it SessionItem
		if err := rows.Scan(&it.ProductID, &it.ProductName, &it.SKU, &it.Barcode, &it.UOMName, &it.OpeningQty); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot product: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ResolveScopeName returns a human-readable name for a scope reference, or an
// empty string when the referenced record does not exist. The table name is
// drawn only from the whitelisted switch below to avoid SQL injection.
func (r *Repository) ResolveScopeName(ctx context.Context, q queryer, scopeType string, scopeID int64) (string, error) {
	if scopeID <= 0 {
		return "", nil
	}
	var table string
	switch scopeType {
	case "store":
		table = "stores"
	case "warehouse":
		table = "warehouses"
	case "category":
		table = "categories"
	case "brand":
		table = "brands"
	case "supplier":
		table = "suppliers"
	case "product":
		table = "products"
	case "location":
		table = "storage_locations"
	default:
		return "", nil
	}
	var name string
	err := q.QueryRow(ctx, "SELECT name FROM "+table+" WHERE id = $1", scopeID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve scope name for %s #%d: %w", scopeType, scopeID, err)
	}
	return name, nil
}

// ScopeProductIDs returns the product universe covered by a scope.
func (r *Repository) ScopeProductIDs(ctx context.Context, q queryer, scope Scope) ([]int, error) {
	var query string
	var args []interface{}
	switch scope.ScopeType {
	case "store":
		query = `SELECT id FROM products WHERE store_id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "warehouse":
		query = `SELECT DISTINCT product_id FROM product_stock WHERE warehouse_id = $1`
	case "category":
		query = `SELECT id FROM products WHERE category_id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "brand":
		query = `SELECT id FROM products WHERE brand_id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "supplier":
		query = `SELECT DISTINCT product_id FROM product_suppliers WHERE supplier_id = $1`
	case "product":
		query = `SELECT id FROM products WHERE id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "location":
		query = `SELECT DISTINCT product_id FROM product_stock WHERE location_id = $1`
	case "manual":
		query = `SELECT id FROM products WHERE deleted_at IS NULL AND status = 'active'`
	default:
		return nil, ErrUnsupportedScope
	}
	if scope.ScopeType != "manual" {
		args = append(args, scope.ScopeID)
	}
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scope products for %s #%d: %w", scope.ScopeType, scope.ScopeID, err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan scope product: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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

// GetProductSKUs returns a product id -> sku map for the given ids.
func (r *Repository) GetProductSKUs(ctx context.Context, productIDs []int) (map[int]string, error) {
	out := make(map[int]string, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `SELECT id, sku FROM products WHERE id = ANY($1::int[])`, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load product skus: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var sku string
		if err := rows.Scan(&id, &sku); err != nil {
			return nil, fmt.Errorf("failed to scan product sku: %w", err)
		}
		out[id] = sku
	}
	return out, rows.Err()
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

func (r *Repository) getAdjustment(ctx context.Context, q queryer, where string, arg interface{}) (*Adjustment, error) {
	var adj Adjustment
	var createdAt time.Time
	err := q.QueryRow(ctx, `
		SELECT a.id, a.adjustment_number, a.session_id, COALESCE(o.session_number,''), a.status,
		       COALESCE(a.notes,''), COALESCE(a.created_by,0), COALESCE(u.username,''), a.created_at
		FROM inventory_adjustments a
		LEFT JOIN stock_opnames o ON o.id = a.session_id
		LEFT JOIN users u ON u.id = a.created_by
		`+where, arg).
		Scan(&adj.ID, &adj.AdjustmentNumber, &adj.SessionID, &adj.SessionNumber, &adj.Status,
			&adj.Notes, &adj.CreatedBy, &adj.CreatedByName, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdjustmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load adjustment: %w", err)
	}
	adj.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	items, err := r.getAdjustmentItems(ctx, q, adj.ID)
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

func (r *Repository) getAdjustmentItems(ctx context.Context, q queryer, adjustmentID int) ([]AdjustmentItem, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.adjustment_id, i.product_id, COALESCE(p.name,''), COALESCE(p.sku,''),
		       i.warehouse_id, i.store_id, i.expected_qty, i.physical_qty, i.difference_qty,
		       i.adjustment_qty, i.unit_cost, i.line_total, COALESCE(i.reason,'')
		FROM inventory_adjustment_items i
		LEFT JOIN products p ON p.id = i.product_id
		WHERE i.adjustment_id = $1
		ORDER BY i.id ASC
	`, adjustmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load adjustment items: %w", err)
	}
	defer rows.Close()
	var out []AdjustmentItem
	for rows.Next() {
		var it AdjustmentItem
		var warehouseID, storeID sql.NullInt64
		if err := rows.Scan(&it.ID, &it.AdjustmentID, &it.ProductID, &it.ProductName, &it.SKU,
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
	}
	return out, rows.Err()
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
		       COALESCE(a.notes,''), COALESCE(a.created_by,0), COALESCE(u.username,''), a.created_at,
		       COALESCE(SUM(i.difference_qty),0), COALESCE(SUM(i.adjustment_qty),0)
		FROM inventory_adjustments a
		LEFT JOIN stock_opnames o ON o.id = a.session_id
		LEFT JOIN users u ON u.id = a.created_by
		LEFT JOIN inventory_adjustment_items i ON i.adjustment_id = a.id
		`+whereSQL+`
		GROUP BY a.id, o.session_number, u.username
		ORDER BY a.created_at DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)-1)+` OFFSET $`+fmt.Sprintf("%d", len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list adjustments: %w", err)
	}
	defer rows.Close()

	var out []Adjustment
	for rows.Next() {
		var adj Adjustment
		var createdAt time.Time
		if err := rows.Scan(&adj.ID, &adj.AdjustmentNumber, &adj.SessionID, &adj.SessionNumber, &adj.Status,
			&adj.Notes, &adj.CreatedBy, &adj.CreatedByName, &createdAt, &adj.TotalDifference, &adj.TotalAdjustment); err != nil {
			return nil, 0, fmt.Errorf("failed to scan adjustment: %w", err)
		}
		adj.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		out = append(out, adj)
	}
	return out, total, rows.Err()
}
