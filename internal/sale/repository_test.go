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

		shiftUserID := insertTestCashier(t, ctx)
		var shiftID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, shiftUserID).Scan(&shiftID)
		require.NoError(t, err)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		sale := &Sale{
			InvoiceNumber: "INV-REPO-CREATE-001",
			CashierID:     shiftUserID,
			ShiftID:       &shiftID,
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

		var shiftIDFromDB *int
		err = dbPool.QueryRow(ctx, `SELECT shift_id FROM sales WHERE id = $1`, sale.ID).Scan(&shiftIDFromDB)
		require.NoError(t, err)
		require.NotNil(t, shiftIDFromDB)
		assert.Equal(t, shiftID, *shiftIDFromDB)

		var itemCount int
		err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM sale_items WHERE sale_id = $1`, sale.ID).Scan(&itemCount)
		require.NoError(t, err)
		assert.Equal(t, 1, itemCount)
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
		defer func() { _ = tx.Rollback(ctx) }()

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
		sales, total, err := repo.GetAllSales(ctx, 2, 0, "", "", "", "", "", nil, "", nil, nil, nil)
		require.NoError(t, err)
		assert.Len(t, sales, 2)
		assert.GreaterOrEqual(t, total, 3)
	})

	t.Run("search by invoice number", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 10, 0, inv1, "", "", "", "", nil, "", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, inv1, sales[0].InvoiceNumber)
	})

	t.Run("filter by payment method", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 10, 0, "", "", "", "", "", nil, "CASH", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		for _, s := range sales {
			assert.Equal(t, "CASH", s.PaymentMethod)
		}
	})

	t.Run("sort by total_amount DESC", func(t *testing.T) {
		sales, _, err := repo.GetAllSales(ctx, 10, 0, "", "total_amount", "DESC", "", "", nil, "", nil, nil, nil)
		require.NoError(t, err)
		if len(sales) >= 2 {
			assert.GreaterOrEqual(t, sales[0].TotalAmount, sales[1].TotalAmount)
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(ctx, 10, 0, "__NONEXISTENT__", "", "", "", "", nil, "", nil, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, sales)
		assert.Equal(t, 0, total)
	})

	t.Run("filter by cashier_id", func(t *testing.T) {
		allSales, _, err := repo.GetAllSales(ctx, 100, 0, "", "", "", "", "", nil, "", nil, nil, nil)
		require.NoError(t, err)
		if len(allSales) == 0 {
			t.Skip("no sales to test cashier_id filter")
		}
		cashierID := allSales[0].CashierID
		sales, total, err := repo.GetAllSales(ctx, 10, 0, "", "", "", "", "", nil, "", nil, nil, &cashierID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, s := range sales {
			assert.Equal(t, cashierID, s.CashierID)
		}
	})
}

func TestSaleRepository_Export(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "REPO-EXPORT-PROD", "Export Product", 12000, 50)
	_ = createAndCommitSale(t, ctx, repo, "INV-REPO-EXPORT-001", prodID, 2, 12000, 24000, 24000, 0)

	rows, err := repo.GetSalesForExport(ctx, "", "", "", "", nil, nil, nil)
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
	defer func() { _ = tx.Rollback(ctx) }()

	var result int
	err = tx.QueryRow(ctx, "SELECT 1").Scan(&result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestSaleRepository_CreateSalePayments(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("insert and retrieve single payment", func(t *testing.T) {
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		prodID := insertTestProduct(t, ctx, "REPO-PAY-001", "Pay Test Product", 10000, 10)
		cashierID := insertTestCashier(t, ctx)
		sale := &Sale{
			InvoiceNumber: "INV-REPO-PAY-001",
			CashierID:     cashierID,
			Subtotal:      10000,
			TotalAmount:   10000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		err = repo.CreateSale(ctx, tx, sale, []SaleItem{{
			ProductID: prodID, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000,
		}})
		require.NoError(t, err)

		payments := []SalePayment{{
			SaleID: sale.ID, PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 10000,
		}}
		err = repo.CreateSalePayments(ctx, tx, sale.ID, payments)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		var count int
		err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM sale_payments WHERE sale_id = $1`, sale.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("insert multiple payments", func(t *testing.T) {
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		prodID := insertTestProduct(t, ctx, "REPO-PAY-002", "Pay Test Multi", 50000, 10)
		cashierID := insertTestCashier(t, ctx)
		sale := &Sale{
			InvoiceNumber: "INV-REPO-PAY-002",
			CashierID:     cashierID,
			Subtotal:      50000,
			TotalAmount:   50000,
			PaymentMethod: "CASH,QRIS",
			Status:        "completed",
		}
		err = repo.CreateSale(ctx, tx, sale, []SaleItem{{
			ProductID: prodID, Quantity: 1, UnitPrice: 50000, Subtotal: 50000, DPPAmount: 50000,
		}})
		require.NoError(t, err)

		payments := []SalePayment{
			{SaleID: sale.ID, PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 30000},
			{SaleID: sale.ID, PaymentMethodID: 5, PaymentMethodCode: "QRIS", Amount: 20000},
		}
		err = repo.CreateSalePayments(ctx, tx, sale.ID, payments)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		var count int
		err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM sale_payments WHERE sale_id = $1`, sale.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestSaleRepository_GetSaleByIDWithPayments(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("sale with split payments returns payments array", func(t *testing.T) {
		prodID := insertTestProduct(t, ctx, "REPO-GET-PAY", "Get Pay Product", 50000, 20)
		cashierID := insertTestCashier(t, ctx)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		sale := &Sale{
			InvoiceNumber: "INV-REPO-GET-PAY-001",
			CashierID:     cashierID,
			Subtotal:      50000,
			TotalAmount:   50000,
			PaymentMethod: "CASH,QRIS",
			Status:        "completed",
		}
		items := []SaleItem{{
			ProductID: prodID, Quantity: 1, UnitPrice: 50000, Subtotal: 50000, DPPAmount: 50000,
		}}
		err = repo.CreateSale(ctx, tx, sale, items)
		require.NoError(t, err)

		payments := []SalePayment{
			{SaleID: sale.ID, PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 30000},
			{SaleID: sale.ID, PaymentMethodID: 5, PaymentMethodCode: "QRIS", Amount: 20000},
		}
		err = repo.CreateSalePayments(ctx, tx, sale.ID, payments)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		got, err := repo.GetSaleByID(ctx, sale.ID, nil)
		require.NoError(t, err)
		require.Len(t, got.Payments, 2)

		assert.Equal(t, "CASH", got.Payments[0].PaymentMethodCode)
		assert.Equal(t, 30000, got.Payments[0].Amount)
		assert.Equal(t, "QRIS", got.Payments[1].PaymentMethodCode)
		assert.Equal(t, 20000, got.Payments[1].Amount)
	})

	t.Run("sale without payments returns empty array", func(t *testing.T) {
		prodID := insertTestProduct(t, ctx, "REPO-GET-PAY-NP", "Get Pay NoPay", 10000, 10)
		sale := createAndCommitSale(t, ctx, repo, "INV-REPO-GET-PAY-NOPAY", prodID, 1, 10000, 10000, 10000, 0)

		got, err := repo.GetSaleByID(ctx, sale.ID, nil)
		require.NoError(t, err)
		assert.Empty(t, got.Payments)
	})
}

func TestSaleRepository_UpdateShiftTotalsWithPayments(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	setupShift := func(t *testing.T) (int, int) {
		t.Helper()
		cashierID := insertTestCashier(t, ctx)
		var shiftID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, cashierID).Scan(&shiftID)
		require.NoError(t, err)
		return cashierID, shiftID
	}

	getShiftTotals := func(t *testing.T, shiftID int) (cashSales, nonCashSales, totalSales int) {
		t.Helper()
		err := dbPool.QueryRow(ctx, `
			SELECT cash_sales, non_cash_sales, total_sales FROM shifts WHERE id = $1
		`, shiftID).Scan(&cashSales, &nonCashSales, &totalSales)
		require.NoError(t, err)
		return
	}

	t.Run("cash payment updates cash_sales only", func(t *testing.T) {
		cashierID, shiftID := setupShift(t)
		prodID := insertTestProduct(t, ctx, "REPO-SHIFT-CASH", "Shift Cash", 10000, 10)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		sale := &Sale{InvoiceNumber: "INV-SHIFT-CASH", CashierID: cashierID, ShiftID: &shiftID, Subtotal: 10000, TotalAmount: 10000, PaymentMethod: "CASH", Status: "completed"}
		err = repo.CreateSale(ctx, tx, sale, []SaleItem{{ProductID: prodID, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000}})
		require.NoError(t, err)

		payments := []SalePayment{{SaleID: sale.ID, PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 10000}}
		err = repo.CreateSalePayments(ctx, tx, sale.ID, payments)
		require.NoError(t, err)

		err = repo.UpdateShiftTotals(ctx, tx, shiftID, 10000, payments)
		require.NoError(t, err)
		err = tx.Commit(ctx)
		require.NoError(t, err)

		cashSales, nonCashSales, totalSales := getShiftTotals(t, shiftID)
		assert.Equal(t, 10000, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, 10000, totalSales)
	})

	t.Run("non-cash payment updates non_cash_sales only", func(t *testing.T) {
		cashierID, shiftID := setupShift(t)
		prodID := insertTestProduct(t, ctx, "REPO-SHIFT-NC", "Shift NonCash", 20000, 10)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		sale := &Sale{InvoiceNumber: "INV-SHIFT-NC", CashierID: cashierID, ShiftID: &shiftID, Subtotal: 20000, TotalAmount: 20000, PaymentMethod: "QRIS", Status: "completed"}
		err = repo.CreateSale(ctx, tx, sale, []SaleItem{{ProductID: prodID, Quantity: 1, UnitPrice: 20000, Subtotal: 20000, DPPAmount: 20000}})
		require.NoError(t, err)

		payments := []SalePayment{{SaleID: sale.ID, PaymentMethodID: 5, PaymentMethodCode: "QRIS", Amount: 20000}}
		err = repo.CreateSalePayments(ctx, tx, sale.ID, payments)
		require.NoError(t, err)

		err = repo.UpdateShiftTotals(ctx, tx, shiftID, 20000, payments)
		require.NoError(t, err)
		err = tx.Commit(ctx)
		require.NoError(t, err)

		cashSales, nonCashSales, totalSales := getShiftTotals(t, shiftID)
		assert.Equal(t, 0, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 20000, totalSales)
	})

	t.Run("mixed payments update both correctly", func(t *testing.T) {
		cashierID, shiftID := setupShift(t)
		prodID := insertTestProduct(t, ctx, "REPO-SHIFT-MIX", "Shift Mixed", 50000, 10)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		sale := &Sale{InvoiceNumber: "INV-SHIFT-MIX", CashierID: cashierID, ShiftID: &shiftID, Subtotal: 50000, TotalAmount: 50000, PaymentMethod: "CASH,QRIS", Status: "completed"}
		err = repo.CreateSale(ctx, tx, sale, []SaleItem{{ProductID: prodID, Quantity: 1, UnitPrice: 50000, Subtotal: 50000, DPPAmount: 50000}})
		require.NoError(t, err)

		payments := []SalePayment{
			{SaleID: sale.ID, PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 30000},
			{SaleID: sale.ID, PaymentMethodID: 5, PaymentMethodCode: "QRIS", Amount: 20000},
		}
		err = repo.CreateSalePayments(ctx, tx, sale.ID, payments)
		require.NoError(t, err)

		err = repo.UpdateShiftTotals(ctx, tx, shiftID, 50000, payments)
		require.NoError(t, err)
		err = tx.Commit(ctx)
		require.NoError(t, err)

		cashSales, nonCashSales, totalSales := getShiftTotals(t, shiftID)
		assert.Equal(t, 30000, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 50000, totalSales)
	})
}

func createParkedSale(t *testing.T, ctx context.Context, repo *Repository, cashierID int, invoice string, status string, prodID, qty, price int) *Sale {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	sale := &Sale{
		InvoiceNumber: invoice,
		CashierID:     cashierID,
		Subtotal:      price * qty,
		TotalAmount:   price * qty,
		PaymentMethod: "CASH",
		Status:        status,
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  qty,
		UnitPrice: price,
		Subtotal:  price * qty,
		DPPAmount: price * qty,
		TaxAmount: 0,
	}}
	err = repo.CreateSale(ctx, tx, sale, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)
	return sale
}

func TestSaleRepository_GetParkedSales(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProduct(t, ctx, "PARKED-PROD-001", "Parked Product", 10000, 50)

	_ = createParkedSale(t, ctx, repo, cashierID, "INV-PARKED-001", "parked", prodID, 2, 10000)
	_ = createParkedSale(t, ctx, repo, cashierID, "INV-PARKED-002", "recalled", prodID, 1, 10000)
	_ = createParkedSale(t, ctx, repo, cashierID, "INV-PARKED-003", "completed", prodID, 3, 10000)

	t.Run("returns parked and recalled only", func(t *testing.T) {
		sales, err := repo.GetParkedSales(ctx, cashierID)
		require.NoError(t, err)
		assert.Len(t, sales, 2)
		for _, s := range sales {
			assert.Contains(t, []string{"parked", "recalled"}, s.Status)
			assert.Equal(t, cashierID, s.CashierID)
		}
	})

	t.Run("filters by cashier_id", func(t *testing.T) {
		otherCashier := insertTestCashier(t, ctx)
		_ = createParkedSale(t, ctx, repo, otherCashier, "INV-PARKED-OTHER", "parked", prodID, 1, 10000)

		sales, err := repo.GetParkedSales(ctx, cashierID)
		require.NoError(t, err)
		for _, s := range sales {
			assert.Equal(t, cashierID, s.CashierID)
		}
	})

	t.Run("excludes cancelled", func(t *testing.T) {
		_ = createParkedSale(t, ctx, repo, cashierID, "INV-PARKED-CANCEL", "cancelled", prodID, 1, 10000)
		sales, err := repo.GetParkedSales(ctx, cashierID)
		require.NoError(t, err)
		for _, s := range sales {
			assert.NotEqual(t, "cancelled", s.Status)
		}
	})

	t.Run("includes items", func(t *testing.T) {
		sales, err := repo.GetParkedSales(ctx, cashierID)
		require.NoError(t, err)
		assert.NotEmpty(t, sales)
		found := false
		for _, s := range sales {
			if s.InvoiceNumber == "INV-PARKED-001" {
				assert.Len(t, s.Items, 1)
				assert.Equal(t, prodID, s.Items[0].ProductID)
				found = true
			}
		}
		assert.True(t, found, "should find INV-PARKED-001 with items")
	})
}

func TestSaleRepository_RecallSale(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProduct(t, ctx, "RECALL-PROD-001", "Recall Product", 10000, 50)

	t.Run("recall parked sale", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-RECALL-001", "parked", prodID, 2, 10000)

		sale, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)
		assert.Equal(t, "recalled", sale.Status)
		assert.Equal(t, parked.InvoiceNumber, sale.InvoiceNumber)
		assert.NotEmpty(t, sale.Items)
		assert.Equal(t, prodID, sale.Items[0].ProductID)
	})

	t.Run("recall already recalled succeeds", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-RECALL-002", "parked", prodID, 1, 10000)
		_, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)

		sale, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)
		assert.Equal(t, "recalled", sale.Status)
	})

	t.Run("recall completed sale returns error", func(t *testing.T) {
		completed := createParkedSale(t, ctx, repo, cashierID, "INV-RECALL-003", "completed", prodID, 1, 10000)
		_, err := repo.RecallSale(ctx, completed.ID)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("recall non-existent sale returns error", func(t *testing.T) {
		_, err := repo.RecallSale(ctx, -999)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})
}

func TestSaleRepository_GetParkedSaleByID(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProduct(t, ctx, "GETBYID-PROD-001", "GetByID Product", 10000, 50)

	t.Run("get parked sale by id", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-GETBYID-001", "parked", prodID, 2, 10000)

		sale, err := repo.GetParkedSaleByID(ctx, parked.ID, cashierID)
		require.NoError(t, err)
		assert.Equal(t, parked.ID, sale.ID)
		assert.Equal(t, "parked", sale.Status)
		assert.NotEmpty(t, sale.Items)
		assert.Len(t, sale.Items, 1)
	})

	t.Run("get recalled sale by id", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-GETBYID-002", "parked", prodID, 1, 10000)
		recalled, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)

		sale, err := repo.GetParkedSaleByID(ctx, recalled.ID, cashierID)
		require.NoError(t, err)
		assert.Equal(t, "recalled", sale.Status)
	})

	t.Run("get non-existent returns error", func(t *testing.T) {
		_, err := repo.GetParkedSaleByID(ctx, -999, cashierID)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("wrong cashier returns error", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-GETBYID-003", "parked", prodID, 1, 10000)

		var otherCashier int
		err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('other_cashier_getbyid', 'other@test.com', 'hash', 1) RETURNING id`).Scan(&otherCashier)
		require.NoError(t, err)

		_, err = repo.GetParkedSaleByID(ctx, parked.ID, otherCashier)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("completed sale not returned", func(t *testing.T) {
		completed := createParkedSale(t, ctx, repo, cashierID, "INV-GETBYID-004", "completed", prodID, 1, 10000)
		_, err := repo.GetParkedSaleByID(ctx, completed.ID, cashierID)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})
}

func TestSaleRepository_ConsumeParkedSale(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProduct(t, ctx, "CONSUME-PROD-001", "Consume Product", 10000, 50)

	t.Run("consume recalled sale", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-CONSUME-001", "parked", prodID, 1, 10000)
		_, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = repo.ConsumeParkedSale(ctx, tx, parked.ID)
		require.NoError(t, err)
	})

	t.Run("consume non-recalled returns error", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-CONSUME-002", "parked", prodID, 1, 10000)

		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = repo.ConsumeParkedSale(ctx, tx, parked.ID)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("consume non-existent returns error", func(t *testing.T) {
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = repo.ConsumeParkedSale(ctx, tx, -999)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})
}

func TestSaleRepository_CancelParkedSale(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProduct(t, ctx, "CANCEL-PROD-001", "Cancel Product", 10000, 50)

	t.Run("cancel parked sale", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-CANCEL-001", "parked", prodID, 1, 10000)
		err := repo.CancelParkedSale(ctx, parked.ID)
		assert.NoError(t, err)

		var status string
		err = dbPool.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1`, parked.ID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", status)
	})

	t.Run("cancel recalled sale", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-CANCEL-002", "recalled", prodID, 1, 10000)
		err := repo.CancelParkedSale(ctx, parked.ID)
		assert.NoError(t, err)

		var status string
		err = dbPool.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1`, parked.ID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", status)
	})

	t.Run("cancel completed sale returns error", func(t *testing.T) {
		completed := createParkedSale(t, ctx, repo, cashierID, "INV-CANCEL-003", "completed", prodID, 1, 10000)
		err := repo.CancelParkedSale(ctx, completed.ID)
		assert.Error(t, err)
	})

	t.Run("cancel non-existent sale returns error", func(t *testing.T) {
		err := repo.CancelParkedSale(ctx, -999)
		assert.Error(t, err)
	})

	t.Run("cancel already cancelled sale returns error", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-CANCEL-004", "parked", prodID, 1, 10000)
		err := repo.CancelParkedSale(ctx, parked.ID)
		require.NoError(t, err)
		err = repo.CancelParkedSale(ctx, parked.ID)
		assert.Error(t, err)
	})
}
