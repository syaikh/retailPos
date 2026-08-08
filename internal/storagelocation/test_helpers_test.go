package storagelocation

import "retail-pos-system/internal/store"

// newTestRepository returns a Repository wired with the store-owned
// StoreExistenceProvider port, mirroring the internal/wiring composition. Tests
// exercise the same provider implementation that runs in production.
func newTestRepository() *Repository {
	repo := NewRepository(dbPool)
	repo.SetStoreExistenceProvider(store.StoreExistenceProvider{})
	return repo
}
