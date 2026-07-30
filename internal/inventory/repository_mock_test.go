package inventory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryRepository_GetStockByProductID_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "product_id", "warehouse_id", "store_id", "quantity", "reorder_point", "reorder_quantity", "last_restocked_at", "created_at", "updated_at"}).
		AddRow(1, 10, 5, 6, 100, 10, 20, now, now, now)
	mock.ExpectQuery("SELECT id, product_id, warehouse_id").WithArgs(10).WillReturnRows(rows)

	repo := NewRepository(mock)
	stock, err := repo.GetStockByProductID(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 100, stock.Quantity)
	require.NotNil(t, stock.WarehouseID)
	assert.Equal(t, 5, *stock.WarehouseID)
	require.NotNil(t, stock.StoreID)
	assert.Equal(t, 6, *stock.StoreID)
}

func TestInventoryRepository_GetStockByProductID_NotFound_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "product_id", "warehouse_id", "store_id", "quantity", "reorder_point", "reorder_quantity", "last_restocked_at", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, product_id, warehouse_id").WithArgs(999).WillReturnRows(rows)

	repo := NewRepository(mock)
	_, err = repo.GetStockByProductID(context.Background(), 999)
	assert.Error(t, err)
}

func TestInventoryRepository_AdjustStock_BeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestInventoryRepository_AdjustStock_QueryRowError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnError(fmt.Errorf("query failed"))

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load product stock")
}

func TestInventoryRepository_AdjustStock_UpsertError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	emptyRows := pgxmock.NewRows([]string{"quantity"})
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(emptyRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(5, 1).WillReturnResult(pgxmock.NewResult("U", 0))
	mock.ExpectExec("INSERT INTO product_stock").WithArgs(1, 5).WillReturnError(fmt.Errorf("insert failed"))

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert stock")
}

func TestInventoryRepository_AdjustStock_SyncError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	emptyRows := pgxmock.NewRows([]string{"quantity"})
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(emptyRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(5, 1).WillReturnResult(pgxmock.NewResult("U", 0))
	mock.ExpectExec("INSERT INTO product_stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("sync failed"))

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync product stock")
}

func TestInventoryRepository_AdjustStock_MovementError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	emptyRows := pgxmock.NewRows([]string{"quantity"})
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(emptyRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(5, 1).WillReturnResult(pgxmock.NewResult("U", 0))
	mock.ExpectExec("INSERT INTO product_stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("movement failed"))

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record inventory movement")
}

func TestInventoryRepository_AdjustStock_CommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	emptyRows := pgxmock.NewRows([]string{"quantity"})
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(emptyRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(5, 1).WillReturnResult(pgxmock.NewResult("U", 0))
	mock.ExpectExec("INSERT INTO product_stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("commit failed"))

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit stock adjustment")
}

func TestInventoryRepository_AdjustStock_NoRowsPath_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	emptyRows := pgxmock.NewRows([]string{"quantity"})
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(emptyRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(pgxmock.AnyArg(), 1).WillReturnResult(pgxmock.NewResult("U", 0))
	mock.ExpectExec("INSERT INTO product_stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.NoError(t, err)
}

func TestInventoryRepository_AdjustStock_Success_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	existingRows := pgxmock.NewRows([]string{"quantity"}).AddRow(10)
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(existingRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(15, 1).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "test")
	assert.NoError(t, err)
}

func TestInventoryRepository_AdjustStock_Decrease_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	existingRows := pgxmock.NewRows([]string{"quantity"}).AddRow(10)
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(existingRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(5, 1).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, -5, nil, "test")
	assert.NoError(t, err)
}

func TestInventoryRepository_AdjustStock_InsufficientStock_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	existingRows := pgxmock.NewRows([]string{"quantity"}).AddRow(3)
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(existingRows)

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, -10, nil, "test")
	assert.ErrorContains(t, err, "insufficient stock")
}

func TestInventoryRepository_AdjustStock_NoRowsExisting_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	emptyRows := pgxmock.NewRows([]string{"quantity"})
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(emptyRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(5, 1).WillReturnResult(pgxmock.NewResult("U", 0))
	mock.ExpectExec("INSERT INTO product_stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, 5, nil, "initial stock")
	assert.NoError(t, err)
}

func TestInventoryRepository_AdjustStock_WithUserID_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	uid := 42
	mock.ExpectBegin()
	existingRows := pgxmock.NewRows([]string{"quantity"}).AddRow(50)
	mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnRows(existingRows)
	mock.ExpectExec("UPDATE product_stock").WithArgs(40, 1).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("UPDATE products SET stock").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("U", 1))
	mock.ExpectExec("INSERT INTO inventory_movements").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("I", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	err = repo.AdjustStock(context.Background(), 1, -10, &uid, "user adj")
	assert.NoError(t, err)
}
