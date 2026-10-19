package shift

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// ActiveShiftFinder is salereturn's gate that a return may only be processed
// on the processing cashier's own open shift: 0 means "no open shift" and the
// caller rejects the return with ErrShiftNotOpen.
func TestActiveShiftFinder_ActiveShiftID(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	finder := ActiveShiftFinder{DB: dbPool}

	t.Run("returns the user's open shift id", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, newTestRepo(t), userID)

		id, err := finder.ActiveShiftID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, id)
	})

	t.Run("returns the most recently opened shift when several rows are open", func(t *testing.T) {
		// OpenShift enforces one open shift per user, so a duplicate can only
		// arrive via a race or a direct write; the ORDER BY opened_at DESC is
		// what makes that case deterministic. Seed it with raw SQL on purpose.
		userID := insertTestUser(ctx, t, 1)
		var olderID, newerID int
		require.NoError(t, dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW() - INTERVAL '1 hour') RETURNING id
		`, userID).Scan(&olderID))
		require.NoError(t, dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, userID).Scan(&newerID))

		id, err := finder.ActiveShiftID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, newerID, id)
	})

	t.Run("returns zero when the user has no shift at all", func(t *testing.T) {
		id, err := finder.ActiveShiftID(ctx, 999999)
		require.NoError(t, err, "no open shift is a normal outcome, not an error")
		assert.Zero(t, id)
	})

	t.Run("returns zero when the user's only shift is closed", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		repo := newTestRepo(t)
		shift := createOpenShift(ctx, t, repo, userID)
		_, err := repo.CloseShift(ctx, shift.ID, userID, 100000, nil)
		require.NoError(t, err)

		id, err := finder.ActiveShiftID(ctx, userID)
		require.NoError(t, err)
		assert.Zero(t, id, "a closed shift must not fund a refund")
	})

	t.Run("returns zero for a foreign user's open shift", func(t *testing.T) {
		ownerID := insertTestUser(ctx, t, 1)
		createOpenShift(ctx, t, newTestRepo(t), ownerID)
		otherID := insertTestUser(ctx, t, 1)

		id, err := finder.ActiveShiftID(ctx, otherID)
		require.NoError(t, err)
		assert.Zero(t, id)
	})

	t.Run("wraps database errors with the user id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		t.Cleanup(func() { mock.Close() })
		boom := errors.New("connection lost")
		mock.ExpectQuery("SELECT id").WithArgs(42).WillReturnError(boom)

		f := ActiveShiftFinder{DB: mock}
		id, err := f.ActiveShiftID(ctx, 42)
		require.Error(t, err)
		assert.Zero(t, id)
		assert.ErrorIs(t, err, boom, "the sentinel must survive wrapping so callers can inspect it")
		assert.Contains(t, err.Error(), "failed to find active shift for user 42")
	})
}
