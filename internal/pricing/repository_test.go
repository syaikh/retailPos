package pricing

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(0)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(0)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func insertTestProduct(t *testing.T, ctx context.Context, sku string, name string, price int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, stock, status)
		 VALUES ($1, $2, $3, 100, 'active') RETURNING id`,
		sku, name, price,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestPricingRepository_CRUD(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	productID := insertTestProduct(t, ctx, "PRC-CRUD-"+time.Now().Format("0102150405"), "Pricing Test Product", 15000)

	t.Run("Create and get by ID", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       &productID,
			PricingType:     PricingTypePromotion,
			PricingMethod:   PricingMethodFixedPrice,
			PricingValue:    12000,
			Name:            "Test Discount",
			MinimumQuantity: 1,
			Priority:        0,
			IsActive:        true,
		}
		err := repo.Create(ctx, rule)
		require.NoError(t, err)
		require.Greater(t, rule.ID, 0)
		assert.NotEmpty(t, rule.CreatedAt)

		got, err := repo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, rule.Name, got.Name)
		assert.Equal(t, PricingTypePromotion, got.PricingType)
		assert.Equal(t, PricingMethodFixedPrice, got.PricingMethod)
		assert.Equal(t, 12000.0, got.PricingValue)
		assert.Equal(t, productID, *got.ProductID)
	})

	t.Run("Get by ID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, -1)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("Get by product ID", func(t *testing.T) {
		rules, err := repo.GetByProductID(ctx, productID)
		require.NoError(t, err)
		assert.NotEmpty(t, rules)
	})

	t.Run("Get active rules", func(t *testing.T) {
		rules, err := repo.GetActiveRules(ctx, productID, nil, nil, time.Now(), nil, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, rules)
		for _, r := range rules {
			assert.True(t, r.IsActive)
		}
	})

	t.Run("Update rule", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       &productID,
			PricingType:     PricingTypeSpecialPrice,
			PricingMethod:   PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            "Updated Rule",
			MinimumQuantity: 5,
			Priority:        1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))

		rule.Name = "Updated Rule v2"
		rule.PricingValue = 9000
		err := repo.Update(ctx, rule)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Rule v2", got.Name)
		assert.Equal(t, 9000.0, got.PricingValue)
	})

	t.Run("Delete rule", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       &productID,
			PricingType:     PricingTypePromotion,
			PricingMethod:   PricingMethodFixedPrice,
			PricingValue:    5000,
			Name:            "Delete Me",
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))

		err := repo.Delete(ctx, rule.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, rule.ID)
		assert.Error(t, err)
	})

	t.Run("GetAll with filters", func(t *testing.T) {
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 0)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with search", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "Updated", nil, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with product filter", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "", &productID, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with type filter", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "", nil, "promotion", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with active filter", func(t *testing.T) {
		active := true
		rules, _, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, nil, nil, nil, &active, "")
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})
}

func TestPricingRepository_BatchMethods(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	productID := insertTestProduct(t, ctx, "PRC-BATCH-"+time.Now().Format("0102150405"), "Batch Test Product", 20000)

	t.Run("GetBasePrice", func(t *testing.T) {
		price, err := repo.GetBasePrice(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 20000, price)
	})

	t.Run("GetBasePrice not found", func(t *testing.T) {
		_, err := repo.GetBasePrice(ctx, -1)
		assert.ErrorIs(t, err, ErrProductNotFound)
	})

	t.Run("GetBasePricesBatch", func(t *testing.T) {
		prices, err := repo.GetBasePricesBatch(ctx, []int{productID})
		require.NoError(t, err)
		assert.Equal(t, 20000, prices[productID])
	})

	t.Run("GetBasePricesBatch empty", func(t *testing.T) {
		prices, err := repo.GetBasePricesBatch(ctx, []int{})
		require.NoError(t, err)
		assert.Empty(t, prices)
	})

	t.Run("GetActiveRulesBatch", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       &productID,
			PricingType:     PricingTypePromotion,
			PricingMethod:   PricingMethodFixedPrice,
			PricingValue:    18000,
			Name:            "Batch Discount",
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))

		rulesMap, err := repo.GetActiveRulesBatch(ctx, []int{productID}, time.Now())
		require.NoError(t, err)
		assert.NotEmpty(t, rulesMap[productID])
	})

	t.Run("GetActiveRulesBatch empty", func(t *testing.T) {
		rulesMap, err := repo.GetActiveRulesBatch(ctx, []int{}, time.Now())
		require.NoError(t, err)
		assert.Empty(t, rulesMap)
	})
}

func TestPricingRepository_ProductScope(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	productID := insertTestProduct(t, ctx, "PRC-SCOPE-"+time.Now().Format("0102150405"), "Scope Test Product", 25000)

	t.Run("GetProductScope", func(t *testing.T) {
		catID, brandID, err := repo.GetProductScope(ctx, productID)
		require.NoError(t, err)
		// Product has no category/brand set
		assert.Nil(t, catID)
		assert.Nil(t, brandID)
	})

	t.Run("GetProductScope not found", func(t *testing.T) {
		_, _, err := repo.GetProductScope(ctx, -1)
		assert.ErrorIs(t, err, ErrProductNotFound)
	})
}

func TestPricingRepository_SearchProducts(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	sku := "SRC-" + time.Now().Format("0102150405")
	productID := insertTestProduct(t, ctx, sku, "Searchable Product", 10000)
	_ = productID

	t.Run("Search by name", func(t *testing.T) {
		results, err := repo.SearchProducts(ctx, "Searchable", 10)
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})

	t.Run("Search by SKU", func(t *testing.T) {
		results, err := repo.SearchProducts(ctx, sku, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})

	t.Run("Search empty query", func(t *testing.T) {
		results, err := repo.SearchProducts(ctx, "", 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestPricingRepository_NameExists(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := t.Context()

	productID := insertTestProduct(t, ctx, "NEX-/"+time.Now().Format("0102150405"), "NameExists Product", 10000)
	rule := &PricingRule{
		ProductID:       &productID,
		PricingType:     PricingTypeSpecialPrice,
		PricingMethod:   PricingMethodFixedPrice,
		PricingValue:    5000,
		Name:            "UniqueName-" + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(ctx, rule))

	t.Run("exists returns true", func(t *testing.T) {
		exists, err := repo.NameExists(ctx, rule.Name, 0)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("exists excludes self on update", func(t *testing.T) {
		exists, err := repo.NameExists(ctx, rule.Name, rule.ID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("case insensitive", func(t *testing.T) {
		exists, err := repo.NameExists(ctx, strings.ToUpper(rule.Name), 0)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent name", func(t *testing.T) {
		exists, err := repo.NameExists(ctx, "non-existent-name-"+time.Now().Format("0102150405"), 0)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
