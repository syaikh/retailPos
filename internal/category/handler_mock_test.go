package category

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
)

type mockCategoryService struct {
	listFn      func(ctx context.Context) ([]Category, error)
	getByIDFn   func(ctx context.Context, id int) (*Category, error)
	getAllFn    func(ctx context.Context, limit, offset int, search string) ([]Category, int, error)
	createFn    func(ctx context.Context, req *CategoryCreateRequest) (*Category, error)
	updateFn    func(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error)
	deleteFn    func(ctx context.Context, id int) error
}

func (m *mockCategoryService) ListCategories(ctx context.Context) ([]Category, error) {
	return m.listFn(ctx)
}
func (m *mockCategoryService) GetCategoryByID(ctx context.Context, id int) (*Category, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockCategoryService) GetAllCategories(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
	return m.getAllFn(ctx, limit, offset, search)
}
func (m *mockCategoryService) CreateCategory(ctx context.Context, req *CategoryCreateRequest) (*Category, error) {
	return m.createFn(ctx, req)
}
func (m *mockCategoryService) UpdateCategory(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error) {
	return m.updateFn(ctx, id, req)
}
func (m *mockCategoryService) DeleteCategory(ctx context.Context, id int) error {
	return m.deleteFn(ctx, id)
}

var _ CategoryService = (*mockCategoryService)(nil)

func setupMockRouter(svc CategoryService) *gin.Engine {
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

func TestMockHandler_ListCategories(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCategoryService{
			listFn: func(ctx context.Context) ([]Category, error) {
				return []Category{{ID: 1, Name: "Electronics"}}, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nil becomes empty array", func(t *testing.T) {
		svc := &mockCategoryService{
			listFn: func(ctx context.Context) ([]Category, error) {
				return nil, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("error", func(t *testing.T) {
		svc := &mockCategoryService{
			listFn: func(ctx context.Context) ([]Category, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_ListCategoriesManagement(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCategoryService{
			getAllFn: func(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
				assert.Equal(t, 50, limit)
				assert.Equal(t, 0, offset)
				return []Category{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories/manage", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("limit clamped", func(t *testing.T) {
		svc := &mockCategoryService{
			getAllFn: func(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
				assert.Equal(t, 200, limit)
				return []Category{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories/manage?limit=999", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative offset clamped", func(t *testing.T) {
		svc := &mockCategoryService{
			getAllFn: func(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
				assert.Equal(t, 0, offset)
				return []Category{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories/manage?offset=-10", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("search param", func(t *testing.T) {
		svc := &mockCategoryService{
			getAllFn: func(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
				assert.Equal(t, "food", search)
				return []Category{}, 0, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories/manage?search=food", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCategoryService{
			getAllFn: func(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/categories/manage", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_CreateCategory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCategoryService{
			createFn: func(ctx context.Context, req *CategoryCreateRequest) (*Category, error) {
				assert.Equal(t, "New Category", req.Name)
				return &Category{ID: 10, Name: "New Category", Slug: "new-category"}, nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"New Category","description":"test"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/categories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(10), data["id"])
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockRouter(&mockCategoryService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/categories", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCategoryService{
			createFn: func(ctx context.Context, req *CategoryCreateRequest) (*Category, error) {
				return nil, errors.New("duplicate slug")
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"Duplicate"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/categories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_UpdateCategory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCategoryService{
			getByIDFn: func(ctx context.Context, id int) (*Category, error) {
				return &Category{ID: 1, Name: "Old"}, nil
			},
			updateFn: func(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error) {
				assert.Equal(t, 1, id)
				return &Category{ID: 1, Name: req.Name}, nil
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"Updated","description":"new desc"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/categories/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockRouter(&mockCategoryService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/categories/abc", strings.NewReader(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := &mockCategoryService{
			getByIDFn: func(ctx context.Context, id int) (*Category, error) {
				return &Category{ID: 1}, nil
			},
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/categories/1", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCategoryService{
			getByIDFn: func(ctx context.Context, id int) (*Category, error) {
				return &Category{ID: 1}, nil
			},
			updateFn: func(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockRouter(svc)
		body := `{"name":"Updated"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/categories/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_DeleteCategory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockCategoryService{
			getByIDFn: func(ctx context.Context, id int) (*Category, error) {
				return &Category{ID: 1, Name: "ToDelete"}, nil
			},
			deleteFn: func(ctx context.Context, id int) error { return nil },
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/categories/1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockRouter(&mockCategoryService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/categories/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockCategoryService{
			getByIDFn: func(ctx context.Context, id int) (*Category, error) {
				return &Category{ID: 1}, nil
			},
			deleteFn: func(ctx context.Context, id int) error { return errors.New("fk constraint") },
		}
		r := setupMockRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/categories/1", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_UpdateCategory_InvalidJSON(t *testing.T) {
	svc := &mockCategoryService{
		getByIDFn: func(ctx context.Context, id int) (*Category, error) {
			return &Category{ID: 1}, nil
		},
	}
	r := setupMockRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/categories/1", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestMockHandler_DeleteCategory_ServiceError(t *testing.T) {
	svc := &mockCategoryService{
		getByIDFn: func(ctx context.Context, id int) (*Category, error) {
			return &Category{ID: 1, Name: "TestCat"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			return errors.New("foreign key constraint violation")
		},
	}
	r := setupMockRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/categories/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}
