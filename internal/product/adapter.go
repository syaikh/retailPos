package product

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"
)

// CategoryRef is the consumer-side view of a category needed for import/export
// reference resolution. Ports return these instead of the category module's
// entity so product never imports brand/category/uom directly.
type CategoryRef struct {
	ID   int
	Name string
}

type BrandRef struct {
	ID   int
	Name string
}

type UOMRef struct {
	ID   int
	Code string
}

type CategoryRefRepo interface {
	GetCategoryIDByName(ctx context.Context, name string) (int, error)
	GetCategoryIDsByNames(ctx context.Context, names []string) (map[string]int, error)
	GetAllCategoriesForExport(ctx context.Context) ([]CategoryRef, error)
}

type BrandRefRepo interface {
	GetIDByName(ctx context.Context, name string) (int, error)
	GetIDsByNames(ctx context.Context, names []string) (map[string]int, error)
	GetAllForExport(ctx context.Context) ([]BrandRef, error)
}

type UOMRefRepo interface {
	GetIDByCode(ctx context.Context, code string) (int, error)
	GetIDsByCodes(ctx context.Context, codes []string) (map[string]int, error)
	GetAllForExport(ctx context.Context) ([]UOMRef, error)
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
		return nil, fmt.Errorf("name is required")
	}
	rowNum, _ := row["_row"].(int)
	barcode, _ := row["Barcode"].(string)
	categoryName, _ := row["Category"].(string)
	brandName, _ := row["Brand"].(string)
	price := floatToInt(row["Price"])
	cost := floatToInt(row["Cost"])
	stock := floatToInt(row["Stock"])
	status := "active"
	if v, ok := row["Status"]; ok {
		status = fmt.Sprintf("%v", v)
	}
	unitOfMeasure, _ := row["UnitOfMeasure"].(string)
	weightGrams, _ := strconv.Atoi(fmt.Sprintf("%v", row["WeightGrams"]))
	description, _ := row["Description"].(string)
	storeID, _ := row["_store_id"].(int)

	return ImportRow{
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
	catMap, brandMap, uomMap, err := r.batchResolveAll(ctx, entities)
	if err != nil {
		return 0, err
	}
	payloads := make([]ImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(ImportRow)
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		if err != nil {
			return len(payloads), err
		}
		payloads = append(payloads, *payload)
	}
	return r.repo.BulkInsertProducts(ctx, payloads)
}

func (r *productRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	catMap, brandMap, uomMap, err := r.batchResolveAll(ctx, entities)
	if err != nil {
		return 0, err
	}
	payloads := make([]ImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(ImportRow)
		payload, err := r.resolveReferences(row, catMap, brandMap, uomMap)
		if err != nil {
			return len(payloads), err
		}
		payloads = append(payloads, *payload)
	}
	return r.repo.BulkUpdateProducts(ctx, payloads)
}

func (r *productRepoAdapter) batchResolveAll(ctx context.Context, entities []interface{}) (catMap, brandMap, uomMap map[string]int, err error) {
	catNames := make(map[string]bool)
	brandNames := make(map[string]bool)
	uomCodes := make(map[string]bool)
	for _, e := range entities {
		row := e.(ImportRow)
		if row.Category != "" {
			catNames[row.Category] = true
		}
		if row.Brand != "" {
			brandNames[row.Brand] = true
		}
		if row.UnitOfMeasure != "" {
			uomCodes[row.UnitOfMeasure] = true
		}
	}
	uniqueCatNames := keysOf(catNames)
	uniqueBrandNames := keysOf(brandNames)
	uniqueUomCodes := keysOf(uomCodes)
	catMap, err = r.categoryRepo.GetCategoryIDsByNames(ctx, uniqueCatNames)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch resolve categories: %w", err)
	}
	brandMap, err = r.brandRepo.GetIDsByNames(ctx, uniqueBrandNames)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch resolve brands: %w", err)
	}
	uomMap, err = r.uomRepo.GetIDsByCodes(ctx, uniqueUomCodes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch resolve UoMs: %w", err)
	}
	return catMap, brandMap, uomMap, nil
}

func keysOf(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

func (r *productRepoAdapter) resolveReferences(row ImportRow, catMap, brandMap, uomMap map[string]int) (*ImportPayload, error) {
	status := strings.ToLower(row.Status)
	if status == "" {
		status = "active"
	}

	var categoryID *int
	if row.Category != "" {
		id, ok := catMap[row.Category]
		if !ok {
			return nil, fmt.Errorf("category %q not found", row.Category)
		}
		categoryID = &id
	}

	var brandID *int
	if row.Brand != "" {
		id, ok := brandMap[row.Brand]
		if !ok {
			return nil, fmt.Errorf("brand %q not found", row.Brand)
		}
		brandID = &id
	}

	var uomID *int
	if row.UnitOfMeasure != "" {
		id, ok := uomMap[row.UnitOfMeasure]
		if !ok {
			return nil, fmt.Errorf("unit of measure %q not found", row.UnitOfMeasure)
		}
		uomID = &id
	}

	var storeID *int
	if row.StoreID > 0 {
		storeID = &row.StoreID
	}

	if strings.TrimSpace(row.Name) == "" {
		return nil, fmt.Errorf("name is required at row %d", row.Row)
	}
	if row.Price < 0 {
		return nil, fmt.Errorf("price must not be negative at row %d", row.Row)
	}
	if row.Cost < 0 {
		return nil, fmt.Errorf("cost must not be negative at row %d", row.Row)
	}
	if row.Stock < 0 {
		return nil, fmt.Errorf("stock must not be negative at row %d", row.Row)
	}

	return &ImportPayload{
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
		WeightGrams:     ptr(row.WeightGrams),
		Description:     strPtr(row.Description),
		StoreID:         storeID,
	}, nil
}

func (r *productRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	storeID := middleware.StoreIDFromContext(ctx)
	products, err := r.repo.GetAllProductsForExport(ctx, storeID)
	if err != nil {
		return nil, err
	}
	viewCost := ownership.CanAccessAll(middleware.PermissionsFromContext(ctx), permissions.ProductCostView)
	result := make([]map[string]interface{}, len(products))
	for i, p := range products {
		row := map[string]interface{}{
			"SKU":           p.SKU,
			"Name":          p.Name,
			"Barcode":       nilStr(p.Barcode),
			"Category":      nilStr(p.CategoryName),
			"Brand":         nilStr(p.BrandName),
			"Price":         p.Price,
			"Stock":         p.Stock,
			"Status":        p.Status,
			"UnitOfMeasure": nilStr(p.UnitOfMeasure),
			"WeightGrams":   nilInt(p.WeightGrams),
			"Description":   nilStr(p.Description),
		}
		if viewCost {
			row["Cost"] = p.Cost
		}
		result[i] = row
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

func floatToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(math.Round(val))
	case int:
		return val
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}
