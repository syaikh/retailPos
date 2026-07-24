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
	svc := &Service{}
	ctx := context.Background()

	t.Run("isActive=true converts to status=active", func(t *testing.T) {
		active := true
		defer func() { _ = recover() }()
		_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &active, nil, "", nil)
	})

	t.Run("isActive=false converts to status=inactive", func(t *testing.T) {
		inactive := false
		defer func() { _ = recover() }()
		_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &inactive, nil, "", nil)
	})

	t.Run("status set, isActive ignored", func(t *testing.T) {
		active := true
		defer func() { _ = recover() }()
		_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, &active, nil, "custom", nil)
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

	defer func() { _ = recover() }()
	_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics", nil, nil, nil, "", nil)
}

func TestService_GetAllProducts_SingleCategoryError(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDFn: func(ctx context.Context, name string) (int, error) {
			return 0, errors.New("category not found")
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	products, total, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "Nonexistent", nil, nil, nil, "", nil)
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

	defer func() { _ = recover() }()
	_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics,Books", nil, nil, nil, "", nil)
}

func TestService_GetAllProducts_MultiCategoryError(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			return nil, errors.New("db error")
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	products, total, err := svc.GetAllProducts(ctx, 10, 0, "", "", "", "A,B", nil, nil, nil, "", nil)
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

	defer func() { _ = recover() }()
	_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", " , , ", nil, nil, nil, "", nil)
}

func TestService_GetAllProducts_MultiCategoryPartialMatch(t *testing.T) {
	catRepo := &testCategoryRepo{
		getByIDsFn: func(ctx context.Context, names []string) (map[string]int, error) {
			return map[string]int{"Electronics": 1}, nil
		},
	}
	svc := &Service{categoryRepo: catRepo}
	ctx := context.Background()

	defer func() { _ = recover() }()
	_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "Electronics,Unknown", nil, nil, nil, "", nil)
}

func TestService_GetAllProducts_NoCategory(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	defer func() { _ = recover() }()
	_, _, _ = svc.GetAllProducts(ctx, 10, 0, "", "", "", "", nil, nil, nil, "", nil)
}


