package customer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// TestGroupLookup_CustomerGroupID covers the sale module's CustomerGroupReader
// implementation used by the cart customer-swap re-pricing path (backend
// review B5): a grouped customer resolves to its group; an ungrouped customer
// and a nonexistent id both resolve to (nil, nil) — base pricing is never an
// error.
func TestGroupLookup_CustomerGroupID(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	require.NoError(t, shared.TruncateTestData(dbPool))
	ctx := context.Background()

	var groupID int
	err := dbPool.QueryRow(ctx, `INSERT INTO customer_groups (name) VALUES ('grouplookup-group') RETURNING id`).Scan(&groupID)
	require.NoError(t, err)

	var groupedID, plainID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO customers (name, phone, email, is_walk_in, is_active, store_id, customer_group_id)
		 VALUES ('Grouped Lookup', '081110000001', 'grouped-lookup@test.com', false, true, 1, $1) RETURNING id`,
		groupID,
	).Scan(&groupedID)
	require.NoError(t, err)
	err = dbPool.QueryRow(ctx,
		`INSERT INTO customers (name, phone, email, is_walk_in, is_active, store_id)
		 VALUES ('Plain Lookup', '081110000002', 'plain-lookup@test.com', false, true, 1) RETURNING id`,
	).Scan(&plainID)
	require.NoError(t, err)

	lookup := GroupLookup{}

	t.Run("grouped customer resolves to its group", func(t *testing.T) {
		got, err := lookup.CustomerGroupID(ctx, dbPool, groupedID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, groupID, *got)
	})

	t.Run("ungrouped customer resolves to nil", func(t *testing.T) {
		got, err := lookup.CustomerGroupID(ctx, dbPool, plainID)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("nonexistent customer resolves to nil, never an error", func(t *testing.T) {
		got, err := lookup.CustomerGroupID(ctx, dbPool, 99999999)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
