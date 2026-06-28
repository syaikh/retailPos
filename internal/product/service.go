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

type Service struct {
	repo        *Repository
	categoryRepo CategoryRepo
	eventBus    EventBus
}

func NewService(repo *Repository, categoryRepo CategoryRepo, eventBus EventBus) *Service {
	return &Service{repo: repo, categoryRepo: categoryRepo, eventBus: eventBus}
}

func (s *Service) GetProductByID(ctx context.Context, id, storeID int) (*Product, error) {
	return s.repo.GetProductByID(ctx, id, ptr(storeID))
}

func (s *Service) GetProductBySKU(ctx context.Context, sku string, storeID int) (*Product, error) {
	return s.repo.GetProductBySKU(ctx, sku, ptr(storeID))
}

func (s *Service) GetAllProducts(ctx context.Context, limit, offset int, search, sortBy, sortDir, category, brand string, storeID *int, isActive *bool, minPrice, maxPrice *float64) ([]Product, int, error) {
	status := ""
	if isActive != nil {
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
				return []Product{}, 0, nil
			}
			categoryIDs = append(categoryIDs, id)
		}
	}
	return s.repo.GetAllProducts(ctx, limit, offset, search, categoryIDs, sortBy, sortDir, nil, storeID, status)
}

func (s *Service) CreateProduct(ctx context.Context, product *Product) error {
	if err := s.repo.CreateProduct(ctx, product); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "product.created", product)
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

func (s *Service) DeleteProduct(ctx context.Context, id int) error {
	if err := s.repo.DeleteProduct(ctx, id, nil); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "product.deleted", id)
}

func (s *Service) BulkUpdateProductStatus(ctx context.Context, ids []int, isActive bool) error {
	status := "active"
	if !isActive { status = "inactive" }
	return s.repo.BulkUpdateProductStatus(ctx, ids, status, nil)
}

func (s *Service) GetNextSKU(ctx context.Context) (string, error) { return s.repo.GetNextSKU(ctx) }
func (s *Service) GetBrandByID(ctx context.Context, id int) (*Brand, error) { return s.repo.GetBrandByID(ctx, id) }
func (s *Service) GetAllBrands(ctx context.Context) ([]Brand, error) { return s.repo.GetAllBrands(ctx) }
func (s *Service) CreateBrand(ctx context.Context, brand *Brand) error {
	if err := s.repo.CreateBrand(ctx, brand); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "brand.created", brand)
}
func (s *Service) UpdateBrand(ctx context.Context, brand *Brand) error {
	old, err := s.repo.GetBrandByID(ctx, brand.ID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateBrand(ctx, brand); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "brand.updated", eventbus.UpdatePayload{Old: old, New: brand})
}
func (s *Service) DeleteBrand(ctx context.Context, id int) error {
	if err := s.repo.DeleteBrand(ctx, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "brand.deleted", id)
}
func (s *Service) GetTaxClassByID(ctx context.Context, id int) (*TaxClass, error) { return s.repo.GetTaxClassByID(ctx, id) }
func (s *Service) GetAllTaxClasses(ctx context.Context) ([]TaxClass, error) { return s.repo.GetAllTaxClasses(ctx) }
func (s *Service) GetUnitOfMeasureByID(ctx context.Context, id int) (*UnitOfMeasure, error) { return s.repo.GetUnitOfMeasureByID(ctx, id) }
func (s *Service) GetAllUnitsOfMeasure(ctx context.Context) ([]UnitOfMeasure, error) { return s.repo.GetAllUnitsOfMeasure(ctx) }
func (s *Service) CreateUnitOfMeasure(ctx context.Context, uom *UnitOfMeasure) error {
	if err := s.repo.CreateUnitOfMeasure(ctx, uom); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "uom.created", uom)
}
func (s *Service) UpdateUnitOfMeasure(ctx context.Context, uom *UnitOfMeasure) error {
	old, err := s.repo.GetUnitOfMeasureByID(ctx, uom.ID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUnitOfMeasure(ctx, uom); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "uom.updated", eventbus.UpdatePayload{Old: old, New: uom})
}
func (s *Service) DeleteUnitOfMeasure(ctx context.Context, id int) error {
	if err := s.repo.DeleteUnitOfMeasure(ctx, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "uom.deleted", id)
}
func (s *Service) GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error) { return s.repo.GetWarehouseByID(ctx, id) }
func (s *Service) GetAllWarehouses(ctx context.Context) ([]Warehouse, error) { return s.repo.GetAllWarehouses(ctx, nil) }

func ptr(i int) *int {
	if i == 0 { return nil }
	return &i
}
