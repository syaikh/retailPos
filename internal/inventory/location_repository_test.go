package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func createTestWarehouse(t *testing.T, ctx context.Context, code string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO warehouses (name, code) VALUES ($1, $2) RETURNING id`,
		"Test Warehouse "+code, code).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestStore(t *testing.T, ctx context.Context, name string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO stores (name) VALUES ($1) RETURNING id`,
		"Test Store "+name).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestLocation(t *testing.T, ctx context.Context, code, name string, warehouseID, storeID *int, active bool) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO storage_locations (code, name, warehouse_id, store_id, is_active)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		code, name, warehouseID, storeID, active).Scan(&id)
	require.NoError(t, err)
	return id
}

func globalStock(t *testing.T, ctx context.Context, productID int) (int, bool) {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, productID).Scan(&qty)
	if err != nil {
		return 0, false
	}
	return qty, true
}

func rackStock(t *testing.T, ctx context.Context, productID, locationID int) (int, bool) {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND location_id = $2
	`, productID, locationID).Scan(&qty)
	if err != nil {
		return 0, false
	}
	return qty, true
}

func TestLocationStock_SetRecordsRackRow_GlobalUnchanged(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-SET-001")
	insertTestStock(t, ctx, productID, 50)
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-SET-WH")
	locID := createTestLocation(t, ctx, "LOC-SET-RACK", "Rack Set", &whID, nil, true)

	require.NoError(t, repo.SetLocationStock(ctx, productID, locID, 12, 1))

	qty, ok := rackStock(t, ctx, productID, locID)
	require.True(t, ok)
	assert.Equal(t, 12, qty)

	global, ok := globalStock(t, ctx, productID)
	require.True(t, ok)
	assert.Equal(t, 50, global)

	var mvCount int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND type = 'location_set' AND reference_table = 'storage_locations'`,
		productID).Scan(&mvCount))
	assert.Equal(t, 1, mvCount)

	// first set records the signed delta from an empty rack: +12
	var change int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT quantity_change FROM inventory_movements WHERE product_id = $1 AND type = 'location_set'`,
		productID).Scan(&change))
	assert.Equal(t, 12, change)
}

func TestLocationStock_SetUpdatesExistingRow(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-SET-002")
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-SET-WH2")
	locID := createTestLocation(t, ctx, "LOC-SET-RACK2", "Rack Set 2", &whID, nil, true)

	require.NoError(t, repo.SetLocationStock(ctx, productID, locID, 5, 1))
	require.NoError(t, repo.SetLocationStock(ctx, productID, locID, 8, 1))

	qty, ok := rackStock(t, ctx, productID, locID)
	require.True(t, ok)
	assert.Equal(t, 8, qty)

	// updates record signed deltas: 0->5 is +5, 5->8 is +3
	var changes []int
	rows, err := dbPool.Query(ctx,
		`SELECT quantity_change FROM inventory_movements WHERE product_id = $1 AND type = 'location_set' ORDER BY id`,
		productID)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var c int
		require.NoError(t, rows.Scan(&c))
		changes = append(changes, c)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []int{5, 3}, changes)

	global, ok := globalStock(t, ctx, productID)
	require.False(t, ok)
	assert.Equal(t, 0, global)
}

func TestLocationStock_SetErrors(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-SET-003")
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-SET-WH3")
	activeLoc := createTestLocation(t, ctx, "LOC-SET-ACT", "Rack Active", &whID, nil, true)
	inactiveLoc := createTestLocation(t, ctx, "LOC-SET-INA", "Rack Inactive", &whID, nil, false)

	t.Run("negative quantity", func(t *testing.T) {
		assert.Error(t, repo.SetLocationStock(ctx, productID, activeLoc, -1, 1))
	})
	t.Run("missing location", func(t *testing.T) {
		assert.ErrorIs(t, repo.SetLocationStock(ctx, productID, 999999, 5, 1), ErrLocationNotFound)
	})
	t.Run("inactive location", func(t *testing.T) {
		assert.ErrorIs(t, repo.SetLocationStock(ctx, productID, inactiveLoc, 5, 1), ErrLocationInactive)
	})
}

func TestLocationStock_TransferMovesBetweenRacks_GlobalUnchanged(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-TFR-001")
	insertTestStock(t, ctx, productID, 100)
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-TFR-WH")
	from := createTestLocation(t, ctx, "LOC-TFR-A", "Rack A", &whID, nil, true)
	to := createTestLocation(t, ctx, "LOC-TFR-B", "Rack B", &whID, nil, true)

	require.NoError(t, repo.SetLocationStock(ctx, productID, from, 30, 1))
	require.NoError(t, repo.SetLocationStock(ctx, productID, to, 10, 1))
	require.NoError(t, repo.TransferLocationStock(ctx, productID, from, to, 7, 1))

	fromQty, _ := rackStock(t, ctx, productID, from)
	toQty, _ := rackStock(t, ctx, productID, to)
	assert.Equal(t, 23, fromQty)
	assert.Equal(t, 17, toQty)

	global, ok := globalStock(t, ctx, productID)
	require.True(t, ok)
	assert.Equal(t, 100, global)

	var mvCount int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND type = 'location_transfer'`,
		productID).Scan(&mvCount))
	assert.Equal(t, 2, mvCount)
}

func TestLocationStock_TransferErrors(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-TFR-002")
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-TFR-WH2")
	from := createTestLocation(t, ctx, "LOC-TFR-A2", "Rack A2", &whID, nil, true)
	to := createTestLocation(t, ctx, "LOC-TFR-B2", "Rack B2", &whID, nil, true)

	t.Run("insufficient source", func(t *testing.T) {
		require.NoError(t, repo.SetLocationStock(ctx, productID, from, 3, 1))
		err := repo.TransferLocationStock(ctx, productID, from, to, 5, 1)
		assert.ErrorIs(t, err, ErrInsufficientLocationStock)
	})
	t.Run("same location", func(t *testing.T) {
		err := repo.TransferLocationStock(ctx, productID, from, from, 1, 1)
		assert.ErrorIs(t, err, ErrSameLocation)
	})
	t.Run("missing location", func(t *testing.T) {
		err := repo.TransferLocationStock(ctx, productID, from, 999999, 1, 1)
		assert.ErrorIs(t, err, ErrLocationNotFound)
	})
}

func TestLocationStock_ListFilters(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productA := insertTestProduct(t, ctx, "LOC-LST-001")
	productB := insertTestProduct(t, ctx, "LOC-LST-002")
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-LST-WH")
	locA := createTestLocation(t, ctx, "LOC-LST-A", "Rack List A", &whID, nil, true)
	locB := createTestLocation(t, ctx, "LOC-LST-B", "Rack List B", &whID, nil, true)

	require.NoError(t, repo.SetLocationStock(ctx, productA, locA, 4, 1))
	require.NoError(t, repo.SetLocationStock(ctx, productA, locB, 6, 1))
	require.NoError(t, repo.SetLocationStock(ctx, productB, locA, 9, 1))

	t.Run("all", func(t *testing.T) {
		items, err := repo.ListLocationStock(ctx, 0, 0)
		require.NoError(t, err)
		assert.Len(t, items, 3)
	})
	t.Run("by product", func(t *testing.T) {
		items, err := repo.ListLocationStock(ctx, productA, 0)
		require.NoError(t, err)
		assert.Len(t, items, 2)
		for _, it := range items {
			assert.Equal(t, productA, it.ProductID)
		}
	})
	t.Run("by location", func(t *testing.T) {
		items, err := repo.ListLocationStock(ctx, 0, locA)
		require.NoError(t, err)
		assert.Len(t, items, 2)
		for _, it := range items {
			assert.Equal(t, locA, it.LocationID)
			assert.NotEmpty(t, it.LocationCode)
			assert.NotEmpty(t, it.LocationName)
		}
	})
	t.Run("by product and location", func(t *testing.T) {
		items, err := repo.ListLocationStock(ctx, productA, locB)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, 6, items[0].Quantity)
	})
}

func TestLocationStock_TransferToEmptyRackCreatesRow(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-TFR-003")
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-TFR-WH3")
	from := createTestLocation(t, ctx, "LOC-TFR-A3", "Rack A3", &whID, nil, true)
	to := createTestLocation(t, ctx, "LOC-TFR-B3", "Rack B3", &whID, nil, true)

	require.NoError(t, repo.SetLocationStock(ctx, productID, from, 10, 1))
	require.NoError(t, repo.TransferLocationStock(ctx, productID, from, to, 4, 1))

	fromQty, _ := rackStock(t, ctx, productID, from)
	toQty, ok := rackStock(t, ctx, productID, to)
	require.True(t, ok, "destination rack row must be created on transfer")
	assert.Equal(t, 6, fromQty)
	assert.Equal(t, 4, toQty)
}

func TestLocationStock_TransferInvalidQuantity(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-TFR-004")
	insertTestUser(t, ctx, 1)
	whID := createTestWarehouse(t, ctx, "LOC-TFR-WH4")
	from := createTestLocation(t, ctx, "LOC-TFR-A4", "Rack A4", &whID, nil, true)
	to := createTestLocation(t, ctx, "LOC-TFR-B4", "Rack B4", &whID, nil, true)

	require.NoError(t, repo.SetLocationStock(ctx, productID, from, 5, 1))

	t.Run("zero quantity", func(t *testing.T) {
		assert.Error(t, repo.TransferLocationStock(ctx, productID, from, to, 0, 1))
	})
	t.Run("negative quantity", func(t *testing.T) {
		assert.Error(t, repo.TransferLocationStock(ctx, productID, from, to, -2, 1))
	})
}

func TestLocationStock_ListEmpty(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	items, err := repo.ListLocationStock(ctx, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestLocationStock_StoreOnlyRack(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	productID := insertTestProduct(t, ctx, "LOC-STO-001")
	insertTestUser(t, ctx, 1)
	storeID := createTestStore(t, ctx, "LOC-STO")
	locID := createTestLocation(t, ctx, "LOC-STO-RACK", "Store Rack", nil, &storeID, true)

	require.NoError(t, repo.SetLocationStock(ctx, productID, locID, 7, 1))

	qty, ok := rackStock(t, ctx, productID, locID)
	require.True(t, ok)
	assert.Equal(t, 7, qty)

	var wh, st *int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT warehouse_id, store_id FROM product_stock WHERE product_id = $1 AND location_id = $2`,
		productID, locID).Scan(&wh, &st))
	assert.Nil(t, wh)
	require.NotNil(t, st)
	assert.Equal(t, storeID, *st)
}
