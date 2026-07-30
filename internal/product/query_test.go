package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedTestProduct(t *testing.T, repo *Repository, ctx context.Context, p *Product) {
	t.Helper()
	err := repo.CreateProduct(ctx, p)
	require.NoError(t, err)
	require.Greater(t, p.ID, 0)
}

func createTestCategory(t *testing.T, ctx context.Context, name string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO categories (name, slug, is_active) VALUES ($1, $1, true) RETURNING id`, name).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestGetAllProducts_SearchFilter(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	skuA := uniqueSKU("QF-SEARCH-A")
	skuB := uniqueSKU("QF-SEARCH-B")
	skuC := uniqueSKU("QF-SEARCH-C")
	skuD := uniqueSKU("QF-SEARCH-D")
	barcodeD := uniqueSKU("BC-SEARCH-D")

	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuA, Name: "Alpha Gadget", Price: 10000, Cost: 5000,
		Stock: 10, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuB, Name: "Beta Gadget", Price: 20000, Cost: 10000,
		Stock: 5, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuC, Name: "Omega Widget", Price: 30000, Cost: 15000,
		Stock: 3, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuD, Name: "Delta Barcode", Price: 40000, Cost: 20000,
		Stock: 7, Status: "active", Barcode: &barcodeD,
	})

	t.Run("search matches by name", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "Alpha", nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, products, 1)
		assert.Equal(t, skuA, products[0].SKU)
	})

	t.Run("search matches by SKU", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, skuB, nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, products, 1)
		assert.Equal(t, "Beta Gadget", products[0].Name)
	})

	t.Run("search matches by partial SKU", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "QF-SEARCH", nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		// All 4 seeded products have "QF-SEARCH" prefix in their SKU
		assert.GreaterOrEqual(t, total, 4, "expected at least 4 products (the QF-SEARCH group) to match partial SKU")
		assert.GreaterOrEqual(t, len(products), 4)
	})

	t.Run("search matches by barcode", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, barcodeD, nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, products, 1)
		assert.Equal(t, skuD, products[0].SKU)
	})

	t.Run("search matches by partial barcode", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "BC-SEARCH", nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, products, 1)
		assert.Equal(t, skuD, products[0].SKU)
	})

	t.Run("search matches multiple via tsquery", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "Gadget", nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, products, 2)
	})

	t.Run("search with no match", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "nonexistent-xyz", nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, products)
	})
}

func TestGetAllProducts_CategoryFilter(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	catID1 := createTestCategory(t, ctx, uniqueSKU("QF-Cat-Filter-1"))
	catID2 := createTestCategory(t, ctx, uniqueSKU("QF-Cat-Filter-2"))

	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-CAT-A"), Name: "CatFilterA", CategoryID: &catID1,
		Price: 1000, Cost: 500, Stock: 10, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-CAT-B"), Name: "CatFilterB", CategoryID: &catID2,
		Price: 2000, Cost: 1000, Stock: 5, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-CAT-C"), Name: "CatFilterC", CategoryID: &catID1,
		Price: 3000, Cost: 1500, Stock: 7, Status: "active",
	})

	t.Run("single category filter", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "", []int{catID1}, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, products, 2)
	})

	t.Run("multiple category filter", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "", []int{catID1, catID2}, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, products, 3)
	})

	t.Run("category filter with no match", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "", []int{999999}, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, products)
	})
}

func TestGetAllProducts_StatusFilter(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-STATUS-A"), Name: "StatusActive", Price: 1000, Cost: 500, Stock: 10, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-STATUS-I"), Name: "StatusInactive", Price: 2000, Cost: 1000, Stock: 5, Status: "inactive",
	})

	t.Run("filter by active", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "", nil, "", "", nil, nil, "active", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, p := range products {
			assert.Equal(t, "active", p.Status)
		}
	})

	t.Run("filter by inactive", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "", nil, "", "", nil, nil, "inactive", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, p := range products {
			assert.Equal(t, "inactive", p.Status)
		}
	})
}

func TestGetAllProducts_MaxStockFilter(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	skuA := uniqueSKU("QF-STOCK-A")
	skuB := uniqueSKU("QF-STOCK-B")

	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuA, Name: "StockFilterA", Price: 1000, Cost: 500, Stock: 1, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuB, Name: "StockFilterB", Price: 2000, Cost: 1000, Stock: 9999, Status: "active",
	})

	maxStock := 1
	products, _, err := repo.GetAllProducts(ctx, 100, 0, "", nil, "", "", &maxStock, nil, "", nil)
	require.NoError(t, err)

	found := false
	for _, p := range products {
		if p.SKU == skuA {
			found = true
		}
		if p.SKU == skuB {
			t.Error("StockFilterB (stock=9999) should not appear with maxStock=1")
		}
	}
	assert.True(t, found, "StockFilterA (stock=1) should appear with maxStock=1")
}

func TestGetAllProducts_Pagination(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		seedTestProduct(t, repo, ctx, &Product{
			SKU: uniqueSKU("QF-PAGE"), Name: "PageTest",
			Price: 1000 * (i + 1), Cost: 500 * (i + 1), Stock: 10, Status: "active",
		})
	}

	t.Run("first page", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 2, 0, "", nil, "v.id", "ASC", nil, nil, "", nil)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(products), 2)
		assert.GreaterOrEqual(t, total, 3)
	})

	t.Run("second page", func(t *testing.T) {
		products, _, err := repo.GetAllProducts(ctx, 2, 2, "", nil, "v.id", "ASC", nil, nil, "", nil)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(products), 2)
	})

	t.Run("offset beyond total", func(t *testing.T) {
		products, _, err := repo.GetAllProducts(ctx, 10, 100, "", nil, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Empty(t, products)
	})
}

func TestGetAllProducts_SortOptions(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-SORT-1"), Name: "Zebra Item", Price: 5000, Cost: 2500, Stock: 10, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-SORT-2"), Name: "Apple Item", Price: 3000, Cost: 1500, Stock: 20, Status: "active",
	})

	t.Run("sort by name ascending", func(t *testing.T) {
		products, _, err := repo.GetAllProducts(ctx, 20, 0, "", nil, "v.name", "ASC", nil, nil, "", nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(products), 2)
		for i := 1; i < len(products); i++ {
			assert.LessOrEqual(t, products[i-1].Name, products[i].Name)
		}
	})

	t.Run("sort by price descending", func(t *testing.T) {
		products, _, err := repo.GetAllProducts(ctx, 20, 0, "", nil, "v.price", "DESC", nil, nil, "", nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(products), 2)
		for i := 1; i < len(products); i++ {
			assert.GreaterOrEqual(t, products[i-1].Price, products[i].Price)
		}
	})

	t.Run("invalid sort falls back to default", func(t *testing.T) {
		products, _, err := repo.GetAllProducts(ctx, 20, 0, "", nil, "evil_column; DROP TABLE", "ASC", nil, nil, "", nil)
		require.NoError(t, err)
		assert.NotNil(t, products)
	})
}

func TestGetAllProducts_CombinedFilters(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	catID := createTestCategory(t, ctx, uniqueSKU("QF-Combined-Cat"))
	skuA := uniqueSKU("QF-COMB-A")
	skuB := uniqueSKU("QF-COMB-B")

	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuA, Name: "Combo Alpha", CategoryID: &catID,
		Price: 5000, Cost: 2500, Stock: 3, Status: "active",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: skuB, Name: "Combo Beta", CategoryID: &catID,
		Price: 5000, Cost: 2500, Stock: 50, Status: "inactive",
	})
	seedTestProduct(t, repo, ctx, &Product{
		SKU: uniqueSKU("QF-COMB-C"), Name: "Combo Gamma",
		Price: 5000, Cost: 2500, Stock: 3, Status: "active",
	})

	t.Run("category + status + maxStock combined", func(t *testing.T) {
		maxStock := 10
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "", []int{catID}, "", "", &maxStock, nil, "active", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, products, 1)
		assert.Equal(t, skuA, products[0].SKU)
	})

	t.Run("search + category combined", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "Alpha", []int{catID}, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, products, 1)
		assert.Equal(t, skuA, products[0].SKU)
	})

	t.Run("ILIKE partial SKU + status combined", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "QF-COMB", nil, "", "", nil, nil, "active", nil)
		require.NoError(t, err)
		// skuA (active) should match; skuB (inactive) should not
		assert.GreaterOrEqual(t, total, 1)
		for _, p := range products {
			assert.Equal(t, "active", p.Status)
			assert.Contains(t, p.SKU, "QF-COMB")
		}
	})

	t.Run("ILIKE partial SKU + category combined", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 20, 0, "QF-COMB", []int{catID}, "", "", nil, nil, "", nil)
		require.NoError(t, err)
		// skuA and skuB have category catID and contain "QF-COMB"; skuC (no category) should not appear
		assert.GreaterOrEqual(t, total, 2)
		for _, p := range products {
			assert.Contains(t, p.SKU, "QF-COMB")
		}
	})
}

func TestGetAllProducts_EmptyResult(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	products, total, err := repo.GetAllProducts(ctx, 10, 0, "zzz-no-match-zzz", nil, "", "", nil, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Nil(t, products)
}
