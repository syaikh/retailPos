package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository_GetByUsername(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting existing user
	user, err := repo.GetByUsername(context.Background(), "superadmin")
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "superadmin", user.Username)
	assert.Equal(t, "superadmin@retailpos.local", user.Email)
	assert.True(t, user.IsActive)
	assert.NotEmpty(t, user.Password)
}

func TestPostgresRepository_GetByUsername_NotFound(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting non-existent user
	user, err := repo.GetByUsername(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestPostgresRepository_GetAllUsers(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting all users
	users, total, err := repo.GetAllUsers(context.Background(), 10, 0, "")
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	assert.Len(t, users, total)
	assert.Equal(t, "superadmin", users[0].Username)
}

func TestPostgresRepository_GetAllUsers_WithSearch(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test search functionality
	users, total, err := repo.GetAllUsers(context.Background(), 10, 0, "admin")
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	assert.Len(t, users, total)

	// Should find superadmin and admin
	foundSuperAdmin := false
	foundAdmin := false
	for _, user := range users {
		if user.Username == "superadmin" {
			foundSuperAdmin = true
		}
		if user.Username == "admin" {
			foundAdmin = true
		}
	}
	assert.True(t, foundSuperAdmin || foundAdmin)
}

func TestPostgresRepository_GetRolePermissions(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting permissions for superadmin role (ID: 1)
	permissions, err := repo.GetRolePermissions(context.Background(), 1)
	require.NoError(t, err)
	assert.Greater(t, len(permissions), 0)

	// Superadmin should have all permissions
	codes := make([]string, len(permissions))
	for i, p := range permissions {
		codes[i] = p.Code
	}
	assert.Contains(t, codes, "product:create")
	assert.Contains(t, codes, "user:manage")
	assert.Contains(t, codes, "sale:create")
}

func TestPostgresRepository_GetRolePermissions_Cashier(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Debug: Check what roles exist
	var roleCount int
	err := testDB.Pool().QueryRow(context.Background(), "SELECT COUNT(*) FROM roles").Scan(&roleCount)
	require.NoError(t, err)
	t.Logf("Total roles in DB: %d", roleCount)

	// Debug: Check what permissions exist
	var permCount int
	err = testDB.Pool().QueryRow(context.Background(), "SELECT COUNT(*) FROM permissions").Scan(&permCount)
	require.NoError(t, err)
	t.Logf("Total permissions in DB: %d", permCount)

	// Test getting permissions for cashier role (ID: 4)
	permissions, err := repo.GetRolePermissions(context.Background(), 4)
	require.NoError(t, err)

	// Cashier should have multiple permissions according to seed
	codes := make([]string, len(permissions))
	for i, p := range permissions {
		codes[i] = p.Code
	}
	t.Logf("Cashier permissions: %v", codes)
	assert.Contains(t, codes, "sale:create")
	assert.Contains(t, codes, "product:read")
	assert.Contains(t, codes, "sale:read")
	assert.Contains(t, codes, "pos:access")
}

func TestPostgresRepository_GetAllRoles(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	// Call method directly on concrete type
	repo := NewPostgresRepository(testDB.Pool())

	// Debug: Check raw SQL
	var count int
	err := testDB.Pool().QueryRow(context.Background(), "SELECT COUNT(*) FROM roles").Scan(&count)
	require.NoError(t, err)
	t.Logf("Raw SQL count: %d", count)

	// Test getting all roles
	roles, err := repo.GetAllRoles(context.Background())
	require.NoError(t, err)
	t.Logf("GetAllRoles returned %d roles", len(roles))
	for i, role := range roles {
		t.Logf("Role %d: ID=%d, Name=%s", i, role.ID, role.Name)
	}

	assert.Greater(t, len(roles), 0)

	// Should have the default roles
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}
	assert.Contains(t, roleNames, "superadmin")
	assert.Contains(t, roleNames, "admin")
	assert.Contains(t, roleNames, "manager")
	assert.Contains(t, roleNames, "cashier")
}

func TestPostgresRepository_GetAllPermissions(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	// Call method directly on concrete type
	repo := NewPostgresRepository(testDB.Pool())

	// Test getting all permissions
	permissions, err := repo.GetAllPermissions(context.Background())
	require.NoError(t, err)
	assert.Greater(t, len(permissions), 0)

	// Should have various permissions
	codes := make([]string, len(permissions))
	for i, p := range permissions {
		codes[i] = p.Code
	}
	assert.Contains(t, codes, "product:create")
	assert.Contains(t, codes, "user:manage")
	assert.Contains(t, codes, "sale:create")
	assert.Contains(t, codes, "report:view")
	assert.Contains(t, codes, "inventory:adjust")
}

func TestPostgresRepository_GetAllProducts(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting all products
	products, total, err := repo.GetAllProducts(context.Background(), 10, 0, "", []int{}, "name", "asc", nil, nil, "")
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	assert.Len(t, products, total)

	// Verify product structure
	for _, product := range products {
		assert.NotEmpty(t, product.SKU)
		assert.NotEmpty(t, product.Name)
		assert.Greater(t, product.Price, 0)
		assert.Greater(t, product.Stock, 0)
	}
}

func TestPostgresRepository_GetProductBySKU(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting product by SKU
	product, err := repo.GetProductBySKU(context.Background(), "SKU-001", nil)
	require.NoError(t, err)
	assert.NotNil(t, product)
	assert.Equal(t, "SKU-001", product.SKU)
	assert.Equal(t, "Indomie Goreng", product.Name)
}

func TestPostgresRepository_GetProductBySKU_NotFound(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// Test getting non-existent product
	product, err := repo.GetProductBySKU(context.Background(), "NON-EXISTENT", nil)
	assert.Error(t, err)
	assert.Nil(t, product)
}

func TestPostgresRepository_GetAllProducts_WithMultipleCategoryFilter(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Close(t)

	repo := NewPostgresRepository(testDB.Pool())

	// First, get a category ID to test with
	var foodCategoryID int
	err := testDB.Pool().QueryRow(context.Background(),
		"SELECT id FROM categories WHERE name = 'Makanan' LIMIT 1").Scan(&foodCategoryID)
	if err != nil {
		t.Skip("Makanan category not found in test data")
		return
	}

	var beverageCategoryID int
	err = testDB.Pool().QueryRow(context.Background(),
		"SELECT id FROM categories WHERE name = 'Minuman' LIMIT 1").Scan(&beverageCategoryID)
	if err != nil {
		t.Skip("Minuman category not found in test data")
		return
	}

	// Test getting products with multiple category filter
	products, total, err := repo.GetAllProducts(
		context.Background(),
		10, 0, "", []int{foodCategoryID, beverageCategoryID}, "name", "asc", nil, nil, "",
	)
	require.NoError(t, err)
	if total > 0 {
		// Verify all returned products belong to the filtered categories
		for _, product := range products {
			if product.CategoryID != nil {
				assert.Contains(t, []int{foodCategoryID, beverageCategoryID}, *product.CategoryID)
			}
		}
	}
}
