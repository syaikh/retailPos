package inventory

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	"retail-pos-system/internal/product"
	"retail-pos-system/internal/storagelocation"
)

// newTestRepo builds a Repository over the test database wired with every port
// the composition root (internal/wiring) wires in production: the
// storage_locations read port (storagelocation.RackProvider) and the products
// sku/name read port (product.MetaLookup).
func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo := NewRepository(dbPool)
	wireAllProviders(repo)
	return repo
}

// newMockRepo builds a Repository over a pgxmock pool wired with the same
// production providers. The real providers issue SQL against the mock, so mock
// tests that exercise port-backed methods must set matching expectations.
func newMockRepo(mock pgxmock.PgxPoolIface) *Repository {
	repo := NewRepository(mock)
	wireAllProviders(repo)
	return repo
}

func wireAllProviders(repo *Repository) {
	repo.SetLocationRackProvider(storagelocation.RackProvider{})
	repo.SetProductMetaProvider(product.MetaLookup{})
}
