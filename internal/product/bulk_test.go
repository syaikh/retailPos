package product

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestBulkUpdateProductStatus(t *testing.T) {
	repo := testRepo()
	ctx := context.Background()

	t.Run("update multiple products to inactive", func(t *testing.T) {
		p1 := &Product{SKU: uniqueSKU("BULK-ST-A"), Name: "BulkStatusA", Price: 1000, Cost: 500, Stock: 10, Status: "active"}
		p2 := &Product{SKU: uniqueSKU("BULK-ST-B"), Name: "BulkStatusB", Price: 2000, Cost: 1000, Stock: 20, Status: "active"}
		seedTestProduct(ctx, repo, t, p1)
		seedTestProduct(ctx, repo, t, p2)

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
		p := &Product{SKU: uniqueSKU("BULK-ST-C"), Name: "BulkStatusC", Price: 3000, Cost: 1500, Stock: 5, Status: "inactive"}
		seedTestProduct(ctx, repo, t, p)

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
	repo := testRepo()
	ctx := context.Background()

	t.Run("inserts a new product", func(t *testing.T) {
		sku := uniqueSKU("BULK-UPSERT-NEW")
		p := ImportPayload{
			SKU:    sku,
			Name:   "Upsert New Product",
			Price:  15000,
			Cost:   8000,
			Stock:  25,
			Status: "active",
		}

		inserted, err := repo.BulkUpsertProduct(ctx, p)
		require.NoError(t, err)
		assert.True(t, inserted)

		got, err := repo.GetProductBySKU(ctx, sku, nil)
		require.NoError(t, err)
		assert.Equal(t, "Upsert New Product", got.Name)
		assert.Equal(t, 15000, got.Price)
		assert.Equal(t, 8000, got.Cost)
		assert.Equal(t, 25, got.Stock)
	})

	t.Run("insert with all optional fields", func(t *testing.T) {
		sku := uniqueSKU("BULK-UPSERT-FULL")
		barcode := uniqueSKU("BC-FULL")
		desc := "Full upsert product"
		weight := 500

		p := ImportPayload{
			SKU:         sku,
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

		got, err := repo.GetProductBySKU(ctx, sku, nil)
		require.NoError(t, err)
		assert.Equal(t, "Full Upsert", got.Name)
		require.NotNil(t, got.Barcode)
		assert.Equal(t, barcode, *got.Barcode)
		require.NotNil(t, got.Description)
		assert.Equal(t, "Full upsert product", *got.Description)
		require.NotNil(t, got.WeightGrams)
		assert.Equal(t, 500, *got.WeightGrams)
	})
}

func TestBulkUpsertProduct_Update(t *testing.T) {
	repo := testRepo()
	ctx := context.Background()

	t.Run("updates an existing product", func(t *testing.T) {
		sku := uniqueSKU("BULK-UPSERT-EXIST")
		existing := &Product{
			SKU:    sku,
			Name:   "Original Name",
			Price:  10000,
			Cost:   5000,
			Stock:  30,
			Status: "active",
		}
		seedTestProduct(ctx, repo, t, existing)

		updated := ImportPayload{
			SKU:    sku,
			Name:   "Updated Name",
			Price:  25000,
			Cost:   12000,
			Stock:  30,
			Status: "active",
		}

		inserted, err := repo.BulkUpsertProduct(ctx, updated)
		require.NoError(t, err)
		assert.False(t, inserted)

		got, err := repo.GetProductBySKU(ctx, sku, nil)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", got.Name)
		assert.Equal(t, 25000, got.Price)
		assert.Equal(t, 12000, got.Cost)
	})
}

func TestBulkInsertProducts(t *testing.T) {
	repo := testRepo()
	ctx := context.Background()

	t.Run("inserts multiple new products", func(t *testing.T) {
		sku1 := uniqueSKU("BULK-INS-1")
		sku2 := uniqueSKU("BULK-INS-2")
		sku3 := uniqueSKU("BULK-INS-3")
		payloads := []ImportPayload{
			{SKU: sku1, Name: "BulkInsert1", Price: 1000, Cost: 500, Stock: 10, Status: "active"},
			{SKU: sku2, Name: "BulkInsert2", Price: 2000, Cost: 1000, Stock: 20, Status: "active"},
			{SKU: sku3, Name: "BulkInsert3", Price: 3000, Cost: 1500, Stock: 30, Status: "active"},
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
		sku := uniqueSKU("BULK-INS-EXIST")
		existing := &Product{
			SKU: sku, Name: "AlreadyExists", Price: 5000, Cost: 2500, Stock: 5, Status: "active",
		}
		seedTestProduct(ctx, repo, t, existing)

		newSKU := uniqueSKU("BULK-INS-NEW1")
		payloads := []ImportPayload{
			{SKU: sku, Name: "ShouldSkip", Price: 999, Cost: 99, Stock: 1, Status: "active"},
			{SKU: newSKU, Name: "ShouldInsert", Price: 1000, Cost: 500, Stock: 10, Status: "active"},
		}

		count, err := repo.BulkInsertProducts(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		got, err := repo.GetProductBySKU(ctx, sku, nil)
		require.NoError(t, err)
		assert.Equal(t, "AlreadyExists", got.Name)
	})

	t.Run("empty payloads returns 0", func(t *testing.T) {
		count, err := repo.BulkInsertProducts(ctx, []ImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestBulkUpdateProducts(t *testing.T) {
	repo := testRepo()
	ctx := context.Background()

	t.Run("empty payloads returns 0", func(t *testing.T) {
		count, err := repo.BulkUpdateProducts(ctx, []ImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestBuildInsertValues_ChunkBoundary(t *testing.T) {
	payloads := make([]ImportPayload, bulkChunkSize)
	for i := range payloads {
		payloads[i] = ImportPayload{SKU: fmt.Sprintf("CHUNK-I-%d", i), Name: "C", Price: 1000, Cost: 500, Stock: 1, Status: "active"}
	}
	valueStrings, valueArgs := buildInsertValues(payloads)
	assert.Len(t, valueStrings, bulkChunkSize)
	assert.Len(t, valueArgs, bulkChunkSize*12)
	assert.Less(t, len(valueArgs), 65535, "a single chunk must stay below the bind-param limit")
}

func TestBuildUpdateValues_ChunkBoundary(t *testing.T) {
	updates := make([]updateItem, bulkChunkSize)
	for i := range updates {
		updates[i] = updateItem{id: i + 1, payload: ImportPayload{Name: "C", Price: 1000, Cost: 500, Status: "active"}}
	}
	valueStrings, valueArgs := buildUpdateValues(updates)
	assert.Len(t, valueStrings, bulkChunkSize)
	assert.Len(t, valueArgs, bulkChunkSize*11)
	assert.Less(t, len(valueArgs), 65535, "a single chunk must stay below the bind-param limit")
}

func TestBulkInsertUpdate_LargeBatch(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := testRepo()
	ctx := context.Background()

	// Remove the seeded rows so later tests that assume an empty table (e.g.
	// pagination offset-beyond-total) are unaffected. product_stock cascades.
	t.Cleanup(func() {
		_, _ = dbPool.Exec(context.Background(), `DELETE FROM products WHERE sku LIKE 'BULK-LG-IO-%'`)
	})

	// > 5,050 rows forces the insert and update paths across multiple chunk
	// boundaries (5 full chunks of 1000 + a partial chunk).
	const total = 5050
	insertPayloads := make([]ImportPayload, 0, total)
	for i := 0; i < total; i++ {
		sku := fmt.Sprintf("BULK-LG-IO-%05d", i)
		insertPayloads = append(insertPayloads, ImportPayload{
			SKU: sku, Name: "Before", Price: 1000, Cost: 500, Stock: 3, Status: "active",
		})
	}

	inserted, err := repo.BulkInsertProducts(ctx, insertPayloads)
	require.NoError(t, err)
	assert.Equal(t, total, inserted)

	var productCount int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE sku LIKE 'BULK-LG-IO-%'`).Scan(&productCount))
	assert.Equal(t, total, productCount)

	var stockCount int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM product_stock WHERE quantity = 3`).Scan(&stockCount))
	assert.Equal(t, total, stockCount)

	updatePayloads := make([]ImportPayload, 0, total)
	for _, p := range insertPayloads {
		updatePayloads = append(updatePayloads, ImportPayload{
			SKU: p.SKU, Name: "After", Price: 2000, Cost: 1000, Stock: 3, Status: "active",
		})
	}
	updated, err := repo.BulkUpdateProducts(ctx, updatePayloads)
	require.NoError(t, err)
	assert.Equal(t, total, updated)

	var matched int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE name = 'After' AND price = 2000`).Scan(&matched))
	assert.Equal(t, total, matched)
}
