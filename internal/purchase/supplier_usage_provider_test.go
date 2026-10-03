package purchase

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var poUsageSeq int64

func poUsageSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func insertUsagePO(ctx context.Context, t *testing.T, supplierID, userID int, status string) {
	t.Helper()
	// po_number is varchar(30); keep the token short while staying unique.
	seq := (time.Now().UnixNano()%1000000)*100 + atomic.AddInt64(&poUsageSeq, 1)
	poNumber := fmt.Sprintf("USG-%08d-%s", seq, status)
	_, err := dbPool.Exec(ctx, `
		INSERT INTO purchase_orders (po_number, supplier_id, store_id, status, created_by, updated_by)
		VALUES ($1, $2, 1, $3, $4, $4)
	`, poNumber, supplierID, status, userID)
	require.NoError(t, err)
}

// TestUsageProvider_CountOpenPurchaseOrdersBySupplier pins the
// deactivation contract: draft, confirmed, and partial_received orders still owe
// goods and block deactivation; fully_received and cancelled are terminal and
// must not.
func TestUsageProvider_CountOpenPurchaseOrdersBySupplier(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "PO Usage Supplier "+poUsageSuffix())
	userID := insertTestUser(ctx, t, "po_usage_user_"+poUsageSuffix())

	insertUsagePO(ctx, t, supplierID, userID, StatusDraft)
	insertUsagePO(ctx, t, supplierID, userID, StatusConfirmed)
	insertUsagePO(ctx, t, supplierID, userID, StatusPartialReceived)
	insertUsagePO(ctx, t, supplierID, userID, StatusFullyReceived)
	insertUsagePO(ctx, t, supplierID, userID, StatusCancelled)

	count, err := (UsageProvider{}).CountOpenPurchaseOrdersBySupplier(ctx, dbPool, supplierID)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "only draft/confirmed/partial_received are open")

	empty, err := (UsageProvider{}).CountOpenPurchaseOrdersBySupplier(ctx, dbPool, -1)
	require.NoError(t, err)
	assert.Equal(t, 0, empty)
}
