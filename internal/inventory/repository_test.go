package inventory

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func insertTestProduct(ctx context.Context, t *testing.T, sku string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price) VALUES ($1, $2, $3) RETURNING id`,
		sku, "Test Product "+sku, 10000,
	).Scan(&id)
	require.NoError(t, err)
	require.Greater(t, id, 0)
	return id
}

func insertTestStock(ctx context.Context, t *testing.T, productID, quantity int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)`,
		productID, quantity,
	)
	require.NoError(t, err)
}

func insertTestUser(ctx context.Context, t *testing.T, id int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, role_id) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`,
		id, "testuser", "testuser@test.com", "hash", 1,
	)
	require.NoError(t, err)
}

func TestInventoryRepository_GetStockByProductID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-GET-001")
		insertTestStock(ctx, t, productID, 25)

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, productID, stock.ProductID)
		assert.Equal(t, 25, stock.Quantity)
		assert.Nil(t, stock.WarehouseID)
		assert.Nil(t, stock.StoreID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetStockByProductID(ctx, -1)
		assert.Error(t, err)
	})

	t.Run("with nullable columns populated", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-GET-NP-001")
		_, err := dbPool.Exec(ctx,
			`INSERT INTO product_stock (product_id, reorder_point, reorder_quantity, last_restocked_at, quantity)
			 VALUES ($1, 5, 10, NOW(), 100)`,
			productID,
		)
		require.NoError(t, err)

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, productID, stock.ProductID)
		assert.Equal(t, 100, stock.Quantity)
		assert.Equal(t, 5, stock.ReorderPoint)
		assert.Equal(t, 10, stock.ReorderQuantity)
		assert.NotEmpty(t, stock.LastRestockedAt)
	})
}

func TestInventoryRepository_AdjustStock(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	t.Run("increase stock", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-ADJ-INC-001")
		insertTestStock(ctx, t, productID, 10)

		err := repo.AdjustStock(ctx, productID, 5, nil, "test increase")
		require.NoError(t, err)

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 15, stock.Quantity)
	})

	t.Run("decrease stock", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-ADJ-DEC-001")
		insertTestStock(ctx, t, productID, 20)

		err := repo.AdjustStock(ctx, productID, -8, nil, "test decrease")
		require.NoError(t, err)

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 12, stock.Quantity)
	})

	t.Run("insufficient stock", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-ADJ-INSF-001")
		insertTestStock(ctx, t, productID, 5)

		err := repo.AdjustStock(ctx, productID, -10, nil, "overdraft")
		assert.ErrorContains(t, err, "insufficient stock")

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 5, stock.Quantity)
	})

	t.Run("zero change returns error", func(t *testing.T) {
		err := repo.AdjustStock(ctx, 1, 0, nil, "no change")
		assert.ErrorContains(t, err, "must not be zero")
	})

	t.Run("creates row if none exists", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-ADJ-CREATE-001")

		err := repo.AdjustStock(ctx, productID, 30, nil, "initial stock")
		require.NoError(t, err)

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 30, stock.Quantity)
	})

	t.Run("with user ID", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "REPO-ADJ-USER-001")
		insertTestStock(ctx, t, productID, 100)
		insertTestUser(ctx, t, 1)
		userID := 1

		err := repo.AdjustStock(ctx, productID, -10, &userID, "user adjustment")
		require.NoError(t, err)

		stock, err := repo.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 90, stock.Quantity)
	})
}
