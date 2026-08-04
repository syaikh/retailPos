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

type Repository struct {
	db shared.DBPool
}

// scopeNameExpr resolves a human-readable name for a session's primary scope
// from the id stored in scope_id. It is a correlated subquery over the outer
// stock_opnames row, so it must only be appended to SELECTs that read from
// stock_opnames without an alias.
const scopeNameExpr = `
	COALESCE(CASE scope_type
		WHEN 'store' THEN (SELECT name FROM stores WHERE id = scope_id)
		WHEN 'warehouse' THEN (SELECT name FROM warehouses WHERE id = scope_id)
		WHEN 'category' THEN (SELECT name FROM categories WHERE id = scope_id)
		WHEN 'brand' THEN (SELECT name FROM brands WHERE id = scope_id)
		WHEN 'supplier' THEN (SELECT name FROM suppliers WHERE id = scope_id)
		WHEN 'product' THEN (SELECT name FROM products WHERE id = scope_id)
		WHEN 'manual' THEN scope_name
	END, '') AS scope_name`

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) GetNextSessionNumber(ctx context.Context) (string, error) {
	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('so_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next stock opname number: %w", err)
	}
	year := time.Now().In(shared.JakartaLocation()).Year()
	return fmt.Sprintf("SO-%d-%06d", year, seq), nil
}

// LoadSnapshotProducts returns the general-stock snapshot for all active products.
func (r *Repository) LoadSnapshotProducts(ctx context.Context) ([]SessionItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ps.product_id, p.name, p.sku, COALESCE(p.barcode, ''), COALESCE(u.name, 'pcs'), COALESCE(ps.quantity, 0)
		FROM product_stock ps
		JOIN products p ON p.id = ps.product_id
		LEFT JOIN units_of_measure u ON u.id = p.unit_of_measure_id
		WHERE ps.warehouse_id IS NULL AND ps.store_id IS NULL
		  AND p.status = 'active' AND p.deleted_at IS NULL
		ORDER BY p.name ASC
	`)
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

// ListAssignableUsers returns active users eligible for assignment to a stock
// opname session (counters and supervisors). Superadmins are excluded as they
// sit outside the day-to-day assignment flow.
func (r *Repository) ListAssignableUsers(ctx context.Context, search string) ([]AssignableUser, error) {
	query := `
		SELECT u.id, u.username, u.email, u.role_id, r.name
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL AND u.is_active = true
		  AND r.name IN ('cashier', 'staff', 'manager', 'admin')
		  AND ($1 = '' OR u.username ILIKE '%' || $1 || '%' OR u.email ILIKE '%' || $1 || '%')
		ORDER BY u.username ASC`
	rows, err := r.db.Query(ctx, query, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignable users: %w", err)
	}
	defer rows.Close()

	var users []AssignableUser
	for rows.Next() {
		var u AssignableUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.RoleID, &u.RoleName); err != nil {
			return nil, fmt.Errorf("failed to scan assignable user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserRoleName returns the role name for a user, or empty string when the
// user does not exist or is inactive.
func (r *Repository) GetUserRoleName(ctx context.Context, userID int) (string, error) {
	var roleName string
	err := r.db.QueryRow(ctx, `
		SELECT r.name
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL AND u.is_active = true`, userID).Scan(&roleName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrAssigneeNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}
	return roleName, nil
}

func (r *Repository) CreateSession(ctx context.Context, tx pgx.Tx, s *Session) error {
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO stock_opnames (session_number, scope_type, scope_id, warehouse_id, store_id, location_id, blind_count, status, created_by, scope_name, title, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, ''), NULLIF($12, ''))
		RETURNING id, created_at, updated_at
	`, s.SessionNumber, s.ScopeType, s.ScopeID, s.WarehouseID, s.StoreID, s.LocationID, s.BlindCount, s.Status, s.CreatedBy,
		s.ScopeName, s.Title, s.Notes).
		Scan(&s.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert stock opname session: %w", err)
	}
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) InsertSessionItems(ctx context.Context, tx pgx.Tx, sessionID int, items []SessionItem) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(items))
	for i, it := range items {
		rows[i] = []interface{}{
			sessionID, it.ProductID, it.OpeningQty, it.ProductName, it.SKU, it.Barcode, it.UOMName,
		}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"stock_opname_items"},
		[]string{"stock_opname_id", "product_id", "opening_qty", "product_name", "sku", "barcode", "uom_name"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert stock opname items: %w", err)
	}
	return nil
}

func (r *Repository) GetSession(ctx context.Context, id int) (*Session, error) {
	s, err := r.getSessionHeader(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	items, err := r.getItems(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	s.Items = items
	scopes, err := r.LoadSessionScopes(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	s.Scopes = scopes
	assignments, err := r.ListAssignments(ctx, id)
	if err != nil {
		return nil, err
	}
	s.Assignments = assignments
	return s, nil
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *Repository) getSessionHeader(ctx context.Context, q queryer, id int) (*Session, error) {
	var s Session
	var warehouseID, storeID, locationID, approvedBy, openedBy, verifiedBy, postedBy, closedBy sql.NullInt64
	var approvedAt, cancelledAt, createdAt, updatedAt, openedAt, verifiedAt, postedAt, closedAt sql.NullTime
	err := q.QueryRow(ctx, `
		SELECT id, session_number, scope_type, scope_id, warehouse_id, store_id, location_id, blind_count, status,
		       COALESCE(title,''), COALESCE(notes,''),
		       created_by, approved_by, approved_at, cancelled_at, created_at, updated_at,
		       opened_by, opened_at, verified_by, verified_at,
		       posted_by, posted_at, closed_by, closed_at,
		       total_difference, total_adjustment,`+scopeNameExpr+`
		FROM stock_opnames
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&s.ID, &s.SessionNumber, &s.ScopeType, &s.ScopeID, &warehouseID, &storeID, &locationID, &s.BlindCount,
		&s.Status, &s.Title, &s.Notes, &s.CreatedBy, &approvedBy, &approvedAt, &cancelledAt, &createdAt, &updatedAt,
		&openedBy, &openedAt, &verifiedBy, &verifiedAt,
		&postedBy, &postedAt, &closedBy, &closedAt,
		&s.TotalDifference, &s.TotalAdjustment, &s.ScopeName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		s.WarehouseID = &v
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		s.StoreID = &v
	}
	if locationID.Valid {
		v := int(locationID.Int64)
		s.LocationID = &v
	}
	assignAuditCols(&s.ApprovedBy, approvedBy, &s.ApprovedAt, approvedAt)
	assignAuditCols(&s.OpenedBy, openedBy, &s.OpenedAt, openedAt)
	assignAuditCols(&s.VerifiedBy, verifiedBy, &s.VerifiedAt, verifiedAt)
	assignAuditCols(&s.PostedBy, postedBy, &s.PostedAt, postedAt)
	assignAuditCols(&s.ClosedBy, closedBy, &s.ClosedAt, closedAt)
	if cancelledAt.Valid {
		s.CancelledAt = cancelledAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if createdAt.Valid {
		s.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if updatedAt.Valid {
		s.UpdatedAt = updatedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	return &s, nil
}

// assignAuditCols copies a nullable user id / timestamp pair onto the session
// audit fields when the database value is present.
func assignAuditCols(userField **int, user sql.NullInt64, timeField *string, ts sql.NullTime) {
	if user.Valid {
		v := int(user.Int64)
		*userField = &v
	}
	if ts.Valid {
		*timeField = ts.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
}

func (r *Repository) getItems(ctx context.Context, q queryer, sessionID int) ([]SessionItem, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.stock_opname_id, i.product_id, i.product_name, i.sku, i.barcode, i.uom_name,
		       i.opening_qty, i.expected_qty, i.physical_qty, i.difference_qty, i.adjustment_qty, i.status,
		       COALESCE((SELECT MAX(c.count_sequence) FROM stock_opname_counts c WHERE c.stock_opname_item_id = i.id), 0),
		       (SELECT MAX(c.counted_by) FROM stock_opname_counts c WHERE c.stock_opname_item_id = i.id),
		       (SELECT MAX(c.counted_at) FROM stock_opname_counts c WHERE c.stock_opname_item_id = i.id)
		FROM stock_opname_items i
		WHERE i.stock_opname_id = $1
		ORDER BY i.product_name ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session items: %w", err)
	}
	defer rows.Close()

	var items []SessionItem
	for rows.Next() {
		var it SessionItem
		var lastCountedBy sql.NullInt64
		var lastCountedAt sql.NullTime
		if err := rows.Scan(&it.ID, &it.StockOpnameID, &it.ProductID, &it.ProductName, &it.SKU, &it.Barcode, &it.UOMName,
			&it.OpeningQty, &it.ExpectedQty, &it.PhysicalQty, &it.DifferenceQty, &it.AdjustmentQty, &it.Status,
			&it.CountSequence, &lastCountedBy, &lastCountedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session item: %w", err)
		}
		if lastCountedBy.Valid {
			v := int(lastCountedBy.Int64)
			it.LastCountedBy = &v
		}
		if lastCountedAt.Valid {
			it.LastCountedAt = lastCountedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *Repository) ListSessions(ctx context.Context, limit, offset int, status, search string) ([]Session, int, error) {
	var where []string
	var args []interface{}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		where = append(where, fmt.Sprintf("LOWER(session_number) LIKE $%d", len(args)))
	}
	where = append(where, "deleted_at IS NULL")
	whereSQL := strings.Join(where, " AND ")

	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM stock_opnames WHERE `+whereSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, `
		SELECT id, session_number, scope_type, scope_id, warehouse_id, store_id, location_id, blind_count, status,
		       COALESCE(title,''), COALESCE(notes,''), created_by,
		       approved_by, approved_at, cancelled_at, created_at, updated_at,
		       opened_by, opened_at, verified_by, verified_at,
		       posted_by, posted_at, closed_by, closed_at,
		       total_difference, total_adjustment,`+scopeNameExpr+`
		FROM stock_opnames
		WHERE `+whereSQL+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)-1)+` OFFSET $`+fmt.Sprintf("%d", len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var warehouseID, storeID, locationID, approvedBy, openedBy, verifiedBy, postedBy, closedBy sql.NullInt64
		var approvedAt, cancelledAt, createdAt, updatedAt, openedAt, verifiedAt, postedAt, closedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.SessionNumber, &s.ScopeType, &s.ScopeID, &warehouseID, &storeID, &locationID, &s.BlindCount,
			&s.Status, &s.Title, &s.Notes, &s.CreatedBy, &approvedBy, &approvedAt, &cancelledAt, &createdAt, &updatedAt,
			&openedBy, &openedAt, &verifiedBy, &verifiedAt,
			&postedBy, &postedAt, &closedBy, &closedAt,
			&s.TotalDifference, &s.TotalAdjustment, &s.ScopeName); err != nil {
			return nil, 0, fmt.Errorf("failed to scan session: %w", err)
		}
		if warehouseID.Valid {
			v := int(warehouseID.Int64)
			s.WarehouseID = &v
		}
		if storeID.Valid {
			v := int(storeID.Int64)
			s.StoreID = &v
		}
		if locationID.Valid {
			v := int(locationID.Int64)
			s.LocationID = &v
		}
		assignAuditCols(&s.ApprovedBy, approvedBy, &s.ApprovedAt, approvedAt)
		assignAuditCols(&s.OpenedBy, openedBy, &s.OpenedAt, openedAt)
		assignAuditCols(&s.VerifiedBy, verifiedBy, &s.VerifiedAt, verifiedAt)
		assignAuditCols(&s.PostedBy, postedBy, &s.PostedAt, postedAt)
		assignAuditCols(&s.ClosedBy, closedBy, &s.ClosedAt, closedAt)
		if cancelledAt.Valid {
			s.CancelledAt = cancelledAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if createdAt.Valid {
			s.CreatedAt = createdAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if updatedAt.Valid {
			s.UpdatedAt = updatedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		sessions = append(sessions, s)
	}
	return sessions, total, rows.Err()
}

func (r *Repository) CancelSession(ctx context.Context, id, userID int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE stock_opnames
		SET status = 'cancelled', cancelled_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status IN ('draft', 'open', 'counting', 'needs_recount') AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("failed to cancel session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		exists, err := r.sessionExists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return ErrInvalidState
	}
	return nil
}

func (r *Repository) GetSessionStatus(ctx context.Context, id int) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `SELECT status FROM stock_opnames WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to load session status: %w", err)
	}
	return status, nil
}

// GetSessionBroadcastMeta returns the session number and store_id in a single
// query, used to build real-time status event payloads.
func (r *Repository) GetSessionBroadcastMeta(ctx context.Context, id int) (string, *int, error) {
	var number string
	var storeID sql.NullInt64
	err := r.db.QueryRow(ctx, `
		SELECT session_number, store_id FROM stock_opnames WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&number, &storeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil, ErrNotFound
		}
		return "", nil, fmt.Errorf("failed to load session broadcast meta: %w", err)
	}
	if !storeID.Valid {
		return number, nil, nil
	}
	v := int(storeID.Int64)
	return number, &v, nil
}

// GetWarehouseStoreID returns the store_id linked to a warehouse, or nil when
// the warehouse does not exist or has no linked store.
func (r *Repository) GetWarehouseStoreID(ctx context.Context, warehouseID int) (*int, error) {
	var storeID sql.NullInt64
	err := r.db.QueryRow(ctx, `SELECT store_id FROM warehouses WHERE id = $1`, warehouseID).Scan(&storeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load warehouse store: %w", err)
	}
	if !storeID.Valid {
		return nil, nil
	}
	v := int(storeID.Int64)
	return &v, nil
}

func (r *Repository) CountPendingItems(ctx context.Context, sessionID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM stock_opname_items WHERE stock_opname_id = $1 AND status <> 'counted'
	`, sessionID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending items: %w", err)
	}
	return n, nil
}

func (r *Repository) sessionExists(ctx context.Context, id int) (bool, error) {
	var one int
	err := r.db.QueryRow(ctx, `SELECT 1 FROM stock_opnames WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&one)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	return true, nil
}

// UpdateStatus is a guarded status transition (WHERE status = currentStatus).
func (r *Repository) UpdateStatus(ctx context.Context, id int, currentStatus, nextStatus string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE stock_opnames SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3 AND deleted_at IS NULL
	`, nextStatus, id, currentStatus)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		exists, err := r.sessionExists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return ErrInvalidState
	}
	return nil
}

func (r *Repository) InsertAssignment(ctx context.Context, tx pgx.Tx, sessionID, userID int, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO stock_opname_assignments (stock_opname_id, user_id, role)
		VALUES ($1, $2, $3)
	`, sessionID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to insert assignment: %w", err)
	}
	return nil
}

func (r *Repository) UpdateAssignmentRole(ctx context.Context, tx pgx.Tx, sessionID, assignmentID int, role string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE stock_opname_assignments SET role = $1 WHERE id = $2 AND stock_opname_id = $3
	`, role, assignmentID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAssignmentNotFound
	}
	return nil
}

// GetAssignmentUserID returns the user assigned to an assignment, or
// ErrAssignmentNotFound when it does not exist.
func (r *Repository) GetAssignmentUserID(ctx context.Context, sessionID, assignmentID int) (int, error) {
	var userID int
	err := r.db.QueryRow(ctx, `
		SELECT user_id FROM stock_opname_assignments WHERE id = $1 AND stock_opname_id = $2`,
		assignmentID, sessionID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrAssignmentNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get assignment user: %w", err)
	}
	return userID, nil
}

func (r *Repository) ListAssignments(ctx context.Context, sessionID int) ([]Assignment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.stock_opname_id, a.user_id, a.role, a.assigned_at, COALESCE(u.username, '')
		FROM stock_opname_assignments a
		LEFT JOIN users u ON u.id = a.user_id
		WHERE a.stock_opname_id = $1
		ORDER BY a.assigned_at ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var out []Assignment
	for rows.Next() {
		var a Assignment
		var assignedAt time.Time
		if err := rows.Scan(&a.ID, &a.StockOpnameID, &a.UserID, &a.Role, &assignedAt, &a.Username); err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		a.AssignedAt = assignedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) IsAssigned(ctx context.Context, sessionID, userID int) (bool, error) {
	var one int
	err := r.db.QueryRow(ctx, `
		SELECT 1 FROM stock_opname_assignments WHERE stock_opname_id = $1 AND user_id = $2 LIMIT 1
	`, sessionID, userID).Scan(&one)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check assignment: %w", err)
	}
	return true, nil
}

func (r *Repository) IsCounterAssigned(ctx context.Context, sessionID, userID int) (bool, error) {
	var one int
	err := r.db.QueryRow(ctx, `
		SELECT 1 FROM stock_opname_assignments
		WHERE stock_opname_id = $1 AND user_id = $2 AND role = 'counter' LIMIT 1
	`, sessionID, userID).Scan(&one)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check counter assignment: %w", err)
	}
	return true, nil
}

// GetItemForCount returns an item joined with its session state.
func (r *Repository) GetItemForCount(ctx context.Context, itemID int) (*SessionItem, *Session, error) {
	var it SessionItem
	var s Session
	var createdAt, updatedAt time.Time
	var warehouseID sql.NullInt64
	err := r.db.QueryRow(ctx, `
		SELECT i.id, i.stock_opname_id, i.product_id, i.product_name, i.sku, i.barcode, i.uom_name,
		       i.opening_qty, i.physical_qty, i.status,
		       o.id, o.session_number, o.scope_type, o.scope_id, o.warehouse_id, o.blind_count, o.status,
		       o.created_at, o.updated_at
		FROM stock_opname_items i
		JOIN stock_opnames o ON o.id = i.stock_opname_id
		WHERE i.id = $1 AND o.deleted_at IS NULL
	`, itemID).Scan(&it.ID, &it.StockOpnameID, &it.ProductID, &it.ProductName, &it.SKU, &it.Barcode, &it.UOMName,
		&it.OpeningQty, &it.PhysicalQty, &it.Status,
		&s.ID, &s.SessionNumber, &s.ScopeType, &s.ScopeID, &warehouseID, &s.BlindCount, &s.Status,
		&createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrItemNotFound
		}
		return nil, nil, fmt.Errorf("failed to load item: %w", err)
	}
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		s.WarehouseID = &v
	}
	return &it, &s, nil
}

func (r *Repository) NextCountSequence(ctx context.Context, tx pgx.Tx, itemID int) (int, error) {
	var seq int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(count_sequence), 0) + 1 FROM stock_opname_counts WHERE stock_opname_item_id = $1
	`, itemID).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("failed to compute count sequence: %w", err)
	}
	return seq, nil
}

func (r *Repository) SaveCount(ctx context.Context, tx pgx.Tx, itemID, seq int, qty float64, userID int, remarks string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO stock_opname_counts (stock_opname_item_id, count_sequence, physical_qty, counted_by, remarks)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
	`, itemID, seq, qty, userID, remarks)
	if err != nil {
		return fmt.Errorf("failed to insert count record: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE stock_opname_items SET physical_qty = $2, status = 'counted', updated_at = NOW()
		WHERE id = $1
	`, itemID, qty)
	if err != nil {
		return fmt.Errorf("failed to update item physical qty: %w", err)
	}
	return nil
}

func (r *Repository) LockItemForCount(ctx context.Context, tx pgx.Tx, itemID int) error {
	_, err := tx.Exec(ctx, `SELECT id FROM stock_opname_items WHERE id = $1 FOR UPDATE`, itemID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrItemNotFound
		}
		return fmt.Errorf("failed to lock item: %w", err)
	}
	return nil
}

func (r *Repository) GetCountHistory(ctx context.Context, itemID int) ([]CountRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.stock_opname_item_id, c.count_sequence, c.physical_qty, c.counted_by,
		       COALESCE(u.username, ''), c.counted_at, COALESCE(c.remarks, '')
		FROM stock_opname_counts c
		LEFT JOIN users u ON u.id = c.counted_by
		WHERE c.stock_opname_item_id = $1
		ORDER BY c.count_sequence ASC
	`, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to load count history: %w", err)
	}
	defer rows.Close()

	var out []CountRecord
	for rows.Next() {
		var c CountRecord
		var countedAt time.Time
		if err := rows.Scan(&c.ID, &c.StockOpnameItemID, &c.CountSequence, &c.PhysicalQty, &c.CountedBy,
			&c.CountedByUser, &countedAt, &c.Remarks); err != nil {
			return nil, fmt.Errorf("failed to scan count record: %w", err)
		}
		c.CountedAt = countedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		out = append(out, c)
	}
	return out, rows.Err()
}

// --- approval transaction support ---

type approvalItem struct {
	ID         int
	ProductID  int
	PhysicalQy float64
	UnitCost   float64
}

func (r *Repository) LockSessionForApproval(ctx context.Context, tx pgx.Tx, id int) (*Session, error) {
	var s Session
	var warehouseID, storeID, locationID sql.NullInt64
	err := tx.QueryRow(ctx, `
		SELECT id, session_number, status, blind_count, warehouse_id, store_id, location_id FROM stock_opnames
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&s.ID, &s.SessionNumber, &s.Status, &s.BlindCount, &warehouseID, &storeID, &locationID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to lock session: %w", err)
	}
	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		s.WarehouseID = &v
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		s.StoreID = &v
	}
	if locationID.Valid {
		v := int(locationID.Int64)
		s.LocationID = &v
	}
	return &s, nil
}

func (r *Repository) LoadApprovalItems(ctx context.Context, tx pgx.Tx, sessionID int) ([]approvalItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT i.id, i.product_id, i.physical_qty, COALESCE(p.cost, 0)
		FROM stock_opname_items i
		LEFT JOIN products p ON p.id = i.product_id
		WHERE i.stock_opname_id = $1
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load approval items: %w", err)
	}
	defer rows.Close()

	var items []approvalItem
	for rows.Next() {
		var it approvalItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.PhysicalQy, &it.UnitCost); err != nil {
			return nil, fmt.Errorf("failed to scan approval item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *Repository) LockStockForProducts(ctx context.Context, tx pgx.Tx, productIDs []int) (map[int]int, error) {
	stock := make(map[int]int, len(productIDs))
	if len(productIDs) == 0 {
		return stock, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT product_id, quantity FROM product_stock
		WHERE product_id = ANY($1::int[]) AND warehouse_id IS NULL AND store_id IS NULL
		FOR UPDATE
	`, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to lock product stock: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			return nil, fmt.Errorf("failed to scan product stock: %w", err)
		}
		stock[pid] = qty
	}
	return stock, rows.Err()
}

func (r *Repository) UpdateItemAdjustment(ctx context.Context, tx pgx.Tx, itemID int, expected, diff, adj float64, reason string) error {
	_, err := tx.Exec(ctx, `
		UPDATE stock_opname_items
		SET expected_qty = $2, difference_qty = $3, adjustment_qty = $4, reason = NULLIF($5, ''), updated_at = NOW()
		WHERE id = $1
	`, itemID, expected, diff, adj, reason)
	if err != nil {
		return fmt.Errorf("failed to update item adjustment: %w", err)
	}
	return nil
}

func (r *Repository) UpdateProductStock(ctx context.Context, tx pgx.Tx, productID, newQty int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL
	`, newQty, productID)
	if err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, NOW())
		`, productID, newQty)
		if err != nil {
			return fmt.Errorf("failed to insert product stock: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE products SET stock = $1, updated_at = NOW() WHERE id = $2`, newQty, productID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock column: %w", err)
	}
	return nil
}

type movementRow struct {
	ProductID      int
	QuantityChange int
	Notes          string
}

func (r *Repository) InsertMovements(ctx context.Context, tx pgx.Tx, sessionID, userID int, rowsData []movementRow) error {
	if len(rowsData) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(rowsData))
	for i, m := range rowsData {
		rows[i] = []interface{}{m.ProductID, m.QuantityChange, MovementTypeStockOpname, sessionID, "stock_opnames", userID, m.Notes}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"inventory_movements"},
		[]string{"product_id", "quantity_change", "type", "reference_id", "reference_table", "user_id", "notes"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert inventory movements: %w", err)
	}
	return nil
}
