package category

import (
	"context"
	"testing"

	importexportshared "retail-pos-system/internal/shared/importexport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "Hello World", "hello-world"},
		{"lowercase", "UPPER CASE", "upper-case"},
		{"apostrophe removed", "Men's Wear", "mens-wear"},
		{"double quotes removed", `He said "Hi"`, "he-said-hi"},
		{"ampersand", "Food & Drink", "food-and-drink"},
		{"slash", "Men/Women", "men-women"},
		{"plus", "C++", "cplusplus"},
		{"equals", "a=b", "aequalsb"},
		{"question mark removed", "What?", "what"},
		{"exclamation removed", "Wow!", "wow"},
		{"at sign", "user@home", "userathome"},
		{"hash", "#1 Product", "number1-product"},
		{"percent", "100%", "100percent"},
		{"parentheses removed", "Items (Sale)", "items-sale"},
		{"double dash collapsed", "a---b", "a-b"},
		{"trimmed dashes", "-hello-", "hello"},
		{"spaces become dashes", "a b c", "a-b-c"},
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
		{"only special chars", "!@#$%", "atnumber$percent"},
		{"120 char input truncated", "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmno", "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijkl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSlug(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateSlug_Truncation(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "a"
	}
	slug := generateSlug(long)
	assert.LessOrEqual(t, len(slug), 120)
	assert.Equal(t, 120, len(slug))
}

func TestCategoryParseBool(t *testing.T) {
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
		{"True", false},
		{"No", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseBool(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestImportResult_AddError(t *testing.T) {
	r := &ImportResult{}
	assert.Empty(t, r.Errors)

	r.AddError(3, "name is required")
	assert.Equal(t, 1, len(r.Errors))
	assert.Equal(t, "row 3: name is required", r.Errors[0])

	r.AddError(7, "duplicate entry")
	assert.Equal(t, 2, len(r.Errors))
	assert.Equal(t, "row 7: duplicate entry", r.Errors[1])
}

func TestCategoryAdapter_LoadReferences(t *testing.T) {
	a := &adapter{}
	refs, err := a.Repository().(*categoryRepoAdapter).LoadReferences(context.TODO(), importexportshared.ModuleSchema{})
	assert.NoError(t, err)
	assert.Nil(t, refs)
}

func TestCategoryMapToEntity(t *testing.T) {
	a := &adapter{}

	t.Run("valid row", func(t *testing.T) {
		row := map[string]interface{}{
			"Name":        "Electronics",
			"Slug":        "electronics",
			"Description": "All electronics",
			"IsActive":    "true",
			"_row":        1,
		}
		entity, err := a.MapToEntity(context.TODO(), importexportshared.ModuleSchema{}, row)
		require.NoError(t, err)
		cRow, ok := entity.(CategoryImportRow)
		require.True(t, ok)
		assert.Equal(t, "Electronics", cRow.Name)
		assert.Equal(t, "electronics", cRow.Slug)
		assert.Equal(t, "All electronics", cRow.Description)
		assert.True(t, cRow.IsActive)
		assert.Equal(t, 1, cRow.Row)
	})

	t.Run("empty name", func(t *testing.T) {
		row := map[string]interface{}{"Name": ""}
		_, err := a.MapToEntity(context.TODO(), importexportshared.ModuleSchema{}, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Name is required")
	})
}
