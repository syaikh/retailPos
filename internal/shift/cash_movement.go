package shift

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type CashMovement struct {
	ID          int       `json:"id"`
	ShiftID     int       `json:"shift_id"`
	UserID      int       `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	Type        string    `json:"type"`
	Amount      int       `json:"amount"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CashMovementSummary struct {
	CashDrops int `json:"cash_drops"`
	PaidIns   int `json:"paid_ins"`
	PaidOuts  int `json:"paid_outs"`
	NetEffect int `json:"net_effect"`
}

var (
	ErrShiftClosed         = errors.New("cannot record movement on a closed shift")
	ErrInvalidMovementType = errors.New("invalid movement type")
	ErrNotShiftOwner       = errors.New("only the shift owner can record movements")
)

func (r *Repository) CreateCashMovement(ctx context.Context, tx pgx.Tx, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error) {
	if movementType != "cash_drop" && movementType != "paid_in" && movementType != "paid_out" {
		return nil, ErrInvalidMovementType
	}

	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM shifts WHERE id = $1 FOR UPDATE`, shiftID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("shift not found")
		}
		return nil, fmt.Errorf("failed to check shift status: %w", err)
	}
	if status != "open" {
		return nil, ErrShiftClosed
	}

	var ownerID int
	err = tx.QueryRow(ctx, `SELECT user_id FROM shifts WHERE id = $1`, shiftID).Scan(&ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shift owner: %w", err)
	}
	if ownerID != userID {
		return nil, ErrNotShiftOwner
	}

	var m CashMovement
	err = tx.QueryRow(ctx, `
		INSERT INTO cash_movements (shift_id, user_id, type, amount, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, shift_id, user_id, type, amount, description, created_at
	`, shiftID, userID, movementType, amount, description).Scan(
		&m.ID, &m.ShiftID, &m.UserID, &m.Type, &m.Amount, &m.Description, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cash movement: %w", err)
	}
	return &m, nil
}

func (r *Repository) ListCashMovements(ctx context.Context, shiftID int) ([]CashMovement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cm.id, cm.shift_id, cm.user_id, COALESCE(u.username, ''), cm.type, cm.amount, cm.description, cm.created_at
		FROM cash_movements cm
		LEFT JOIN users u ON u.id = cm.user_id
		WHERE cm.shift_id = $1
		ORDER BY cm.created_at ASC
	`, shiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cash movements: %w", err)
	}
	defer rows.Close()

	var movements []CashMovement
	for rows.Next() {
		var m CashMovement
		if err := rows.Scan(&m.ID, &m.ShiftID, &m.UserID, &m.Username, &m.Type, &m.Amount, &m.Description, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cash movement: %w", err)
		}
		movements = append(movements, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cash movements: %w", err)
	}
	return movements, nil
}

func (r *Repository) ShiftCashMovementSummary(ctx context.Context, tx pgx.Tx, shiftID int) (CashMovementSummary, error) {
	var s CashMovementSummary
	err := tx.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'cash_drop' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'paid_in' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'paid_out' THEN amount ELSE 0 END), 0)
		FROM cash_movements WHERE shift_id = $1
	`, shiftID).Scan(&s.CashDrops, &s.PaidIns, &s.PaidOuts)
	if err != nil {
		return CashMovementSummary{}, fmt.Errorf("failed to get cash movement summary: %w", err)
	}
	s.NetEffect = -s.CashDrops + s.PaidIns - s.PaidOuts
	return s, nil
}
