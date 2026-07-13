package customer

import (
	"testing"

	importexportshared "retail-pos-system/internal/shared/importexport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCustomerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valid", "John Doe", ""},
		{"single char", "A", ""},
		{"whitespace only", "   ", "name is required"},
		{"empty", "", "name is required"},
		{"200 chars ok", string(make([]byte, 200)), ""},
		{"201 chars", string(make([]byte, 201)), "name must be at most 200 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomerName(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCustomerEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valid simple", "user@example.com", ""},
		{"valid with dots", "user.name@domain.co", ""},
		{"valid with plus", "user+tag@example.com", ""},
		{"valid subdomain", "user@sub.domain.com", ""},
		{"empty", "", "email is required"},
		{"whitespace only", "  ", "email is required"},
		{"no at sign", "userexample.com", "invalid email format"},
		{"no domain", "user@", "invalid email format"},
		{"no tld", "user@example", "invalid email format"},
		{"space in email", "user @example.com", "invalid email format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomerEmail(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCustomerPhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valid digits", "08123456789", ""},
		{"valid with spaces", "0812 345 6789", ""},
		{"valid with dashes", "0812-345-6789", ""},
		{"valid with plus", "+62812345678", ""},
		{"valid with parens", "(021) 555-1234", ""},
		{"empty", "", "phone is required"},
		{"whitespace only", "  ", "phone is required"},
		{"too short", "123", "invalid phone format"},
		{"letters", "abcdefghij", "invalid phone format"},
		{"special chars", "0812@345#678", "invalid phone format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomerPhone(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomerStrPtr(t *testing.T) {
	p := strPtr("hello")
	require.NotNil(t, p)
	assert.Equal(t, "hello", *p)

	p = strPtr("")
	assert.Nil(t, p)
}

func TestExtractStoreID(t *testing.T) {
	t.Run("nil storeID", func(t *testing.T) {
		rows := []CustomerImportRow{
			{Name: "A", StoreID: nil},
			{Name: "B", StoreID: nil},
		}
		assert.Nil(t, extractStoreID(rows))
	})

	t.Run("first non-nil", func(t *testing.T) {
		sid1 := 1
		sid2 := 2
		rows := []CustomerImportRow{
			{Name: "A", StoreID: nil},
			{Name: "B", StoreID: &sid1},
			{Name: "C", StoreID: &sid2},
		}
		got := extractStoreID(rows)
		require.NotNil(t, got)
		assert.Equal(t, 1, *got)
	})

	t.Run("empty slice", func(t *testing.T) {
		assert.Nil(t, extractStoreID(nil))
	})
}

func TestCustomerParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"Yes", true},
		{"TRUE", true},
		{"YES", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"bogus", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseBool(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCustomerImportResult_AddError(t *testing.T) {
	r := &ImportResult{}
	r.AddError(5, "name is required")
	assert.Equal(t, 1, len(r.Errors))
	assert.Equal(t, "row 5: name is required", r.Errors[0])

	r.AddError(10, "duplicate phone")
	assert.Equal(t, 2, len(r.Errors))
}

func TestCustomerAdapter_LoadReferences(t *testing.T) {
	a := &adapter{}
	refs, err := a.Repository().(*customerRepoAdapter).LoadReferences(nil, importexportshared.ModuleSchema{})
	assert.NoError(t, err)
	assert.Nil(t, refs)
}

func TestCustomerStrVal(t *testing.T) {
	assert.Equal(t, "", strVal(nil))
	s := "hello"
	assert.Equal(t, "hello", strVal(&s))
}

func TestCustomerMapToEntity(t *testing.T) {
	a := &adapter{}

	t.Run("valid row", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":     "John",
			"Phone":    "081234",
			"Email":    "john@example.com",
			"Address":  "123 St",
			"Note":     "VIP",
			"IsActive": "true",
			"_row":     1,
		}
		entity, err := a.MapToEntity(nil, importexportshared.ModuleSchema{}, row)
		require.NoError(t, err)
		cRow, ok := entity.(CustomerImportRow)
		require.True(t, ok)
		assert.Equal(t, "John", cRow.Name)
		assert.Equal(t, "081234", cRow.Phone)
		assert.Equal(t, "john@example.com", cRow.Email)
		assert.Equal(t, "123 St", cRow.Address)
		assert.Equal(t, "VIP", cRow.Note)
		assert.True(t, cRow.IsActive)
	})

	t.Run("empty name", func(t *testing.T) {
		row := map[string]interface{}{"Name": ""}
		_, err := a.MapToEntity(nil, importexportshared.ModuleSchema{}, row)
		assert.Error(t, err)
	})

	t.Run("storeID", func(t *testing.T) {
		row := map[string]interface{}{"Name": "Jane", "_store_id": 42}
		entity, err := a.MapToEntity(nil, importexportshared.ModuleSchema{}, row)
		require.NoError(t, err)
		cRow := entity.(CustomerImportRow)
		require.NotNil(t, cRow.StoreID)
		assert.Equal(t, 42, *cRow.StoreID)
	})
}
