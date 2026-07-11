package product

import (
	"context"
	"strings"

	"retail-pos-system/internal/eventbus"
)

type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}

type CategoryRepo interface {
	GetCategoryIDByName(ctx context.Context, name string) (int, error)
}

type BrandRepo interface {
	GetIDByName(ctx context.Context, name string) (int, error)
}

type UOMRepo interface {
	GetIDByCode(ctx context.Context, code string) (int, error)
}

type Service struct {
	repo        *Repository
	categoryRepo CategoryRepo
	brandRepo   BrandRepo
	uomRepo     UOMRepo
	eventBus    EventBus
}

func NewService(repo *Repository, categoryRepo CategoryRepo, brandRepo BrandRepo, uomRepo UOMRepo, eventBus EventBus) *Service {
	return &Service{repo: repo, categoryRepo: categoryRepo, brandRepo: brandRepo, uomRepo: uomRepo, eventBus: eventBus}
}

func (s *Service) GetProductByID(ctx context.Context, id, storeID int) (*Product, error) {
	return s.repo.GetProductByID(ctx, id, ptr(storeID))
}

func (s *Service) GetProductBySKU(ctx context.Context, sku string, storeID int) (*Product, error) {
	return s.repo.GetProductBySKU(ctx, sku, ptr(storeID))
}

func (s *Service) GetAllProducts(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
	if status == "" && isActive != nil {
		if *isActive { status = "active" } else { status = "inactive" }
	}
	var categoryIDs []int
	if category != "" {
		names := strings.Split(category, ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			id, err := s.categoryRepo.GetCategoryIDByName(ctx, name)
			if err != nil {
				return nil, 0, err
			}
			categoryIDs = append(categoryIDs, id)
		}
	}
	return s.repo.GetAllProducts(ctx, limit, offset, search, categoryIDs, sortBy, sortDir, maxStock, storeID, status)
}

func (s *Service) CreateProduct(ctx context.Context, product *Product) error {
	return s.repo.CreateProduct(ctx, product)
}

func (s *Service) UpdateProduct(ctx context.Context, product *Product) error {
	old, err := s.repo.GetProductByID(ctx, product.ID, product.StoreID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateProduct(ctx, product, product.StoreID); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "product.updated", eventbus.UpdatePayload{Old: old, New: product})
}

func (s *Service) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	return s.repo.DeleteProduct(ctx, id, storeID)
}

func (s *Service) BulkUpdateProductStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	status := "active"
	if !isActive { status = "inactive" }
	return s.repo.BulkUpdateProductStatus(ctx, ids, status, storeID)
}

func (s *Service) GetNextSKU(ctx context.Context) (string, error) { return s.repo.GetNextSKU(ctx) }
func (s *Service) GetTaxClassByID(ctx context.Context, id int) (*TaxClass, error) { return s.repo.GetTaxClassByID(ctx, id) }
func (s *Service) GetAllTaxClasses(ctx context.Context) ([]TaxClass, error) { return s.repo.GetAllTaxClasses(ctx) }
func (s *Service) GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error) { return s.repo.GetWarehouseByID(ctx, id) }
func (s *Service) GetAllWarehouses(ctx context.Context) ([]Warehouse, error) { return s.repo.GetAllWarehouses(ctx, nil) }



func (s *Service) resolveCategoryID(ctx context.Context, name string) (int, error) {
	return s.categoryRepo.GetCategoryIDByName(ctx, name)
}

func (s *Service) resolveBrandID(ctx context.Context, name string) (int, error) {
	return s.brandRepo.GetIDByName(ctx, name)
}

func (s *Service) resolveUnitOfMeasureID(ctx context.Context, code string) (int, error) {
	return s.uomRepo.GetIDByCode(ctx, code)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func ptr(i int) *int {
	if i == 0 { return nil }
	return &i
}
