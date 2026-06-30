package product

import (
	"context"
	"fmt"
	"strings"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/importutil"
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

func (s *Service) GetAllProducts(ctx context.Context, limit, offset int, search, sortBy, sortDir, category, brand string, storeID *int, isActive *bool, minPrice, maxPrice *float64, maxStock *int, status string) ([]Product, int, error) {
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
				return []Product{}, 0, nil
			}
			categoryIDs = append(categoryIDs, id)
		}
	}
	return s.repo.GetAllProducts(ctx, limit, offset, search, categoryIDs, sortBy, sortDir, maxStock, storeID, status)
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
func (s *Service) GetTaxClassByID(ctx context.Context, id int) (*TaxClass, error) { return s.repo.GetTaxClassByID(ctx, id) }
func (s *Service) GetAllTaxClasses(ctx context.Context) ([]TaxClass, error) { return s.repo.GetAllTaxClasses(ctx) }
func (s *Service) GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error) { return s.repo.GetWarehouseByID(ctx, id) }
func (s *Service) GetAllWarehouses(ctx context.Context) ([]Warehouse, error) { return s.repo.GetAllWarehouses(ctx, nil) }

func (s *Service) GetAllProductsForExport(ctx context.Context) ([]Product, error) {
	return s.repo.GetAllProductsForExport(ctx)
}

func (s *Service) ImportProducts(ctx context.Context, records []ProductImportRow) importutil.ImportResult {
	result := importutil.ImportResult{Errors: []string{}}

	for _, rec := range records {
		if rec.SKU == "" {
			result.AddError(rec.Row, "SKU is required")
			continue
		}
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}
		if rec.Price <= 0 {
			result.AddError(rec.Row, "Price must be greater than 0")
			continue
		}
		if rec.Stock < 0 {
			result.AddError(rec.Row, "Stock must not be negative")
			continue
		}
		status := rec.Status
		if status == "" {
			status = "active"
		}
		validStatuses := map[string]bool{"active": true, "inactive": true, "draft": true, "archived": true}
		if !validStatuses[status] {
			result.AddError(rec.Row, "Status must be one of: active, inactive, draft, archived")
			continue
		}

		var categoryID *int
		if rec.Category != "" {
			id, err := s.resolveCategoryID(ctx, rec.Category)
			if err != nil {
				result.AddError(rec.Row, fmt.Sprintf("category error: %v", err))
				continue
			}
			categoryID = &id
		}

		var brandID *int
		if rec.Brand != "" {
			id, err := s.resolveBrandID(ctx, rec.Brand)
			if err != nil {
				result.AddError(rec.Row, fmt.Sprintf("brand error: %v", err))
				continue
			}
			brandID = &id
		}

		var uomID *int
		if rec.UnitOfMeasure != "" {
			id, err := s.resolveUnitOfMeasureID(ctx, rec.UnitOfMeasure)
			if err != nil {
				result.AddError(rec.Row, fmt.Sprintf("unit of measure error: %v", err))
				continue
			}
			uomID = &id
		}

		inserted, err := s.repo.BulkUpsertProduct(ctx, ProductImportPayload{
			SKU:           rec.SKU,
			Name:          rec.Name,
			Barcode:       strPtr(rec.Barcode),
			CategoryID:    categoryID,
			BrandID:       brandID,
			Price:         rec.Price,
			Cost:          rec.Cost,
			Stock:         rec.Stock,
			Status:        status,
			UnitOfMeasureID: uomID,
			WeightGrams:   intPtr(rec.WeightGrams),
			Description:   strPtr(rec.Description),
		})

		if err != nil {
			result.AddError(rec.Row, fmt.Sprintf("failed to upsert: %v", err))
			continue
		}

		if inserted {
			result.Inserted++
		} else {
			result.Updated++
		}
	}

	return result
}

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
