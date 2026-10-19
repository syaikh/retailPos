package shift

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// ActiveShiftFinder implements salereturn.ActiveShiftFinder (structural
// typing — no import of internal/salereturn; internal/wiring passes it to the
// salereturn service). A return may only be processed on the processing
// cashier's own open shift, because that shift's drawer funds the refund.
type ActiveShiftFinder struct {
	DB shared.DBPool
}

// ActiveShiftID returns the user's currently open shift id, or 0 when there
// is none (the caller treats 0 as "no open shift" and rejects the return).
func (f ActiveShiftFinder) ActiveShiftID(ctx context.Context, userID int) (int, error) {
	var id int
	err := f.DB.QueryRow(ctx, `
		SELECT id
		FROM shifts
		WHERE user_id = $1 AND status = 'open'
		ORDER BY opened_at DESC
		LIMIT 1
	`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to find active shift for user %d: %w", userID, err)
	}
	return id, nil
}
