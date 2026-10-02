package stockopname

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScopeProductIDs_SupplierScopeIsStoreScoped covers the second place a store
// could learn about another store's supplier terms.
//
// A "supplier" opname scope asks for every product that supplier supplies, and
// that answer came from product_suppliers with no store predicate. After
// migration 058 a link row names the store whose terms it holds, so an unscoped
// answer pulled another store's private supplier relationships into this store's
// stock count -- the same cross-store visibility the supplier module fixes, but
// reached through the stock opname product universe instead of a supplier route.
//
// The test drives the real product.MetaLookup through the wired repository, so it
// covers the port signature as well as the SQL.
func TestScopeProductIDs_SupplierScopeIsStoreScoped(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	repo := newTestRepository()

	newStore := func(suffix string) int {
		t.Helper()
		var id int
		require.NoError(t, dbPool.QueryRow(ctx,
			`INSERT INTO stores (name) VALUES ($1) RETURNING id`,
			fmt.Sprintf("OpnameScope-%s-%d", suffix, time.Now().UnixNano())).Scan(&id))
		t.Cleanup(func() {
			_, _ = dbPool.Exec(context.Background(), `DELETE FROM stores WHERE id = $1`, id)
		})
		return id
	}
	newSupplier := func() int {
		t.Helper()
		var id int
		ts := time.Now().UnixNano()
		require.NoError(t, dbPool.QueryRow(ctx,
			`INSERT INTO suppliers (name, code) VALUES ($1, $2) RETURNING id`,
			fmt.Sprintf("OpnameScopeSupplier-%d", ts), fmt.Sprintf("OPNSC-%d", ts)).Scan(&id))
		t.Cleanup(func() {
			_, _ = dbPool.Exec(context.Background(), `DELETE FROM suppliers WHERE id = $1`, id)
		})
		return id
	}
	newProduct := func(suffix string) int {
		t.Helper()
		ts := time.Now().UnixNano()
		var id int
		require.NoError(t, dbPool.QueryRow(ctx,
			`INSERT INTO products (sku, name, price, status) VALUES ($1, $2, 10000, 'active') RETURNING id`,
			fmt.Sprintf("OPNSC-%s-%d", suffix, ts), "Opname Scope "+suffix).Scan(&id))
		// These products carry store_id IS NULL, which is the "visible in every
		// store-scoped read" case. Left behind they leak into unrelated tests the
		// same way, so remove them here rather than relying on another test's reset.
		t.Cleanup(func() {
			_, _ = dbPool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id)
		})
		return id
	}

	storeA := newStore("A")
	storeB := newStore("B")
	supplierID := newSupplier()

	// onlyStoreB is linked to the supplier in store B alone.
	onlyStoreB := newProduct("only-b")
	storeBPtr := storeB
	_, err := dbPool.Exec(ctx,
		`INSERT INTO product_suppliers (product_id, supplier_id, unit_cost, store_id) VALUES ($1, $2, 5000, $3)`,
		onlyStoreB, supplierID, storeBPtr)
	require.NoError(t, err)

	// globalLink carries the estate-wide default, inherited by every store.
	globalLink := newProduct("global")
	_, err = dbPool.Exec(ctx,
		`INSERT INTO product_suppliers (product_id, supplier_id, unit_cost, store_id) VALUES ($1, $2, 5000, NULL)`,
		globalLink, supplierID)
	require.NoError(t, err)

	resolve := func(storeID *int) []int {
		t.Helper()
		ids, err := repo.ScopeProductIDs(ctx, dbPool,
			Scope{ScopeType: "supplier", ScopeID: int64(supplierID)}, storeID)
		require.NoError(t, err)
		return ids
	}
	contains := func(ids []int, want int) bool {
		for _, id := range ids {
			if id == want {
				return true
			}
		}
		return false
	}

	t.Run("a store's count excludes another store's supplier terms", func(t *testing.T) {
		storeAPtr := storeA
		ids := resolve(&storeAPtr)
		assert.False(t, contains(ids, onlyStoreB),
			"store B's private supplier link must not enter store A's stock count")
		assert.True(t, contains(ids, globalLink),
			"the estate-wide default applies to every store")
	})

	t.Run("the owning store counts its own link", func(t *testing.T) {
		ids := resolve(&storeBPtr)
		assert.True(t, contains(ids, onlyStoreB))
		assert.True(t, contains(ids, globalLink))
	})

	t.Run("an unrestricted caller still sees every link", func(t *testing.T) {
		ids := resolve(nil)
		assert.True(t, contains(ids, onlyStoreB))
		assert.True(t, contains(ids, globalLink))
	})

	t.Run("a non-supplier scope is unaffected", func(t *testing.T) {
		// category scope reads products, not the link table; the added parameter
		// must not change its behaviour.
		storeAPtr := storeA
		withStore, err := repo.ScopeProductIDs(ctx, dbPool,
			Scope{ScopeType: "category", ScopeID: 1}, &storeAPtr)
		require.NoError(t, err)
		withoutStore, err := repo.ScopeProductIDs(ctx, dbPool,
			Scope{ScopeType: "category", ScopeID: 1}, nil)
		require.NoError(t, err)
		assert.Equal(t, withoutStore, withStore)
	})
}
