package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertAuditLog inserts an audit log entry at a specific time for testing.
func insertAuditLog(t *testing.T, pool *pgxpool.Pool, userID *int, action string, when time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO audit_logs (user_id, role, action, entity_type, entity_id, created_at)
		VALUES ($1, 'admin', $2, 'test', 0, $3)
	`, userID, action, when)
	require.NoError(t, err, "insert audit log %q", action)
}

// TestGetAllSales_JakartaDateBoundary verifies that GetAllSales date filtering
// uses Asia/Jakarta timezone: a sale at 2025-06-15 23:59:59 UTC (= 2025-06-16 06:59:59 WIB)
// should be included when filtering by Jakarta date 2025-06-16, not 2025-06-15.
func TestGetAllSales_JakartaDateBoundary(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())
	ctx := context.Background()
	jkt := mustLoadJakarta()

	// Sale at 2025-06-15 23:59:59 UTC = 2025-06-16 06:59:59 WIB (next Jakarta day)
	lateUtc := time.Date(2025, 6, 15, 23, 59, 59, 0, time.UTC)
	insertSale(t, testDB.Pool(), "TZ-BOUNDARY-LATE", nil, 50000, lateUtc)

	// Sale at 2025-06-15 16:59:59 UTC = 2025-06-15 23:59:59 WIB (same Jakarta day)
	earlyUtc := time.Date(2025, 6, 15, 16, 59, 59, 0, time.UTC)
	insertSale(t, testDB.Pool(), "TZ-BOUNDARY-EARLY", nil, 30000, earlyUtc)

	t.Run("late sale included in next Jakarta day", func(t *testing.T) {
		// Filter Jakarta 2025-06-16: should include lateUtc sale
		sales, total, err := repo.GetAllSales(
			ctx, 100, 0, "", "", "",
			"2025-06-16", "2025-06-17", nil, "", nil, nil,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, total, "should find exactly 1 sale on 2025-06-16 Jakarta")
		if len(sales) > 0 {
			assert.Equal(t, "TZ-BOUNDARY-LATE", sales[0].InvoiceNumber)
		}
	})

	t.Run("late sale excluded from previous Jakarta day", func(t *testing.T) {
		// Filter Jakarta 2025-06-15: should NOT include lateUtc sale
		sales, total, err := repo.GetAllSales(
			ctx, 100, 0, "", "", "",
			"2025-06-15", "2025-06-16", nil, "", nil, nil,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, total, "should find exactly 1 sale on 2025-06-15 Jakarta")
		if len(sales) > 0 {
			assert.Equal(t, "TZ-BOUNDARY-EARLY", sales[0].InvoiceNumber)
		}
	})

	t.Run("both sales in combined range", func(t *testing.T) {
		sales, total, err := repo.GetAllSales(
			ctx, 100, 0, "", "", "",
			"2025-06-15", "2025-06-17", nil, "", nil, nil,
		)
		require.NoError(t, err)
		assert.Equal(t, 2, total, "should find both sales across 2 Jakarta days")
		_ = sales
	})

	// Verify the UTC timestamps are correct for the boundary test
	t.Run("sanity check UTC timestamps", func(t *testing.T) {
		assert.Equal(t, 23, lateUtc.Hour())
		assert.Equal(t, 59, lateUtc.Minute())
		assert.Equal(t, 59, lateUtc.Second())

		lateInJkt := lateUtc.In(jkt)
		assert.Equal(t, 6, lateInJkt.Hour(), "23:59 UTC = 06:59 WIB next day")
		assert.Equal(t, 16, lateInJkt.Day(), "should be 16th in Jakarta")

		earlyInJkt := earlyUtc.In(jkt)
		assert.Equal(t, 15, earlyInJkt.Day(), "16:59 UTC = 23:59 WIB on 15th")
	})
}

// TestGetAvailableYears_JakartaBoundary verifies that GetAvailableYears uses
// AT TIME ZONE 'Asia/Jakarta' so a sale at 2024-12-31 18:00:00 UTC
// (= 2025-01-01 01:00:00 WIB) counts as year 2025, not 2024.
func TestGetAvailableYears_JakartaBoundary(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())
	ctx := context.Background()

	// Sale at 2024-12-31 18:00:00 UTC = 2025-01-01 01:00:00 WIB (next Jakarta year!)
	crossYear := time.Date(2024, 12, 31, 18, 0, 0, 0, time.UTC)
	insertSale(t, testDB.Pool(), "TZ-YEAR-CROSS", nil, 10000, crossYear)

	years, err := repo.GetAvailableYears(ctx, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(years), 1, "should have at least 1 year")

	assert.Contains(t, years, 2025,
		"cross-year sale should be counted as 2025 in Jakarta timezone")
	assert.NotContains(t, years, 2024,
		"cross-year sale should NOT be counted as 2024 in Jakarta timezone")

	// Sanity check
	jkt := mustLoadJakarta()
	inJkt := crossYear.In(jkt)
	assert.Equal(t, 2025, inJkt.Year(), "2024-12-31 18:00 UTC = 2025-01-01 WIB")
	assert.Equal(t, time.January, inJkt.Month())
	assert.Equal(t, 1, inJkt.Day())
}

// TestGetDualChartData_JakartaCrossMidnight verifies that GetDualChartData
// correctly aggregates revenue by Jakarta date, even when sales cross
// Jakarta midnight (17:00 UTC).
func TestGetDualChartData_JakartaCrossMidnight(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())
	ctx := context.Background()
	jkt := mustLoadJakarta()

	// Sale June 15 16:00 UTC = June 15 23:00 WIB (same Jakarta day)
	insertSale(t, testDB.Pool(), "TZ-CHART-EARLY", nil, 1000,
		time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC))

	// Sale June 15 17:00 UTC = June 16 00:00 WIB (next Jakarta day!)
	insertSale(t, testDB.Pool(), "TZ-CHART-MIDNIGHT", nil, 2000,
		time.Date(2025, 6, 15, 17, 0, 0, 0, time.UTC))

	// Sale June 16 16:00 UTC = June 16 23:00 WIB (same Jakarta day)
	insertSale(t, testDB.Pool(), "TZ-CHART-LATE", nil, 3000,
		time.Date(2025, 6, 16, 16, 0, 0, 0, time.UTC))

	// Current period: June 15–16 (Jakarta dates), previous: June 8–9 (no data)
	currentStart := time.Date(2025, 6, 15, 0, 0, 0, 0, jkt)
	currentEnd := time.Date(2025, 6, 17, 0, 0, 0, 0, jkt)
	previousStart := time.Date(2025, 6, 8, 0, 0, 0, 0, jkt)
	previousEnd := time.Date(2025, 6, 10, 0, 0, 0, 0, jkt)

	current, previous, err := repo.GetDualChartData(ctx, currentStart, currentEnd, previousStart, previousEnd)
	require.NoError(t, err)
	// currentEnd=June 17 exclusive → series includes June 15, 16, 17 (3 entries)
	require.Equal(t, 3, len(current), "current period has 3 days (June 15-17)")
	require.Equal(t, 3, len(previous), "previous period has 3 days")

	// Build a map for assertions
	currentByDate := make(map[string]int)
	for _, dp := range current {
		currentByDate[dp.Date] = dp.Total
	}
	previousByDate := make(map[string]int)
	for _, dp := range previous {
		previousByDate[dp.Date] = dp.Total
	}

	t.Logf("Current data: %+v", currentByDate)

	// June 15: only the early sale (1000)
	assert.Equal(t, 1000, currentByDate["2025-06-15"],
		"June 15 should only have the early sale (1000)")

	// June 16: midnight-crossing (2000) + late (3000) = 5000
	assert.Equal(t, 5000, currentByDate["2025-06-16"],
		"June 16 should have midnight-crossing + late sales (2000+3000)")

	// June 17: exclusive boundary, no data
	assert.Equal(t, 0, currentByDate["2025-06-17"],
		"June 17 is the exclusive end, should have no data")

	// Previous period should have no data
	for _, dp := range previous {
		assert.Equal(t, 0, dp.Total, "previous period should have no sales")
	}
}

// TestAuditLogBoundary_Integration verifies that the audit log GetAll boundary
// < endDate.Add(24h) correctly includes all entries for the end date.
// A log created at 2025-06-15 23:59:59 UTC (= 2025-06-16 06:59:59 WIB) should
// be included when endDate = 2025-06-16 but excluded when endDate = 2025-06-15.
func TestAuditLogBoundary_Integration(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())
	ctx := context.Background()
	jkt := mustLoadJakarta()

	// Audit log at 2025-06-15 23:59:59 UTC = 2025-06-16 06:59:59 WIB
	lateUtc := time.Date(2025, 6, 15, 23, 59, 59, 0, time.UTC)
	userID := 1 // superadmin from seed data
	insertAuditLog(t, testDB.Pool(), &userID, "tz-test-late", lateUtc)

	// Audit log at 2025-06-15 16:59:59 UTC = 2025-06-15 23:59:59 WIB
	earlyUtc := time.Date(2025, 6, 15, 16, 59, 59, 0, time.UTC)
	insertAuditLog(t, testDB.Pool(), &userID, "tz-test-early", earlyUtc)

	t.Run("late log included when endDate=2025-06-16", func(t *testing.T) {
		endDate := time.Date(2025, 6, 16, 0, 0, 0, 0, jkt)
		logs, total, err := repo.GetAll(ctx, 100, 0, nil, "", "", "", nil, &endDate)
		require.NoError(t, err)
		// Should include lateUtc log (June 16 Jakarta)
		found := false
		for _, l := range logs {
			if l.Action == "tz-test-late" {
				found = true
				break
			}
		}
		assert.True(t, found, "late log should be included when endDate=2025-06-16")
		_ = total
	})

	t.Run("late log excluded when endDate=2025-06-15", func(t *testing.T) {
		endDate := time.Date(2025, 6, 15, 0, 0, 0, 0, jkt)
		logs, total, err := repo.GetAll(ctx, 100, 0, nil, "", "", "", nil, &endDate)
		require.NoError(t, err)
		found := false
		for _, l := range logs {
			if l.Action == "tz-test-late" {
				found = true
				break
			}
		}
		assert.False(t, found, "late log should be excluded when endDate=2025-06-15")
		_ = total
	})

	t.Run("early log included when endDate=2025-06-15", func(t *testing.T) {
		endDate := time.Date(2025, 6, 15, 0, 0, 0, 0, jkt)
		logs, total, err := repo.GetAll(ctx, 100, 0, nil, "", "", "", nil, &endDate)
		require.NoError(t, err)
		found := false
		for _, l := range logs {
			if l.Action == "tz-test-early" {
				found = true
				break
			}
		}
		assert.True(t, found, "early log should be included when endDate=2025-06-15")
		_ = total
	})
}
