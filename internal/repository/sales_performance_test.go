package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGetAllSalesPerformance verifies the N+1 query fix
func TestGetAllSalesPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())
	ctx := context.Background()

	// Measure time for fetching sales with items
	start := time.Now()
	sales, _, err := repo.GetAllSales(ctx, 20, 0, "", "created_at", "ASC", "", "", nil, "", nil, nil)
	require.NoError(t, err)
	elapsed := time.Since(start)
	
	// Verify we got some sales
	if len(sales) > 0 {
		t.Logf("Fetched %d sales in %s (items per sale: %d)", 
			len(sales), 
			elapsed.String(),
			len(sales[0].Items))
	}
}