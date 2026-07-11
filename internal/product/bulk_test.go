package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkUpdateProductStatus(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("update multiple products to inactive", func(t *testing.T) {
		p1 := &Product{SKU: "BULK-ST-A", Name: "BulkStatusA", Price: 1000, Cost: 500, Stock: 10, Status: "active"}
		p2 := &Product{SKU: "BULK-ST-B", Name: "BulkStatusB", Price: 2000, Cost: 1000, Stock: 20, Status: "active"}
		seedTestProduct(t, repo, ctx, p1)
		seedTestProduct(t, repo, ctx, p2)

		err := repo.BulkUpdateProductStatus(ctx, []int{p1.ID, p2.ID}, "inactive", nil)
		require.NoError(t, err)

		got1, err := repo.GetProductByID(ctx, p1.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "inactive", got1.Status)

		got2, err := repo.GetProductByID(ctx, p2.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "inactive", got2.Status)
	})

	t.Run("empty ids list is no-op", func(t *testing.T) {
		err := repo.BulkUpdateProductStatus(ctx, []int{}, "inactive", nil)
		require.NoError(t, err)
	})

	t.Run("update to active from inactive", func(t *testing.T) {
		p := &Product{SKU: "BULK-ST-C", Name: "BulkStatusC", Price: 3000, Cost: 1500, Stock: 5, Status: "inactive"}
		seedTestProduct(t, repo, ctx, p)

		err := repo.BulkUpdateProductStatus(ctx, []int{p.ID}, "active", nil)
		require.NoError(t, err)

		got, err := repo.GetProductByID(ctx, p.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "active", got.Status)
	})

	t.Run("nonexistent ids succeed silently", func(t *testing.T) {
		err := repo.BulkUpdateProductStatus(ctx, []int{-999, -998}, "archived", nil)
		require.NoError(t, err)
	})
}

func TestBulkUpsertProduct_Insert(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("inserts a new product", func(t *testing.T) {
		p := ProductImportPayload{
			SKU:    "BULK-UPSERT-NEW",
			Name:   "Upsert New Product",
			Price:  15000,
			Cost:   8000,
			Stock:  25,
			Status: "active",
		}

		inserted, err := repo.BulkUpsertProduct(ctx, p)
		require.NoError(t, err)
		assert.True(t, inserted)

		got, err := repo.GetProductBySKU(ctx, "BULK-UPSERT-NEW", nil)
		require.NoError(t, err)
		assert.Equal(t, "Upsert New Product", got.Name)
		assert.Equal(t, 15000, got.Price)
		assert.Equal(t, 8000, got.Cost)
		assert.Equal(t, 25, got.Stock)
	})

	t.Run("insert with all optional fields", func(t *testing.T) {
		barcode := "BULK-UPSERT-BC"
		desc := "Full upsert product"
		weight := 500

		p := ProductImportPayload{
			SKU:         "BULK-UPSERT-FULL",
			Name:        "Full Upsert",
			Barcode:     &barcode,
			Price:       20000,
			Cost:        10000,
			Stock:       15,
			Status:      "active",
			Description: &desc,
			WeightGrams: &weight,
		}

		inserted, err := repo.BulkUpsertProduct(ctx, p)
		require.NoError(t, err)
		assert.True(t, inserted)

		got, err := repo.GetProductBySKU(ctx, "BULK-UPSERT-FULL", nil)
		require.NoError(t, err)
		assert.Equal(t, "Full Upsert", got.Name)
		require.NotNil(t, got.Barcode)
		assert.Equal(t, "BULK-UPSERT-BC", *got.Barcode)
		require.NotNil(t, got.Description)
		assert.Equal(t, "Full upsert product", *got.Description)
		require.NotNil(t, got.WeightGrams)
		assert.Equal(t, 500, *got.WeightGrams)
	})
}

func TestBulkUpsertProduct_Update(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("updates an existing product", func(t *testing.T) {
		// First insert via CreateProduct
		existing := &Product{
			SKU:    "BULK-UPSERT-EXIST",
			Name:   "Original Name",
			Price:  10000,
			Cost:   5000,
			Stock:  30,
			Status: "active",
		}
		seedTestProduct(t, repo, ctx, existing)

		// Now upsert with same SKU
		updated := ProductImportPayload{
			SKU:    "BULK-UPSERT-EXIST",
			Name:   "Updated Name",
			Price:  25000,
			Cost:   12000,
			Stock:  30,
			Status: "active",
		}

		inserted, err := repo.BulkUpsertProduct(ctx, updated)
		require.NoError(t, err)
		assert.False(t, inserted)

		got, err := repo.GetProductBySKU(ctx, "BULK-UPSERT-EXIST", nil)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", got.Name)
		assert.Equal(t, 25000, got.Price)
		assert.Equal(t, 12000, got.Cost)
	})
}

func TestBulkInsertProducts(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("inserts multiple new products", func(t *testing.T) {
		payloads := []ProductImportPayload{
			{SKU: "BULK-INS-1", Name: "BulkInsert1", Price: 1000, Cost: 500, Stock: 10, Status: "active"},
			{SKU: "BULK-INS-2", Name: "BulkInsert2", Price: 2000, Cost: 1000, Stock: 20, Status: "active"},
			{SKU: "BULK-INS-3", Name: "BulkInsert3", Price: 3000, Cost: 1500, Stock: 30, Status: "active"},
		}

		count, err := repo.BulkInsertProducts(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		for _, p := range payloads {
			got, err := repo.GetProductBySKU(ctx, p.SKU, nil)
			require.NoError(t, err)
			assert.Equal(t, p.Name, got.Name)
			assert.Equal(t, p.Price, got.Price)
		}
	})

	t.Run("skips already existing products", func(t *testing.T) {
		// Create one existing
		existing := &Product{
			SKU: "BULK-INS-EXIST", Name: "AlreadyExists", Price: 5000, Cost: 2500, Stock: 5, Status: "active",
		}
		seedTestProduct(t, repo, ctx, existing)

		payloads := []ProductImportPayload{
			{SKU: "BULK-INS-EXIST", Name: "ShouldSkip", Price: 999, Cost: 99, Stock: 1, Status: "active"},
			{SKU: "BULK-INS-NEW1", Name: "ShouldInsert", Price: 1000, Cost: 500, Stock: 10, Status: "active"},
		}

		count, err := repo.BulkInsertProducts(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Original should be unchanged
		got, err := repo.GetProductBySKU(ctx, "BULK-INS-EXIST", nil)
		require.NoError(t, err)
		assert.Equal(t, "AlreadyExists", got.Name)
	})

	t.Run("empty payloads returns 0", func(t *testing.T) {
		count, err := repo.BulkInsertProducts(ctx, []ProductImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestBulkUpdateProducts(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("empty payloads returns 0", func(t *testing.T) {
		count, err := repo.BulkUpdateProducts(ctx, []ProductImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
