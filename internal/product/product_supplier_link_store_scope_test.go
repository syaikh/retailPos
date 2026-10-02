package product

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// These tests pin the store-scope contract of migration
// 058_supplier_terms_store_scope.sql (audit D3, Option C) at the SQL that
// implements it. The rule under test throughout: a store inherits the global
// terms it has not overridden, may write only its own row, and cannot reach the
// global one — which belongs to the whole estate.

func scopeIntPtr(v int) *int { return &v }

// insertScopeStore creates a store to scope against. Only name is required, and
// lower(name) is unique, so a timestamp suffix is enough to stay distinct
// across runs.
func insertScopeStore(t *testing.T, suffix string) int {
	t.Helper()
	name := fmt.Sprintf("ScopeStore-%s-%d", suffix, time.Now().UnixNano())
	var id int
	require.NoError(t, dbPool.QueryRow(context.Background(),
		`INSERT INTO stores (name) VALUES ($1) RETURNING id`, name).Scan(&id))
	return id
}

func insertScopeProduct(t *testing.T, suffix string) int {
	t.Helper()
	var id int
	require.NoError(t, dbPool.QueryRow(context.Background(),
		`INSERT INTO products (sku, name, price, status) VALUES ($1, $2, 10000, 'active') RETURNING id`,
		uniqueSKU("SCOPE-P-"+suffix), "Scope Product "+suffix).Scan(&id))
	return id
}

func insertScopeSupplier(t *testing.T, suffix string) int {
	t.Helper()
	var id int
	require.NoError(t, dbPool.QueryRow(context.Background(),
		`INSERT INTO suppliers (name, code) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("ScopeSupplier-%s-%d", suffix, time.Now().UnixNano()),
		fmt.Sprintf("SCOPE-%s-%d", suffix, time.Now().UnixNano())).Scan(&id))
	return id
}

// globalLinkCost reads a product's global row straight from the table, bypassing
// the store, so a refusal can be proven to have left the row untouched.
func globalLinkCost(t *testing.T, productID, supplierID int) (int, bool) {
	t.Helper()
	var cost int
	err := dbPool.QueryRow(context.Background(),
		`SELECT unit_cost FROM product_suppliers
		 WHERE product_id = $1 AND supplier_id = $2 AND store_id IS NULL`,
		productID, supplierID).Scan(&cost)
	if err != nil {
		return 0, false
	}
	return cost, true
}

func TestSupplierLinkStore_GlobalTermsAreInheritedAndOverridden(t *testing.T) {
	store := SupplierLinkStore{}
	ctx := context.Background()

	productID := insertScopeProduct(t, "INHERIT")
	supplierID := insertScopeSupplier(t, "INHERIT")
	storeA := insertScopeStore(t, "A")
	storeB := insertScopeStore(t, "B")

	// A global row: the default every store inherits.
	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: productID, SupplierID: supplierID, UnitCost: 1000,
	}))

	t.Run("store inherits the global default", func(t *testing.T) {
		link, err := store.GetLink(ctx, dbPool, productID, supplierID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 1000, link.UnitCost)
		assert.Nil(t, link.StoreID, "the inherited row is the global one")

		links, err := store.ListLinksByProduct(ctx, dbPool, productID, &storeA)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 1000, links[0].UnitCost)
	})

	// The override is the point of the whole migration: store A negotiates its
	// own cost, and the global row must stop being what A sees.
	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: productID, SupplierID: supplierID, UnitCost: 2000,
		StoreID: &storeA,
	}))

	t.Run("own row wins over the global default", func(t *testing.T) {
		link, err := store.GetLink(ctx, dbPool, productID, supplierID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 2000, link.UnitCost, "store A's own terms, not the shadowed global row")
		require.NotNil(t, link.StoreID)
		assert.Equal(t, storeA, *link.StoreID)
	})

	t.Run("list collapses the shadowed global row", func(t *testing.T) {
		links, err := store.ListLinksByProduct(ctx, dbPool, productID, &storeA)
		require.NoError(t, err)
		require.Len(t, links, 1, "the shadowed global default must not sit next to the override")
		assert.Equal(t, 2000, links[0].UnitCost)
	})

	t.Run("one store's override does not reprice another", func(t *testing.T) {
		link, err := store.GetLink(ctx, dbPool, productID, supplierID, &storeB)
		require.NoError(t, err)
		assert.Equal(t, 1000, link.UnitCost, "store B still inherits the global default")
	})

	t.Run("unscoped read sees both rows", func(t *testing.T) {
		links, err := store.ListLinksByProduct(ctx, dbPool, productID, nil)
		require.NoError(t, err)
		assert.Len(t, links, 2, "superadmin is unrestricted and sees the override and the default")
	})
}

func TestSupplierLinkStore_StoreScopedWritesCannotTouchGlobalTerms(t *testing.T) {
	store := SupplierLinkStore{}
	ctx := context.Background()

	productID := insertScopeProduct(t, "GUARD")
	supplierID := insertScopeSupplier(t, "GUARD")
	storeA := insertScopeStore(t, "GUARD")

	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: productID, SupplierID: supplierID, UnitCost: 1000, IsPreferred: true,
	}))

	t.Run("update is refused", func(t *testing.T) {
		err := store.UpdateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 9999,
			StoreID: &storeA, // store A does not own this row
		})
		require.ErrorIs(t, err, shared.ErrProductSupplierNotFound)

		cost, _ := globalLinkCost(t, productID, supplierID)
		assert.Equal(t, 1000, cost, "the global terms must be untouched")
	})

	t.Run("delete is refused", func(t *testing.T) {
		err := store.DeleteLink(ctx, dbPool, productID, supplierID, &storeA)
		require.ErrorIs(t, err, shared.ErrProductSupplierNotFound)

		_, found := globalLinkCost(t, productID, supplierID)
		assert.True(t, found, "the global row must survive a store-scoped unlink")
	})

	t.Run("set preferred is refused", func(t *testing.T) {
		err := store.SetPreferredLink(ctx, dbPool, productID, supplierID, &storeA)
		require.ErrorIs(t, err, shared.ErrProductSupplierNotFound)

		link, err := store.GetPreferredLink(ctx, dbPool, productID, nil)
		require.NoError(t, err)
		assert.True(t, link.IsPreferred, "the global preference must survive")
	})

	t.Run("an own row is writable", func(t *testing.T) {
		require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 3000, StoreID: &storeA,
		}))
		require.NoError(t, store.UpdateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 3500, StoreID: &storeA,
		}))

		link, err := store.GetLink(ctx, dbPool, productID, supplierID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 3500, link.UnitCost)

		cost, _ := globalLinkCost(t, productID, supplierID)
		assert.Equal(t, 1000, cost, "editing store A's row must not move the global one")
	})
}

func TestSupplierLinkStore_PreferredSupplierIsPerStore(t *testing.T) {
	store := SupplierLinkStore{}
	ctx := context.Background()

	productID := insertScopeProduct(t, "PREF")
	supplierOne := insertScopeSupplier(t, "PREF-1")
	supplierTwo := insertScopeSupplier(t, "PREF-2")
	storeA := insertScopeStore(t, "PREF-A")
	storeB := insertScopeStore(t, "PREF-B")

	for _, sid := range []int{supplierOne, supplierTwo} {
		require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: sid, UnitCost: 1000,
		}))
	}

	// The estate-wide default.
	require.NoError(t, store.SetPreferredLink(ctx, dbPool, productID, supplierOne, nil))

	t.Run("a store cannot prefer a link it does not own", func(t *testing.T) {
		// Store A has no row of its own for either supplier yet, so this is a
		// refusal, not a silent no-op. A store expresses a preference by linking
		// the supplier first, which is also what the handler tells the caller.
		require.ErrorIs(t,
			store.SetPreferredLink(ctx, dbPool, productID, supplierTwo, &storeA),
			shared.ErrProductSupplierNotFound)
	})

	// Store A links both suppliers for itself, then picks between them.
	for _, sid := range []int{supplierOne, supplierTwo} {
		require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: sid, UnitCost: 900, StoreID: &storeA,
		}))
	}

	t.Run("a store may override the default", func(t *testing.T) {
		require.NoError(t, store.SetPreferredLink(ctx, dbPool, productID, supplierTwo, &storeA))
	})

	t.Run("the overriding store sees its own choice", func(t *testing.T) {
		link, err := store.GetPreferredLink(ctx, dbPool, productID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, supplierTwo, link.SupplierID)
	})

	t.Run("other stores keep the global default", func(t *testing.T) {
		link, err := store.GetPreferredLink(ctx, dbPool, productID, &storeB)
		require.NoError(t, err)
		assert.Equal(t, supplierOne, link.SupplierID, "store A changing its mind is not the estate's business")
	})

	t.Run("unscoped read leads with the global default", func(t *testing.T) {
		link, err := store.GetPreferredLink(ctx, dbPool, productID, nil)
		require.NoError(t, err)
		assert.Equal(t, supplierOne, link.SupplierID)
	})

	t.Run("a store has at most one preferred supplier", func(t *testing.T) {
		require.NoError(t, store.SetPreferredLink(ctx, dbPool, productID, supplierOne, &storeA))

		link, err := store.GetPreferredLink(ctx, dbPool, productID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, supplierOne, link.SupplierID)

		links, err := store.ListLinksByProduct(ctx, dbPool, productID, &storeA)
		require.NoError(t, err)
		preferred := 0
		for _, l := range links {
			if l.IsPreferred {
				preferred++
			}
		}
		assert.Equal(t, 1, preferred)
	})

	t.Run("HasPreferredLink follows the effective choice", func(t *testing.T) {
		has, err := store.HasPreferredLink(ctx, dbPool, productID, &storeB)
		require.NoError(t, err)
		assert.True(t, has)

		noPreference := insertScopeProduct(t, "PREF-NONE")
		has, err = store.HasPreferredLink(ctx, dbPool, noPreference, &storeB)
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestSupplierLinkStore_UnscopedWriteTargetsGlobalTerms(t *testing.T) {
	store := SupplierLinkStore{}
	ctx := context.Background()

	productID := insertScopeProduct(t, "UNSCOPED")
	supplierID := insertScopeSupplier(t, "UNSCOPED")
	storeA := insertScopeStore(t, "UNSCOPED")

	// A store row alone: an unscoped edit has no global row to hit.
	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: productID, SupplierID: supplierID, UnitCost: 4000, StoreID: &storeA,
	}))

	t.Run("unscoped update misses a store row", func(t *testing.T) {
		err := store.UpdateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 4500,
		})
		require.ErrorIs(t, err, shared.ErrProductSupplierNotFound)
	})

	t.Run("unscoped delete misses a store row", func(t *testing.T) {
		require.ErrorIs(t, store.DeleteLink(ctx, dbPool, productID, supplierID, nil),
			shared.ErrProductSupplierNotFound)
	})

	t.Run("unscoped update hits the global row once it exists", func(t *testing.T) {
		require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 5000,
		}))
		require.NoError(t, store.UpdateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 5500,
		}))

		link, err := store.GetLink(ctx, dbPool, productID, supplierID, nil)
		require.NoError(t, err)
		assert.Equal(t, 5500, link.UnitCost)

		own, err := store.GetLink(ctx, dbPool, productID, supplierID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 4000, own.UnitCost, "the store row is a separate row, not a view of the global one")
	})
}

func TestSupplierLinkStore_DuplicateLinkRejectedPerScope(t *testing.T) {
	store := SupplierLinkStore{}
	ctx := context.Background()

	productID := insertScopeProduct(t, "DUP")
	supplierID := insertScopeSupplier(t, "DUP")
	storeA := insertScopeStore(t, "DUP")

	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: productID, SupplierID: supplierID, UnitCost: 1000,
	}))

	// The old constraint was UNIQUE (product_id, supplier_id), which made a
	// store override impossible to express at all. The new one is unique per
	// scope, so the same pair is rejected twice in one scope and accepted once
	// per scope.
	t.Run("duplicate global link is rejected", func(t *testing.T) {
		err := store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 1001,
		})
		require.Error(t, err)
	})

	t.Run("the same pair in another store is accepted", func(t *testing.T) {
		require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 2000, StoreID: &storeA,
		}))
	})

	t.Run("duplicate store link is rejected", func(t *testing.T) {
		err := store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
			ProductID: productID, SupplierID: supplierID, UnitCost: 2001, StoreID: &storeA,
		})
		require.Error(t, err)
	})
}

// TestGetAllProducts_SupplierFilterIsStoreScoped guards a leak that lived outside
// the supplier module entirely, so the link store's own tests could not see it:
// the product list's ?supplier_id= filter used an EXISTS subquery over
// product_suppliers with no store predicate. A store filtering "products from
// supplier X" was therefore shown products whose only link to X was another
// store's private terms -- exactly the visibility migration 058 was written to
// close, reachable through a different route.
func TestGetAllProducts_SupplierFilterIsStoreScoped(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	storeA := insertScopeStore(t, "A")
	storeB := insertScopeStore(t, "B")
	supplierID := insertScopeSupplier(t, "filter")
	repo := NewRepository(dbPool)
	store := SupplierLinkStore{}
	ctx := context.Background()

	// linkedToB is reachable only through store B's own terms.
	linkedToB := insertScopeProduct(t, "linked-to-b")
	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: linkedToB, SupplierID: supplierID, UnitCost: 7000, StoreID: scopeIntPtr(storeB),
	}))

	// globallyLinked carries the estate-wide default, so every store inherits it.
	globallyLinked := insertScopeProduct(t, "globally-linked")
	require.NoError(t, store.CreateLink(ctx, dbPool, &shared.ProductSupplier{
		ProductID: globallyLinked, SupplierID: supplierID, UnitCost: 7000,
	}))

	visibleTo := func(storeID *int) map[int]bool {
		t.Helper()
		rows, total, err := repo.GetAllProducts(ctx, 200, 0, "", nil, "name", "asc",
			nil, storeID, "", scopeIntPtr(supplierID), nil, "")
		require.NoError(t, err)
		require.GreaterOrEqual(t, total, 1)

		ids := make(map[int]bool, len(rows))
		for _, p := range rows {
			ids[p.ID] = true
		}
		return ids
	}

	t.Run("a store cannot find a product through another store's link", func(t *testing.T) {
		seen := visibleTo(scopeIntPtr(storeA))
		assert.False(t, seen[linkedToB], "store B's private terms must not surface in store A")
		assert.True(t, seen[globallyLinked], "the estate-wide default is inherited by every store")
	})

	t.Run("the owning store sees its own link", func(t *testing.T) {
		seen := visibleTo(scopeIntPtr(storeB))
		assert.True(t, seen[linkedToB])
		assert.True(t, seen[globallyLinked])
	})

	t.Run("superadmin stays unrestricted", func(t *testing.T) {
		seen := visibleTo(nil)
		assert.True(t, seen[linkedToB])
		assert.True(t, seen[globallyLinked])
	})

	t.Run("the total count agrees with the page", func(t *testing.T) {
		// The count query is a separate statement from the data query and had the
		// same unscoped subquery; a fix applied to only one of them would make
		// pagination lie, so both are asserted through the same call.
		_, total, err := repo.GetAllProducts(ctx, 1, 0, "", nil, "name", "asc",
			nil, scopeIntPtr(storeA), "", scopeIntPtr(supplierID), nil, "")
		require.NoError(t, err)
		assert.Equal(t, 1, total, "store A inherits only the global link")
	})
}
