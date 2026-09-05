package shift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/user"
)

func TestShiftService_OpenShift_ValidatesOpeningBalance(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
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
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	svc := NewService(repo)
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	createOpenShift(ctx, t, repo, userID)

	shifts, err := svc.ExportShifts(ctx, ownership.Scope{UserID: &userID}, "", nil, "")
	require.NoError(t, err)
	assert.NotEmpty(t, shifts)
}

func TestShiftService_GetDiscrepancyThreshold(t *testing.T) {
	t.Run("returns default when settings is nil", func(t *testing.T) {
		repo := &Repository{}
		svc := &service{repo: repo}
		threshold := svc.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, defaultDiscrepancyThreshold, threshold)
	})

	t.Run("returns default when settings returns error", func(t *testing.T) {
		repo := &Repository{}
		svc := &service{repo: repo, settings: &mockSettings{err: assert.AnError}}
		threshold := svc.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, defaultDiscrepancyThreshold, threshold)
	})

	t.Run("returns default when key missing from settings", func(t *testing.T) {
		repo := &Repository{}
		svc := &service{repo: repo, settings: &mockSettings{data: map[string]string{}}}
		threshold := svc.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, defaultDiscrepancyThreshold, threshold)
	})

	t.Run("returns default when value is not a number", func(t *testing.T) {
		repo := &Repository{}
		svc := &service{repo: repo, settings: &mockSettings{data: map[string]string{"shift_discrepancy_threshold": "abc"}}}
		threshold := svc.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, defaultDiscrepancyThreshold, threshold)
	})

	t.Run("returns default when value is zero or negative", func(t *testing.T) {
		repo := &Repository{}
		svc := &service{repo: repo, settings: &mockSettings{data: map[string]string{"shift_discrepancy_threshold": "0"}}}
		threshold := svc.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, defaultDiscrepancyThreshold, threshold)

		svc2 := &service{repo: repo, settings: &mockSettings{data: map[string]string{"shift_discrepancy_threshold": "-100"}}}
		threshold2 := svc2.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, defaultDiscrepancyThreshold, threshold2)
	})

	t.Run("returns configured value from settings", func(t *testing.T) {
		repo := &Repository{}
		svc := &service{repo: repo, settings: &mockSettings{data: map[string]string{"shift_discrepancy_threshold": "75000"}}}
		threshold := svc.GetDiscrepancyThreshold(context.Background())
		assert.Equal(t, 75000, threshold)
	})
}

func TestShiftService_SetSettingsProvider(t *testing.T) {
	repo := &Repository{}
	svc := &service{repo: repo}
	assert.Nil(t, svc.settings)

	provider := &mockSettings{data: map[string]string{}}
	svc.SetSettingsProvider(provider)
	assert.NotNil(t, svc.settings)
}

type mockSettings struct {
	data map[string]string
	err  error
}

func (m *mockSettings) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[string]string)
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			result[k] = v
		}
	}
	return result, nil
}

func TestShiftService_FlagForReview(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("flag closed shift sets needs_review", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		closed, err := svc.CloseShift(ctx, shift.ID, userID, 200000, nil)
		require.NoError(t, err)
		assert.True(t, closed.NeedsReview)

		err = svc.FlagForReview(ctx, closed.ID)
		require.NoError(t, err)

		got, err := svc.GetShiftByID(ctx, ownership.Scope{}, closed.ID)
		require.NoError(t, err)
		assert.True(t, got.NeedsReview)
	})

	t.Run("flag open shift does nothing (no matching row)", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		err := svc.FlagForReview(ctx, shift.ID)
		require.NoError(t, err)

		got, err := svc.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
		require.NoError(t, err)
		assert.False(t, got.NeedsReview, "open shift must not be affected")
	})

	t.Run("flag non-existent shift does nothing", func(t *testing.T) {
		err := svc.FlagForReview(ctx, 999999)
		require.NoError(t, err)
	})
}
