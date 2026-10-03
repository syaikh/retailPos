package supplier

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// govSuffix gives every fixture a unique code/SKU across runs: TruncateTestData
// intentionally does not truncate suppliers or product_suppliers (they are not
// in its table list), so a fixed code would collide with a previous run's live
// row on the code-active partial unique index.
func govSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func createGovSupplier(ctx context.Context, t *testing.T, repo *Repository, code string, active bool) *Supplier {
	t.Helper()
	s := &Supplier{Name: "Gov " + code, Code: code, IsActive: active}
	require.NoError(t, repo.Create(ctx, s))
	require.Equal(t, 1, s.Version, "a new supplier starts at version 1")
	return s
}

func insertGovUser(ctx context.Context, t *testing.T) int {
	t.Helper()
	var id int
	username := "gov_user_" + govSuffix()
	err := dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ($1, $2, 'hash', 1)
		RETURNING id
	`, username, username+"@test.com").Scan(&id)
	require.NoError(t, err)
	return id
}

func linkProductToSupplier(ctx context.Context, t *testing.T, productID, supplierID int) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO product_suppliers (product_id, supplier_id, unit_cost)
		VALUES ($1, $2, 1000)
	`, productID, supplierID)
	require.NoError(t, err)
}

func insertOpenPO(ctx context.Context, t *testing.T, supplierID, userID int, status string) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO purchase_orders (po_number, supplier_id, store_id, status, created_by, updated_by)
		VALUES ($1, $2, 1, $3, $4, $4)
	`, "PO-GOV-"+govSuffix(), supplierID, status, userID)
	require.NoError(t, err)
}

func insertConsignmentArrangement(ctx context.Context, t *testing.T, supplierID, userID int, status string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO consignment_arrangements (supplier_id, store_id, status, created_by)
		VALUES ($1, 1, $2, $3)
		RETURNING id
	`, supplierID, status, userID).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertConsignmentStock(ctx context.Context, t *testing.T, arrangementID, supplierID, productID int, available int) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO consignment_stock (product_id, supplier_id, arrangement_id, store_id, available_qty)
		VALUES ($1, $2, $3, 1, $4)
	`, productID, supplierID, arrangementID, available)
	require.NoError(t, err)
}

func TestSupplierGovernance_CodeReuseAfterSoftDelete(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	code := "SUP-GOV-REUSE-" + govSuffix()

	first := createGovSupplier(ctx, t, repo, code, true)

	// A second live row with the same code is refused by the partial unique
	// index installed by 060_supplier_governance.sql.
	dup := &Supplier{Name: "Dup", Code: code, IsActive: true}
	err := repo.Create(ctx, dup)
	require.Error(t, err, "a duplicate live code must be rejected")

	// Soft-delete frees the code: GetByCode only sees live rows, and the
	// partial index only covers deleted_at IS NULL.
	require.NoError(t, repo.Delete(ctx, first.ID))
	_, err = repo.GetByCode(ctx, code)
	assert.ErrorIs(t, err, ErrSupplierNotFound)

	second := createGovSupplier(ctx, t, repo, code, true)
	assert.NotEqual(t, first.ID, second.ID)
	got, err := repo.GetByCode(ctx, code)
	require.NoError(t, err)
	assert.Equal(t, second.ID, got.ID)
}

func TestSupplierGovernance_VersionGuard(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	s := createGovSupplier(ctx, t, repo, "SUP-GOV-VER-"+govSuffix(), true)
	require.Equal(t, 1, s.Version)

	loaded, err := repo.GetByID(ctx, s.ID)
	require.NoError(t, err)
	require.Equal(t, 1, loaded.Version)

	// Correct expected version succeeds and bumps the stored version.
	loaded.Name = "Renamed"
	require.NoError(t, repo.Update(ctx, loaded))
	assert.Equal(t, 2, loaded.Version)

	// Replaying the stale version is refused as a conflict, not a 404.
	stale := &Supplier{ID: s.ID, Name: "Stale", Code: s.Code, IsActive: true, Version: 1}
	err = repo.Update(ctx, stale)
	assert.ErrorIs(t, err, ErrSupplierVersionConflict)

	// Version 0 opts out of the check (legacy callers / seeder) and still bumps.
	loaded.Name = "NoVersionCheck"
	loaded.Version = 0
	require.NoError(t, repo.Update(ctx, loaded))
	assert.Equal(t, 3, loaded.Version)

	// A version-guarded update of a missing supplier is reported as not found.
	missing := &Supplier{ID: -1, Name: "Missing", Code: "X", IsActive: true, Version: 5}
	assert.ErrorIs(t, repo.Update(ctx, missing), ErrSupplierVersionConflict)
}

func TestSupplierGovernance_ProvenanceStamped(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	creator := insertGovUser(ctx, t)
	editor := insertGovUser(ctx, t)

	s := &Supplier{Name: "Provenance", Code: "SUP-GOV-PROV-" + govSuffix(), IsActive: true, CreatedBy: &creator, UpdatedBy: &creator}
	require.NoError(t, repo.Create(ctx, s))

	got, err := repo.GetByID(ctx, s.ID)
	require.NoError(t, err)
	require.NotNil(t, got.CreatedBy)
	assert.Equal(t, creator, *got.CreatedBy)
	require.NotNil(t, got.UpdatedBy)
	assert.Equal(t, creator, *got.UpdatedBy)

	got.UpdatedBy = &editor
	require.NoError(t, repo.Update(ctx, got))

	after, err := repo.GetByID(ctx, s.ID)
	require.NoError(t, err)
	require.NotNil(t, after.UpdatedBy)
	assert.Equal(t, editor, *after.UpdatedBy)
}

func TestSupplierService_DeleteGuard(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	svc := NewService(repo)

	t.Run("blocked by a product link", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEL-LINK-"+govSuffix(), true)
		productID := insertTestProduct(ctx, t, "SKU-GOV-DEL-"+govSuffix(), "Gov Del Product", 1000)
		linkProductToSupplier(ctx, t, productID, s.ID)

		err := svc.Delete(ctx, s.ID)
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, 1, inUse.Usage.ProductLinks)
		assert.False(t, inUse.Deactivating)

		// Still present after a blocked delete.
		_, err = repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
	})

	t.Run("blocked by an open purchase order", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEL-PO-"+govSuffix(), true)
		userID := insertGovUser(ctx, t)
		insertOpenPO(ctx, t, s.ID, userID, "confirmed")

		err := svc.Delete(ctx, s.ID)
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, 1, inUse.Usage.OpenPurchaseOrders)
	})

	t.Run("blocked by a live consignment", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEL-CONS-"+govSuffix(), true)
		userID := insertGovUser(ctx, t)
		insertConsignmentArrangement(ctx, t, s.ID, userID, "active")

		err := svc.Delete(ctx, s.ID)
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, 1, inUse.Usage.ActiveConsignments)
	})

	t.Run("succeeds when unreferenced", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEL-OK-"+govSuffix(), true)
		require.NoError(t, svc.Delete(ctx, s.ID))

		_, err := repo.GetByID(ctx, s.ID)
		assert.ErrorIs(t, err, ErrSupplierNotFound)
	})

	t.Run("missing supplier is not found, not a silent success", func(t *testing.T) {
		err := svc.Delete(ctx, -1)
		assert.ErrorIs(t, err, ErrSupplierNotFound)
	})
}

func TestSupplierService_DeactivateGuard(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	svc := NewService(repo)

	t.Run("product links alone do not block deactivation", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEACT-LINK-"+govSuffix(), true)
		productID := insertTestProduct(ctx, t, "SKU-GOV-DEACT-"+govSuffix(), "Gov Deact Product", 1000)
		linkProductToSupplier(ctx, t, productID, s.ID)

		loaded, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		loaded.IsActive = false
		require.NoError(t, svc.Update(ctx, loaded))

		after, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		assert.False(t, after.IsActive)
	})

	t.Run("an open purchase order blocks deactivation", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEACT-PO-"+govSuffix(), true)
		userID := insertGovUser(ctx, t)
		insertOpenPO(ctx, t, s.ID, userID, "draft")

		loaded, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		loaded.IsActive = false

		err = svc.Update(ctx, loaded)
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.True(t, inUse.Deactivating)
		assert.Equal(t, 1, inUse.Usage.OpenPurchaseOrders)

		after, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		assert.True(t, after.IsActive, "a blocked deactivation must not change the row")
	})

	t.Run("an ended arrangement holding stock still blocks", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-DEACT-STOCK-"+govSuffix(), true)
		userID := insertGovUser(ctx, t)
		arrangementID := insertConsignmentArrangement(ctx, t, s.ID, userID, "ended")
		productID := insertTestProduct(ctx, t, "SKU-GOV-STOCK-"+govSuffix(), "Gov Stock Product", 1000)
		insertConsignmentStock(ctx, t, arrangementID, s.ID, productID, 3)

		loaded, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		loaded.IsActive = false

		err = svc.Update(ctx, loaded)
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, 1, inUse.Usage.ActiveConsignments)
	})
}

func TestSupplierService_BulkAllOrNothing(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	svc := NewService(repo)

	t.Run("bulk delete refuses the whole batch when any row is referenced", func(t *testing.T) {
		clean := createGovSupplier(ctx, t, repo, "SUP-GOV-BDEL-OK-"+govSuffix(), true)
		blocked := createGovSupplier(ctx, t, repo, "SUP-GOV-BDEL-BLK-"+govSuffix(), true)
		productID := insertTestProduct(ctx, t, "SKU-GOV-BDEL-"+govSuffix(), "Gov BDel Product", 1000)
		linkProductToSupplier(ctx, t, productID, blocked.ID)

		_, err := svc.BulkDelete(ctx, []int{clean.ID, blocked.ID})
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Contains(t, inUse.BlockedIDs, blocked.ID)
		assert.NotContains(t, inUse.BlockedIDs, clean.ID)
		assert.Equal(t, 1, inUse.Usage.ProductLinks)

		// The clean sibling must survive: all-or-nothing.
		_, err = repo.GetByID(ctx, clean.ID)
		require.NoError(t, err, "a clean supplier in a blocked batch must not be deleted")
	})

	t.Run("bulk deactivate ignores already-inactive rows", func(t *testing.T) {
		blocked := createGovSupplier(ctx, t, repo, "SUP-GOV-BDEACT-BLK-"+govSuffix(), true)
		alreadyInactive := createGovSupplier(ctx, t, repo, "SUP-GOV-BDEACT-INACTIVE-"+govSuffix(), false)
		userID := insertGovUser(ctx, t)
		insertOpenPO(ctx, t, blocked.ID, userID, "confirmed")
		insertOpenPO(ctx, t, alreadyInactive.ID, userID, "confirmed")

		_, err := svc.BulkUpdate(ctx, []int{blocked.ID, alreadyInactive.ID}, false, &userID)
		var inUse *InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, []int{blocked.ID}, inUse.BlockedIDs)
	})
}

func TestSupplierService_GetUsage(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepo(t)
	svc := NewService(repo)

	t.Run("reports the cross-module breakdown", func(t *testing.T) {
		s := createGovSupplier(ctx, t, repo, "SUP-GOV-USAGE-"+govSuffix(), true)
		userID := insertGovUser(ctx, t)
		productID := insertTestProduct(ctx, t, "SKU-GOV-USAGE-"+govSuffix(), "Gov Usage Product", 1000)
		linkProductToSupplier(ctx, t, productID, s.ID)
		insertOpenPO(ctx, t, s.ID, userID, "partial_received")
		insertConsignmentArrangement(ctx, t, s.ID, userID, "active")

		usage, err := svc.GetUsage(ctx, s.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, usage.ProductLinks)
		assert.Equal(t, 1, usage.OpenPurchaseOrders)
		assert.Equal(t, 1, usage.ActiveConsignments)
		assert.Equal(t, 3, usage.Total())
		assert.Equal(t, 2, usage.InFlight())
	})

	t.Run("missing supplier is not found", func(t *testing.T) {
		_, err := svc.GetUsage(ctx, -1)
		assert.ErrorIs(t, err, ErrSupplierNotFound)
	})
}
