package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"retail-pos-system/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func mustLoadJakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(fmt.Sprintf("failed to load Asia/Jakarta: %v", err))
	}
	return loc
}

func insertSaleTest(t *testing.T, pool *pgxpool.Pool, invoice string, storeID *int, total int, when time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO sales (invoice_number, cashier_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'completed', $9)
	`, invoice, 1, storeID, total, 0, 0, total, "cash", when)
	require.NoError(t, err, "insert sale %q", invoice)
}

// TestExcelExport_ValuesMatchWeb verifies that the Excel export produces the
// same KPI values as GetPeriodComparison for the same date range.
func TestExcelExport_ValuesMatchWeb(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	ctx := context.Background()
	jkt := mustLoadJakarta()
	svc := NewExcelService(repo)

	// Seed data: 3 sales across 2 Jakarta days
	// Day 1 (Jan 15 WIB):
	//   Sale A at Jan 15 16:00 UTC = Jan 15 23:00 WIB, 100000
	// Day 2 (Jan 16 WIB):
	//   Sale B at Jan 15 17:00 UTC = Jan 16 00:00 WIB, 200000
	//   Sale C at Jan 16 08:00 UTC = Jan 16 15:00 WIB, 300000
	insertSaleTest(t, testDB.Pool(), "EXCEL-TEST-A", nil, 100000,
		time.Date(2026, 1, 15, 16, 0, 0, 0, time.UTC))
	insertSaleTest(t, testDB.Pool(), "EXCEL-TEST-B", nil, 200000,
		time.Date(2026, 1, 15, 17, 0, 0, 0, time.UTC))
	insertSaleTest(t, testDB.Pool(), "EXCEL-TEST-C", nil, 300000,
		time.Date(2026, 1, 16, 8, 0, 0, 0, time.UTC))

	// Period: Jan 15–16 WIB (current), Jan 8–9 WIB (previous, no data)
	currentStart := time.Date(2026, 1, 15, 0, 0, 0, 0, jkt)
	currentEnd := time.Date(2026, 1, 17, 0, 0, 0, 0, jkt) // exclusive
	previousStart := time.Date(2026, 1, 8, 0, 0, 0, 0, jkt)
	previousEnd := time.Date(2026, 1, 10, 0, 0, 0, 0, jkt)

	// Get period comparison from API
	comparison, err := repo.GetPeriodComparison(ctx,
		currentStart, currentEnd, previousStart, previousEnd)
	require.NoError(t, err)

	// Verify the comparison data matches expectations
	assert.Equal(t, 600000, comparison.CurrentRevenue,
		"total revenue: 100000 + 200000 + 300000 = 600000")
	assert.Equal(t, 3, comparison.CurrentOrders)
	assert.Equal(t, 200000, comparison.CurrentAOV)
	assert.Equal(t, 300000, comparison.RevenuePerDay, "600000 / 2 days")

	// Generate Excel export (passing inclusive end dates -1 day to match handler)
	params := DashboardExportParams{
		PeriodLabel: "daily",
		StartDate:   currentStart,
		EndDate:     currentEnd.AddDate(0, 0, -1),
		PrevStart:   previousStart,
		PrevEnd:     previousEnd.AddDate(0, 0, -1),
		IsHourly:    false,
	}
	excelBytes, err := svc.GenerateDashboardExport(ctx, params)
	require.NoError(t, err)
	require.Greater(t, len(excelBytes), 0, "excel bytes should be non-empty")

	// Parse Excel and verify KPI values
	f, err := excelize.OpenReader(bytes.NewReader(excelBytes))
	require.NoError(t, err)
	defer f.Close()

	dashboard := "Dashboard"
	actualRevenue, _ := f.GetCellValue(dashboard, "B5")
	actualOrders, _ := f.GetCellValue(dashboard, "B6")
	actualAOV, _ := f.GetCellValue(dashboard, "B7")
	actualRevPerDay, _ := f.GetCellValue(dashboard, "B8")

	assert.Equal(t, fmt.Sprintf("%d", comparison.CurrentRevenue), actualRevenue,
		"Excel revenue should match API")
	assert.Equal(t, fmt.Sprintf("%d", comparison.CurrentOrders), actualOrders,
		"Excel orders should match API")
	assert.Equal(t, fmt.Sprintf("%d", comparison.CurrentAOV), actualAOV,
		"Excel AOV should match API")
	assert.Equal(t, fmt.Sprintf("%d", comparison.RevenuePerDay), actualRevPerDay,
		"Excel revenue per day should match API")
}
