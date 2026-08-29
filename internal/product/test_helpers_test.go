package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/inventory"
)

// testRepo builds a Repository over the test database wired with the
// production providers the composition root (internal/wiring) wires: the
// product_stock row writer (inventory.StockWriter). Test constructors
// must use testRepo instead of NewRepository whenever a stock write path
// (Create/Update/Restore/BulkUpsert/BulkInsert) may run — an unwired writer
// fails fast.
func testRepo() *Repository {
	repo := NewRepository(dbPool)
	repo.SetProductStockWriter(inventory.StockWriter{})
	return repo
}

func insertTestStore(ctx context.Context, t *testing.T, name string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ($1, true) RETURNING id`, name).Scan(&id)
	require.NoError(t, err)
	return id
}
