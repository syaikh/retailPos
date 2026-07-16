package supplier

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

func TestSupplierRepository_CRUD(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		s := &Supplier{
			Name:        "Test Supplier",
			Code:        "SUP-CR-" + time.Now().Format("0102150405"),
			ContactName: strPtr("John Doe"),
			Email:       strPtr("john@test.com"),
			Phone:       strPtr("08123456789"),
			Address:     strPtr("Jakarta"),
			IsActive:    true,
		}
		err := repo.Create(ctx, s)
		require.NoError(t, err)
		require.Greater(t, s.ID, 0)
		assert.NotEmpty(t, s.CreatedAt)

		got, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		assert.Equal(t, s.Name, got.Name)
		assert.Equal(t, s.Code, got.Code)
		assert.Equal(t, "John Doe", *got.ContactName)
	})

	t.Run("Get by ID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, -1)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("Get by code", func(t *testing.T) {
		code := "SUP-CODE-" + time.Now().Format("0102150405")
		s := &Supplier{Name: "Code Test", Code: code, IsActive: true}
		require.NoError(t, repo.Create(ctx, s))

		got, err := repo.GetByCode(ctx, code)
		require.NoError(t, err)
		assert.Equal(t, s.Name, got.Name)
	})

	t.Run("Get by code not found", func(t *testing.T) {
		_, err := repo.GetByCode(ctx, "NONEXISTENT-CODE")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("Update supplier", func(t *testing.T) {
		s := &Supplier{
			Name:     "Before Update",
			Code:     "SUP-UPD-" + time.Now().Format("0102150405"),
			IsActive: true,
		}
		require.NoError(t, repo.Create(ctx, s))

		s.Name = "After Update"
		s.Email = strPtr("updated@test.com")
		err := repo.Update(ctx, s)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		assert.Equal(t, "After Update", got.Name)
		assert.Equal(t, "updated@test.com", *got.Email)
	})

	t.Run("Delete supplier (soft delete)", func(t *testing.T) {
		s := &Supplier{
			Name:     "Delete Me",
			Code:     "SUP-DEL-" + time.Now().Format("0102150405"),
			IsActive: true,
		}
		require.NoError(t, repo.Create(ctx, s))

		err := repo.Delete(ctx, s.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, s.ID)
		assert.Error(t, err)
	})

	t.Run("GetAll", func(t *testing.T) {
		suppliers, total, err := repo.GetAll(ctx, 10, 0, "", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 0)
		assert.NotNil(t, suppliers)
	})

	t.Run("GetAll with search", func(t *testing.T) {
		suppliers, _, err := repo.GetAll(ctx, 10, 0, "After", nil)
		require.NoError(t, err)
		assert.NotNil(t, suppliers)
	})

	t.Run("GetAll with active filter", func(t *testing.T) {
		active := true
		suppliers, _, err := repo.GetAll(ctx, 10, 0, "", &active)
		require.NoError(t, err)
		assert.NotNil(t, suppliers)
	})
}

func TestSupplierRepository_ProductSupplierLinking(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	s := &Supplier{
		Name:     "Link Test Supplier",
		Code:     "SUP-LINK-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(t, ctx, "SUP-PROD-"+time.Now().Format("0102150405"), "Supplier Link Product", 10000)

	t.Run("Link product", func(t *testing.T) {
		ps := &ProductSupplier{
			ProductID:    productID,
			SupplierID:   s.ID,
			SupplierSKU:  strPtr("SUP-SKU-001"),
			UnitCost:     8000,
			LeadTimeDays: 7,
			IsPreferred:  true,
		}
		err := repo.LinkProduct(ctx, ps)
		require.NoError(t, err)
		assert.Greater(t, ps.ID, 0)
	})

	t.Run("Get product supplier", func(t *testing.T) {
		ps, err := repo.GetProductSupplier(ctx, productID, s.ID)
		require.NoError(t, err)
		assert.Equal(t, 8000, ps.UnitCost)
		assert.True(t, ps.IsPreferred)
	})

	t.Run("Get preferred supplier", func(t *testing.T) {
		ps, err := repo.GetPreferredSupplier(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, s.ID, ps.SupplierID)
	})

	t.Run("Has preferred supplier", func(t *testing.T) {
		has, err := repo.HasPreferredSupplier(ctx, productID)
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("Set preferred supplier", func(t *testing.T) {
		s2 := &Supplier{
			Name:     "Preferred Test 2",
			Code:     "SUP-PREF-" + time.Now().Format("0102150405"),
			IsActive: true,
		}
		require.NoError(t, repo.Create(ctx, s2))

		ps2 := &ProductSupplier{
			ProductID:   productID,
			SupplierID:  s2.ID,
			UnitCost:    9000,
			IsPreferred: false,
		}
		require.NoError(t, repo.LinkProduct(ctx, ps2))

		err := repo.SetPreferredSupplier(ctx, productID, s2.ID)
		require.NoError(t, err)

		got, err := repo.GetPreferredSupplier(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, s2.ID, got.SupplierID)
	})

	t.Run("Get suppliers by product ID", func(t *testing.T) {
		suppliers, err := repo.GetSuppliersByProductID(ctx, productID)
		require.NoError(t, err)
		assert.NotEmpty(t, suppliers)
		assert.NotNil(t, suppliers[0].SupplierName)
	})

	t.Run("Get products by supplier ID", func(t *testing.T) {
		products, err := repo.GetProductsBySupplierID(ctx, s.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, products)
		assert.NotNil(t, products[0].ProductName)
	})

	t.Run("Unlink product", func(t *testing.T) {
		err := repo.UnlinkProduct(ctx, productID, s.ID)
		require.NoError(t, err)

		_, err = repo.GetProductSupplier(ctx, productID, s.ID)
		assert.Error(t, err)
	})
}
