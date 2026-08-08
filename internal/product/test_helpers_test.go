package product

import (
	"retail-pos-system/internal/inventory"
)

// testRepo builds a Repository over the test database wired with the
// production providers the composition root (internal/wiring) wires: the
// product_stock row writer (inventory.ProductStockWriter). Test constructors
// must use testRepo instead of NewRepository whenever a stock write path
// (Create/Update/Restore/BulkUpsert/BulkInsert) may run — an unwired writer
// fails fast.
func testRepo() *Repository {
	repo := NewRepository(dbPool)
	repo.SetProductStockWriter(inventory.ProductStockWriter{})
	return repo
}
