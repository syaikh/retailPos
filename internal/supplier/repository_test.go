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

func TestRepository_GetNextSupplierCode(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	code1, err := repo.GetNextSupplierCode(ctx)
	require.NoError(t, err)
	assert.Regexp(t, `^SUP-\d+$`, code1)

	code2, err := repo.GetNextSupplierCode(ctx)
	require.NoError(t, err)
	assert.Regexp(t, `^SUP-\d+$`, code2)
	assert.NotEqual(t, code1, code2, "supplier codes should be sequential and unique")
}

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

func TestSupplierRepository_CRUD(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
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
		assert.GreaterOrEqual(t, total, 1)
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
	repo := newTestRepo(t)
	ctx := context.Background()

	s := &Supplier{
		Name:     "Link Test Supplier",
		Code:     "SUP-LINK-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	var s2 *Supplier
	productID := insertTestProduct(ctx, t, "SUP-PROD-"+time.Now().Format("0102150405"), "Supplier Link Product", 10000)

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
		s2 = &Supplier{
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

	t.Run("Update product supplier", func(t *testing.T) {
		ps, err := repo.GetProductSupplier(ctx, productID, s2.ID)
		require.NoError(t, err)

		ps.UnitCost = 9500
		ps.LeadTimeDays = 10
		err = repo.UpdateProductSupplier(ctx, ps)
		require.NoError(t, err)

		got, err := repo.GetProductSupplier(ctx, productID, s2.ID)
		require.NoError(t, err)
		assert.Equal(t, 9500, got.UnitCost)
		assert.Equal(t, 10, got.LeadTimeDays)
	})

	t.Run("Unlink product", func(t *testing.T) {
		err := repo.UnlinkProduct(ctx, productID, s.ID)
		require.NoError(t, err)

		_, err = repo.GetProductSupplier(ctx, productID, s.ID)
		assert.Error(t, err)
	})
}

func TestSupplierRepository_GetAllInactiveFilter(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
	ctx := context.Background()

	s := &Supplier{
		Name:     "Inactive Filter Supplier",
		Code:     "SUP-INACT-" + time.Now().Format("0102150405"),
		IsActive: false,
	}
	require.NoError(t, repo.Create(ctx, s))

	inactive := false
	suppliers, total, err := repo.GetAll(ctx, 10, 0, "", &inactive)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.NotNil(t, suppliers)

	found := false
	for _, sup := range suppliers {
		if sup.ID == s.ID {
			found = true
			assert.False(t, sup.IsActive)
			break
		}
	}
	assert.True(t, found, "inactive supplier should be in results")
}

func TestSupplierRepository_BulkUpdate(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
	ctx := context.Background()

	s1 := &Supplier{Name: "Bulk Upd 1", Code: "SUP-BU1-" + time.Now().Format("0102150405"), IsActive: true}
	s2 := &Supplier{Name: "Bulk Upd 2", Code: "SUP-BU2-" + time.Now().Format("0102150405"), IsActive: true}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	t.Run("activate multiple", func(t *testing.T) {
		count, err := repo.BulkUpdate(ctx, []int{s1.ID, s2.ID}, true)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("deactivate multiple", func(t *testing.T) {
		count, err := repo.BulkUpdate(ctx, []int{s1.ID, s2.ID}, false)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		got, err := repo.GetByID(ctx, s1.ID)
		require.NoError(t, err)
		assert.False(t, got.IsActive)
	})

	t.Run("empty ids", func(t *testing.T) {
		count, err := repo.BulkUpdate(ctx, []int{}, false)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestSupplierRepository_BulkDelete(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
	ctx := context.Background()

	s1 := &Supplier{Name: "Bulk Del 1", Code: "SUP-BD1-" + time.Now().Format("0102150405"), IsActive: true}
	s2 := &Supplier{Name: "Bulk Del 2", Code: "SUP-BD2-" + time.Now().Format("0102150405"), IsActive: true}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	count, err := repo.BulkDelete(ctx, []int{s1.ID, s2.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = repo.GetByID(ctx, s1.ID)
	assert.Error(t, err)
	_, err = repo.GetByID(ctx, s2.ID)
	assert.Error(t, err)

	t.Run("empty ids", func(t *testing.T) {
		count, err := repo.BulkDelete(ctx, []int{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestSupplierRepository_BulkInsertSuppliers(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
	ctx := context.Background()

	t.Run("insert multiple", func(t *testing.T) {
		payloads := []ImportPayload{
			{
				Code:        "SUP-BI1-" + time.Now().Format("0102150405"),
				Name:        "Bulk Insert 1",
				ContactName: strPtr("Contact 1"),
				Phone:       strPtr("08111111111"),
				Email:       strPtr("bulk1@test.com"),
				Address:     strPtr("Address 1"),
				Notes:       strPtr("Note 1"),
				IsActive:    true,
			},
			{
				Code:     "SUP-BI2-" + time.Now().Format("0102150405"),
				Name:     "Bulk Insert 2",
				IsActive: false,
			},
		}
		count, err := repo.BulkInsertSuppliers(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("empty payloads", func(t *testing.T) {
		count, err := repo.BulkInsertSuppliers(ctx, []ImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestSupplierRepository_BulkUpdateSuppliers(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
	ctx := context.Background()

	code := "SUP-BUS-" + time.Now().Format("0102150405")
	s := &Supplier{Name: "Bulk Update Supplier", Code: code, IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	t.Run("update existing", func(t *testing.T) {
		payloads := []ImportPayload{
			{
				Code:     code,
				Name:     "Bulk Updated Supplier",
				IsActive: false,
			},
		}
		count, err := repo.BulkUpdateSuppliers(ctx, payloads)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		got, err := repo.GetByCode(ctx, code)
		require.NoError(t, err)
		assert.Equal(t, "Bulk Updated Supplier", got.Name)
		assert.False(t, got.IsActive)
	})

	t.Run("empty payloads", func(t *testing.T) {
		count, err := repo.BulkUpdateSuppliers(ctx, []ImportPayload{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestSupplierRepository_GetAllForExport(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := newTestRepo(t)
	ctx := context.Background()

	s := &Supplier{
		Name:     "Export Test Supplier",
		Code:     "SUP-EXPFULL-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	suppliers, err := repo.GetAllForExport(ctx)
	require.NoError(t, err)
	assert.NotNil(t, suppliers)

	found := false
	for _, sup := range suppliers {
		if sup.ID == s.ID {
			found = true
			assert.Equal(t, s.Code, sup.Code)
			break
		}
	}
	assert.True(t, found)
}
