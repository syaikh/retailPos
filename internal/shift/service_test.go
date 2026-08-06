package shift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/shared"
)

func TestShiftService_OpenShift_ValidatesOpeningBalance(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("zero balance returns error", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift, err := svc.OpenShift(ctx, userID, nil, 0)
		assert.Error(t, err)
		assert.Nil(t, shift)
		assert.Contains(t, err.Error(), "must be greater than zero")
	})

	t.Run("negative balance returns error", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift, err := svc.OpenShift(ctx, userID, nil, -50000)
		assert.Error(t, err)
		assert.Nil(t, shift)
		assert.Contains(t, err.Error(), "must be greater than zero")
	})

	t.Run("positive balance succeeds", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift, err := svc.OpenShift(ctx, userID, nil, 100000)
		require.NoError(t, err)
		require.NotNil(t, shift)
		assert.Equal(t, 100000, shift.OpeningBalance)
		assert.Equal(t, "open", shift.Status)
	})
}

func TestShiftService_CloseShift_ValidatesClosingBalance(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("negative closing balance returns error", func(t *testing.T) {
		shift, err := svc.CloseShift(ctx, 1, 1, -1, nil)
		assert.Error(t, err)
		assert.Nil(t, shift)
		assert.Contains(t, err.Error(), "closing balance must not be negative")
	})
}

func TestShiftService_ReviewShift(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("review shift marks as reviewed", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift, err := svc.OpenShift(ctx, userID, nil, 100000)
		require.NoError(t, err)

		closed, err := svc.CloseShift(ctx, shift.ID, userID, 200000, nil)
		require.NoError(t, err)
		assert.True(t, closed.NeedsReview)

		reviewerID := insertTestUser(ctx, t, 2)
		reviewed, err := svc.ReviewShift(ctx, closed.ID, reviewerID)
		require.NoError(t, err)
		assert.False(t, reviewed.NeedsReview)
		require.NotNil(t, reviewed.ReviewedBy)
		assert.Equal(t, reviewerID, *reviewed.ReviewedBy)
	})

	t.Run("review non-existent shift returns error", func(t *testing.T) {
		_, err := svc.ReviewShift(ctx, 999999, 1)
		assert.Error(t, err)
	})
}

func TestShiftService_GetActiveShift(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	createOpenShift(ctx, t, repo, userID)

	shift, err := svc.GetActiveShift(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, shift)
	assert.Equal(t, "open", shift.Status)
}

func TestShiftService_GetShiftByID(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	got, err := svc.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
	require.NoError(t, err)
	assert.Equal(t, shift.ID, got.ID)
}

func TestShiftService_ListShifts(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	createOpenShift(ctx, t, repo, userID)

	shifts, total, err := svc.ListShifts(ctx, ownership.Scope{UserID: &userID}, "", nil, "", 10, 0, "opened_at", "DESC")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.NotEmpty(t, shifts)
}

func TestShiftService_AuditShift(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	_, _, err := svc.AuditShift(ctx, shift.ID)
	require.NoError(t, err)
}

func TestShiftService_ExportShifts(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	createOpenShift(ctx, t, repo, userID)

	shifts, err := svc.ExportShifts(ctx, ownership.Scope{UserID: &userID}, "", nil, "")
	require.NoError(t, err)
	assert.NotEmpty(t, shifts)
}

func TestShiftService_CloseAll(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("returns closed shift ids", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		createOpenShift(ctx, t, repo, userID)

		ids, err := svc.CloseAll(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, ids, 1)
	})
}
