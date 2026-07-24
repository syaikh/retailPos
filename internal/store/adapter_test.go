package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_NewAdapter(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	assert.NotNil(t, a)
}

func TestAdapter_ModuleName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	assert.Equal(t, "stores", a.ModuleName())
}

func TestAdapter_ValidateBusiness(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	result := a.ValidateBusiness(context.Background(), Schema, nil)
	assert.Nil(t, result)
}

func TestAdapter_MapToEntity(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)

	t.Run("valid row", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "Test Store",
			"Address":  "123 St",
			"Phone":    "08123",
			"IsActive": true,
			"_row":     1,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr, ok := entity.(StoreImportRow)
		require.True(t, ok)
		assert.Equal(t, "Test Store", sr.Name)
		assert.Equal(t, "123 St", sr.Address)
		assert.Equal(t, "08123", sr.Phone)
		assert.True(t, sr.IsActive)
		assert.Equal(t, 1, sr.Row)
	})

	t.Run("missing name", func(t *testing.T) {
		row := map[string]interface{}{
			"Name": "",
			"_row": 2,
		}
		_, err := a.MapToEntity(context.Background(), Schema, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("name nil", func(t *testing.T) {
		row := map[string]interface{}{
			"_row": 3,
		}
		_, err := a.MapToEntity(context.Background(), Schema, row)
		assert.Error(t, err)
	})

	t.Run("is_active true", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "Active Store",
			"IsActive": "true",
			"_row":     4,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr := entity.(StoreImportRow)
		assert.True(t, sr.IsActive)
	})

	t.Run("is_active false", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "Inactive Store",
			"IsActive": "false",
			"_row":     5,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr := entity.(StoreImportRow)
		assert.False(t, sr.IsActive)
	})

	t.Run("is_active 1", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "One Store",
			"IsActive": "1",
			"_row":     6,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr := entity.(StoreImportRow)
		assert.True(t, sr.IsActive)
	})

	t.Run("is_active 0", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "Zero Store",
			"IsActive": "0",
			"_row":     7,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr := entity.(StoreImportRow)
		assert.False(t, sr.IsActive)
	})

	t.Run("is_active yes", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "Yes Store",
			"IsActive": "yes",
			"_row":     8,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr := entity.(StoreImportRow)
		assert.True(t, sr.IsActive)
	})

	t.Run("no IsActive defaults true", func(t *testing.T) {
		row := map[string]interface{}{
			"Name": "Default Active",
			"_row": 9,
		}
		entity, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		sr := entity.(StoreImportRow)
		assert.True(t, sr.IsActive)
	})
}

func TestAdapter_Repository(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	r := a.Repository()
	assert.NotNil(t, r)
}

func TestAdapter_Insert(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	r := a.Repository()

	entities := []interface{}{
		StoreImportRow{Row: 1, Name: "Adapter Store 1", Address: "Addr1", Phone: "111", IsActive: true},
		StoreImportRow{Row: 2, Name: "Adapter Store 2", Address: "Addr2", Phone: "222", IsActive: false},
	}
	count, err := r.Insert(context.Background(), entities)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	s1, err := repo.GetByName(context.Background(), "Adapter Store 1")
	require.NoError(t, err)
	assert.Equal(t, "Addr1", s1.Address)
	assert.True(t, s1.IsActive)

	s2, err := repo.GetByName(context.Background(), "Adapter Store 2")
	require.NoError(t, err)
	assert.False(t, s2.IsActive)
}

func TestAdapter_Update_Existing(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	r := a.Repository()

	existing := &Store{Name: "Adapter Update Target", Address: "Old Addr", Phone: "000", IsActive: true}
	err := repo.Create(context.Background(), existing)
	require.NoError(t, err)

	entities := []interface{}{
		StoreImportRow{Row: 1, Name: "Adapter Update Target", Address: "New Addr", Phone: "999", IsActive: false},
	}
	count, err := r.Update(context.Background(), entities)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	updated, err := repo.GetByName(context.Background(), "Adapter Update Target")
	require.NoError(t, err)
	assert.Equal(t, "New Addr", updated.Address)
	assert.Equal(t, "999", updated.Phone)
	assert.False(t, updated.IsActive)
}

func TestAdapter_Update_InsertWhenNotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	r := a.Repository()

	entities := []interface{}{
		StoreImportRow{Row: 1, Name: "Adapter Upsert Brand New", Address: "Brand New Addr", Phone: "555", IsActive: true},
	}
	count, err := r.Update(context.Background(), entities)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	found, err := repo.GetByName(context.Background(), "Adapter Upsert Brand New")
	require.NoError(t, err)
	assert.Equal(t, "Brand New Addr", found.Address)
}

func TestAdapter_ExportData(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	r := a.Repository()

	data, err := r.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestAdapter_LoadReferences(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	r := a.Repository()

	refs, err := r.LoadReferences(context.Background(), Schema)
	require.NoError(t, err)
	assert.Nil(t, refs)
}

func Test_parseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"something", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseBool(tt.input))
		})
	}
}

func TestStoreImportRow_Structure(t *testing.T) {
	skipIfNoDB(t)
	row := StoreImportRow{
		Row:      1,
		Name:     "Struct",
		Address:  "Addr",
		Phone:    "Phone",
		IsActive: true,
	}
	assert.Equal(t, 1, row.Row)
	assert.Equal(t, "Struct", row.Name)
	assert.Equal(t, "Addr", row.Address)
	assert.Equal(t, "Phone", row.Phone)
	assert.True(t, row.IsActive)
}
