package shift

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"retail-pos-system/internal/shared"
)

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existingID int
	err = tx.QueryRow(ctx, `
		SELECT id FROM shifts
		WHERE user_id = $1 AND status = 'open'
		LIMIT 1
	`, userID).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("user already has an open shift")
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
		return nil, fmt.Errorf("failed to open shift: %w", err)
	}

	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		shift.StoreID = &v
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit shift: %w", err)
	}

	shift.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &shift, nil
}

func (r *Repository) CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var shift Shift
	var storeID sql.NullInt64
	var storeName sql.NullString
	var openedAt, createdAt time.Time

	err = tx.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.store_id, st.name, s.status, s.opening_balance, s.opened_at, s.created_at
		FROM shifts s
		LEFT JOIN stores st ON st.id = s.store_id
		WHERE s.id = $1 AND s.user_id = $2 AND s.status = 'open'
		FOR UPDATE OF s
	`, shiftID, userID).Scan(
		&shift.ID, &shift.UserID, &storeID, &storeName, &shift.Status, &shift.OpeningBalance,
		&openedAt, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("shift not found or not open")
	}

	shift.OpenedAt = openedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	shift.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	if storeID.Valid {
		v := int(storeID.Int64)
		shift.StoreID = &v
	}
	if storeName.Valid {
		shift.StoreName = storeName.String
	}

	var summary ShiftSummary
	err = tx.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN LOWER(payment_method) = 'cash' THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN LOWER(payment_method) != 'cash' THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(total_amount), 0),
			COUNT(*)
		FROM sales
		WHERE shift_id = $1
		  AND status = 'completed'
	`, shiftID).Scan(
		&summary.TotalCashSales, &summary.TotalNonCashSales,
		&summary.TotalSales, &summary.TotalTransactions,
	)
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

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit shift close: %w", err)
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
	var storeName sql.NullString
	var notes sql.NullString
	var openedAt, createdAt, updatedAt time.Time
	var closedAt, reviewedAt sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.store_id, st.name, s.status, s.opening_balance, s.closing_balance,
		       s.cash_sales, s.non_cash_sales, s.total_sales, s.transaction_count,
		       s.discrepancy, s.notes, s.needs_review, s.reviewed_by, s.reviewed_at,
		       s.opened_at, s.closed_at, s.created_at, s.updated_at
		FROM shifts s
		LEFT JOIN stores st ON st.id = s.store_id
		WHERE s.user_id = $1 AND s.status = 'open'
		LIMIT 1
	`, userID).Scan(
		&shift.ID, &shift.UserID, &storeID, &storeName, &shift.Status, &shift.OpeningBalance,
		&closingBalance, &shift.CashSales, &shift.NonCashSales, &shift.TotalSales,
		&shift.TransactionCount, &discrepancy, &notes,
		&shift.NeedsReview, &reviewedBy, &reviewedAt,
		&openedAt, &closedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN LOWER(payment_method) = 'cash' THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN LOWER(payment_method) != 'cash' THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(total_amount), 0),
			COUNT(*)
		FROM sales
		WHERE shift_id = $1
		  AND status = 'completed'
	`, shift.ID).Scan(
		&shift.CashSales, &shift.NonCashSales,
		&shift.TotalSales, &shift.TransactionCount,
	); err != nil {
		log.Printf("failed to scan live sales for active shift %d: %v", shift.ID, err)
	}

	if storeID.Valid {
		v := int(storeID.Int64)
		shift.StoreID = &v
	}
	if storeName.Valid {
		shift.StoreName = storeName.String
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

func (r *Repository) ListShifts(ctx context.Context, userID *int, status string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if userID != nil {
		where += fmt.Sprintf(" AND s.user_id = $%d", argIdx)
		args = append(args, *userID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND s.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
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
		SELECT s.id, s.user_id, u.username, s.store_id, st.name, s.status,
		       s.opening_balance, s.closing_balance, s.cash_sales, s.non_cash_sales,
		       s.total_sales, s.transaction_count, s.discrepancy, s.notes,
		       s.needs_review, s.reviewed_by, s.reviewed_at,
		       s.opened_at, s.closed_at, s.created_at, s.updated_at
		FROM shifts s
		LEFT JOIN users u ON u.id = s.user_id
		LEFT JOIN stores st ON st.id = s.store_id
		WHERE %s
		ORDER BY s.%s %s
		LIMIT $%d OFFSET $%d
	`, where, sortBy, sortDir, argIdx, argIdx+1), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shifts: %w", err)
	}
	defer rows.Close()

	var shifts []Shift
	for rows.Next() {
		var s Shift
		var storeID, closingBalance, discrepancy, reviewedBy sql.NullInt64
		var storeName sql.NullString
		var notes sql.NullString
		var openedAt, createdAt, updatedAt time.Time
		var closedAt, reviewedAt sql.NullTime

		err := rows.Scan(
			&s.ID, &s.UserID, &s.Username, &storeID, &storeName, &s.Status,
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
		}
		if storeName.Valid {
			s.StoreName = storeName.String
		}
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

	return shifts, total, nil
}

func (r *Repository) GetShiftByID(ctx context.Context, shiftID int) (*Shift, error) {
	var s Shift
	var storeID, closingBalance, discrepancy, reviewedBy sql.NullInt64
	var storeName sql.NullString
	var notes sql.NullString
	var openedAt, createdAt, updatedAt time.Time
	var closedAt, reviewedAt sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.user_id, u.username, s.store_id, st.name, s.status,
		       s.opening_balance, s.closing_balance, s.cash_sales, s.non_cash_sales,
		       s.total_sales, s.transaction_count, s.discrepancy, s.notes,
		       s.needs_review, s.reviewed_by, s.reviewed_at,
		       s.opened_at, s.closed_at, s.created_at, s.updated_at
		FROM shifts s
		LEFT JOIN users u ON u.id = s.user_id
		LEFT JOIN stores st ON st.id = s.store_id
		WHERE s.id = $1
	`, shiftID).Scan(
		&s.ID, &s.UserID, &s.Username, &storeID, &storeName, &s.Status,
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
	}
	if storeName.Valid {
		s.StoreName = storeName.String
	}
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
	_, err := r.db.Exec(ctx, `
		UPDATE shifts
		SET needs_review = false,
		    reviewed_by = $1,
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, reviewerID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to review shift: %w", err)
	}

	return r.GetShiftByID(ctx, shiftID)
}
