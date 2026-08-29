package supplier

import (
	"testing"

	"retail-pos-system/internal/product"
)

// newTestRepo returns a repository wired with the product-owned
// ProductSupplierStore port, mirroring the composition-root wiring in
// internal/wiring/wiring.go.
func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo := NewRepository(dbPool)
	repo.SetProductSupplierStore(product.SupplierLinkStore{})
	return repo
}
