package product

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	importexportshared "retail-pos-system/internal/shared/importexport"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/uom"
)

type CategoryRefRepo interface {
	GetCategoryIDByName(ctx context.Context, name string) (int, error)
	GetAllCategoriesForExport(ctx context.Context) ([]category.Category, error)
}

type BrandRefRepo interface {
	GetIDByName(ctx context.Context, name string) (int, error)
	GetAllForExport(ctx context.Context) ([]brand.Brand, error)
}

type UOMRefRepo interface {
	GetIDByCode(ctx context.Context, code string) (int, error)
	GetAllForExport(ctx context.Context) ([]uom.UnitOfMeasure, error)
}

type adapter struct {
	repo         *Repository
	categoryRepo CategoryRefRepo
	brandRepo    BrandRefRepo
	uomRepo      UOMRefRepo
}

func NewAdapter(repo *Repository, categoryRepo CategoryRefRepo, brandRepo BrandRefRepo, uomRepo UOMRefRepo) importexportshared.Adapter {
	return &adapter{
		repo:         repo,
		categoryRepo: categoryRepo,
		brandRepo:    brandRepo,
		uomRepo:      uomRepo,
	}
}

func (a *adapter) ModuleName() string {
	return "products"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ importexportshared.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	sku, _ := row["SKU"].(string)
	if sku == "" {
		return nil, fmt.Errorf("SKU is required")
	}
	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	rowNum, _ := row["_row"].(int)
	barcode, _ := row["Barcode"].(string)
	categoryName, _ := row["Category"].(string)
	brandName, _ := row["Brand"].(string)
	price, _ := strconv.Atoi(fmt.Sprintf("%v", row["Price"]))
	cost, _ := strconv.Atoi(fmt.Sprintf("%v", row["Cost"]))
	stock, _ := strconv.Atoi(fmt.Sprintf("%v", row["Stock"]))
	status := "active"
	if v, ok := row["Status"]; ok {
		status = fmt.Sprintf("%v", v)
	}
	unitOfMeasure, _ := row["UnitOfMeasure"].(string)
	weightGrams, _ := strconv.Atoi(fmt.Sprintf("%v", row["WeightGrams"]))
	description, _ := row["Description"].(string)
	storeID, _ := row["_store_id"].(int)

	return ProductImportRow{
		Row:           rowNum,
		SKU:           sku,
		Name:          name,
		Barcode:       barcode,
		Category:      categoryName,
		Brand:         brandName,
		Price:         price,
		Cost:          cost,
		Stock:         stock,
		Status:        status,
		UnitOfMeasure: unitOfMeasure,
		WeightGrams:   weightGrams,
		Description:   description,
		StoreID:       storeID,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &productRepoAdapter{
		repo:         a.repo,
		categoryRepo: a.categoryRepo,
		brandRepo:    a.brandRepo,
		uomRepo:      a.uomRepo,
	}
}

type productRepoAdapter struct {
	repo         *Repository
	categoryRepo CategoryRefRepo
	brandRepo    BrandRefRepo
	uomRepo      UOMRefRepo
}

func (r *productRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	payloads := make([]ProductImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(ProductImportRow)
		payload, err := r.resolveReferences(ctx, row)
		if err != nil {
			return len(payloads), err
		}
		payloads = append(payloads, *payload)
	}
	return r.repo.BulkInsertProducts(ctx, payloads)
}

func (r *productRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	payloads := make([]ProductImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(ProductImportRow)
		payload, err := r.resolveReferences(ctx, row)
		if err != nil {
			return len(payloads), err
		}
		payloads = append(payloads, *payload)
	}
	return r.repo.BulkUpdateProducts(ctx, payloads)
}

func (r *productRepoAdapter) resolveReferences(ctx context.Context, row ProductImportRow) (*ProductImportPayload, error) {
	status := strings.ToLower(row.Status)
	if status == "" {
		status = "active"
	}

	var categoryID *int
	if row.Category != "" {
		id, err := r.categoryRepo.GetCategoryIDByName(ctx, row.Category)
		if err != nil {
			return nil, fmt.Errorf("category %q: %w", row.Category, err)
		}
		categoryID = &id
	}

	var brandID *int
	if row.Brand != "" {
		id, err := r.brandRepo.GetIDByName(ctx, row.Brand)
		if err != nil {
			return nil, fmt.Errorf("brand %q: %w", row.Brand, err)
		}
		brandID = &id
	}

	var uomID *int
	if row.UnitOfMeasure != "" {
		id, err := r.uomRepo.GetIDByCode(ctx, row.UnitOfMeasure)
		if err != nil {
			return nil, fmt.Errorf("unit of measure %q: %w", row.UnitOfMeasure, err)
		}
		uomID = &id
	}

	var storeID *int
	if row.StoreID > 0 {
		storeID = &row.StoreID
	}

	return &ProductImportPayload{
		SKU:             row.SKU,
		Name:            row.Name,
		Barcode:         strPtr(row.Barcode),
		CategoryID:      categoryID,
		BrandID:         brandID,
		Price:           row.Price,
		Cost:            row.Cost,
		Stock:           row.Stock,
		Status:          status,
		UnitOfMeasureID: uomID,
		WeightGrams:     intPtr(row.WeightGrams),
		Description:     strPtr(row.Description),
		StoreID:         storeID,
	}, nil
}

func (r *productRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	products, err := r.repo.GetAllProductsForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(products))
	for i, p := range products {
		result[i] = map[string]interface{}{
			"SKU":           p.SKU,
			"Name":          p.Name,
			"Barcode":       nilStr(p.Barcode),
			"Category":      nilStr(p.CategoryName),
			"Brand":         nilStr(p.BrandName),
			"Price":         p.Price,
			"Cost":          p.Cost,
			"Stock":         p.Stock,
			"Status":        p.Status,
			"UnitOfMeasure": nilStr(p.UnitOfMeasure),
			"WeightGrams":   nilInt(p.WeightGrams),
			"Description":   nilStr(p.Description),
		}
	}
	return result, nil
}

func (r *productRepoAdapter) LoadReferences(ctx context.Context, s importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	refs := make(map[string][]importexportshared.ReferenceItem)

	for _, ref := range s.References {
		switch ref.ReferenceModule {
		case "categories":
			categories, err := r.categoryRepo.GetAllCategoriesForExport(ctx)
			if err != nil {
				return nil, fmt.Errorf("load categories: %w", err)
			}
			for _, c := range categories {
				refs["categories"] = append(refs["categories"], importexportshared.ReferenceItem{
					Key: c.Name, Value: c.ID,
				})
			}

		case "brands":
			brands, err := r.brandRepo.GetAllForExport(ctx)
			if err != nil {
				return nil, fmt.Errorf("load brands: %w", err)
			}
			for _, b := range brands {
				refs["brands"] = append(refs["brands"], importexportshared.ReferenceItem{
					Key: b.Name, Value: b.ID,
				})
			}

		case "uoms":
			units, err := r.uomRepo.GetAllForExport(ctx)
			if err != nil {
				return nil, fmt.Errorf("load uoms: %w", err)
			}
			for _, u := range units {
				refs["uoms"] = append(refs["uoms"], importexportshared.ReferenceItem{
					Key: u.Code, Value: u.ID,
				})
			}
		}
	}

	return refs, nil
}

func nilStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}
