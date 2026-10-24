package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// TestNameLookup_ProductIDsByName_Cap pins the P1 bound: a free-text search
// resolves to at most maxSearchIDs product IDs, ordered by id so the
// truncation is deterministic. The caller feeds the result into an
// id = ANY(...) filter, so an uncapped resolution would produce a
// pathologically large IN list.
func TestNameLookup_ProductIDsByName_Cap(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, shared.TruncateTestData(dbPool))

	// 1001 products sharing one name prefix — exactly one over the cap —
	// seeded in a single statement so the fixture stays cheap.
	_, err := dbPool.Exec(ctx, `
		INSERT INTO products (sku, name, price, status)
		SELECT 'CAP-' || lpad(i::text, 4, '0'), 'CapPrefix product ' || i, 1000, 'active'
		FROM generate_series(1, 1001) AS i`)
	require.NoError(t, err)

	ids, err := (NameLookup{}).ProductIDsByName(ctx, dbPool, "CapPrefix%")
	require.NoError(t, err)
	require.Len(t, ids, maxSearchIDs, "a 1001-match search must truncate to the cap")
	for i := 1; i < len(ids); i++ {
		require.Greater(t, ids[i], ids[i-1], "truncated result must stay ordered by id")
	}
}

// TestNameLookup_ProductIDsByName_UnderCap guards the other side of the LIMIT:
// a narrow search returns every match, unchanged by the cap.
func TestNameLookup_ProductIDsByName_UnderCap(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, shared.TruncateTestData(dbPool))

	_, err := dbPool.Exec(ctx, `
		INSERT INTO products (sku, name, price, status)
		VALUES ('UNDERCAP-1', 'UnderCap product 1', 1000, 'active'),
		       ('UNDERCAP-2', 'UnderCap product 2', 1000, 'active'),
		       ('UNDERCAP-3', 'UnderCap product 3', 1000, 'active')`)
	require.NoError(t, err)

	ids, err := (NameLookup{}).ProductIDsByName(ctx, dbPool, "UnderCap%")
	require.NoError(t, err)
	require.Len(t, ids, 3, "a below-cap search must return every match")

	// No match at all yields an empty slice, not an error.
	none, err := (NameLookup{}).ProductIDsByName(ctx, dbPool, "NoSuchPrefixAnywhere%")
	require.NoError(t, err)
	require.Empty(t, none)
}
