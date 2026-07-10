package sale

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
		os.Exit(0)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(0)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func insertTestProduct(t *testing.T, ctx context.Context, sku, name string, price, stock int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id
	`, sku, name, price, price/2).Scan(&id)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity)
		VALUES ($1, $2)
	`, id, stock)
	require.NoError(t, err)
	return id
}

func insertTestCashier(t *testing.T, ctx context.Context) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('sale_test_cashier', 'sale_cashier@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&id)
	require.NoError(t, err)
	return id
}

func createAndCommitSale(t *testing.T, ctx context.Context, repo *Repository, invoice string, prodID, qty, price, subtotal, dpp, tax int) *Sale {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	cashierID := insertTestCashier(t, ctx)
	sale := &Sale{
		InvoiceNumber: invoice,
		CashierID:     cashierID,
		Subtotal:      subtotal,
		Discount:      0,
		Tax:           tax,
		TotalAmount:   subtotal + tax,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  qty,
		UnitPrice: price,
		Subtotal:  subtotal,
		DPPAmount: dpp,
		TaxAmount: tax,
	}}

	err = repo.CreateSale(ctx, tx, sale, items)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
	return sale
}

func TestSaleRepository_CreateAndGet(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("create sale with items", func(t *testing.T) {
		prodID := insertTestProduct(t, ctx, "REPO-CREATE-001", "Create Test Product", 10000, 50)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		sale := &Sale{
			InvoiceNumber: "INV-REPO-CREATE-001",
			CashierID:     insertTestCashier(t, ctx),
			Subtotal:      50000,
			Discount:      0,
			Tax:           5000,
			TotalAmount:   55000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  5,
			UnitPrice: 10000,
			Subtotal:  50000,
			DPPAmount: 50000,
			TaxAmount: 0,
		}}

		err = repo.CreateSale(ctx, tx, sale, items)
		require.NoError(t, err)
		require.Greater(t, sale.ID, 0)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		var qty int
		err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&qty)
		require.NoError(t, err)
		assert.Equal(t, 45, qty)

		var movCount int
		err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND type = 'sale' AND reference_id = $2`, prodID, sale.ID).Scan(&movCount)
		require.NoError(t, err)
		assert.Equal(t, 1, movCount)
	})

	t.Run("get sale by ID with items", func(t *testing.T) {
		prodID := insertTestProduct(t, ctx, "REPO-GET-001", "Get Test Product", 20000, 30)
		sale := createAndCommitSale(t, ctx, repo, "INV-REPO-GET-001", prodID, 2, 20000, 40000, 40000, 0)

		got, err := repo.GetSaleByID(ctx, sale.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, sale.InvoiceNumber, got.InvoiceNumber)
		assert.Equal(t, sale.CashierID, got.CashierID)
		assert.Equal(t, sale.Subtotal, got.Subtotal)
		assert.Equal(t, sale.TotalAmount, got.TotalAmount)
		assert.Equal(t, "CASH", got.PaymentMethod)
		assert.Equal(t, "completed", got.Status)
		assert.NotEmpty(t, got.CreatedAt)
		assert.NotEmpty(t, got.UpdatedAt)
		assert.Len(t, got.Items, 1)
		assert.Equal(t, prodID, got.Items[0].ProductID)
		assert.Equal(t, 2, got.Items[0].Quantity)
	})

	t.Run("get sale by ID not found", func(t *testing.T) {
		_, err := repo.GetSaleByID(ctx, -1, nil)
		assert.ErrorContains(t, err, "sale not found")
	})

	t.Run("duplicate invoice number", func(t *testing.T) {
		prodID := insertTestProduct(t, ctx, "REPO-DUP-001", "Duplicate Test", 15000, 10)
		_ = createAndCommitSale(t, ctx, repo, "INV-REPO-DUP-001", prodID, 1, 15000, 15000, 15000, 0)

		prodID2 := insertTestProduct(t, ctx, "REPO-DUP-002", "Duplicate Test 2", 15000, 10)
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		dup := &Sale{
			InvoiceNumber: "INV-REPO-DUP-001",
			CashierID:     insertTestCashier(t, ctx),
			Subtotal:      15000,
			TotalAmount:   15000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		dupItems := []SaleItem{{
			ProductID: prodID2,
			Quantity:  1,
			UnitPrice: 15000,
			Subtotal:  15000,
			DPPAmount: 15000,
			TaxAmount: 0,
		}}

		err = repo.CreateSale(ctx, tx, dup, dupItems)
		assert.Error(t, err)
	})
}

func TestSaleRepository_ListSales(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "REPO-LIST-PROD", "List Test Product", 10000, 100)
	inv1, inv2, inv3 := "INV-REPO-LIST-001", "INV-REPO-LIST-002", "INV-REPO-LIST-003"
	_ = createAndCommitSale(t, ctx, repo, inv1, prodID, 1, 10000, 10000, 10000, 0)
	_ = createAndCommitSale(t, ctx, repo, inv2, prodID, 2, 10000, 20000, 20000, 0)
	_ = createAndCommitSale(t, ctx, repo, inv3, prodID, 3, 10000, 30000, 30000, 0)

	t.Run("pagination", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 2, 0, "", "", "", "", "", nil, "", nil, nil)
		require.NoError(t, err)
		assert.Len(t, sales, 2)
		assert.GreaterOrEqual(t, total, 3)
	})

	t.Run("search by invoice number", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 10, 0, inv1, "", "", "", "", nil, "", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, inv1, sales[0].InvoiceNumber)
	})

	t.Run("filter by payment method", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 10, 0, "", "", "", "", "", nil, "CASH", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		for _, s := range sales {
			assert.Equal(t, "CASH", s.PaymentMethod)
		}
	})

	t.Run("sort by total_amount DESC", func(t *testing.T) {
		sales, _, err := repo.GetAllSales(ctx, 10, 0, "", "total_amount", "DESC", "", "", nil, "", nil, nil)
		require.NoError(t, err)
		if len(sales) >= 2 {
			assert.GreaterOrEqual(t, sales[0].TotalAmount, sales[1].TotalAmount)
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 10, 0, "__NONEXISTENT__", "", "", "", "", nil, "", nil, nil)
		require.NoError(t, err)
		assert.Empty(t, sales)
		assert.Equal(t, 0, total)
	})
}

func TestSaleRepository_Export(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "REPO-EXPORT-PROD", "Export Product", 12000, 50)
	_ = createAndCommitSale(t, ctx, repo, "INV-REPO-EXPORT-001", prodID, 2, 12000, 24000, 24000, 0)

	rows, err := repo.GetSalesForExport(ctx, "", "", "", "", nil, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, rows)

	found := false
	for _, r := range rows {
		if r.InvoiceNumber == "INV-REPO-EXPORT-001" {
			found = true
			assert.Equal(t, 1, r.ItemCount)
			assert.Equal(t, "CASH", r.PaymentMethod)
			assert.Equal(t, 24000, r.TotalAmount)
			break
		}
	}
	assert.True(t, found)
}

func TestSaleRepository_InvoiceNumber(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	inv, err := repo.GetNextInvoiceNumber(ctx)
	require.NoError(t, err)
	assert.Contains(t, inv, "INV-")
}

func TestSaleRepository_PaymentMethods(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("get all active", func(t *testing.T) {
		methods, err := repo.GetAllActive(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(methods), 5)
		codes := make(map[string]bool)
		for _, m := range methods {
			codes[m.Code] = true
		}
		assert.True(t, codes["CASH"])
		assert.True(t, codes["QRIS"])
	})

	t.Run("get by code", func(t *testing.T) {
		pm, err := repo.GetPaymentMethodByCode(ctx, "CASH")
		require.NoError(t, err)
		assert.Equal(t, "Cash", pm.Name)
		assert.True(t, pm.IsActive)
		assert.False(t, pm.RequiresReference)
	})

	t.Run("get by code not found", func(t *testing.T) {
		_, err := repo.GetPaymentMethodByCode(ctx, "NONEXISTENT")
		assert.Error(t, err)
	})

	t.Run("get by ID", func(t *testing.T) {
		pm, err := repo.GetPaymentMethodByID(ctx, 1)
		require.NoError(t, err)
		assert.NotEmpty(t, pm.Code)
	})

	t.Run("get by ID not found", func(t *testing.T) {
		_, err := repo.GetPaymentMethodByID(ctx, -1)
		assert.Error(t, err)
	})
}

func TestSaleRepository_BeginTx(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	var result int
	err = tx.QueryRow(ctx, "SELECT 1").Scan(&result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}
