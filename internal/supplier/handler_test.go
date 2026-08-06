package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"pricing.view", "pricing.create", "pricing.update", "pricing.delete"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(_ permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func setupSupplierRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListSuppliers(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/suppliers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Supplier `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_CreateSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	t.Run("success", func(t *testing.T) {
		code := "HDL-SUP-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		body := `{"name":"Handler Supplier","code":"` + code + `","contact_name":"Jane","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Supplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Supplier", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		body := `{"contact_name":"no name or code"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	repo := NewRepository(dbPool)
	s := &Supplier{
		Name:     "Before Update",
		Code:     "HDL-UPD-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("success", func(t *testing.T) {
		body := `{"name":"After Update","code":"` + s.Code + `","contact_name":"Updated","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/"+strconv.Itoa(s.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Supplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "After Update", resp.Data.Name)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/abc", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	repo := NewRepository(dbPool)
	s := &Supplier{
		Name:     "Delete Handler",
		Code:     "HDL-DEL-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/"+strconv.Itoa(s.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_GetSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)

	s := &Supplier{
		Name:     "Get Handler Supplier",
		Code:     "HDL-GET-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers/"+strconv.Itoa(s.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Supplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, s.Name, resp.Data.Name)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_GetProductsBySupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)

	s := &Supplier{
		Name:     "Products For Supplier",
		Code:     "HDL-PFS-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers/"+strconv.Itoa(s.ID)+"/products", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []ProductSupplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers/abc/products", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_GetSuppliersByProduct(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	ctx := t.Context()
	productID := insertTestProduct(ctx, t, "HDL-SP-"+strconv.FormatInt(time.Now().UnixNano(), 10), "Handler SP Product", 5000)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/"+strconv.Itoa(productID)+"/suppliers", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []ProductSupplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/abc/suppliers", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_LinkProduct(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)
	ctx := t.Context()

	s := &Supplier{
		Name:     "Link Handler Supplier",
		Code:     "HDL-LINK-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "HDL-LP-"+strconv.FormatInt(time.Now().UnixNano(), 10), "Link Product", 7000)

	t.Run("success", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"unit_cost":6000,"lead_time_days":5,"is_preferred":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/"+strconv.Itoa(s.ID)+"/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data ProductSupplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, productID, resp.Data.ProductID)
		assert.Equal(t, s.ID, resp.Data.SupplierID)
	})

	t.Run("invalid supplier id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/abc/products", strings.NewReader(`{"product_id":1,"unit_cost":1000}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UnlinkProduct(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)
	ctx := t.Context()

	s := &Supplier{
		Name:     "Unlink Handler Supplier",
		Code:     "HDL-UNLINK-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "HDL-UP-"+strconv.FormatInt(time.Now().UnixNano(), 10), "Unlink Product", 8000)

	ps := &ProductSupplier{
		ProductID:  productID,
		SupplierID: s.ID,
		UnitCost:   5000,
	}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/"+strconv.Itoa(s.ID)+"/products/"+strconv.Itoa(productID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("invalid supplier id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/abc/products/"+strconv.Itoa(productID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid product id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/"+strconv.Itoa(s.ID)+"/products/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateProductSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)
	ctx := t.Context()

	s := &Supplier{
		Name:     "UPS Handler Supplier",
		Code:     "HDL-UPS-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "HDL-UPS-P-"+strconv.FormatInt(time.Now().UnixNano(), 10), "UPS Product", 9000)

	ps := &ProductSupplier{
		ProductID:  productID,
		SupplierID: s.ID,
		UnitCost:   5000,
	}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	t.Run("success", func(t *testing.T) {
		body := `{"unit_cost":7000,"lead_time_days":3,"is_preferred":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/"+strconv.Itoa(s.ID)+"/products/"+strconv.Itoa(productID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data ProductSupplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 7000, resp.Data.UnitCost)
	})

	t.Run("invalid supplier id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/abc/products/"+strconv.Itoa(productID), strings.NewReader(`{"unit_cost":1000}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid product id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/"+strconv.Itoa(s.ID)+"/products/abc", strings.NewReader(`{"unit_cost":1000}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_SetPreferredSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)
	ctx := t.Context()

	s := &Supplier{
		Name:     "Pref Handler Supplier",
		Code:     "HDL-PREF-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "HDL-PREF-P-"+strconv.FormatInt(time.Now().UnixNano(), 10), "Pref Product", 10000)

	ps := &ProductSupplier{
		ProductID:  productID,
		SupplierID: s.ID,
		UnitCost:   6000,
	}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/"+strconv.Itoa(s.ID)+"/products/"+strconv.Itoa(productID)+"/preferred", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "updated", resp.Status)
	})

	t.Run("invalid supplier id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/abc/products/"+strconv.Itoa(productID)+"/preferred", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid product id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/"+strconv.Itoa(s.ID)+"/products/abc/preferred", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_BulkUpdate(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)
	ctx := t.Context()

	s1 := &Supplier{Name: "BU-H1", Code: "HDL-BU1-" + strconv.FormatInt(time.Now().UnixNano(), 10), IsActive: true}
	s2 := &Supplier{Name: "BU-H2", Code: "HDL-BU2-" + strconv.FormatInt(time.Now().UnixNano(), 10), IsActive: true}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	t.Run("success", func(t *testing.T) {
		body := `{"ids":[` + strconv.Itoa(s1.ID) + `,` + strconv.Itoa(s2.ID) + `],"is_active":false}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/bulk", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Updated int `json:"updated"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Updated)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/bulk", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_BulkDelete(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()
	repo := NewRepository(dbPool)
	ctx := t.Context()

	s1 := &Supplier{Name: "BD-H1", Code: "HDL-BD1-" + strconv.FormatInt(time.Now().UnixNano(), 10), IsActive: true}
	s2 := &Supplier{Name: "BD-H2", Code: "HDL-BD2-" + strconv.FormatInt(time.Now().UnixNano(), 10), IsActive: true}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	t.Run("success", func(t *testing.T) {
		body := `{"ids":[` + strconv.Itoa(s1.ID) + `,` + strconv.Itoa(s2.ID) + `]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/bulk", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Deleted int `json:"deleted"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Deleted)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/bulk", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_ListSuppliers_IsActiveFilter(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/suppliers?is_active=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateSupplier_ErrorBranches(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	repo := NewRepository(dbPool)
	s := &Supplier{
		Name:     "Branch Fill",
		Code:     "HDL-FILL-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/"+strconv.Itoa(s.ID), strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fill name from existing", func(t *testing.T) {
		body := `{"code":"` + s.Code + `","contact_name":"Only Code"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/"+strconv.Itoa(s.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data Supplier `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Branch Fill", resp.Data.Name)
	})

	t.Run("validation fails on missing supplier", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/999999999", strings.NewReader(`{"contact_name":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_LinkProduct_ErrorBranches(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	t.Run("bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/1/products", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation fails", func(t *testing.T) {
		body := `{"product_id":0,"supplier_id":0,"unit_cost":-5}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/1/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateProductSupplier_ErrorBranches(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	t.Run("bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/1/products/1", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation fails", func(t *testing.T) {
		body := `{"product_id":0,"supplier_id":0,"unit_cost":-5}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/1/products/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func setupSupplierMockRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(svc, nil)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_MockErrorBranches(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
			return nil, 0, assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers", nil)
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("list nil suppliers", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
			return nil, 0, nil
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers", nil)
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{getByIDFn: func(ctx context.Context, id int) (*Supplier, error) {
			return nil, assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/suppliers/1", nil)
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{deleteFn: func(ctx context.Context, id int) error {
			return assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/1", nil)
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("unlink error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{unlinkProductFn: func(ctx context.Context, productID, supplierID int) error {
			return assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/1/products/2", nil)
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("set preferred error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{setPreferredSupplierFn: func(ctx context.Context, productID, supplierID int) error {
			return assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers/1/products/2/preferred", nil)
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("bulk update error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{bulkUpdateFn: func(ctx context.Context, ids []int, isActive bool) (int, error) {
			return 0, assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/bulk", strings.NewReader(`{"ids":[1],"is_active":true}`))
		req.Header.Set("Content-Type", "application/json")
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bulk delete error", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{bulkDeleteFn: func(ctx context.Context, ids []int) (int, error) {
			return 0, assert.AnError
		}}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/bulk", strings.NewReader(`{"ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")
		setupSupplierMockRouter(svc).ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
