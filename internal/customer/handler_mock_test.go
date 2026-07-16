package customer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCustomerService struct {
	getAllFn     func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error)
	getByIDFn    func(ctx context.Context, id int, storeID *int) (*Customer, error)
	createFn     func(ctx context.Context, customer *Customer, storeID *int) error
	updateFn     func(ctx context.Context, customer *Customer, id int, storeID *int) error
	deleteFn     func(ctx context.Context, id int, storeID *int) error
	bulkStatusFn func(ctx context.Context, ids []int, isActive bool, storeID *int) error
	bulkDeleteFn func(ctx context.Context, ids []int, storeID *int) error
}

func (m *mockCustomerService) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
	return m.getAllFn(ctx, limit, offset, search, isActive, storeID)
}
func (m *mockCustomerService) GetCustomerByID(ctx context.Context, id int, storeID *int) (*Customer, error) {
	return m.getByIDFn(ctx, id, storeID)
}
func (m *mockCustomerService) CreateCustomer(ctx context.Context, customer *Customer, storeID *int) error {
	return m.createFn(ctx, customer, storeID)
}
func (m *mockCustomerService) UpdateCustomer(ctx context.Context, customer *Customer, id int, storeID *int) error {
	return m.updateFn(ctx, customer, id, storeID)
}
func (m *mockCustomerService) DeleteCustomer(ctx context.Context, id int, storeID *int) error {
	return m.deleteFn(ctx, id, storeID)
}
func (m *mockCustomerService) BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	return m.bulkStatusFn(ctx, ids, isActive, storeID)
}
func (m *mockCustomerService) BulkDeleteCustomers(ctx context.Context, ids []int, storeID *int) error {
	return m.bulkDeleteFn(ctx, ids, storeID)
}

var _ CustomerService = (*mockCustomerService)(nil)

func setupMockRouter(svc CustomerService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestMockHandler_GetCustomers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, 10, limit)
				return []Customer{{ID: 1, Name: "Test"}}, 1, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?limit=10", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["total"])
	})

	t.Run("nil customers becomes empty array", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				return nil, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("limit clamped to 200", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, 200, limit)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?limit=500", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("is_active param parsed", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				require.NotNil(t, isActive)
				assert.True(t, *isActive)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?is_active=true", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative offset clamped", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, 0, offset)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?offset=-5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMockHandler_GetCustomerByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				assert.Equal(t, 42, id)
				return &Customer{ID: 42, Name: "Found"}, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers/42", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers/99", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_CreateCustomer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			createFn: func(ctx context.Context, customer *Customer, storeID *int) error {
				customer.ID = 100
				return nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"New Customer","phone":"0812345","email":"new@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty name", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		body := `{"name":"","phone":"0812345","email":"e@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "name is required")
	})

	t.Run("invalid email", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		body := `{"name":"Test","phone":"0812345","email":"not-an-email"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email")
	})

	t.Run("invalid phone", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		body := `{"name":"Test","phone":"12","email":"e@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid phone")
	})

	t.Run("empty phone", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		body := `{"name":"Test","phone":"","email":"e@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCustomerService{
			createFn: func(ctx context.Context, customer *Customer, storeID *int) error {
				return errors.New("duplicate")
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"Test","phone":"0812345","email":"e@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_UpdateCustomer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Old"}, nil
			},
			updateFn: func(ctx context.Context, customer *Customer, id int, storeID *int) error {
				return nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"Updated","email":"u@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/abc", strings.NewReader(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("customer not found", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/99", strings.NewReader(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("walk-in customer forbidden", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Walk-in", IsWalkIn: true}, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "walk-in")
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1}, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid name in update", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Old"}, nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"   ","phone":"0812345"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid email in update", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Old"}, nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"email":"bad"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid phone in update", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Old"}, nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"phone":"x"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestMockHandler_DeleteCustomer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "ToDelete"}, nil
			},
			deleteFn: func(ctx context.Context, id int, storeID *int) error { return nil },
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/99", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("walk-in forbidden", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Walk-in", IsWalkIn: true}, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("delete error", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "X"}, nil
			},
			deleteFn: func(ctx context.Context, id int, storeID *int) error { return errors.New("fk constraint") },
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_BulkUpdateCustomerStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			bulkStatusFn: func(ctx context.Context, ids []int, isActive bool, storeID *int) error {
				assert.Equal(t, []int{1, 2}, ids)
				assert.False(t, isActive)
				return nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"ids":[1,2],"is_active":false}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ids", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		body := `{"ids":[],"is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "no customer IDs")
	})

	t.Run("too many ids", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		ids := strings.Repeat("1,", 200) + "1"
		body := `{"ids":[` + ids + `],"is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "too many")
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/status", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCustomerService{
			bulkStatusFn: func(ctx context.Context, ids []int, isActive bool, storeID *int) error {
				return errors.New("db error")
			},
		}
		r := setupMockRouter(svc)
		body := `{"ids":[1],"is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_BulkDeleteCustomers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCustomerService{
			bulkDeleteFn: func(ctx context.Context, ids []int, storeID *int) error {
				assert.Equal(t, []int{1, 2}, ids)
				return nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"ids":[1,2]}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/delete", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ids", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/delete", strings.NewReader(`{"ids":[]}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("too many ids", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		ids := strings.Repeat("1,", 200) + "1"
		body := `{"ids":[` + ids + `]}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/delete", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/delete", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCustomerService{
			bulkDeleteFn: func(ctx context.Context, ids []int, storeID *int) error {
				return errors.New("db error")
			},
		}
		r := setupMockRouter(svc)
		body := `{"ids":[1]}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers/bulk/delete", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetCustomers_ExtraPaths(t *testing.T) {
	t.Run("isActive camelCase param", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				require.NotNil(t, isActive)
				assert.False(t, *isActive)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?isActive=false", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("default limit when no param", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, 50, limit)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid limit falls back to default", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, 50, limit)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?limit=abc", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("search param passed through", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, "john", search)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?search=john", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative offset clamped to zero", func(t *testing.T) {
		svc := &mockCustomerService{
			getAllFn: func(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
				assert.Equal(t, 0, offset)
				return []Customer{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/customers?offset=-5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMockHandler_CreateCustomer_ExtraPaths(t *testing.T) {
	t.Run("name too long", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		longName := strings.Repeat("a", 201)
		body := `{"name":"` + longName + `","phone":"0812345678","email":"a@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "200 characters")
	})

	t.Run("empty email", func(t *testing.T) {
		r := setupMockRouter(&mockCustomerService{})
		body := `{"name":"Test","phone":"0812345678","email":""}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "email is required")
	})
}

func TestMockHandler_UpdateCustomer_ServiceError(t *testing.T) {
	t.Run("update service error", func(t *testing.T) {
		svc := &mockCustomerService{
			getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
				return &Customer{ID: 1, Name: "Existing"}, nil
			},
			updateFn: func(ctx context.Context, customer *Customer, id int, storeID *int) error {
				return errors.New("update failed")
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"Updated"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_DeleteCustomer_WalkIn(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return &Customer{ID: 1, Name: "Walk-In", IsWalkIn: true}, nil
		},
	}
	r := setupMockRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "walk-in")
}

func TestMockHandler_DeleteCustomer_ServiceError(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return &Customer{ID: 1, Name: "Real"}, nil
		},
		deleteFn: func(ctx context.Context, id int, storeID *int) error {
			return errors.New("fk constraint")
		},
	}
	r := setupMockRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMockHandler_DeleteCustomer_NotFound(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return nil, errors.New("not found")
		},
	}
	r := setupMockRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/999", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}
