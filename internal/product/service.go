package product

import (
	"context"
	"strings"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
)

type CategoryRepo interface {
	GetCategoryIDByName(ctx context.Context, name string) (int, error)
	GetCategoryIDsByNames(ctx context.Context, names []string) (map[string]int, error)
}

type BrandRepo interface {
	GetIDByName(ctx context.Context, name string) (int, error)
}

type UOMRepo interface {
	GetIDByCode(ctx context.Context, code string) (int, error)
}

type service struct {
	repo         *Repository
	categoryRepo CategoryRepo
	brandRepo    BrandRepo
	uomRepo      UOMRepo
	eventBus     shared.EventBus
}

func NewService(repo *Repository, categoryRepo CategoryRepo, brandRepo BrandRepo, uomRepo UOMRepo, eventBus shared.EventBus) Service {
	return &service{repo: repo, categoryRepo: categoryRepo, brandRepo: brandRepo, uomRepo: uomRepo, eventBus: eventBus}
}

func (s *service) GetProductByID(ctx context.Context, id, storeID int) (*Product, error) {
	return s.repo.GetProductByID(ctx, id, ptr(storeID))
}

func (s *service) GetProductsByIDs(ctx context.Context, ids []int) ([]Product, error) {
	return s.repo.GetProductsByIDs(ctx, ids)
}

func (s *service) GetProductBySKU(ctx context.Context, sku string, storeID int) (*Product, error) {
	return s.repo.GetProductBySKU(ctx, sku, ptr(storeID))
}

func (s *service) GetAllProducts(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
	if status == "" && isActive != nil {
		if *isActive {
			status = "active"
		} else {
			status = "inactive"
		}
	}
	var categoryIDs []int
	if category != "" {
		names := strings.Split(category, ",")
		unique := make([]string, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			unique = append(unique, name)
		}
		if len(unique) == 1 {
			id, err := s.categoryRepo.GetCategoryIDByName(ctx, unique[0])
			if err != nil {
				return nil, 0, err
			}
			categoryIDs = append(categoryIDs, id)
		} else if len(unique) > 1 {
			ids, err := s.categoryRepo.GetCategoryIDsByNames(ctx, unique)
			if err != nil {
				return nil, 0, err
			}
			for _, name := range unique {
				if id, ok := ids[name]; ok {
					categoryIDs = append(categoryIDs, id)
				}
			}
		}
	}
	return s.repo.GetAllProducts(ctx, limit, offset, search, categoryIDs, sortBy, sortDir, maxStock, storeID, status, supplierID)
}

func (s *service) CreateProduct(ctx context.Context, product *Product) error {
	return s.repo.CreateProduct(ctx, product)
}

func (s *service) UpdateProduct(ctx context.Context, product *Product) error {
	// Guard against updating a non-existent product. The fetched row is not
	// published anymore (the DTO carries only the new state), but the lookup
	// preserves the prior behavior of failing instead of silently succeeding.
	if _, err := s.repo.GetProductByID(ctx, product.ID, product.StoreID); err != nil {
		return err
	}
	if err := s.repo.UpdateProduct(ctx, product, product.StoreID); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, events.TopicProductUpdated, &events.ProductUpdated{
		ID:      product.ID,
		SKU:     product.SKU,
		Stock:   product.Stock,
		Price:   product.Price,
		StoreID: product.StoreID,
	})
}

func (s *service) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	return s.repo.DeleteProduct(ctx, id, storeID)
}

func (s *service) BulkUpdateProductStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	status := "active"
	if !isActive {
		status = "inactive"
	}
	return s.repo.BulkUpdateProductStatus(ctx, ids, status, storeID)
}

func (s *service) GetNextSKU(ctx context.Context) (string, error) { return s.repo.GetNextSKU(ctx) }
func (s *service) GetTaxClassByID(ctx context.Context, id int) (*TaxClass, error) {
	return s.repo.GetTaxClassByID(ctx, id)
}
func (s *service) GetAllTaxClasses(ctx context.Context) ([]TaxClass, error) {
	return s.repo.GetAllTaxClasses(ctx)
}
func (s *service) GetActiveProductOptions(ctx context.Context) ([]Option, error) {
	return s.repo.GetActiveProductOptions(ctx)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}
