package sale

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression: AtomicGetOrCreateOpenCart previously scanned expired_at into *string,
// which fails for any open cart carrying a non-NULL expired_at (500 on add-to-cart),
// and reused expired carts instead of starting a fresh one.
func TestAtomicGetOrCreateOpenCart_ExpiredCartIsExpiredAndReplaced(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	cashierID := insertTestCashier(ctx, t)

	first, err := repo.AtomicGetOrCreateOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = dbPool.Exec(ctx, `DELETE FROM cart_items WHERE cart_session_id = $1`, first.ID)
		_, _ = dbPool.Exec(ctx, `DELETE FROM cart_sessions WHERE cashier_id = $1`, cashierID)
	})

	_, err = dbPool.Exec(ctx, `UPDATE cart_sessions SET expired_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, first.ID)
	require.NoError(t, err)

	second, err := repo.AtomicGetOrCreateOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID, "expired cart must not be reused")
	assert.Nil(t, second.ExpiredAt)

	var status string
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT status FROM cart_sessions WHERE id = $1`, first.ID).Scan(&status))
	assert.Equal(t, "expired", status)

	open, err := repo.GetOpenCartByCashier(ctx, cashierID)
	if assert.NoError(t, err) {
		assert.Equal(t, second.ID, open.ID, "open-cart lookup must return the fresh cart")
	}
}

func TestAtomicGetOrCreateOpenCart_LiveCartReusedWithFormattedExpiry(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	cashierID := insertTestCashier(ctx, t)

	first, err := repo.AtomicGetOrCreateOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = dbPool.Exec(ctx, `DELETE FROM cart_items WHERE cart_session_id = $1`, first.ID)
		_, _ = dbPool.Exec(ctx, `DELETE FROM cart_sessions WHERE cashier_id = $1`, cashierID)
	})

	future := time.Now().Add(30 * time.Minute)
	_, err = dbPool.Exec(ctx, `UPDATE cart_sessions SET expired_at = $1 WHERE id = $2`, future, first.ID)
	require.NoError(t, err)

	second, err := repo.AtomicGetOrCreateOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "live cart must be reused")
	if assert.NotNil(t, second.ExpiredAt) {
		parsed, err := time.Parse(time.RFC3339, *second.ExpiredAt)
		require.NoError(t, err)
		assert.WithinDuration(t, future, parsed, 2*time.Second)
	}
}

func TestGetOpenCartByCashier_HidesExpiredCart(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	cashierID := insertTestCashier(ctx, t)

	cart, err := repo.AtomicGetOrCreateOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = dbPool.Exec(ctx, `DELETE FROM cart_items WHERE cart_session_id = $1`, cart.ID)
		_, _ = dbPool.Exec(ctx, `DELETE FROM cart_sessions WHERE cashier_id = $1`, cashierID)
	})

	_, err = dbPool.Exec(ctx, `UPDATE cart_sessions SET expired_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, cart.ID)
	require.NoError(t, err)

	_, err = repo.GetOpenCartByCashier(ctx, cashierID)
	assert.True(t, errors.Is(err, ErrCartNotFound), "expired cart must not surface as the open cart, got: %v", err)
}
