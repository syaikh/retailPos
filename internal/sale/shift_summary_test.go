package sale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func insertTestUser(t *testing.T) int {
	t.Helper()
	ctx := context.Background()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('summary_cashier', 'summary_cashier@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestPaymentMethod(t *testing.T, code string) int {
	t.Helper()
	ctx := context.Background()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO payment_methods (code, name, is_active, requires_reference) VALUES ($1, $2, true, false) ON CONFLICT (code) DO UPDATE SET is_active = true RETURNING id`, code, code).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestShiftSummary_SplitPayment_DoesNotInflateTotalSales(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	userID := insertTestUser(t)
	var custID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, email, phone)
		VALUES ('Summary Customer', 'summary@test.com', '0812')
		RETURNING id
	`).Scan(&custID)
	require.NoError(t, err)

	var shiftID int
	err = dbPool.QueryRow(ctx, `
		INSERT INTO shifts (user_id, status, opening_balance, opened_at)
		VALUES ($1, 'open', 0, NOW()) RETURNING id
	`, userID).Scan(&shiftID)
	require.NoError(t, err)

	cashID := insertTestPaymentMethod(t, "CASH")
	cardID := insertTestPaymentMethod(t, "CARD")

	var saleID int
	err = dbPool.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
		                   subtotal, discount, tax, total_amount, created_at)
		VALUES ('SPLIT-001', $1, $2, $3, 'Cash', 'completed', 100000, 0, 10000, 110000, NOW())
		RETURNING id
	`, userID, custID, shiftID).Scan(&saleID)
	require.NoError(t, err)

	for _, p := range []struct {
		methodID int
		code     string
		amount   int
	}{
		{cashID, "CASH", 50000},
		{cardID, "CARD", 60000},
	} {
		_, err = dbPool.Exec(ctx, `
			INSERT INTO sale_payments (sale_id, payment_method_id, payment_method_code, amount)
			VALUES ($1, $2, $3, $4)
		`, saleID, p.methodID, p.code, p.amount)
		require.NoError(t, err)
	}

	summary, err := (ShiftSummaryProvider{}).ShiftSummary(ctx, dbPool, shiftID)
	require.NoError(t, err)

	// TotalSales must reflect the single sale's total, not one row per payment.
	require.Equal(t, 110000, summary.TotalSales)
	require.Equal(t, 1, summary.TotalTransactions)
	// Payment breakdown still sums both payment rows.
	require.Equal(t, 50000, summary.TotalCashSales)
	require.Equal(t, 60000, summary.TotalNonCashSales)
}

func TestShiftSummary_LegacySaleWithoutPayments_FallsBackToPaymentMethod(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	userID := insertTestUser(t)
	var custID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, email, phone)
		VALUES ('Legacy Customer', 'legacy@test.com', '0813')
		RETURNING id
	`).Scan(&custID)
	require.NoError(t, err)

	var shiftID int
	err = dbPool.QueryRow(ctx, `
		INSERT INTO shifts (user_id, status, opening_balance, opened_at)
		VALUES ($1, 'open', 0, NOW()) RETURNING id
	`, userID).Scan(&shiftID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
		                   subtotal, discount, tax, total_amount, created_at)
		VALUES ('LEGACY-001', $1, $2, $3, 'Cash', 'completed', 50000, 0, 0, 50000, NOW())
	`, userID, custID, shiftID)
	require.NoError(t, err)

	summary, err := (ShiftSummaryProvider{}).ShiftSummary(ctx, dbPool, shiftID)
	require.NoError(t, err)

	require.Equal(t, 50000, summary.TotalCashSales)
	require.Equal(t, 0, summary.TotalNonCashSales)
	require.Equal(t, 50000, summary.TotalSales)
	require.Equal(t, 1, summary.TotalTransactions)
}
