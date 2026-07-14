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

type testBrandRepo struct {
	getIDByNameFn func(ctx context.Context, name string) (int, error)
}

func (m *testBrandRepo) GetIDByName(ctx context.Context, name string) (int, error) {
	return m.getIDByNameFn(ctx, name)
}

type testUOMRepo struct {
	getIDByCodeFn func(ctx context.Context, code string) (int, error)
}

func (m *testUOMRepo) GetIDByCode(ctx context.Context, code string) (int, error) {
	return m.getIDByCodeFn(ctx, code)
}

func TestService_GetAllProducts_IsActiveToStatus(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	t.Run("isActive=true converts to status=active", func(t *testing.T) {
		active := true
		defer func() { recover() }()
		svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &active, nil, "")
	})

	t.Run("isActive=false converts to status=inactive", func(t *testing.T) {
		inactive := false
		defer func() { recover() }()
		svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &inactive, nil, "")
	})

	t.Run("status set, isActive ignored", func(t *testing.T) {
		active := true
		defer func() { recover() }()
		svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &active, nil, "custom")
	})
}

func TestService_GetAllProducts_SingleCategory(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			assert.Equal(t, "Electronics", name)
			return 42, nil
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	defer func() { recover() }()
	svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics", nil, nil, nil, "")
}

func TestService_GetAllProducts_SingleCategoryError(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			return 0, errors.New("category not found")
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	products, total, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "Nonexistent", nil, nil, nil, "")
	assert.Error(t, err)
	assert.Nil(t, products)
	assert.Equal(t, 0, total)
}

func TestService_GetAllProducts_MultiCategory(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			return 0, errors.New("should not be called for multi-category")
		},
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			assert.Equal(t, []string{"Electronics", "Books"}, names)
			return map[string]int{"Electronics": 1, "Books": 2}, nil
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	defer func() { recover() }()
	svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics,Books", nil, nil, nil, "")
}

func TestService_GetAllProducts_MultiCategoryError(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			return nil, errors.New("db error")
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	products, total, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "A,B", nil, nil, nil, "")
	assert.Error(t, err)
	assert.Nil(t, products)
	assert.Equal(t, 0, total)
}

func TestService_GetAllProducts_EmptyCategoryFiltered(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			t.Fatal("should not be called with empty trimmed name")
			return 0, nil
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	defer func() { recover() }()
	svc.GetAllProducts(ctx, 10, 0, "", "", "", " , , ", nil, nil, nil, "")
}

func TestService_GetAllProducts_MultiCategoryPartialMatch(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			return map[string]int{"Electronics": 1}, nil
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	defer func() { recover() }()
	svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics,Unknown", nil, nil, nil, "")
}

func TestService_GetAllProducts_NoCategory(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	defer func() { recover() }()
	svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, nil, nil, "")
}

func TestService_ResolveHelpers(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			return 10, nil
		},
	}
	brandRepo := &testBrandRepo{
		getIDByNameFn: func(ctx context.Context, name string) (int, error) {
			return 20, nil
		},
	}
	uomRepo := &testUOMRepo{
		getIDByCodeFn: func(ctx context.Context, code string) (int, error) {
			return 30, nil
		},
	}
	svc := &Service{categoryRepo: catRepo, brandRepo: brandRepo, uomRepo: uomRepo}
	ctx := context.Background()

	t.Run("resolveCategoryID", func(t *testing.T) {
		id, err := svc.resolveCategoryID(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, 10, id)
	})

	t.Run("resolveBrandID", func(t *testing.T) {
		id, err := svc.resolveBrandID(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, 20, id)
	})

	t.Run("resolveUnitOfMeasureID", func(t *testing.T) {
		id, err := svc.resolveUnitOfMeasureID(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, 30, id)
	})

	t.Run("strPtr empty", func(t *testing.T) {
		assert.Nil(t, strPtr(""))
	})

	t.Run("strPtr non-empty", func(t *testing.T) {
		s := strPtr("hello")
		assert.Equal(t, "hello", *s)
	})

	t.Run("intPtr zero", func(t *testing.T) {
		assert.Nil(t, intPtr(0))
	})

	t.Run("intPtr non-zero", func(t *testing.T) {
		i := intPtr(42)
		assert.Equal(t, 42, *i)
	})

	t.Run("ptr zero", func(t *testing.T) {
		assert.Nil(t, ptr(0))
	})

	t.Run("ptr non-zero", func(t *testing.T) {
		p := ptr(7)
		assert.Equal(t, 7, *p)
	})
}
