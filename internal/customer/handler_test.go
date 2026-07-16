package customer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

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
		c.Set("permissions", []string{"customer:create", "customer:read", "customer:update", "customer:delete"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupCustomerRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)

	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_GetCustomers(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	phone1 := "0812AHMAD001"
	phone2 := "0812AHMAD002"
	c1 := &Customer{Name: "Ahmad Fauzi", Phone: &phone1, Email: ptr("ahmad@test.com"), IsActive: true}
	c2 := &Customer{Name: "Ahmad Hidayat", Phone: &phone2, Email: ptr("hidayat@test.com"), IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c1))
	require.NoError(t, repo.CreateCustomer(ctx, c2))

	t.Run("returns list with seeded data", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customers?limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Customer `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, resp.Total, 1)
		assert.LessOrEqual(t, len(resp.Data), 10)
		assert.Greater(t, len(resp.Data), 0)
	})

	t.Run("search by name", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customers?search=Ahmad", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Customer `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, resp.Total, 1)
		assert.Contains(t, resp.Data[0].Name, "Ahmad")
	})

	t.Run("empty search returns empty list", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customers?search=NONEXISTENT_XYZ", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Customer `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})
}

func TestHandler_GetCustomerByID(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	phone := "0811WALKINHDL01"
	c := &Customer{Name: "Pelanggan Umum / Walk-in", Phone: &phone, Email: ptr("walkin@test.com"), IsWalkIn: true, IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	t.Run("found walk-in customer", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customers/"+strconv.Itoa(c.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Customer `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Pelanggan Umum / Walk-in", resp.Data.Name)
		assert.True(t, resp.Data.IsWalkIn)
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customers/999999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customers/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_CreateCustomer(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler Create Test","phone":"0812000001","email":"create@test.com","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Customer `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Create Test", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateCustomer(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	phone := "0812000002"
	c := &Customer{
		Name:     "Handler Before Update",
		Phone:    &phone,
		Email:    ptr("test@example.com"),
		IsActive: true,
	}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler After Update","phone":"0812000002","email":"updated@test.com","is_active":false}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/customers/"+strconv.Itoa(c.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		body := `{"name":"Invalid","phone":"0812TEST002","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/customers/abc", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteCustomer(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	phone := "0812TEST003"
	c := &Customer{
		Name:     "Handler To Delete",
		Phone:    &phone,
		Email:    ptr("test@example.com"),
		IsActive: true,
	}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/customers/"+strconv.Itoa(c.ID), nil)
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
		req, _ := http.NewRequest("DELETE", "/customers/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_BulkUpdateCustomerStatus(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	phone1 := "0812TEST004"
	phone2 := "0812TEST005"
	c1 := &Customer{Name: "Bulk Status 1", Phone: &phone1, Email: ptr("test@example.com"), IsActive: true}
	c2 := &Customer{Name: "Bulk Status 2", Phone: &phone2, Email: ptr("test@example.com"), IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c1))
	require.NoError(t, repo.CreateCustomer(ctx, c2))

	t.Run("success", func(t *testing.T) {
		body := `{"ids":[` + strconv.Itoa(c1.ID) + `,` + strconv.Itoa(c2.ID) + `],"is_active":false}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "updated", resp.Status)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers/bulk/status", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_BulkDeleteCustomers(t *testing.T) {
	skipIfNoDB(t)
	r := setupCustomerRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	phone1 := "0812TEST006"
	phone2 := "0812TEST007"
	c1 := &Customer{Name: "Bulk Delete 1", Phone: &phone1, Email: ptr("test@example.com"), IsActive: true}
	c2 := &Customer{Name: "Bulk Delete 2", Phone: &phone2, Email: ptr("test@example.com"), IsActive: true}
	require.NoError(t, repo.CreateCustomer(ctx, c1))
	require.NoError(t, repo.CreateCustomer(ctx, c2))

	t.Run("success", func(t *testing.T) {
		body := `{"ids":[` + strconv.Itoa(c1.ID) + `,` + strconv.Itoa(c2.ID) + `]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers/bulk/delete", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers/bulk/delete", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
