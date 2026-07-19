package supplier

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrPtr(t *testing.T) {
	t.Run("returns pointer for non-empty string", func(t *testing.T) {
		result := strPtr("hello")
		require.NotNil(t, result)
		assert.Equal(t, "hello", *result)
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		result := strPtr("")
		assert.Nil(t, result)
	})
}

func TestNilStr(t *testing.T) {
	t.Run("returns string from pointer", func(t *testing.T) {
		s := "hello"
		result := nilStr(&s)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns empty for nil pointer", func(t *testing.T) {
		result := nilStr(nil)
		assert.Equal(t, "", result)
	})
}

func TestNewAdapter(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	assert.NotNil(t, a)
}

func TestAdapter_ModuleName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	assert.Equal(t, "suppliers", a.ModuleName())
}

func TestAdapter_ValidateBusiness(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	result := a.ValidateBusiness(context.Background(), Schema, []map[string]interface{}{})
	assert.Nil(t, result)
}

func TestAdapter_MapToEntity(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)

	t.Run("valid row", func(t *testing.T) {
		row := map[string]interface{}{
			"Code":        "SUP-ADP-001",
			"Name":        "Test Adapter Supplier",
			"ContactName": "John",
			"Phone":       "08123456789",
			"Email":       "john@test.com",
			"Address":     "Jakarta",
			"Notes":       "some notes",
			"IsActive":    "true",
			"_row":        1,
		}
		result, err := a.MapToEntity(context.Background(), Schema, row)
		require.NoError(t, err)
		require.NotNil(t, result)
		importRow, ok := result.(SupplierImportRow)
		require.True(t, ok)
		assert.Equal(t, "SUP-ADP-001", importRow.Code)
		assert.Equal(t, "Test Adapter Supplier", importRow.Name)
		assert.Equal(t, "John", importRow.ContactName)
		assert.Equal(t, "08123456789", importRow.Phone)
		assert.Equal(t, "john@test.com", importRow.Email)
		assert.Equal(t, "Jakarta", importRow.Address)
		assert.Equal(t, "some notes", importRow.Notes)
		assert.True(t, importRow.IsActive)
		assert.Equal(t, 1, importRow.Row)
	})

	t.Run("missing Code", func(t *testing.T) {
		row := map[string]interface{}{
			"Name": "Test",
		}
		_, err := a.MapToEntity(context.Background(), Schema, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Code is required")
	})

	t.Run("missing Name", func(t *testing.T) {
		row := map[string]interface{}{
			"Code": "SUP-001",
		}
		_, err := a.MapToEntity(context.Background(), Schema, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Name is required")
	})
}

func TestAdapter_LoadReferences(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	repoActions := a.Repository()
	result, err := repoActions.LoadReferences(context.Background(), Schema)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestAdapter_Repository_Insert(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	repoActions := a.Repository()

	entities := []interface{}{
		SupplierImportRow{
			Row:         1,
			Code:        "SUP-BULK-INS-" + time.Now().Format("0102150405"),
			Name:        "Bulk Insert Supplier",
			ContactName: "Jane",
			Phone:       "08123456789",
			Email:       "jane@test.com",
			Address:     "Bandung",
			Notes:       "test",
			IsActive:    true,
		},
	}
	count, err := repoActions.Insert(context.Background(), entities)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAdapter_Repository_Update(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)

	code := "SUP-BULK-UPD-" + time.Now().Format("0102150405")
	s := &Supplier{
		Name:     "Before Adapter Update",
		Code:     code,
		IsActive: true,
	}
	require.NoError(t, repo.Create(context.Background(), s))

	a := NewAdapter(repo)
	repoActions := a.Repository()

	entities := []interface{}{
		SupplierImportRow{
			Row:         1,
			Code:        code,
			Name:        "After Adapter Update",
			ContactName: "Updated",
			Phone:       "08999999999",
			Email:       "updated@test.com",
			Address:     "Surabaya",
			Notes:       "updated notes",
			IsActive:    false,
		},
	}
	count, err := repoActions.Update(context.Background(), entities)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAdapter_Repository_ExportData(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)

	s := &Supplier{
		Name:     "Export Supplier",
		Code:     "SUP-EXP-" + time.Now().Format("0102150405"),
		IsActive: true,
		Notes:    strPtr("export note"),
	}
	require.NoError(t, repo.Create(context.Background(), s))

	a := NewAdapter(repo)
	repoActions := a.Repository()

	data, err := repoActions.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	assert.NotNil(t, data)

	found := false
	for _, item := range data {
		if item["Code"] == s.Code {
			found = true
			assert.Equal(t, s.Name, item["Name"])
			assert.Equal(t, true, item["IsActive"])
			break
		}
	}
	assert.True(t, found, "exported data should contain the created supplier")
}
