package uom

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

type mockUOMService struct {
	getAllFn          func(ctx context.Context) ([]UnitOfMeasure, error)
	getAllPaginatedFn func(ctx context.Context, limit, offset int, search string) ([]UnitOfMeasure, int, error)
	getByIDFn         func(ctx context.Context, id int) (*UnitOfMeasure, error)
	createFn          func(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error)
	updateFn          func(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error)
	deleteFn          func(ctx context.Context, id int) error
}

func (m *mockUOMService) GetAll(ctx context.Context) ([]UnitOfMeasure, error) {
	return m.getAllFn(ctx)
}
func (m *mockUOMService) GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]UnitOfMeasure, int, error) {
	return m.getAllPaginatedFn(ctx, limit, offset, search)
}
func (m *mockUOMService) GetByID(ctx context.Context, id int) (*UnitOfMeasure, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockUOMService) Create(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error) {
	return m.createFn(ctx, req)
}
func (m *mockUOMService) Update(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
	return m.updateFn(ctx, id, req)
}
func (m *mockUOMService) Delete(ctx context.Context, id int) error {
	return m.deleteFn(ctx, id)
}

var _ UOMService = (*mockUOMService)(nil)

func setupMockUOMRouter(svc UOMService) *gin.Engine {
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

func TestMockUOMHandler_ListUnitsOfMeasure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUOMService{
			getAllPaginatedFn: func(ctx context.Context, limit, offset int, search string) ([]UnitOfMeasure, int, error) {
				return []UnitOfMeasure{{ID: 1, Name: "PCS"}}, 1, nil
			},
		}
		r := setupMockUOMRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/units-of-measure", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUOMService{
			getAllPaginatedFn: func(ctx context.Context, limit, offset int, search string) ([]UnitOfMeasure, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		r := setupMockUOMRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/units-of-measure", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("nil becomes empty", func(t *testing.T) {
		svc := &mockUOMService{
			getAllPaginatedFn: func(ctx context.Context, limit, offset int, search string) ([]UnitOfMeasure, int, error) {
				return nil, 0, nil
			},
		}
		r := setupMockUOMRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/units-of-measure", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})
}

func TestMockUOMHandler_CreateUnitOfMeasure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUOMService{
			createFn: func(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error) {
				return &UnitOfMeasure{ID: 1, Code: req.Code, Name: req.Name}, nil
			},
		}
		r := setupMockUOMRouter(svc)
		body := `{"code":"PCS","name":"Pieces","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/units-of-measure", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockUOMRouter(&mockUOMService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/units-of-measure", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUOMService{
			createFn: func(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error) {
				return nil, errors.New("duplicate")
			},
		}
		r := setupMockUOMRouter(svc)
		body := `{"code":"DUP","name":"Duplicate","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/units-of-measure", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockUOMHandler_UpdateUnitOfMeasure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUOMService{
			updateFn: func(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
				return &UnitOfMeasure{ID: id, Code: req.Code, Name: req.Name}, nil
			},
		}
		r := setupMockUOMRouter(svc)
		body := `{"code":"KG","name":"Kilogram","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/units-of-measure/5", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUOMRouter(&mockUOMService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/units-of-measure/abc", strings.NewReader(`{"code":"X","name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUOMService{
			updateFn: func(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockUOMRouter(svc)
		body := `{"code":"KG","name":"Kilogram","is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/units-of-measure/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockUOMHandler_DeleteUnitOfMeasure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUOMService{
			deleteFn: func(ctx context.Context, id int) error { return nil },
		}
		r := setupMockUOMRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/units-of-measure/5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUOMRouter(&mockUOMService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/units-of-measure/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUOMService{
			deleteFn: func(ctx context.Context, id int) error { return errors.New("fail") },
		}
		r := setupMockUOMRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/units-of-measure/1", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockUOMHandler_UpdateUnitOfMeasure_InvalidJSON(t *testing.T) {
	svc := &mockUOMService{}
	r := setupMockUOMRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/units-of-measure/1", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestMockUOMHandler_DeleteUnitOfMeasure_ServiceError(t *testing.T) {
	svc := &mockUOMService{
		deleteFn: func(ctx context.Context, id int) error {
			return errors.New("foreign key constraint violation")
		},
	}
	r := setupMockUOMRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/units-of-measure/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestMockUOMHandler_UpdateUnitOfMeasure_SuccessIsActive(t *testing.T) {
	svc := &mockUOMService{
		updateFn: func(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
			assert.Equal(t, 5, id)
			assert.Equal(t, "L", req.Code)
			assert.Equal(t, "Liter", req.Name)
			assert.NotNil(t, req.IsActive)
			assert.True(t, *req.IsActive)
			return &UnitOfMeasure{ID: id, Code: req.Code, Name: req.Name, IsActive: *req.IsActive}, nil
		},
	}
	r := setupMockUOMRouter(svc)
	body := `{"code":"L","name":"Liter","is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/units-of-measure/5", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
