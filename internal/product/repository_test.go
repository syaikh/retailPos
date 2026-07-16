package product

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func uniqueSKU(base string) string {
	return base + "-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%09d", time.Now().Nanosecond())
}

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

func TestProductRepository_CRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		p := &Product{
			SKU:    uniqueSKU("TEST-SKU-001"),
			Name:   "Test Product",
			Price:  15000,
			Cost:   10000,
			Stock:  50,
			Status: "active",
		}
		err := repo.CreateProduct(ctx, p)
		require.NoError(t, err)
		require.Greater(t, p.ID, 0)

		got, err := repo.GetProductByID(ctx, p.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, p.SKU, got.SKU)
		assert.Equal(t, p.Name, got.Name)
		assert.Equal(t, p.Price, got.Price)
		assert.Equal(t, p.Stock, got.Stock)
		assert.Equal(t, "active", got.Status)
	})

	t.Run("Get by SKU", func(t *testing.T) {
		p := &Product{
			SKU:    uniqueSKU("TEST-SKU-UNIQUE"),
			Name:   "SKU Lookup Test",
			Price:  20000,
			Cost:   15000,
			Stock:  10,
			Status: "active",
		}
		err := repo.CreateProduct(ctx, p)
		require.NoError(t, err)

		got, err := repo.GetProductBySKU(ctx, p.SKU, nil)
		require.NoError(t, err)
		assert.Equal(t, p.Name, got.Name)
	})

	t.Run("Get product not found", func(t *testing.T) {
		_, err := repo.GetProductByID(ctx, -1, nil)
		assert.ErrorContains(t, err, "product not found")
	})

	t.Run("Update product", func(t *testing.T) {
		p := &Product{
			SKU:    uniqueSKU("TEST-SKU-UPDATE"),
			Name:   "Before Update",
			Price:  10000,
			Cost:   5000,
			Stock:  20,
			Status: "active",
		}
		err := repo.CreateProduct(ctx, p)
		require.NoError(t, err)

		p.Name = "After Update"
		p.Price = 25000

		err = repo.UpdateProduct(ctx, p, nil)
		require.NoError(t, err)

		got, err := repo.GetProductByID(ctx, p.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "After Update", got.Name)
		assert.Equal(t, 25000, got.Price)
	})

	t.Run("Delete product (soft)", func(t *testing.T) {
		p := &Product{
			SKU:    uniqueSKU("TEST-SKU-DELETE"),
			Name:   "To Be Deleted",
			Price:  5000,
			Cost:   2500,
			Stock:  5,
			Status: "active",
		}
		err := repo.CreateProduct(ctx, p)
		require.NoError(t, err)

		err = repo.DeleteProduct(ctx, p.ID, nil)
		require.NoError(t, err)

		got, err := repo.GetProductByID(ctx, p.ID, nil)
		assert.ErrorContains(t, err, "product not found")
		assert.Nil(t, got)
	})

	t.Run("Bulk update status", func(t *testing.T) {
		p1 := &Product{SKU: uniqueSKU("TEST-BULK-1"), Name: "Bulk 1", Price: 1000, Cost: 500, Stock: 1, Status: "active"}
		p2 := &Product{SKU: uniqueSKU("TEST-BULK-2"), Name: "Bulk 2", Price: 2000, Cost: 1000, Stock: 2, Status: "active"}
		require.NoError(t, repo.CreateProduct(ctx, p1))
		require.NoError(t, repo.CreateProduct(ctx, p2))

		err := repo.BulkUpdateProductStatus(ctx, []int{p1.ID, p2.ID}, "inactive", nil)
		require.NoError(t, err)

		got1, _ := repo.GetProductByID(ctx, p1.ID, nil)
		got2, _ := repo.GetProductByID(ctx, p2.ID, nil)
		assert.Equal(t, "inactive", got1.Status)
		assert.Equal(t, "inactive", got2.Status)
	})
}

func TestProductRepository_NextSKU(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	sku, err := repo.GetNextSKU(ctx)
	require.NoError(t, err)
	assert.Contains(t, sku, "SKU-")
}

func TestProductRepository_TaxClassRead(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Get tax class not found", func(t *testing.T) {
		_, err := repo.GetTaxClassByID(ctx, -1)
		assert.ErrorContains(t, err, "tax class not found")
	})

	t.Run("List all tax classes", func(t *testing.T) {
		_, err := dbPool.Exec(ctx, `INSERT INTO tax_classes (name, rate_percent, is_active) VALUES ('TestTaxList', 11, true)`)
		require.NoError(t, err)
		list, err := repo.GetAllTaxClasses(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}

func TestProductRepository_WarehouseCRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Get all warehouses", func(t *testing.T) {
		_, err := dbPool.Exec(ctx, `INSERT INTO warehouses (name, code, is_active) VALUES ('Test WH', 'TWH01', true)`)
		require.NoError(t, err)
		warehouses, err := repo.GetAllWarehouses(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, warehouses)
	})
}

func TestProductRepository_DeletedProductRestore(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Get deleted by barcode", func(t *testing.T) {
		// First create a product with a unique barcode
		barcode := "DEL-BARCODE-001"
		p := &Product{
			SKU:     uniqueSKU("TEST-RESTORE-SKU"),
			Name:    "Restore Test",
			Price:   10000,
			Cost:    5000,
			Stock:   10,
			Status:  "active",
			Barcode: &barcode,
		}
		err := repo.CreateProduct(ctx, p)
		require.NoError(t, err)

		// Soft delete it
		err = repo.DeleteProduct(ctx, p.ID, nil)
		require.NoError(t, err)

		// Look up by barcode (should be in deleted state)
		got, err := repo.GetDeletedProductByBarcode(ctx, barcode, nil)
		require.NoError(t, err)
		assert.Equal(t, p.Name, got.Name)
	})

	t.Run("Restore product", func(t *testing.T) {
		p := &Product{
			SKU:    uniqueSKU("TEST-RESTORE-2"),
			Name:   "To Restore",
			Price:  5000,
			Cost:   2500,
			Stock:  3,
			Status: "archived",
		}
		require.NoError(t, repo.CreateProduct(ctx, p))
		require.NoError(t, repo.DeleteProduct(ctx, p.ID, nil))

		p.Status = "active"
		err := repo.RestoreProduct(ctx, p)
		require.NoError(t, err)

		got, err := repo.GetProductByID(ctx, p.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "active", got.Status)
	})
}

func TestProductRepository_GetAllProductsPagination(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		p := &Product{
			SKU:    uniqueSKU("TEST-PAGING-" + string(rune('A'+i))),
			Name:   "Paging Test " + string(rune('A'+i)),
			Price:  1000 * (i + 1),
			Cost:   500 * (i + 1),
			Stock:  10 * (i + 1),
			Status: "active",
		}
		require.NoError(t, repo.CreateProduct(ctx, p))
	}

	t.Run("limit and offset", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 2, 0, "", nil, "price", "ASC", nil, nil, "")
		require.NoError(t, err)
		assert.LessOrEqual(t, len(products), 2)
		assert.Greater(t, total, 0)
	})

	t.Run("search by name", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(ctx, 10, 0, "Paging", nil, "", "", nil, nil, "")
		require.NoError(t, err)
		assert.Greater(t, total, 0)
		assert.Greater(t, len(products), 0)
	})
}
