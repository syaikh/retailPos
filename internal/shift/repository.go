package shift

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/shared"
)

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique-constraint
// violation (23505), surfaced when the partial unique index rejects a second
// open shift for the same user.
const uniqueViolationCode = "23505"

type Repository struct {
	db                     shared.DBPool
	summaryProvider        SalesSummaryProvider
	storeNameProvider      StoreNameProvider
	usernameProvider       UsernameProvider
	paymentBreakdownProvider PaymentBreakdownProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// SetSalesSummaryProvider wires the sale-owned implementation of the
// SalesSummaryProvider port. It MUST be called before any summary read runs
// (close, live sales); an unwired repository fails fast at the read point.
func (r *Repository) SetSalesSummaryProvider(p SalesSummaryProvider) {
	r.summaryProvider = p
}

// SetStoreNameProvider wires the store-owned implementation of the
// StoreNameProvider port (ADR §2.4). It MUST be called before any read that
// needs a store name (close, active, list, get-by-id); an unwired repository
// fails fast at the read point.
func (r *Repository) SetStoreNameProvider(p StoreNameProvider) {
	r.storeNameProvider = p
}

// SetUsernameProvider wires the user-owned implementation of the
// UsernameProvider port (ADR §2.4). It MUST be called before any read that
// needs a username (list, get-by-id); an unwired repository fails fast at the
// read point.
func (r *Repository) SetUsernameProvider(p UsernameProvider) {
	r.usernameProvider = p
}

func (r *Repository) SetPaymentBreakdownProvider(p PaymentBreakdownProvider) {
	r.paymentBreakdownProvider = p
}

func (r *Repository) OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	shift, err := r.OpenShiftTx(ctx, tx, userID, storeID, openingBalance)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit shift: %w", err)
	}
	return shift, nil
}

// OpenShiftTx opens a shift within an existing transaction.
func (r *Repository) OpenShiftTx(ctx context.Context, tx pgx.Tx, userID int, storeID *int, openingBalance int) (*Shift, error) {
	var existingID int
	err := tx.QueryRow(ctx, `
		SELECT id FROM shifts
		WHERE user_id = $1 AND status = 'open'
		LIMIT 1
	`, userID).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("user already has an open shift")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing open shift: %w", err)
	}

	var shift Shift
	var storeIDVal sql.NullInt64
	if storeID != nil {
		storeIDVal = sql.NullInt64{Int64: int64(*storeID), Valid: true}
	}

	var openedAt, createdAt, updatedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO shifts (user_id, store_id, opening_balance, status, opened_at)
		VALUES ($1, $2, $3, 'open', NOW())
		RETURNING id, user_id, store_id, status, opening_balance, opened_at, created_at, updated_at
	`, userID, storeIDVal, openingBalance).Scan(
		&shift.ID, &shift.UserID, &storeIDVal, &shift.Status, &shift.OpeningBalance,
		&openedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, fmt.Errorf("user already has an open shift")
		}
		return nil, fmt.Errorf("failed to open shift: %w", err)
	}

	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		shift.StoreID = &v
	}

	shift.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &shift, nil
}

func (r *Repository) shiftSalesSummary(ctx context.Context, shiftID int) (shared.ShiftSaleSummary, error) {
	if r.summaryProvider == nil {
		return shared.ShiftSaleSummary{}, errors.New("shift repository: sales summary provider not wired; call SetShiftSalesSummaryProvider")
	}
	return r.summaryProvider.ShiftSummary(ctx, r.db, shiftID)
}

func (r *Repository) shiftSalesSummaryInTx(ctx context.Context, tx pgx.Tx, shiftID int) (shared.ShiftSaleSummary, error) {
	if r.summaryProvider == nil {
		return shared.ShiftSaleSummary{}, errors.New("shift repository: sales summary provider not wired; call SetShiftSalesSummaryProvider")
	}
	return r.summaryProvider.ShiftSummaryInTx(ctx, tx, shiftID)
}

func (r *Repository) storeNamesByIDs(ctx context.Context, ids []int) (map[int]string, error) {
	if r.storeNameProvider == nil {
		return nil, errors.New("shift repository: store name provider not wired; call SetStoreNameProvider")
	}
	return r.storeNameProvider.StoreNamesByIDs(ctx, r.db, ids)
}

func (r *Repository) usernamesByIDs(ctx context.Context, ids []int) (map[int]string, error) {
	if r.usernameProvider == nil {
		return nil, errors.New("shift repository: username provider not wired; call SetUsernameProvider")
	}
	return r.usernameProvider.UsernamesByIDs(ctx, r.db, ids)
}

func (r *Repository) CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	shift, err := r.CloseShiftTx(ctx, tx, shiftID, userID, closingBalance, notes)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit shift close: %w", err)
	}
	return shift, nil
}

// CloseShiftTx closes a shift within an existing transaction.
func (r *Repository) CloseShiftTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	var shift Shift
	var storeID sql.NullInt64
	var openedAt, createdAt time.Time

	err := tx.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.store_id, s.status, s.opening_balance, s.opened_at, s.created_at
		FROM shifts s
		WHERE s.id = $1 AND s.user_id = $2 AND s.status = 'open'
		FOR UPDATE OF s
	`, shiftID, userID).Scan(
		&shift.ID, &shift.UserID, &storeID, &shift.Status, &shift.OpeningBalance,
		&openedAt, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("shift not found or not open")
	}

	shift.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	var openCarts int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM cart_sessions WHERE shift_id = $1 AND status = 'open'`, shiftID).Scan(&openCarts)
	if err != nil {
		return nil, fmt.Errorf("failed to check for active carts: %w", err)
	}
	if openCarts > 0 {
		return nil, fmt.Errorf("cannot close shift: %d active cart(s) still open", openCarts)
	}

	if storeID.Valid {
		v := int(storeID.Int64)
		shift.StoreID = &v
		storeNames, err := r.storeNamesByIDs(ctx, []int{v})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve store name: %w", err)
		}
		shift.StoreName = storeNames[v]
	}

	summary, err := r.shiftSalesSummaryInTx(ctx, tx, shiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate shift summary: %w", err)
	}

	discrepancy := closingBalance - shift.OpeningBalance - summary.TotalCashSales

	const discrepancyThreshold = 50000
	needsReview := discrepancy < -discrepancyThreshold || discrepancy > discrepancyThreshold

	var closedAt, updatedAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE shifts
		SET status = 'closed',
		    closing_balance = $1,
		    cash_sales = $2,
		    non_cash_sales = $3,
		    total_sales = $4,
		    transaction_count = $5,
		    discrepancy = $6,
		    notes = $7,
		    needs_review = $8,
		    closed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $9
		RETURNING closed_at, updated_at
	`, closingBalance, summary.TotalCashSales, summary.TotalNonCashSales,
		summary.TotalSales, summary.TotalTransactions, discrepancy, notes, needsReview, shiftID).Scan(
		&closedAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to close shift: %w", err)
	}

	shift.ClosingBalance = &closingBalance
	shift.CashSales = summary.TotalCashSales
	shift.NonCashSales = summary.TotalNonCashSales
	shift.TotalSales = summary.TotalSales
	shift.TransactionCount = summary.TotalTransactions
	shift.Discrepancy = &discrepancy
	shift.Notes = notes
	shift.NeedsReview = needsReview
	shift.Status = "closed"
	shift.ClosedAt = closedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	return &shift, nil
}

func (r *Repository) GetActiveShiftByUserID(ctx context.Context, userID int) (*Shift, error) {
	var shift Shift
	var storeID, closingBalance, discrepancy, reviewedBy sql.NullInt64
	var notes sql.NullString
	var openedAt, createdAt, updatedAt time.Time
	var closedAt, reviewedAt sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.store_id, s.status, s.opening_balance, s.closing_balance,
		       s.cash_sales, s.non_cash_sales, s.total_sales, s.transaction_count,
		       s.discrepancy, s.notes, s.needs_review, s.reviewed_by, s.reviewed_at,
		       s.opened_at, s.closed_at, s.created_at, s.updated_at
		FROM shifts s
		WHERE s.user_id = $1 AND s.status = 'open'
		LIMIT 1
	`, userID).Scan(
		&shift.ID, &shift.UserID, &storeID, &shift.Status, &shift.OpeningBalance,
		&closingBalance, &shift.CashSales, &shift.NonCashSales, &shift.TotalSales,
		&shift.TransactionCount, &discrepancy, &notes,
		&shift.NeedsReview, &reviewedBy, &reviewedAt,
		&openedAt, &closedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if shift.CashSales == 0 && shift.TransactionCount == 0 {
		summary, err := r.shiftSalesSummary(ctx, shift.ID)
		if err != nil {
			slog.Error("failed to scan live sales for active shift", "shift_id", shift.ID, "error", err)
		} else {
			shift.CashSales = summary.TotalCashSales
			shift.NonCashSales = summary.TotalNonCashSales
			shift.TotalSales = summary.TotalSales
			shift.TransactionCount = summary.TotalTransactions
		}
	}

	if storeID.Valid {
		v := int(storeID.Int64)
		shift.StoreID = &v
		storeNames, err := r.storeNamesByIDs(ctx, []int{v})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve store name: %w", err)
		}
		shift.StoreName = storeNames[v]
	}
	if closingBalance.Valid {
		v := int(closingBalance.Int64)
		shift.ClosingBalance = &v
	}
	if discrepancy.Valid {
		v := int(discrepancy.Int64)
		shift.Discrepancy = &v
	}
	if notes.Valid {
		shift.Notes = &notes.String
	}
	if reviewedBy.Valid {
		v := int(reviewedBy.Int64)
		shift.ReviewedBy = &v
	}
	if reviewedAt.Valid {
		shift.ReviewedAt = reviewedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if closedAt.Valid {
		shift.ClosedAt = closedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}

	shift.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &shift, nil
}

func (r *Repository) ListShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if ownerID, restricted := scope.OwnID(); restricted {
		where += fmt.Sprintf(" AND s.user_id = $%d", argIdx)
		args = append(args, ownerID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND s.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if needsReview != nil {
		where += fmt.Sprintf(" AND s.needs_review = $%d", argIdx)
		args = append(args, *needsReview)
		argIdx++
	}
	if discrepancyFilter != "" {
		switch discrepancyFilter {
		case "balanced":
			where += fmt.Sprintf(" AND COALESCE(s.discrepancy, 0) = $%d", argIdx)
			args = append(args, 0)
			argIdx++
		case "surplus":
			where += fmt.Sprintf(" AND s.discrepancy > $%d", argIdx)
			args = append(args, 0)
			argIdx++
		case "shortage":
			where += fmt.Sprintf(" AND s.discrepancy < $%d", argIdx)
			args = append(args, 0)
			argIdx++
		}
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM shifts s WHERE %s
	`, where), countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count shifts: %w", err)
	}

	allowedSort := map[string]bool{"opened_at": true, "closed_at": true, "status": true}
	if !allowedSort[sortBy] {
		sortBy = "opened_at"
	}
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "DESC"
	}

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT s.id, s.user_id, s.store_id, s.status,
		       s.opening_balance, s.closing_balance, s.cash_sales, s.non_cash_sales,
		       s.total_sales, s.transaction_count, s.discrepancy, s.notes,
		       s.needs_review, s.reviewed_by, s.reviewed_at,
		       s.opened_at, s.closed_at, s.created_at, s.updated_at
		FROM shifts s
		WHERE %s
		ORDER BY s.%s %s
		LIMIT $%d OFFSET $%d
	`, where, sortBy, sortDir, argIdx, argIdx+1), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shifts: %w", err)
	}
	defer rows.Close()

	var shifts []Shift
	var storeIDs, userIDs []int
	for rows.Next() {
		var s Shift
		var storeID, closingBalance, discrepancy, reviewedBy sql.NullInt64
		var notes sql.NullString
		var openedAt, createdAt, updatedAt time.Time
		var closedAt, reviewedAt sql.NullTime

		err := rows.Scan(
			&s.ID, &s.UserID, &storeID, &s.Status,
			&s.OpeningBalance, &closingBalance, &s.CashSales, &s.NonCashSales,
			&s.TotalSales, &s.TransactionCount, &discrepancy, &notes,
			&s.NeedsReview, &reviewedBy, &reviewedAt,
			&openedAt, &closedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan shift: %w", err)
		}

		if storeID.Valid {
			v := int(storeID.Int64)
			s.StoreID = &v
			storeIDs = append(storeIDs, v)
		}
		userIDs = append(userIDs, s.UserID)
		if closingBalance.Valid {
			v := int(closingBalance.Int64)
			s.ClosingBalance = &v
		}
		if discrepancy.Valid {
			v := int(discrepancy.Int64)
			s.Discrepancy = &v
		}
		if notes.Valid {
			s.Notes = &notes.String
		}
		if reviewedBy.Valid {
			v := int(reviewedBy.Int64)
			s.ReviewedBy = &v
		}
		if reviewedAt.Valid {
			s.ReviewedAt = reviewedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if closedAt.Valid {
			s.ClosedAt = closedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}

		s.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		shifts = append(shifts, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate shifts: %w", err)
	}

	storeNames, err := r.storeNamesByIDs(ctx, storeIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to resolve store names: %w", err)
	}
	usernames, err := r.usernamesByIDs(ctx, userIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to resolve usernames: %w", err)
	}
	for i := range shifts {
		if shifts[i].StoreID != nil {
			shifts[i].StoreName = storeNames[*shifts[i].StoreID]
		}
		shifts[i].Username = usernames[shifts[i].UserID]
	}

	return shifts, total, nil
}

func (r *Repository) GetShiftByID(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
	var s Shift
	var storeID, closingBalance, discrepancy, reviewedBy sql.NullInt64
	var notes sql.NullString
	var openedAt, createdAt, updatedAt time.Time
	var closedAt, reviewedAt sql.NullTime

	query := `
		SELECT s.id, s.user_id, s.store_id, s.status,
		       s.opening_balance, s.closing_balance, s.cash_sales, s.non_cash_sales,
		       s.total_sales, s.transaction_count, s.discrepancy, s.notes,
		       s.needs_review, s.reviewed_by, s.reviewed_at,
		       s.opened_at, s.closed_at, s.created_at, s.updated_at
		FROM shifts s
		WHERE s.id = $1`
	args := []interface{}{shiftID}
	if ownerID, restricted := scope.OwnID(); restricted {
		query += " AND s.user_id = $2"
		args = append(args, ownerID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.UserID, &storeID, &s.Status,
		&s.OpeningBalance, &closingBalance, &s.CashSales, &s.NonCashSales,
		&s.TotalSales, &s.TransactionCount, &discrepancy, &notes,
		&s.NeedsReview, &reviewedBy, &reviewedAt,
		&openedAt, &closedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if storeID.Valid {
		v := int(storeID.Int64)
		s.StoreID = &v
		storeNames, err := r.storeNamesByIDs(ctx, []int{v})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve store name: %w", err)
		}
		s.StoreName = storeNames[v]
	}
	usernames, err := r.usernamesByIDs(ctx, []int{s.UserID})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve username: %w", err)
	}
	s.Username = usernames[s.UserID]
	if closingBalance.Valid {
		v := int(closingBalance.Int64)
		s.ClosingBalance = &v
	}
	if discrepancy.Valid {
		v := int(discrepancy.Int64)
		s.Discrepancy = &v
	}
	if notes.Valid {
		s.Notes = &notes.String
	}
	if reviewedBy.Valid {
		v := int(reviewedBy.Int64)
		s.ReviewedBy = &v
	}
	if reviewedAt.Valid {
		s.ReviewedAt = reviewedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if closedAt.Valid {
		s.ClosedAt = closedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}

	s.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}

func (r *Repository) ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE shifts
		SET needs_review = false,
		    reviewed_by = $1,
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2 AND needs_review = true AND status = 'closed'
	`, reviewerID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to review shift: %w", err)
	}
	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("shift not pending review or not found")
	}

	return r.GetShiftByID(ctx, ownership.Scope{}, shiftID)
}

func (r *Repository) FlagForReview(ctx context.Context, shiftID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE shifts
		SET needs_review = true,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'closed'
	`, shiftID)
	if err != nil {
		return fmt.Errorf("failed to flag shift for review: %w", err)
	}
	return nil
}

func (r *Repository) GetShiftWithLiveSales(ctx context.Context, shiftID int) (*Shift, int, error) {
	shift, err := r.GetShiftByID(ctx, ownership.Scope{}, shiftID)
	if err != nil {
		return nil, 0, err
	}

	summary, err := r.shiftSalesSummary(ctx, shiftID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query live cash sales: %w", err)
	}

	return shift, summary.TotalCashSales, nil
}

func (r *Repository) GetShiftReportData(ctx context.Context, shiftID int) (*ShiftReportData, error) {
	shift, err := r.GetShiftByID(ctx, ownership.Scope{}, shiftID)
	if err != nil {
		return nil, err
	}

	report := &ShiftReportData{Shift: *shift}

	if shift.ClosedAt != "" && shift.OpenedAt != "" {
		opened, err := time.Parse(time.RFC3339, shift.OpenedAt)
		if err == nil {
			closed, err := time.Parse(time.RFC3339, shift.ClosedAt)
			if err == nil {
				report.DurationMinutes = int(closed.Sub(opened).Minutes())
			}
		}
	}

	if r.paymentBreakdownProvider != nil {
		breakdown, err := r.paymentBreakdownProvider.PaymentMethodBreakdown(ctx, r.db, shiftID)
		if err == nil {
			report.PaymentBreakdown = breakdown
		}
	}

	if r.usernameProvider != nil {
		names, err := r.usernameProvider.UsernamesByIDs(ctx, r.db, []int{shift.UserID})
		if err == nil {
			if name, ok := names[shift.UserID]; ok {
				report.Username = name
			}
		}
	}
	if r.storeNameProvider != nil && shift.StoreID != nil {
		names, err := r.storeNameProvider.StoreNamesByIDs(ctx, r.db, []int{*shift.StoreID})
		if err == nil {
			if name, ok := names[*shift.StoreID]; ok {
				report.StoreName = name
			}
		}
	}

	return report, nil
}

func (r *Repository) ListOpenShiftsOlderThan(ctx context.Context, threshold time.Time) ([]Shift, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.user_id, s.store_id, s.status, s.opening_balance, s.opened_at
		FROM shifts s
		WHERE s.status = 'open' AND s.opened_at < $1
		ORDER BY s.opened_at ASC
	`, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to list old open shifts: %w", err)
	}
	defer rows.Close()

	var shifts []Shift
	for rows.Next() {
		var s Shift
		var storeID sql.NullInt64
		var openedAt time.Time
		if err := rows.Scan(&s.ID, &s.UserID, &storeID, &s.Status, &s.OpeningBalance, &openedAt); err != nil {
			return nil, fmt.Errorf("failed to scan old open shift: %w", err)
		}
		if storeID.Valid {
			sid := int(storeID.Int64)
			s.StoreID = &sid
		}
		s.OpenedAt = openedAt.Format(time.RFC3339)
		shifts = append(shifts, s)
	}
	return shifts, rows.Err()
}
