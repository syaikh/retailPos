package supplier

import (
	"testing"

	"retail-pos-system/internal/consignment"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/purchase"
)

// newTestRepo returns a repository wired with the product-owned
// ProductSupplierStore port and the purchase/consignment usage counters,
// mirroring the composition-root wiring in internal/wiring/wiring.go.
func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo := NewRepository(dbPool)
	repo.SetProductSupplierStore(product.SupplierLinkStore{})
	repo.SetPurchaseUsageCounter(purchase.SupplierUsageProvider{})
	repo.SetConsignmentUsageCounter(consignment.SupplierUsageProvider{})
	return repo
}
