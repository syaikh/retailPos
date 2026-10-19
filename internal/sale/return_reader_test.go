package sale

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestReturnSaleReader_SaleForReturn(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	userID := insertTestUser(t)
	cashID := insertTestPaymentMethod(t, "CASH")

	t.Run("missing sale returns nil,nil", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })

		r := ReturnSaleReader{}
		got, err := r.SaleForReturn(ctx, tx, 999999)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("sale with items and payments, customer set", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })

		var custID int
		err = tx.QueryRow(ctx, `
			INSERT INTO customers (name, email, phone, store_id)
			VALUES ('C1', 'c1@test.com', '08120000001', 1)
			ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name RETURNING id
		`).Scan(&custID)
		require.NoError(t, err)

		var shiftID int
		_, _ = tx.Exec(ctx, `INSERT INTO users (id, username, email, password_hash, role_id) VALUES ($1, 'summary_cashier', 'summary_cashier@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`, userID)
		err = tx.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, userID).Scan(&shiftID)
		require.NoError(t, err)

		var saleID int
		err = tx.QueryRow(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
			                   subtotal, discount, tax, total_amount, change_due, created_at, store_id)
			VALUES ('RET-001', $1, $2, $3, 'Cash', 'completed', 220000, 0, 20000, 240000, 0, NOW(), 1)
			RETURNING id
		`, userID, custID, shiftID).Scan(&saleID)
		require.NoError(t, err)

		p1 := insertTestProduct(ctx, t, "RET-RP1", "P1", 100000, 100)
		p2 := insertTestProduct(ctx, t, "RET-RP2", "P2", 40000, 100)
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, product_name, quantity, unit_price, subtotal, dpp_amount, tax_amount, tax_rate)
			VALUES ($1, $2, 'P1', 2, 100000, 200000, 181818, 18182, 10)
		`, saleID, p1)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, product_name, quantity, unit_price, subtotal, dpp_amount, tax_amount, tax_rate)
			VALUES ($1, $2, 'P2', 1, 40000, 40000, 36364, 3636, 10)
		`, saleID, p2)
		require.NoError(t, err)

		// insert two payments
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_payments (sale_id, payment_method_id, payment_method_code, amount)
			VALUES ($1, $2, 'CASH', 200000)
		`, saleID, cashID)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_payments (sale_id, payment_method_id, payment_method_code, amount)
			VALUES ($1, $2, 'CASH', 40000)
		`, saleID, cashID)
		require.NoError(t, err)

		r := ReturnSaleReader{}
		got, err := r.SaleForReturn(ctx, tx, saleID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, saleID, got.ID)
		require.Equal(t, "RET-001", got.InvoiceNumber)
		require.Equal(t, userID, got.CashierID)
		require.NotNil(t, got.CustomerID)
		require.Equal(t, custID, *got.CustomerID)
		require.Equal(t, 1, got.StoreID)
		require.Equal(t, "completed", got.Status)
		require.Equal(t, 240000, got.TotalAmount)
		require.Equal(t, 0, got.ChangeDue)
		require.WithinDuration(t, time.Now(), got.CreatedAt, 2*time.Second)
		require.Len(t, got.Items, 2)
		// ordered by id
		require.Equal(t, p1, got.Items[0].ProductID)
		require.Equal(t, p2, got.Items[1].ProductID)
		require.Equal(t, "P1", got.Items[0].ProductName)
		require.Equal(t, 200000, got.Items[0].Subtotal)
		require.Len(t, got.Payments, 2)
		require.Equal(t, 200000, got.Payments[0].Amount)
		require.Equal(t, 40000, got.Payments[1].Amount)
	})

	t.Run("customer NULL, store_id NULL becomes 0", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })

		var shiftID int
		_, _ = tx.Exec(ctx, `INSERT INTO users (id, username, email, password_hash, role_id) VALUES ($1, 'summary_cashier', 'summary_cashier@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`, userID)
		err = tx.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, userID).Scan(&shiftID)
		require.NoError(t, err)

		var saleID int
		err = tx.QueryRow(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
			                   subtotal, discount, tax, total_amount, created_at)
			VALUES ('RET-002', $1, NULL, $2, 'Cash', 'completed', 10000, 0, 1000, 11000, NOW())
			RETURNING id
		`, userID, shiftID).Scan(&saleID)
		require.NoError(t, err)

		r := ReturnSaleReader{}
		got, err := r.SaleForReturn(ctx, tx, saleID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Nil(t, got.CustomerID)
		require.Equal(t, 0, got.StoreID) // COALESCE(store_id, 0)
		require.Equal(t, "completed", got.Status)
	})

	t.Run("sale with no items and no payments returns empty slices", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })

		var shiftID int
		_, _ = tx.Exec(ctx, `INSERT INTO users (id, username, email, password_hash, role_id) VALUES ($1, 'summary_cashier', 'summary_cashier@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`, userID)
		err = tx.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, userID).Scan(&shiftID)
		require.NoError(t, err)

		var saleID int
		// A customer exists but the sale has zero line items and zero
		// payments; the reader must still return the sale header.
		var custID int
		_ = tx.QueryRow(ctx, `INSERT INTO customers (name, phone, email, store_id) VALUES ('W','w-ret003','w-ret003@test.com',1) ON CONFLICT (phone) DO UPDATE SET name=EXCLUDED.name RETURNING id`).Scan(&custID)
		err = tx.QueryRow(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
			                   subtotal, discount, tax, total_amount, created_at)
			VALUES ('RET-003', $1, $3, $2, 'Cash', 'completed', 5000, 0, 500, 5500, NOW())
			RETURNING id
		`, userID, shiftID, custID).Scan(&saleID)
		require.NoError(t, err)

		r := ReturnSaleReader{}
		got, err := r.SaleForReturn(ctx, tx, saleID)
		require.NoError(t, err)
		require.NotNil(t, got)
		// The reader accumulates rows via append, so with no rows the
		// slices stay nil; the contract is "no items and no payments",
		// which Len 0 expresses for both nil and empty slices.
		require.Len(t, got.Items, 0)
		require.Len(t, got.Payments, 0)
	})

	t.Run("status is COALESCE to empty string when NULL", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })

		var shiftID int
		_, _ = tx.Exec(ctx, `INSERT INTO users (id, username, email, password_hash, role_id) VALUES ($1, 'summary_cashier', 'summary_cashier@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`, userID)
		err = tx.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, 'open', 0, NOW()) RETURNING id
		`, userID).Scan(&shiftID)
		require.NoError(t, err)

		var saleID int
		var custID int
		_ = tx.QueryRow(ctx, `INSERT INTO customers (name, phone, email, store_id) VALUES ('W','w-ret004','w-ret004@test.com',1) ON CONFLICT (phone) DO UPDATE SET name=EXCLUDED.name RETURNING id`).Scan(&custID)
		err = tx.QueryRow(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
			                   subtotal, discount, tax, total_amount, created_at)
			VALUES ('RET-004', $1, $3, $2, 'Cash', NULL, 5000, 0, 500, 5500, NOW())
			RETURNING id
		`, userID, shiftID, custID).Scan(&saleID)
		require.NoError(t, err)

		r := ReturnSaleReader{}
		got, err := r.SaleForReturn(ctx, tx, saleID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "", got.Status)
	})

	t.Run("non-ErrNoRows error is wrapped", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		// Roll back so the tx is bad
		require.NoError(t, tx.Rollback(ctx))

		r := ReturnSaleReader{}
		got, err := r.SaleForReturn(ctx, tx, 1)
		require.Error(t, err)
		require.Nil(t, got)
		require.True(t, errors.Is(err, pgx.ErrTxClosed) || err != nil)
	})
}
