package product

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCategoryRepo struct {
	getByIDFn  func(ctx context.Context, name string) (int, error)
	getByIDsFn func(ctx context.Context, names []string) (map[string]int, error)
}

func (m *testCategoryRepo) GetCategoryIDByName(ctx context.Context, name string) (int, error) {
	return m.getByIDFn(ctx, name)
}
func (m *testCategoryRepo) GetCategoryIDsByNames(ctx context.Context, names []string) (map[string]int, error) {
	return m.getByIDsFn(ctx, names)
}

func TestService_GetAllProducts_IsActiveToStatus(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := &service{repo: repo}
	ctx := context.Background()

	t.Run("isActive=true converts to status=active", func(t *testing.T) {
		active := true
		_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &active, nil, "", nil, nil)
		assert.NoError(t, err)
	})

	t.Run("isActive=false converts to status=inactive", func(t *testing.T) {
		inactive := false
		_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &inactive, nil, "", nil, nil)
		assert.NoError(t, err)
	})

	t.Run("status set, isActive ignored", func(t *testing.T) {
		active := true
		_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &active, nil, "custom", nil, nil)
		assert.NoError(t, err)
	})
}

func TestService_GetAllProducts_SingleCategory(t *testing.T) {
	repo := NewRepository(dbPool)
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			assert.Equal(t, "Electronics", name)
			return 42, nil
		},
	}
	svc := &service{repo: repo, categoryRepo: catRepo}
	ctx := context.Background()

	_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics", nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
}

func TestService_GetAllProducts_SingleCategoryError(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			return 0, errors.New("category not found")
		},
	}
	svc := &service{categoryRepo: catRepo}
	ctx := context.Background()

	products, total, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "Nonexistent", nil, nil, nil, "", nil, nil)
	assert.Error(t, err)
	assert.Nil(t, products)
	assert.Equal(t, 0, total)
}

func TestService_GetAllProducts_MultiCategory(t *testing.T) {
	repo := NewRepository(dbPool)
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			return 0, errors.New("should not be called for multi-category")
		},
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			assert.Equal(t, []string{"Electronics", "Books"}, names)
			return map[string]int{"Electronics": 1, "Books": 2}, nil
		},
	}
	svc := &service{repo: repo, categoryRepo: catRepo}
	ctx := context.Background()

	_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics,Books", nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
}

func TestService_GetAllProducts_MultiCategoryError(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			return nil, errors.New("db error")
		},
	}
	svc := &service{categoryRepo: catRepo}
	ctx := context.Background()

	products, total, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "A,B", nil, nil, nil, "", nil, nil)
	assert.Error(t, err)
	assert.Nil(t, products)
	assert.Equal(t, 0, total)
}

func TestService_GetAllProducts_EmptyCategoryFiltered(t *testing.T) {
	repo := NewRepository(dbPool)
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			t.Fatal("should not be called with empty trimmed name")
			return 0, nil
		},
	}
	svc := &service{repo: repo, categoryRepo: catRepo}
	ctx := context.Background()

	_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", " , , ", nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
}

func TestService_GetAllProducts_MultiCategoryPartialMatch(t *testing.T) {
	repo := NewRepository(dbPool)
	catRepo := &testCategoryRepo{
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			return map[string]int{"Electronics": 1}, nil
		},
	}
	svc := &service{repo: repo, categoryRepo: catRepo}
	ctx := context.Background()

	_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics,Unknown", nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
}

func TestService_GetAllProducts_NoCategory(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := &service{repo: repo}
	ctx := context.Background()

	_, _, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
}

func TestService_ResolveCategoryID(t *testing.T) {
	t.Run("resolves category by name when CategoryID is nil", func(t *testing.T) {
		catRepo := &testCategoryRepo{
			getByIDFn: func(ctx context.Context, name string) (int, error) {
				assert.Equal(t, "Electronics", name)
				return 42, nil
			},
		}
		svc := &service{categoryRepo: catRepo}
		ctx := context.Background()

		p := &Product{CategoryName: strPtr("Electronics")}
		err := svc.resolveCategoryID(ctx, p)
		assert.NoError(t, err)
		assert.NotNil(t, p.CategoryID)
		assert.Equal(t, 42, *p.CategoryID)
	})

	t.Run("skips resolution when CategoryID is already set", func(t *testing.T) {
		catRepo := &testCategoryRepo{
			getByIDFn: func(ctx context.Context, name string) (int, error) {
				t.Fatal("should not be called when CategoryID is set")
				return 0, nil
			},
		}
		svc := &service{categoryRepo: catRepo}
		ctx := context.Background()

		existingID := 10
		p := &Product{CategoryID: &existingID, CategoryName: strPtr("Electronics")}
		err := svc.resolveCategoryID(ctx, p)
		assert.NoError(t, err)
		assert.Equal(t, 10, *p.CategoryID)
	})

	t.Run("skips resolution when CategoryName is nil", func(t *testing.T) {
		catRepo := &testCategoryRepo{
			getByIDFn: func(ctx context.Context, name string) (int, error) {
				t.Fatal("should not be called when CategoryName is nil")
				return 0, nil
			},
		}
		svc := &service{categoryRepo: catRepo}
		ctx := context.Background()

		p := &Product{}
		err := svc.resolveCategoryID(ctx, p)
		assert.NoError(t, err)
		assert.Nil(t, p.CategoryID)
	})

	t.Run("skips resolution when CategoryName is empty", func(t *testing.T) {
		catRepo := &testCategoryRepo{
			getByIDFn: func(ctx context.Context, name string) (int, error) {
				t.Fatal("should not be called when CategoryName is empty")
				return 0, nil
			},
		}
		svc := &service{categoryRepo: catRepo}
		ctx := context.Background()

		p := &Product{CategoryName: strPtr("")}
		err := svc.resolveCategoryID(ctx, p)
		assert.NoError(t, err)
		assert.Nil(t, p.CategoryID)
	})

	t.Run("returns error when category not found", func(t *testing.T) {
		catRepo := &testCategoryRepo{
			getByIDFn: func(ctx context.Context, name string) (int, error) {
				return 0, errors.New("category not found")
			},
		}
		svc := &service{categoryRepo: catRepo}
		ctx := context.Background()

		p := &Product{CategoryName: strPtr("Nonexistent")}
		err := svc.resolveCategoryID(ctx, p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "category \"Nonexistent\" not found")
		assert.Nil(t, p.CategoryID)
	})

	t.Run("preserves error chain with %w", func(t *testing.T) {
		innerErr := errors.New("connection refused")
		catRepo := &testCategoryRepo{
			getByIDFn: func(ctx context.Context, name string) (int, error) {
				return 0, innerErr
			},
		}
		svc := &service{categoryRepo: catRepo}
		ctx := context.Background()

		p := &Product{CategoryName: strPtr("Test")}
		err := svc.resolveCategoryID(ctx, p)
		assert.Error(t, err)
		assert.ErrorIs(t, err, innerErr)
	})
}
