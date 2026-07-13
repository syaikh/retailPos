package product

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProduct(t *testing.T) {
	tests := []struct {
		name    string
		product Product
		wantErr string
	}{
		{"valid product", Product{Name: "Widget", Price: 1000, Cost: 500, Stock: 10}, ""},
		{"zero price/cost/stock", Product{Name: "Free Item", Price: 0, Cost: 0, Stock: 0}, ""},
		{"empty name", Product{Name: "", Price: 1000}, "name is required"},
		{"whitespace name", Product{Name: "   ", Price: 1000}, "name is required"},
		{"negative price", Product{Name: "Bad", Price: -1}, "price must not be negative"},
		{"negative cost", Product{Name: "Bad", Cost: -1}, "cost must not be negative"},
		{"negative stock", Product{Name: "Bad", Stock: -1}, "stock must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProduct(&tt.product)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFloatToInt(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int
	}{
		{"float64", 3.7, 4},
		{"float64 exact", 5.0, 5},
		{"float64 half", 2.5, 3},
		{"int", 10, 10},
		{"string numeric", "42", 42},
		{"string non-numeric", "abc", 0},
		{"string empty", "", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := floatToInt(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStrPtr(t *testing.T) {
	p := strPtr("hello")
	require.NotNil(t, p)
	assert.Equal(t, "hello", *p)

	p = strPtr("")
	assert.Nil(t, p)
}

func TestIntPtr(t *testing.T) {
	p := intPtr(42)
	require.NotNil(t, p)
	assert.Equal(t, 42, *p)

	p = intPtr(0)
	assert.Nil(t, p)
}

func TestPtr(t *testing.T) {
	p := ptr(42)
	require.NotNil(t, p)
	assert.Equal(t, 42, *p)

	p = ptr(0)
	assert.Nil(t, p)
}

func TestKeysOf(t *testing.T) {
	result := keysOf(map[string]bool{})
	assert.Empty(t, result)

	result = keysOf(map[string]bool{"a": true})
	assert.Equal(t, []string{"a"}, result)

	result = keysOf(map[string]bool{"a": true, "b": true})
	assert.ElementsMatch(t, []string{"a", "b"}, result)
}

func TestResolveReferences(t *testing.T) {
	r := &productRepoAdapter{}
	catMap := map[string]int{"Electronics": 1}
	brandMap := map[string]int{"Samsung": 2}
	uomMap := map[string]int{"PCS": 3}

	t.Run("valid row", func(t *testing.T) {
		row := ProductImportRow{
			Row: 1, SKU: "SKU001", Name: "Phone", Category: "Electronics",
			Brand: "Samsung", Price: 1000, Cost: 500, Stock: 10,
			Status: "active", UnitOfMeasure: "PCS", WeightGrams: 200,
			Barcode: "123456", Description: "A phone",
		}
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.NoError(t, err)
		assert.Equal(t, "SKU001", payload.SKU)
		assert.Equal(t, "Phone", payload.Name)
		assert.Equal(t, 1000, payload.Price)
		assert.Equal(t, "active", payload.Status)
		assert.NotNil(t, payload.CategoryID)
		assert.Equal(t, 1, *payload.CategoryID)
		assert.NotNil(t, payload.BrandID)
		assert.Equal(t, 2, *payload.BrandID)
		assert.NotNil(t, payload.UnitOfMeasureID)
		assert.Equal(t, 3, *payload.UnitOfMeasureID)
		assert.NotNil(t, payload.Barcode)
		assert.Equal(t, "123456", *payload.Barcode)
	})

	t.Run("missing category", func(t *testing.T) {
		row := ProductImportRow{Row: 2, SKU: "SKU002", Name: "Item", Category: "Nonexistent", Price: 100}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "category")
	})

	t.Run("missing brand", func(t *testing.T) {
		row := ProductImportRow{Row: 3, SKU: "SKU003", Name: "Item", Brand: "Nonexistent", Price: 100}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "brand")
	})

	t.Run("missing UoM", func(t *testing.T) {
		row := ProductImportRow{Row: 4, SKU: "SKU004", Name: "Item", UnitOfMeasure: "NONEXISTENT", Price: 100}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit of measure")
	})

	t.Run("empty name", func(t *testing.T) {
		row := ProductImportRow{Row: 5, SKU: "SKU005", Name: "", Price: 100}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("negative price", func(t *testing.T) {
		row := ProductImportRow{Row: 6, SKU: "SKU006", Name: "Item", Price: -1}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "price must not be negative")
	})

	t.Run("negative cost", func(t *testing.T) {
		row := ProductImportRow{Row: 7, SKU: "SKU007", Name: "Item", Cost: -5}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cost must not be negative")
	})

	t.Run("negative stock", func(t *testing.T) {
		row := ProductImportRow{Row: 8, SKU: "SKU008", Name: "Item", Stock: -10}
		_, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stock must not be negative")
	})

	t.Run("empty refs become nil pointers", func(t *testing.T) {
		row := ProductImportRow{Row: 9, SKU: "SKU009", Name: "Item", Price: 100}
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.NoError(t, err)
		assert.Nil(t, payload.CategoryID)
		assert.Nil(t, payload.BrandID)
		assert.Nil(t, payload.UnitOfMeasureID)
		assert.Nil(t, payload.StoreID)
	})

	t.Run("status normalization", func(t *testing.T) {
		row := ProductImportRow{Row: 10, SKU: "SKU010", Name: "Item", Price: 100, Status: "ACTIVE"}
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.NoError(t, err)
		assert.Equal(t, "active", payload.Status)
	})

	t.Run("default status", func(t *testing.T) {
		row := ProductImportRow{Row: 11, SKU: "SKU011", Name: "Item", Price: 100, Status: ""}
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.NoError(t, err)
		assert.Equal(t, "active", payload.Status)
	})

	t.Run("storeID > 0 becomes pointer", func(t *testing.T) {
		row := ProductImportRow{Row: 12, SKU: "SKU012", Name: "Item", Price: 100, StoreID: 5}
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		require.NoError(t, err)
		assert.NotNil(t, payload.StoreID)
		assert.Equal(t, 5, *payload.StoreID)
	})
}

func TestGetStockThresholds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/stock-thresholds", nil)

	h := &Handler{}
	h.GetStockThresholds(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "warning")
	assert.Contains(t, w.Body.String(), "critical")
}

func TestBulkUpdateProductStatus_EmptyIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/products/bulk/status",
		strings.NewReader(`{"ids":[],"is_active":true}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := &Handler{}
	h.BulkUpdateProductStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "no product IDs")
}

func TestBulkUpdateProductStatus_TooManyIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ids := make([]int, 201)
	idsJSON := "["
	for i := range ids {
		if i > 0 {
			idsJSON += ","
		}
		idsJSON += "1"
	}
	idsJSON += "]"
	body := `{"ids":` + idsJSON + `,"is_active":true}`
	c.Request = httptest.NewRequest(http.MethodPost, "/products/bulk/status", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h := &Handler{}
	h.BulkUpdateProductStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "too many")
}

func TestGetProductByID_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/products/abc", nil)
	c.AddParam("id", "abc")

	h := &Handler{}
	h.GetProductByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid product id")
}

func TestDeleteProduct_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/products/abc", nil)
	c.AddParam("id", "abc")

	h := &Handler{}
	h.DeleteProduct(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid product id")
}

func TestGetStoreID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	t.Run("no storeID in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Nil(t, h.getStoreID(c))
	})

	t.Run("storeID present", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		sid := 42
		c.Set("storeID", &sid)
		got := h.getStoreID(c)
		require.NotNil(t, got)
		assert.Equal(t, 42, *got)
	})

	t.Run("storeID wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("storeID", "not-an-int")
		assert.Nil(t, h.getStoreID(c))
	})
}
