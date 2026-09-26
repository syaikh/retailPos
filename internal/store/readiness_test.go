package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readinessRepo is a store.Repo test double that answers only the two calls
// Readiness makes (GetByID + ReadinessDetails); everything else panics if
// reached.
type readinessRepo struct {
	Repo
	store   *Store
	details *ReadinessDetails
	getErr  error
	detErr  error
}

func (r readinessRepo) GetByID(context.Context, int) (*Store, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.store, nil
}

func (r readinessRepo) ReadinessDetails(context.Context, int) (*ReadinessDetails, error) {
	if r.detErr != nil {
		return nil, r.detErr
	}
	return r.details, nil
}

func readyStore() *Store {
	return &Store{ID: 7, Name: "Store Bandung", Address: "Jl. Merdeka 1", Phone: "0221234", IsActive: true}
}

func readyDetails() *ReadinessDetails {
	return &ReadinessDetails{
		Staff: map[string]int{
			"manager": 1, "supervisor": 1, "cashier": 2, "inventory_staff": 1, "finance": 1,
		},
		ActiveProducts:    4500,
		ZeroStockProducts: 12,
		StorageLocations:  1,
	}
}

func TestReadiness_ReadyWhenEveryConditionMet(t *testing.T) {
	svc := NewService(readinessRepo{store: readyStore(), details: readyDetails()})

	r, err := svc.Readiness(context.Background(), 7)
	require.NoError(t, err)
	assert.True(t, r.Ready)
	assert.Empty(t, r.Blockers)
	assert.Equal(t, "Store Bandung", r.Name)
	assert.Equal(t, RequiredRoles, r.RequiredRoles)
	assert.Equal(t, 2, r.Staff["cashier"])
	assert.Equal(t, 4500, r.Catalog.ActiveProducts)
	assert.Equal(t, 1, r.StorageLocations)
	assert.True(t, r.AddressSet)
	assert.True(t, r.PhoneSet)
}

func TestReadiness_EveryMissingRequiredRoleIsABlocker(t *testing.T) {
	details := readyDetails()
	delete(details.Staff, "finance")
	delete(details.Staff, "supervisor")
	svc := NewService(readinessRepo{store: readyStore(), details: details})

	r, err := svc.Readiness(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, r.Ready)
	assert.Equal(t, []string{"staff.supervisor", "staff.finance"}, r.Blockers)
}

func TestReadiness_StoreProfileBlockers(t *testing.T) {
	store := &Store{ID: 7, Name: "New Store", Address: "  ", Phone: "", IsActive: false}
	details := readyDetails()
	details.StorageLocations = 0

	svc := NewService(readinessRepo{store: store, details: details})
	r, err := svc.Readiness(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, r.Ready)
	assert.Equal(t, []string{"store.inactive", "store.address", "store.phone", "storage_location"}, r.Blockers)
	assert.False(t, r.AddressSet)
	assert.False(t, r.PhoneSet)
}

func TestReadiness_CatalogBlockers(t *testing.T) {
	t.Run("no active products", func(t *testing.T) {
		details := readyDetails()
		details.ActiveProducts = 0
		details.ZeroStockProducts = 0
		svc := NewService(readinessRepo{store: readyStore(), details: details})

		r, err := svc.Readiness(context.Background(), 7)
		require.NoError(t, err)
		assert.Contains(t, r.Blockers, "catalog")
	})

	t.Run("every active product is out of stock", func(t *testing.T) {
		details := readyDetails()
		details.ZeroStockProducts = details.ActiveProducts
		svc := NewService(readinessRepo{store: readyStore(), details: details})

		r, err := svc.Readiness(context.Background(), 7)
		require.NoError(t, err)
		assert.Contains(t, r.Blockers, "catalog")
	})

	t.Run("partially out of stock stays informational", func(t *testing.T) {
		svc := NewService(readinessRepo{store: readyStore(), details: readyDetails()})

		r, err := svc.Readiness(context.Background(), 7)
		require.NoError(t, err)
		assert.NotContains(t, r.Blockers, "catalog")
	})
}

func TestReadiness_NonPositiveStaffCountsDoNotSatisfyARole(t *testing.T) {
	t.Run("zero count", func(t *testing.T) {
		details := readyDetails()
		details.Staff["finance"] = 0
		svc := NewService(readinessRepo{store: readyStore(), details: details})

		r, err := svc.Readiness(context.Background(), 7)
		require.NoError(t, err)
		assert.Contains(t, r.Blockers, "staff.finance")
		_, listed := r.Staff["finance"]
		assert.False(t, listed, "a zero-count role must not appear in the staff map")
	})

	t.Run("negative count", func(t *testing.T) {
		details := readyDetails()
		details.Staff["cashier"] = -1
		svc := NewService(readinessRepo{store: readyStore(), details: details})

		r, err := svc.Readiness(context.Background(), 7)
		require.NoError(t, err)
		assert.Contains(t, r.Blockers, "staff.cashier")
		_, listed := r.Staff["cashier"]
		assert.False(t, listed, "a negative-count role must not appear in the staff map")
	})
}

func TestReadiness_NilStaffMapIsTreatedAsEmpty(t *testing.T) {
	details := readyDetails()
	details.Staff = nil
	svc := NewService(readinessRepo{store: readyStore(), details: details})

	r, err := svc.Readiness(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, r.Ready, "no staff at all can never be ready")
	assert.NotNil(t, r.Staff)
	assert.Empty(t, r.Staff)
	for _, role := range RequiredRoles {
		assert.Contains(t, r.Blockers, "staff."+role)
	}
}

func TestReadinessDetails_UnwiredProvidersFailFast(t *testing.T) {
	ctx := context.Background()

	t.Run("staff counts", func(t *testing.T) {
		repo := NewRepository(nil)
		assert.PanicsWithValue(t,
			"store.Repository: StaffCountsProvider not wired (SetStaffCountsProvider)",
			func() { _, _ = repo.ReadinessDetails(ctx, 7) },
		)
	})

	t.Run("catalog stats", func(t *testing.T) {
		repo := NewRepository(nil)
		repo.SetStaffCountsProvider(stubStaffCounts{"manager": 1})
		assert.PanicsWithValue(t,
			"store.Repository: SellableStockProvider not wired (SetSellableStockProvider)",
			func() { _, _ = repo.ReadinessDetails(ctx, 7) },
		)
	})

	t.Run("storage location count", func(t *testing.T) {
		repo := NewRepository(nil)
		repo.SetStaffCountsProvider(stubStaffCounts{"manager": 1})
		repo.SetSellableStockProvider(stubStockStats{active: 10, zero: 3})
		assert.PanicsWithValue(t,
			"store.Repository: StorageLocationCountProvider not wired (SetStorageLocationCountProvider)",
			func() { _, _ = repo.ReadinessDetails(ctx, 7) },
		)
	})
}

func TestReadiness_MissingStoreIsNotFound(t *testing.T) {
	// The real repository wraps pgx.ErrNoRows; wrapRepoErr maps that exact
	// sentinel to ErrNotFound, so the double must surface it too.
	svc := NewService(readinessRepo{getErr: pgx.ErrNoRows})

	_, err := svc.Readiness(context.Background(), 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestReadiness_DetailLoadFailureIsInternal(t *testing.T) {
	svc := NewService(readinessRepo{store: readyStore(), detErr: errors.New("provider not wired")})

	_, err := svc.Readiness(context.Background(), 7)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInternal)
}
