package brand

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockBrandService struct {
	getAllFn          func(ctx context.Context) ([]Brand, error)
	getAllPaginatedFn func(ctx context.Context, limit, offset int, search string) ([]Brand, int, error)
	getByIDFn         func(ctx context.Context, id int) (*Brand, error)
	createFn          func(ctx context.Context, req *BrandCreateRequest) (*Brand, error)
	updateFn          func(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error)
	deleteFn          func(ctx context.Context, id int) error
}

func (m *mockBrandService) GetAll(ctx context.Context) ([]Brand, error) {
	return m.getAllFn(ctx)
}
func (m *mockBrandService) GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
	return m.getAllPaginatedFn(ctx, limit, offset, search)
}
func (m *mockBrandService) GetByID(ctx context.Context, id int) (*Brand, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockBrandService) Create(ctx context.Context, req *BrandCreateRequest) (*Brand, error) {
	return m.createFn(ctx, req)
}
func (m *mockBrandService) Update(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
	return m.updateFn(ctx, id, req)
}
func (m *mockBrandService) Delete(ctx context.Context, id int) error {
	return m.deleteFn(ctx, id)
}

var _ BrandService = (*mockBrandService)(nil)

func setupMockBrandRouter(svc BrandService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPublicRoutes(r.Group("/"))
	return r
}

func TestMockBrandHandler_ListBrands(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockBrandService{
			getAllPaginatedFn: func(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
				return []Brand{{ID: 1, Name: "Nike"}}, 1, nil
			},
		}
		r := setupMockBrandRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/brands", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockBrandService{
			getAllPaginatedFn: func(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		r := setupMockBrandRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/brands", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("nil becomes empty", func(t *testing.T) {
		svc := &mockBrandService{
			getAllPaginatedFn: func(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
				return nil, 0, nil
			},
		}
		r := setupMockBrandRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/brands", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})
}

func TestMockBrandHandler_CreateBrand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockBrandService{
			createFn: func(ctx context.Context, req *BrandCreateRequest) (*Brand, error) {
				return &Brand{ID: 1, Name: req.Name}, nil
			},
		}
		r := setupMockBrandRouter(svc)
		body := `{"name":"New Brand","description":"test","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/brands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockBrandRouter(&mockBrandService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/brands", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockBrandService{
			createFn: func(ctx context.Context, req *BrandCreateRequest) (*Brand, error) {
				return nil, errors.New("duplicate")
			},
		}
		r := setupMockBrandRouter(svc)
		body := `{"name":"Existing","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/brands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockBrandHandler_UpdateBrand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockBrandService{
			updateFn: func(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
				return &Brand{ID: id, Name: req.Name}, nil
			},
		}
		r := setupMockBrandRouter(svc)
		body := `{"name":"Updated","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/brands/5", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockBrandRouter(&mockBrandService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/brands/abc", strings.NewReader(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockBrandService{
			updateFn: func(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockBrandRouter(svc)
		body := `{"name":"Updated","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/brands/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockBrandHandler_DeleteBrand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockBrandService{
			deleteFn: func(ctx context.Context, id int) error { return nil },
		}
		r := setupMockBrandRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/brands/5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockBrandRouter(&mockBrandService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/brands/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockBrandService{
			deleteFn: func(ctx context.Context, id int) error { return errors.New("fail") },
		}
		r := setupMockBrandRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/brands/1", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockBrandHandler_UpdateBrand_InvalidJSON(t *testing.T) {
	svc := &mockBrandService{}
	r := setupMockBrandRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/brands/1", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestMockBrandHandler_DeleteBrand_ServiceError(t *testing.T) {
	svc := &mockBrandService{
		deleteFn: func(ctx context.Context, id int) error {
			return errors.New("foreign key constraint violation")
		},
	}
	r := setupMockBrandRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/brands/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}
