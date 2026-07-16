package pricing

import (
	"context"
	"os"
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
			ProductID:       productID,
			PricingType:     PricingTypeDiscount,
			Name:            "Test Discount",
			Price:           12000,
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
		assert.Equal(t, PricingTypeDiscount, got.PricingType)
		assert.Equal(t, 12000, got.Price)
		assert.Equal(t, productID, got.ProductID)
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
		rules, err := repo.GetActiveRules(ctx, productID, time.Now())
		require.NoError(t, err)
		assert.NotEmpty(t, rules)
		for _, r := range rules {
			assert.True(t, r.IsActive)
		}
	})

	t.Run("Update rule", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       productID,
			PricingType:     PricingTypeWholesale,
			Name:            "Updated Rule",
			Price:           10000,
			MinimumQuantity: 5,
			Priority:        1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))

		rule.Name = "Updated Rule v2"
		rule.Price = 9000
		err := repo.Update(ctx, rule)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Rule v2", got.Name)
		assert.Equal(t, 9000, got.Price)
	})

	t.Run("Delete rule", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       productID,
			PricingType:     PricingTypePromotion,
			Name:            "Delete Me",
			Price:           5000,
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
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 0)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with search", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "Updated", nil, "", nil)
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with product filter", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "", &productID, "", nil)
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with type filter", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "", nil, "discount", nil)
		require.NoError(t, err)
		assert.NotNil(t, rules)
	})

	t.Run("GetAll with active filter", func(t *testing.T) {
		active := true
		rules, _, err := repo.GetAll(ctx, 10, 0, "", nil, "", &active)
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
		// Create a rule first
		rule := &PricingRule{
			ProductID:       productID,
			PricingType:     PricingTypeDiscount,
			Name:            "Batch Discount",
			Price:           18000,
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
