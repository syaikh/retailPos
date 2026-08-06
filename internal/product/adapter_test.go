package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/platform/importexport/schema"
	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/uom"
)

func TestProductAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "products", a.ModuleName())
}

func TestProductAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil, nil, nil, nil)
	assert.NotNil(t, a)
	assert.Equal(t, "products", a.ModuleName())
}

func TestProductAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestProductAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    ProductImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":          1,
				"SKU":           "SKU-001",
				"Name":          "Widget Pro",
				"Barcode":       "8999999999999",
				"Category":      "Electronics",
				"Brand":         "Samsung",
				"Price":         "50000",
				"Cost":          "35000",
				"Stock":         "100",
				"Status":        "active",
				"UnitOfMeasure": "PCS",
				"WeightGrams":   "250",
				"Description":   "A premium widget",
			},
			want: ProductImportRow{
				Row:           1,
				SKU:           "SKU-001",
				Name:          "Widget Pro",
				Barcode:       "8999999999999",
				Category:      "Electronics",
				Brand:         "Samsung",
				Price:         50000,
				Cost:          35000,
				Stock:         100,
				Status:        "active",
				UnitOfMeasure: "PCS",
				WeightGrams:   250,
				Description:   "A premium widget",
			},
		},
		{
			name: "SKU is required - empty string",
			row: map[string]interface{}{
				"_row":  2,
				"SKU":   "",
				"Name":  "NoSKU",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "SKU key missing",
			row: map[string]interface{}{
				"_row":  3,
				"Name":  "NoSKU",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "Name is required - empty string",
			row: map[string]interface{}{
				"_row":  4,
				"SKU":   "SKU-004",
				"Name":  "",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "Name key missing",
			row: map[string]interface{}{
				"_row":  5,
				"SKU":   "SKU-005",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "Status defaults to active",
			row: map[string]interface{}{
				"_row":  6,
				"SKU":   "SKU-006",
				"Name":  "DefaultStatus",
				"Price": "2000",
			},
			want: ProductImportRow{
				Row:    6,
				SKU:    "SKU-006",
				Name:   "DefaultStatus",
				Price:  2000,
				Status: "active",
			},
		},
		{
			name: "Status explicit inactive",
			row: map[string]interface{}{
				"_row":   7,
				"SKU":    "SKU-007",
				"Name":   "InactiveItem",
				"Price":  "3000",
				"Status": "inactive",
			},
			want: ProductImportRow{
				Row:    7,
				SKU:    "SKU-007",
				Name:   "InactiveItem",
				Price:  3000,
				Status: "inactive",
			},
		},
		{
			name: "zero values for missing numeric fields",
			row: map[string]interface{}{
				"_row":  8,
				"SKU":   "SKU-008",
				"Name":  "Minimal",
				"Price": "0",
			},
			want: ProductImportRow{
				Row:    8,
				SKU:    "SKU-008",
				Name:   "Minimal",
				Status: "active",
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"SKU":   "SKU-009",
				"Name":  "NoRowProduct",
				"Price": "5000",
			},
			want: ProductImportRow{
				Row:    0,
				SKU:    "SKU-009",
				Name:   "NoRowProduct",
				Price:  5000,
				Status: "active",
			},
		},
		{
			name: "float string price gets truncated to int",
			row: map[string]interface{}{
				"_row":  10,
				"SKU":   "SKU-010",
				"Name":  "FloatPrice",
				"Price": "99.99",
			},
			want: ProductImportRow{
				Row:    10,
				SKU:    "SKU-010",
				Name:   "FloatPrice",
				Status: "active",
			},
		},
		{
			name: "non-numeric price becomes 0",
			row: map[string]interface{}{
				"_row":  11,
				"SKU":   "SKU-011",
				"Name":  "BadPrice",
				"Price": "not-a-number",
			},
			want: ProductImportRow{
				Row:    11,
				SKU:    "SKU-011",
				Name:   "BadPrice",
				Status: "active",
			},
		},
	}

	ctx := context.Background()
	a := &adapter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.MapToEntity(ctx, Schema, tt.row)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			importRow, ok := got.(ProductImportRow)
			require.True(t, ok, "expected ProductImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestProductAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}

func TestNilStr(t *testing.T) {
	tests := []struct {
		name string
		s    *string
		want string
	}{
		{"nil pointer", nil, ""},
		{"non-nil", ptrString("hello"), "hello"},
		{"empty string", ptrString(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nilStr(tt.s))
		})
	}
}

func TestNilInt(t *testing.T) {
	tests := []struct {
		name string
		i    *int
		want interface{}
	}{
		{"nil pointer", nil, nil},
		{"non-nil", ptrInt(42), 42},
		{"zero", ptrInt(0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nilInt(tt.i))
		})
	}
}

func ptrString(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func seedProductReferences(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `INSERT INTO categories (name, slug, description, is_active) VALUES ('TestCat', 'testcat', 'Test category', true) ON CONFLICT (name) DO NOTHING`)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `INSERT INTO brands (name, description, is_active) VALUES ('TestBrand', 'Test brand', true) ON CONFLICT (name) DO NOTHING`)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `INSERT INTO units_of_measure (code, name, description, is_active) VALUES ('PCSP', 'Pieces', 'Pieces', true) ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)
}

func TestProductAdapter_Insert_Success(t *testing.T) {
	ctx := context.Background()
	seedProductReferences(t, ctx)

	repo := NewRepository(dbPool)
	catRepo := category.NewRepository(dbPool)
	brandRepo := brand.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)
	catRef, brandRef, uomRef := newTestRefAdapters(catRepo, brandRepo, uomRepo)
	adapter := NewAdapter(repo, catRef, brandRef, uomRef)
	ra := adapter.Repository()

	rows := []interface{}{
		ProductImportRow{
			Row: 1, SKU: "TEST-SKU-001", Name: "Test Product 1",
			Category: "TestCat", Brand: "TestBrand", Price: 10000, Cost: 7000, Stock: 50,
			Status: "active", UnitOfMeasure: "PCSP",
		},
	}

	count, err := ra.Insert(ctx, rows)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	product, err := repo.GetProductBySKU(ctx, "TEST-SKU-001", nil)
	require.NoError(t, err)
	assert.Equal(t, "Test Product 1", product.Name)
	assert.Equal(t, 10000, product.Price)
}

func TestProductAdapter_Insert_MissingCategory(t *testing.T) {
	ctx := context.Background()
	seedProductReferences(t, ctx)

	repo := NewRepository(dbPool)
	catRepo := category.NewRepository(dbPool)
	brandRepo := brand.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)
	catRef, brandRef, uomRef := newTestRefAdapters(catRepo, brandRepo, uomRepo)
	adapter := NewAdapter(repo, catRef, brandRef, uomRef)
	ra := adapter.Repository()

	rows := []interface{}{
		ProductImportRow{
			Row: 1, SKU: "TEST-SKU-002", Name: "Test Product 2",
			Category: "NonExistentCat", Brand: "TestBrand", Price: 10000, Cost: 7000, Stock: 50,
			Status: "active", UnitOfMeasure: "PCSP",
		},
	}

	_, err := ra.Insert(ctx, rows)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category")
}

func TestProductAdapter_Update_Success(t *testing.T) {
	ctx := context.Background()
	seedProductReferences(t, ctx)

	repo := NewRepository(dbPool)
	catRepo := category.NewRepository(dbPool)
	brandRepo := brand.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)
	catRef, brandRef, uomRef := newTestRefAdapters(catRepo, brandRepo, uomRepo)
	adapter := NewAdapter(repo, catRef, brandRef, uomRef)
	ra := adapter.Repository()

	product := &Product{
		SKU: "TEST-SKU-UPDATE", Name: "Before Update", Price: 10000, Cost: 5000, Stock: 20, Status: "active",
	}
	err := repo.CreateProduct(ctx, product)
	require.NoError(t, err)

	rows := []interface{}{
		ProductImportRow{
			Row: 1, SKU: "TEST-SKU-UPDATE", Name: "After Update",
			Category: "TestCat", Brand: "TestBrand", Price: 15000, Cost: 8000, Stock: 30,
			Status: "active", UnitOfMeasure: "PCSP",
		},
	}

	count, err := ra.Update(ctx, rows)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	got, err := repo.GetProductBySKU(ctx, "TEST-SKU-UPDATE", nil)
	require.NoError(t, err)
	assert.Equal(t, "After Update", got.Name)
	assert.Equal(t, 15000, got.Price)
}

func TestProductAdapter_ExportData_Success(t *testing.T) {
	ctx := context.Background()
	seedProductReferences(t, ctx)

	repo := NewRepository(dbPool)
	catRepo := category.NewRepository(dbPool)
	brandRepo := brand.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)
	catRef, brandRef, uomRef := newTestRefAdapters(catRepo, brandRepo, uomRepo)
	adapter := NewAdapter(repo, catRef, brandRef, uomRef)
	ra := adapter.Repository()

	product := &Product{
		SKU: "TEST-SKU-EXPORT", Name: "Export Product", Price: 20000, Cost: 15000, Stock: 100, Status: "active",
	}
	err := repo.CreateProduct(ctx, product)
	require.NoError(t, err)

	data, err := ra.ExportData(ctx, importexportshared.ModuleSchema{})
	require.NoError(t, err)
	require.NotEmpty(t, data)

	found := false
	for _, row := range data {
		if row["SKU"] == "TEST-SKU-EXPORT" {
			found = true
			assert.Equal(t, "Export Product", row["Name"])
			assert.Equal(t, 20000, row["Price"])
		}
	}
	assert.True(t, found, "exported product not found")
}

type testCategoryRefAdapter struct {
	repo *category.Repository
}

func (a *testCategoryRefAdapter) GetCategoryIDByName(ctx context.Context, name string) (int, error) {
	return a.repo.GetCategoryIDByName(ctx, name)
}

func (a *testCategoryRefAdapter) GetCategoryIDsByNames(ctx context.Context, names []string) (map[string]int, error) {
	return a.repo.GetCategoryIDsByNames(ctx, names)
}

func (a *testCategoryRefAdapter) GetAllCategoriesForExport(ctx context.Context) ([]CategoryRef, error) {
	cats, err := a.repo.GetAllCategoriesForExport(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryRef, len(cats))
	for i, c := range cats {
		out[i] = CategoryRef{ID: c.ID, Name: c.Name}
	}
	return out, nil
}

type testBrandRefAdapter struct {
	repo *brand.Repository
}

func (a *testBrandRefAdapter) GetIDByName(ctx context.Context, name string) (int, error) {
	return a.repo.GetIDByName(ctx, name)
}

func (a *testBrandRefAdapter) GetIDsByNames(ctx context.Context, names []string) (map[string]int, error) {
	return a.repo.GetIDsByNames(ctx, names)
}

func (a *testBrandRefAdapter) GetAllForExport(ctx context.Context) ([]BrandRef, error) {
	brands, err := a.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BrandRef, len(brands))
	for i, b := range brands {
		out[i] = BrandRef{ID: b.ID, Name: b.Name}
	}
	return out, nil
}

type testUOMRefAdapter struct {
	repo *uom.Repository
}

func (a *testUOMRefAdapter) GetIDByCode(ctx context.Context, code string) (int, error) {
	return a.repo.GetIDByCode(ctx, code)
}

func (a *testUOMRefAdapter) GetIDsByCodes(ctx context.Context, codes []string) (map[string]int, error) {
	return a.repo.GetIDsByCodes(ctx, codes)
}

func (a *testUOMRefAdapter) GetAllForExport(ctx context.Context) ([]UOMRef, error) {
	uoms, err := a.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UOMRef, len(uoms))
	for i, u := range uoms {
		out[i] = UOMRef{ID: u.ID, Code: u.Code}
	}
	return out, nil
}

func newTestRefAdapters(catRepo *category.Repository, brandRepo *brand.Repository, uomRepo *uom.Repository) (CategoryRefRepo, BrandRefRepo, UOMRefRepo) {
	return &testCategoryRefAdapter{repo: catRepo}, &testBrandRefAdapter{repo: brandRepo}, &testUOMRefAdapter{repo: uomRepo}
}

func TestProductAdapter_LoadReferences_Success(t *testing.T) {
	ctx := context.Background()
	seedProductReferences(t, ctx)

	repo := NewRepository(dbPool)
	catRepo := category.NewRepository(dbPool)
	brandRepo := brand.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)
	catRef, brandRef, uomRef := newTestRefAdapters(catRepo, brandRepo, uomRepo)
	adapter := NewAdapter(repo, catRef, brandRef, uomRef)

	schema := schema.ModuleSchema{
		ModuleName:    "products",
		SchemaVersion: "1.0",
		References: []importexportshared.ReferenceDef{
			{ReferenceModule: "categories"},
			{ReferenceModule: "brands"},
			{ReferenceModule: "uoms"},
		},
	}

	refs, err := adapter.Repository().LoadReferences(ctx, schema)
	require.NoError(t, err)
	require.NotNil(t, refs)
	assert.Contains(t, refs, "categories")
	assert.Contains(t, refs, "brands")
	assert.Contains(t, refs, "uoms")
	assert.NotEmpty(t, refs["categories"])
	assert.NotEmpty(t, refs["brands"])
	assert.NotEmpty(t, refs["uoms"])
}
