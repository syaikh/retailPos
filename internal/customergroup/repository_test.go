package customergroup

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestCustomerGroupRepository_CRUD(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		cg := &CustomerGroup{
			Name:        "Test Group CRUD",
			Description: "Integration test",
			IsActive:    true,
		}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		require.Greater(t, cg.ID, 0)

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Group CRUD", fetched.Name)
		assert.Equal(t, "Integration test", fetched.Description)
		assert.True(t, fetched.IsActive)
	})

	t.Run("Get all with pagination", func(t *testing.T) {
		groups, total, err := repo.GetAll(ctx, 10, 0, "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("Get all with search", func(t *testing.T) {
		groups, total, err := repo.GetAll(ctx, 10, 0, "Test Group CRUD", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("Get all with is_active filter", func(t *testing.T) {
		active := true
		groups, _, err := repo.GetAll(ctx, 10, 0, "", &active, nil)
		require.NoError(t, err)
		for _, g := range groups {
			assert.True(t, g.IsActive)
		}
	})

	t.Run("Get all active", func(t *testing.T) {
		groups, err := repo.GetAllActive(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("NameExists", func(t *testing.T) {
		exists, err := repo.NameExists(ctx, "Test Group CRUD", 0)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Update", func(t *testing.T) {
		// Create a fresh group for update testing
		cg := &CustomerGroup{Name: "To Update", Description: "", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		fetched.Name = "Updated Group"
		err = repo.Update(ctx, fetched)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, fetched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Group", updated.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		cg := &CustomerGroup{Name: "To Delete", Description: "", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)

		err = repo.Delete(ctx, cg.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, cg.ID)
		assert.Error(t, err)
	})

	t.Run("Create with color", func(t *testing.T) {
		cg := &CustomerGroup{
			Name:     "Colored Group",
			IsActive: true,
			Color:    "#FF5733",
		}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		assert.Equal(t, "#FF5733", fetched.Color)
	})

	t.Run("Create with empty color uses default", func(t *testing.T) {
		cg := &CustomerGroup{
			Name:     "Default Color Group",
			IsActive: true,
		}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		assert.Equal(t, "#6C5CE7", fetched.Color)
	})

	t.Run("Update color", func(t *testing.T) {
		cg := &CustomerGroup{Name: "Color Update", IsActive: true, Color: "#6C5CE7"}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		fetched.Color = "#00B894"
		err = repo.Update(ctx, fetched)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		assert.Equal(t, "#00B894", updated.Color)
	})

	t.Run("CustomerCount returns 0 for group with no customers", func(t *testing.T) {
		cg := &CustomerGroup{Name: "No Customers CG", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, fetched.CustomerCount)
	})

	t.Run("GetByName finds existing group", func(t *testing.T) {
		cg := &CustomerGroup{Name: "FindByName Group", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		found, err := repo.GetByName(ctx, "FindByName Group")
		require.NoError(t, err)
		assert.Equal(t, cg.ID, found.ID)
		assert.Equal(t, "FindByName Group", found.Name)
	})

	t.Run("GetByName case insensitive", func(t *testing.T) {
		cg := &CustomerGroup{Name: "CaseInsensitive Group", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		found, err := repo.GetByName(ctx, "caseinsensitive group")
		require.NoError(t, err)
		assert.Equal(t, cg.ID, found.ID)
	})

	t.Run("GetByName returns error for non-existent", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "NonExistent Group 99999")
		assert.Error(t, err)
	})

	t.Run("GetAll with hasCustomers=true returns only groups with customers", func(t *testing.T) {
		hasCust := true
		groups, _, err := repo.GetAll(ctx, 100, 0, "", nil, &hasCust)
		require.NoError(t, err)
		for _, g := range groups {
			assert.Greater(t, g.CustomerCount, 0, "group %s should have customers", g.Name)
		}
	})

	t.Run("GetAll with hasCustomers=false returns groups without customers", func(t *testing.T) {
		hasCust := false
		groups, _, err := repo.GetAll(ctx, 100, 0, "", nil, &hasCust)
		require.NoError(t, err)
		for _, g := range groups {
			assert.Equal(t, 0, g.CustomerCount, "group %s should have 0 customers", g.Name)
		}
	})

	t.Run("GetAll search matches description", func(t *testing.T) {
		cg := &CustomerGroup{Name: "UniqueSearchable", Description: "FindThisDesc", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		groups, total, err := repo.GetAll(ctx, 10, 0, "FindThisDesc", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("BulkUpdate sets is_active", func(t *testing.T) {
		cg1 := &CustomerGroup{Name: "BulkUpd1", IsActive: true}
		cg2 := &CustomerGroup{Name: "BulkUpd2", IsActive: true}
		require.NoError(t, repo.Create(ctx, cg1))
		require.NoError(t, repo.Create(ctx, cg2))
		defer func() { _ = repo.Delete(ctx, cg1.ID) }()
		defer func() { _ = repo.Delete(ctx, cg2.ID) }()

		updated, err := repo.BulkUpdate(ctx, []int{cg1.ID, cg2.ID}, false)
		require.NoError(t, err)
		assert.Equal(t, 2, updated)

		f1, err := repo.GetByID(ctx, cg1.ID)
		require.NoError(t, err)
		assert.False(t, f1.IsActive)
		f2, err := repo.GetByID(ctx, cg2.ID)
		require.NoError(t, err)
		assert.False(t, f2.IsActive)
	})

	t.Run("BulkUpdate with empty IDs returns 0", func(t *testing.T) {
		updated, err := repo.BulkUpdate(ctx, []int{}, true)
		require.NoError(t, err)
		assert.Equal(t, 0, updated)
	})

	t.Run("BulkDelete removes groups", func(t *testing.T) {
		cg1 := &CustomerGroup{Name: "BulkDel1", IsActive: true}
		cg2 := &CustomerGroup{Name: "BulkDel2", IsActive: true}
		require.NoError(t, repo.Create(ctx, cg1))
		require.NoError(t, repo.Create(ctx, cg2))

		deleted, err := repo.BulkDelete(ctx, []int{cg1.ID, cg2.ID})
		require.NoError(t, err)
		assert.Equal(t, 2, deleted)

		_, err = repo.GetByID(ctx, cg1.ID)
		assert.Error(t, err)
		_, err = repo.GetByID(ctx, cg2.ID)
		assert.Error(t, err)
	})

	t.Run("BulkDelete with empty IDs returns 0", func(t *testing.T) {
		deleted, err := repo.BulkDelete(ctx, []int{})
		require.NoError(t, err)
		assert.Equal(t, 0, deleted)
	})

	t.Run("BulkUpsertCustomerGroups inserts new and updates existing", func(t *testing.T) {
		cg := &CustomerGroup{Name: "Upsert Existing", IsActive: true}
		require.NoError(t, repo.Create(ctx, cg))
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		records := []ImportRow{
			{Row: 1, Name: "Upsert Existing", Description: "Updated via upsert", IsActive: false},
			{Row: 2, Name: "Upsert Brand New", Description: "Created via upsert", IsActive: true},
		}
		result := repo.BulkUpsertCustomerGroups(ctx, records)
		assert.Equal(t, 1, result.Inserted)
		assert.Equal(t, 1, result.Updated)
		assert.Empty(t, result.Errors)

		created, err := repo.GetByName(ctx, "Upsert Brand New")
		require.NoError(t, err)
		defer func() { _ = repo.Delete(ctx, created.ID) }()
	})

	t.Run("GetAllForExport returns all groups", func(t *testing.T) {
		groups, err := repo.GetAllForExport(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("GetAllForExport includes color field", func(t *testing.T) {
		cg := &CustomerGroup{Name: "ExportColor", IsActive: true, Color: "#E17055"}
		require.NoError(t, repo.Create(ctx, cg))
		defer func() { _ = repo.Delete(ctx, cg.ID) }()

		groups, err := repo.GetAllForExport(ctx)
		require.NoError(t, err)
		found := false
		for _, g := range groups {
			if g.ID == cg.ID {
				assert.Equal(t, "#E17055", g.Color)
				found = true
				break
			}
		}
		assert.True(t, found, "created group should appear in export")
	})
}

func TestCustomerGroupRepository_GetByID_NotFound(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999999)
	assert.Error(t, err)
}

func TestCustomerGroupRepository_GetAll_AllFiltersCombined(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "AllFiltersCG", Description: "combo test", IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))
	defer func() { _ = repo.Delete(ctx, cg.ID) }()

	active := true
	hasCust := false
	groups, total, err := repo.GetAll(ctx, 10, 0, "AllFiltersCG", &active, &hasCust)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(groups), 1)
	for _, g := range groups {
		assert.True(t, g.IsActive)
		assert.Equal(t, 0, g.CustomerCount)
	}
}
