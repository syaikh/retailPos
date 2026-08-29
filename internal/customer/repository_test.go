package customer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/customergroup"
	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func newWiredRepo() *Repository {
	repo := NewRepository(dbPool)
	repo.SetCustomerGroupNameProvider(customergroup.NameLookup{})
	return repo
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

func TestCustomerRepository_GetByPhone(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		phone := "081234567890"
		err := repo.CreateCustomer(ctx, &Customer{Name: "Ahmad Fauzi", Phone: &phone, Email: ptr("ahmad@test.com"), IsActive: true})
		require.NoError(t, err)

		c, err := repo.GetByPhone(ctx, "081234567890", nil)
		require.NoError(t, err)
		assert.Equal(t, "Ahmad Fauzi", c.Name)
		assert.False(t, c.IsWalkIn)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByPhone(ctx, "999999999999", nil)
		assert.ErrorContains(t, err, "customer not found")
	})
}

func TestCustomerRepository_GetCustomerByID(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		phone := "0811WALKIN001"
		created := &Customer{Name: "Pelanggan Umum / Walk-in", Phone: &phone, Email: ptr("walkin@test.com"), IsWalkIn: true, IsActive: true}
		err := repo.CreateCustomer(ctx, created)
		require.NoError(t, err)

		c, err := repo.GetCustomerByID(ctx, created.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "Pelanggan Umum / Walk-in", c.Name)
		assert.True(t, c.IsWalkIn)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCustomerByID(ctx, -1, nil)
		assert.ErrorContains(t, err, "customer not found")
	})
}

func TestCustomerRepository_GetAllCustomers(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	// TestMain truncates the DB and never loads seeds, so this test creates the
	// customers it relies on instead of assuming 014_customers.sql exists.
	fixtures := []struct{ name, phone string }{
		{"Ahmad Zulkarnaen", "081100000001"},
		{"Budi Santoso", "081100000002"},
		{"Siti Nurhaliza", "081100000003"},
		{"Dewi Lestari", "081100000004"},
		{"Eko Prasetyo", "081100000005"},
		{"Rina Wijaya", "081100000006"},
		{"Hendra Gunawan", "081100000007"},
		{"Maya Anggraini", "081100000008"},
		{"Rizky Pratama", "081100000009"},
	}
	for _, f := range fixtures {
		phone := f.phone
		require.NoError(t, repo.CreateCustomer(ctx, &Customer{Name: f.name, Phone: &phone, Email: ptr("list@test.com"), IsActive: true}))
	}
	walkInPhone := "081100000010"
	require.NoError(t, repo.CreateCustomer(ctx, &Customer{Name: "Walk-in List", Phone: &walkInPhone, Email: ptr("walkin.list@test.com"), IsWalkIn: true, IsActive: true}))

	t.Run("default list excludes walk-in", func(t *testing.T) {
		customers, total, err := repo.GetAllCustomers(ctx, 100, 0, "", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(fixtures))
		assert.GreaterOrEqual(t, len(customers), len(fixtures))
		for _, c := range customers {
			assert.False(t, c.IsWalkIn)
		}
	})

	t.Run("search by name", func(t *testing.T) {
		customers, total, err := repo.GetAllCustomers(ctx, 10, 0, "Ahmad Zulkarnaen", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, customers, 1)
		assert.Contains(t, customers[0].Name, "Ahmad")
	})

	t.Run("filter by active", func(t *testing.T) {
		active := true
		customers, total, err := repo.GetAllCustomers(ctx, 100, 0, "", &active, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, len(fixtures))
		for _, c := range customers {
			assert.True(t, c.IsActive)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		customers, total, err := repo.GetAllCustomers(ctx, 2, 0, "", nil, nil, nil)
		require.NoError(t, err)
		assert.Len(t, customers, 2)
		assert.GreaterOrEqual(t, total, len(fixtures))
	})
}

func TestCustomerRepository_CreateCustomer(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	t.Run("create regular customer", func(t *testing.T) {
		phone := "081111111111"
		c := &Customer{
			Name:     "Test Create",
			Phone:    &phone,
			Email:    ptr("test@example.com"),
			IsActive: true,
		}
		err := repo.CreateCustomer(ctx, c)
		require.NoError(t, err)
		assert.Greater(t, c.ID, 0)

		got, err := repo.GetCustomerByID(ctx, c.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "Test Create", got.Name)
		assert.Equal(t, phone, *got.Phone)
	})

	t.Run("create with all fields", func(t *testing.T) {
		phone := "082222222222"
		email := "test@example.com"
		addr := "Jl. Test No. 1"
		taxID := "12345"
		note := "test note"
		c := &Customer{
			Name:     "Test Full Fields",
			Phone:    &phone,
			Email:    &email,
			Address:  &addr,
			TaxID:    &taxID,
			Note:     &note,
			IsActive: true,
		}
		err := repo.CreateCustomer(ctx, c)
		require.NoError(t, err)
		assert.Greater(t, c.ID, 0)

		got, err := repo.GetCustomerByID(ctx, c.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, email, *got.Email)
		assert.Equal(t, addr, *got.Address)
		assert.Equal(t, taxID, *got.TaxID)
		assert.Equal(t, note, *got.Note)
	})
}

func TestCustomerRepository_UpdateCustomer(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	phone := "083333333333"
	c := &Customer{
		Name:     "Before Update",
		Phone:    &phone,
		Email:    ptr("test@example.com"),
		IsActive: true,
	}
	err := repo.CreateCustomer(ctx, c)
	require.NoError(t, err)

	newPhone := "083333333334"
	c.Name = "After Update"
	c.Phone = &newPhone
	c.IsActive = false

	err = repo.UpdateCustomer(ctx, c, c.ID, nil)
	require.NoError(t, err)

	got, err := repo.GetCustomerByID(ctx, c.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "After Update", got.Name)
	assert.Equal(t, newPhone, *got.Phone)
	assert.False(t, got.IsActive)
}

func TestCustomerRepository_DeleteCustomer(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	phone := "084444444444"
	c := &Customer{
		Name:     "To Be Deleted",
		Phone:    &phone,
		Email:    ptr("test@example.com"),
		IsActive: true,
	}
	err := repo.CreateCustomer(ctx, c)
	require.NoError(t, err)

	err = repo.DeleteCustomer(ctx, c.ID, nil)
	require.NoError(t, err)

	got, err := repo.GetCustomerByID(ctx, c.ID, nil)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
}

func TestCustomerRepository_BulkUpdateCustomersStatus(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	phone1 := "085555555551"
	phone2 := "085555555552"
	c1 := &Customer{Name: "Bulk Update 1", Phone: &phone1, Email: ptr("test@example.com"), IsActive: true}
	c2 := &Customer{Name: "Bulk Update 2", Phone: &phone2, Email: ptr("test@example.com"), IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c1))
	require.NoError(t, repo.CreateCustomer(ctx, c2))

	err := repo.BulkUpdateCustomersStatus(ctx, []int{c1.ID, c2.ID}, false, nil)
	require.NoError(t, err)

	got1, _ := repo.GetCustomerByID(ctx, c1.ID, nil)
	got2, _ := repo.GetCustomerByID(ctx, c2.ID, nil)
	assert.False(t, got1.IsActive)
	assert.False(t, got2.IsActive)
}

func TestCustomerRepository_BulkDeleteCustomers(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	phone1 := "086666666661"
	phone2 := "086666666662"
	c1 := &Customer{Name: "Bulk Delete 1", Phone: &phone1, Email: ptr("test@example.com"), IsActive: true}
	c2 := &Customer{Name: "Bulk Delete 2", Phone: &phone2, Email: ptr("test@example.com"), IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c1))
	require.NoError(t, repo.CreateCustomer(ctx, c2))

	err := repo.BulkDeleteCustomers(ctx, []int{c1.ID, c2.ID}, nil)
	require.NoError(t, err)

	got1, _ := repo.GetCustomerByID(ctx, c1.ID, nil)
	got2, _ := repo.GetCustomerByID(ctx, c2.ID, nil)
	assert.False(t, got1.IsActive)
	assert.False(t, got2.IsActive)
}

func ptr(s string) *string { return &s }

// TestCustomerRepository_GroupNameEnrichment verifies that customer reads
// resolve customer_group_name through the customergroup-owned port instead of
// a direct customer_groups JOIN.
func TestCustomerRepository_GroupNameEnrichment(t *testing.T) {
	repo := newWiredRepo()
	ctx := context.Background()

	var groupID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO customer_groups (name, is_active) VALUES ($1, true) RETURNING id`,
		"Platinum "+time.Now().Format("0102150405.000000")).Scan(&groupID))

	phone := "08777" + time.Now().Format("0102150405")
	c := &Customer{Name: "Grouped Customer", Phone: &phone, Email: ptr("grouped@test.com"), CustomerGroupID: &groupID, IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c))
	customerID := c.ID

	t.Run("by phone", func(t *testing.T) {
		got, err := repo.GetByPhone(ctx, phone, nil)
		require.NoError(t, err)
		require.NotNil(t, got.CustomerGroupID)
		assert.Equal(t, groupID, *got.CustomerGroupID)
		require.NotNil(t, got.CustomerGroupName)
		assert.Contains(t, *got.CustomerGroupName, "Platinum")
	})

	t.Run("by id", func(t *testing.T) {
		got, err := repo.GetCustomerByID(ctx, customerID, nil)
		require.NoError(t, err)
		require.NotNil(t, got.CustomerGroupName)
		assert.Contains(t, *got.CustomerGroupName, "Platinum")
	})

	t.Run("list", func(t *testing.T) {
		customers, _, err := repo.GetAllCustomers(ctx, 10, 0, "", nil, nil, nil)
		require.NoError(t, err)
		var found bool
		for _, row := range customers {
			if row.ID == customerID {
				found = true
				require.NotNil(t, row.CustomerGroupName)
				assert.Contains(t, *row.CustomerGroupName, "Platinum")
			}
		}
		assert.True(t, found)
	})
}
