package product

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testBrandRepo struct {
	getIDByNameFn func(ctx context.Context, name string) (int, error)
}

func (m *testBrandRepo) GetIDByName(ctx context.Context, name string) (int, error) {
	return m.getIDByNameFn(ctx, name)
}

func TestService_ResolveBrandID(t *testing.T) {
	t.Run("resolves brand by name when BrandID is nil", func(t *testing.T) {
		brandRepo := &testBrandRepo{
			getIDByNameFn: func(ctx context.Context, name string) (int, error) {
				assert.Equal(t, "Indofood", name)
				return 55, nil
			},
		}
		svc := &service{brandRepo: brandRepo}
		ctx := context.Background()

		p := &Product{BrandName: strPtr("Indofood")}
		err := svc.resolveBrandID(ctx, p)
		assert.NoError(t, err)
		assert.NotNil(t, p.BrandID)
		assert.Equal(t, 55, *p.BrandID)
	})

	t.Run("skips resolution when BrandID is already set", func(t *testing.T) {
		brandRepo := &testBrandRepo{
			getIDByNameFn: func(ctx context.Context, name string) (int, error) {
				t.Fatal("should not be called when BrandID is set")
				return 0, nil
			},
		}
		svc := &service{brandRepo: brandRepo}
		ctx := context.Background()

		existingID := 10
		p := &Product{BrandID: &existingID, BrandName: strPtr("Indofood")}
		err := svc.resolveBrandID(ctx, p)
		assert.NoError(t, err)
		assert.Equal(t, 10, *p.BrandID)
	})

	t.Run("skips resolution when BrandName is nil", func(t *testing.T) {
		brandRepo := &testBrandRepo{
			getIDByNameFn: func(ctx context.Context, name string) (int, error) {
				t.Fatal("should not be called when BrandName is nil")
				return 0, nil
			},
		}
		svc := &service{brandRepo: brandRepo}
		ctx := context.Background()

		p := &Product{}
		err := svc.resolveBrandID(ctx, p)
		assert.NoError(t, err)
		assert.Nil(t, p.BrandID)
	})

	t.Run("skips resolution when BrandName is empty", func(t *testing.T) {
		brandRepo := &testBrandRepo{
			getIDByNameFn: func(ctx context.Context, name string) (int, error) {
				t.Fatal("should not be called when BrandName is empty")
				return 0, nil
			},
		}
		svc := &service{brandRepo: brandRepo}
		ctx := context.Background()

		p := &Product{BrandName: strPtr("")}
		err := svc.resolveBrandID(ctx, p)
		assert.NoError(t, err)
		assert.Nil(t, p.BrandID)
	})

	t.Run("returns error when brand not found", func(t *testing.T) {
		brandRepo := &testBrandRepo{
			getIDByNameFn: func(ctx context.Context, name string) (int, error) {
				return 0, errors.New("brand not found")
			},
		}
		svc := &service{brandRepo: brandRepo}
		ctx := context.Background()

		p := &Product{BrandName: strPtr("Nonexistent")}
		err := svc.resolveBrandID(ctx, p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "brand \"Nonexistent\" not found")
		assert.Nil(t, p.BrandID)
	})

	t.Run("preserves error chain with %w", func(t *testing.T) {
		innerErr := errors.New("connection refused")
		brandRepo := &testBrandRepo{
			getIDByNameFn: func(ctx context.Context, name string) (int, error) {
				return 0, innerErr
			},
		}
		svc := &service{brandRepo: brandRepo}
		ctx := context.Background()

		p := &Product{BrandName: strPtr("Test")}
		err := svc.resolveBrandID(ctx, p)
		assert.Error(t, err)
		assert.ErrorIs(t, err, innerErr)
	})
}
