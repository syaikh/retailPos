package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUsageProvider_CountActiveConsignmentsBySupplier pins the
// consignment half of the supplier guard: an active arrangement is live, and
// ending an arrangement with no remaining stock clears it.
func TestUsageProvider_CountActiveConsignmentsBySupplier(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	svc, supplierID, _, _ := setupArrangementNoTerms(t)

	count, err := (UsageProvider{}).CountActiveConsignmentsBySupplier(ctx, dbPool, supplierID)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "an active arrangement is a live reference")

	var arrangementID int
	err = dbPool.QueryRow(ctx, `
		SELECT id FROM consignment_arrangements
		WHERE supplier_id = $1 AND status = $2
		ORDER BY id DESC LIMIT 1
	`, supplierID, StatusActive).Scan(&arrangementID)
	require.NoError(t, err)

	_, err = svc.EndArrangement(ctx, arrangementID, nil)
	require.NoError(t, err)

	count, err = (UsageProvider{}).CountActiveConsignmentsBySupplier(ctx, dbPool, supplierID)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "an ended arrangement with no stock is not live")

	empty, err := (UsageProvider{}).CountActiveConsignmentsBySupplier(ctx, dbPool, -1)
	require.NoError(t, err)
	assert.Equal(t, 0, empty)
}
