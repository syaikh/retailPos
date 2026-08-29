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

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func insertTestProduct(ctx context.Context, t *testing.T, sku string, name string, price int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, status)
		 VALUES ($1, $2, $3, 'active') RETURNING id`,
		sku, name, price,
	).Scan(&id)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, 100)`, id)
	require.NoError(t, err)
	return id
}

// newWiredRepo returns a pricing Repository with the product/category/brand
// owner providers wired, mirroring production wiring (internal/wiring).
func newWiredRepo() *Repository {
	repo := NewRepository(dbPool)
	repo.SetProductPricingProvider(product.PricingLookup{})
	repo.SetCategorySearchProvider(category.NamesProvider{})
	repo.SetBrandSearchProvider(brand.NamesProvider{})
	return repo
}

func TestPricingRepository_CRUD(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	productID := insertTestProduct(ctx, t, "PRC-CRUD-"+time.Now().Format("0102150405"), "Pricing Test Product", 15000)

	t.Run("Create and get by ID", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
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
		assert.Equal(t, PricingTypePromotion, got.Type)
		assert.Equal(t, PricingMethodFixedPrice, got.Method)
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
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypeSpecialPrice,
			Method:          PricingMethodFixedPrice,
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
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
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
		assert.GreaterOrEqual(t, total, 1)
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
	repo := newWiredRepo()
	ctx := context.Background()

	productID := insertTestProduct(ctx, t, "PRC-BATCH-"+time.Now().Format("0102150405"), "Batch Test Product", 20000)

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
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
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

func TestPricingRepository_GetAll_Filters(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	productID := insertTestProduct(ctx, t, "PRC-GAF-"+time.Now().Format("0102150405"), "GetAll Filter Product", 15000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            "GetAll Filter Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusApproved,
	}
	require.NoError(t, repo.Create(ctx, rule))

	t.Run("filter by pricing_method", func(t *testing.T) {
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "fixed_price", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, r := range rules {
			assert.Equal(t, PricingMethodFixedPrice, r.Method)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, nil, nil, nil, nil, "approved")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, r := range rules {
			assert.Equal(t, StatusApproved, r.Status)
		}
	})

	t.Run("filter by category_id", func(t *testing.T) {
		catID := 1
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", &catID, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(rules))
	})

	t.Run("filter by brand_id", func(t *testing.T) {
		brandID := 1
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, &brandID, nil, nil, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(rules))
	})

	t.Run("filter by customer_group_id", func(t *testing.T) {
		cgID := 1
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, nil, &cgID, nil, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(rules))
	})

	t.Run("filter by store_id", func(t *testing.T) {
		sid := 1
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, nil, nil, &sid, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(rules))
	})

	t.Run("filter by is_active false", func(t *testing.T) {
		inactive := false
		rules, total, err := repo.GetAll(ctx, 10, 0, "", nil, "", "", nil, nil, nil, nil, &inactive, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(rules))
	})
}

func TestPricingRepository_GetAll_SearchByOwnerNames(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()
	ts := time.Now().Format("0102150405.000000")

	var catID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO categories (name, is_active) VALUES ($1, true) RETURNING id`,
		"ZZZ Category Alpha "+ts).Scan(&catID))
	var brandID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO brands (name, is_active) VALUES ($1, true) RETURNING id`,
		"ZZZ Brand Beta "+ts).Scan(&brandID))
	var productID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, status, category_id, brand_id)
		 VALUES ($1, $2, $3, 'active', $4, $5) RETURNING id`,
		"PRC-SRCH-"+ts, "ZZZ Product Gamma "+ts, 15000, catID, brandID).Scan(&productID))
	_, err := dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, 100)`, productID)
	require.NoError(t, err)

	newRule := func(scopeName string, productID, categoryID, brandID *int) *Rule {
		rule := &Rule{
			ProductID:       productID,
			CategoryID:      categoryID,
			BrandID:         brandID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    12000,
			Name:            "Rule " + scopeName + " Scope " + ts,
			MinimumQuantity: 1,
			Priority:        0,
			IsActive:        true,
			Status:          StatusApproved,
		}
		require.NoError(t, repo.Create(ctx, rule))
		return rule
	}
	productRule := newRule("Product", &productID, nil, nil)
	categoryRule := newRule("Category", nil, &catID, nil)
	brandRule := newRule("Brand", nil, nil, &brandID)

	containsID := func(rules []Rule, id int) bool {
		for _, r := range rules {
			if r.ID == id {
				return true
			}
		}
		return false
	}

	t.Run("match by product name", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "ZZZ Product Gamma "+ts, nil, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.True(t, containsID(rules, productRule.ID), "expected product-scoped rule in results")
		assert.False(t, containsID(rules, categoryRule.ID))
		assert.False(t, containsID(rules, brandRule.ID))
	})

	t.Run("match by category name", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "ZZZ Category Alpha "+ts, nil, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.True(t, containsID(rules, categoryRule.ID), "expected category-scoped rule in results")
		assert.False(t, containsID(rules, productRule.ID))
		assert.False(t, containsID(rules, brandRule.ID))
	})

	t.Run("match by brand name", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "ZZZ Brand Beta "+ts, nil, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.True(t, containsID(rules, brandRule.ID), "expected brand-scoped rule in results")
		assert.False(t, containsID(rules, productRule.ID))
		assert.False(t, containsID(rules, categoryRule.ID))
	})

	t.Run("no match", func(t *testing.T) {
		rules, _, err := repo.GetAll(ctx, 10, 0, "ZZZ NoSuchEntity "+ts, nil, "", "", nil, nil, nil, nil, nil, "")
		require.NoError(t, err)
		assert.False(t, containsID(rules, productRule.ID))
		assert.False(t, containsID(rules, categoryRule.ID))
		assert.False(t, containsID(rules, brandRule.ID))
	})
}

func TestPricingRepository_GetByID_Fields(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	productID := insertTestProduct(ctx, t, "PRC-FID-"+time.Now().Format("0102150405"), "GetByID Fields Product", 15000)
	now := time.Now().In(shared.JakartaLocation())
	from := now.Add(24 * time.Hour)
	until := now.Add(7 * 24 * time.Hour)
	fromStr := from.Format("15:04:05")
	untilStr := until.Format("15:04:05")

	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypeSpecialPrice,
		Method:          PricingMethodDiscountPct,
		PricingValue:    15.0,
		Name:            "Full Fields Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 2,
		Priority:        5,
		IsActive:        true,
		Status:          StatusApproved,
		EffectiveFrom:   &from,
		EffectiveUntil:  &until,
		RecurrenceDays:  []string{"monday", "tuesday"},
		TimeFrom:        &fromStr,
		TimeTo:          &untilStr,
	}
	require.NoError(t, repo.Create(ctx, rule))
	require.Greater(t, rule.ID, 0)

	got, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, rule.ID, got.ID)
	assert.Equal(t, PricingTypeSpecialPrice, got.Type)
	assert.Equal(t, PricingMethodDiscountPct, got.Method)
	assert.Equal(t, 15.0, got.PricingValue)
	assert.Equal(t, 2, got.MinimumQuantity)
	assert.Equal(t, 5, got.Priority)
	assert.True(t, got.IsActive)
	assert.NotNil(t, got.EffectiveFrom)
	assert.NotNil(t, got.EffectiveUntil)
	assert.Equal(t, []string{"monday", "tuesday"}, got.RecurrenceDays)
	assert.NotNil(t, got.TimeFrom)
	assert.NotNil(t, got.TimeTo)
}

func TestPricingRepository_GetByProductID_Empty(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	rules, err := repo.GetByProductID(ctx, -99999)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestPricingRepository_GetBasePricesBatch_Multiple(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	id1 := insertTestProduct(ctx, t, "BATCH1-"+time.Now().Format("0102150405"), "Batch Product 1", 10000)
	id2 := insertTestProduct(ctx, t, "BATCH2-"+time.Now().Format("0102150405"), "Batch Product 2", 20000)

	prices, err := repo.GetBasePricesBatch(ctx, []int{id1, id2})
	require.NoError(t, err)
	assert.Equal(t, 10000, prices[id1])
	assert.Equal(t, 20000, prices[id2])
}

func TestPricingRepository_ProductScope(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	productID := insertTestProduct(ctx, t, "PRC-SCOPE-"+time.Now().Format("0102150405"), "Scope Test Product", 25000)

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
	repo := newWiredRepo()
	ctx := context.Background()

	sku := "SRC-" + time.Now().Format("0102150405")
	productID := insertTestProduct(ctx, t, sku, "Searchable Product", 10000)
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
	repo := newWiredRepo()
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "NEX-/"+time.Now().Format("0102150405"), "NameExists Product", 10000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypeSpecialPrice,
		Method:          PricingMethodFixedPrice,
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

func TestPricingRepository_BulkInsertPricingRules(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "BIPR-"+time.Now().Format("0102150405"), "BulkInsert Product", 15000)

	t.Run("bulk insert", func(t *testing.T) {
		payloads := []RuleImportPayload{
			{
				ProductID:       &productID,
				Type:            string(PricingTypePromotion),
				Method:          string(PricingMethodFixedPrice),
				PricingValue:    10000,
				Name:            "Bulk Insert 1 " + time.Now().Format("0102150405.000"),
				MinimumQuantity: 1,
				IsActive:        true,
			},
			{
				ProductID:       &productID,
				Type:            string(PricingTypeSpecialPrice),
				Method:          string(PricingMethodDiscountAmt),
				PricingValue:    2000,
				Name:            "Bulk Insert 2 " + time.Now().Format("0102150405.000"),
				MinimumQuantity: 5,
				IsActive:        true,
			},
		}
		count, err := repo.BulkInsertPricingRules(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("empty payloads", func(t *testing.T) {
		count, err := repo.BulkInsertPricingRules(ctx, []RuleImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestPricingRepository_BulkUpdatePricingRules(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "BUPR-"+time.Now().Format("0102150405"), "BulkUpdate Product", 20000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    15000,
		Name:            "BulkUpdate Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(ctx, rule))

	t.Run("bulk update", func(t *testing.T) {
		payloads := []RuleImportPayload{
			{
				ProductID:       &productID,
				Type:            string(PricingTypePromotion),
				Method:          string(PricingMethodFixedPrice),
				PricingValue:    12000,
				Name:            rule.Name,
				MinimumQuantity: 1,
				IsActive:        true,
			},
		}
		count, err := repo.BulkUpdatePricingRules(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		got, err := repo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, 12000.0, got.PricingValue)
	})

	t.Run("empty payloads", func(t *testing.T) {
		count, err := repo.BulkUpdatePricingRules(ctx, []RuleImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("no matching rule", func(t *testing.T) {
		nonExistent := -99999
		payloads := []RuleImportPayload{
			{
				ProductID:       &nonExistent,
				Type:            string(PricingTypePromotion),
				Method:          string(PricingMethodFixedPrice),
				PricingValue:    5000,
				Name:            "Non-existent",
				MinimumQuantity: 1,
				IsActive:        true,
			},
		}
		count, err := repo.BulkUpdatePricingRules(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestPricingRepository_GetAllForExport(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "EXP-"+time.Now().Format("0102150405"), "Export Product", 15000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            "Export Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(ctx, rule))

	rules, err := repo.GetAllForExport(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rules), 1)
}

func TestPricingRepository_GetProductCostAndTax(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	productID := insertTestProduct(ctx, t, "PRC-COST-"+time.Now().Format("0102150405"), "Cost Tax Product", 15000)

	t.Run("success", func(t *testing.T) {
		ct, err := repo.GetProductCostAndTax(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 0, ct.Cost)
		assert.Equal(t, "Cost Tax Product", ct.ProductName)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetProductCostAndTax(ctx, -1)
		assert.ErrorIs(t, err, ErrProductNotFound)
	})
}

func TestPricingRepository_GetProductCostAndTaxBatch(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newWiredRepo()
	ctx := context.Background()

	id1 := insertTestProduct(ctx, t, "PRC-CB1-"+time.Now().Format("0102150405"), "CB One", 10000)
	id2 := insertTestProduct(ctx, t, "PRC-CB2-"+time.Now().Format("0102150405"), "CB Two", 20000)

	t.Run("multiple", func(t *testing.T) {
		res, err := repo.GetProductCostAndTaxBatch(ctx, []int{id1, id2})
		require.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, "CB One", res[id1].ProductName)
		assert.Equal(t, "CB Two", res[id2].ProductName)
	})

	t.Run("empty input", func(t *testing.T) {
		res, err := repo.GetProductCostAndTaxBatch(ctx, []int{})
		require.NoError(t, err)
		assert.Empty(t, res)
	})
}
