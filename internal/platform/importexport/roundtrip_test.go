package importexport_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/customer"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/export"
	importer "retail-pos-system/internal/platform/importexport/import"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"
	"retail-pos-system/internal/platform/importexport/validation"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/uom"
)

var (
	dbPool      *pgxpool.Pool
	schemaReg   *schema.Registry
	adapterReg  *importexport.AdapterRegistry
	importEng   *importer.Engine
	templateEng *template.Engine
	exportEng   *export.Engine
	progEng     *progress.Engine
	runID       string
)

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../../database/migrations"); err != nil {
		os.Exit(1)
	}

	_ = shared.TruncateTestData(pool)

	schemaReg = schema.NewRegistry()
	_ = schemaReg.Register(brand.Schema)
	_ = schemaReg.Register(category.Schema)
	_ = schemaReg.Register(uom.Schema)
	_ = schemaReg.Register(customer.Schema)
	_ = schemaReg.Register(product.Schema)

	adapterReg = importexport.NewAdapterRegistry()
	_ = adapterReg.Register(brand.NewAdapter(brand.NewRepository(dbPool)))
	_ = adapterReg.Register(category.NewAdapter(category.NewRepository(dbPool)))
	_ = adapterReg.Register(uom.NewAdapter(uom.NewRepository(dbPool)))
	_ = adapterReg.Register(customer.NewAdapter(customer.NewRepository(dbPool)))
	productRepo := product.NewRepository(dbPool)
	productRepo.SetProductStockWriter(inventory.ProductStockWriter{})
	_ = adapterReg.Register(product.NewAdapter(productRepo, &testCategoryRefRepo{category.NewRepository(dbPool)}, &testBrandRefRepo{brand.NewRepository(dbPool)}, &testUOMRefRepo{uom.NewRepository(dbPool)}))

	val := validation.NewDefaultPipeline()
	progEng = progress.NewEngine(progress.NewInMemoryStore())
	importEng = importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	templateEng = template.NewEngine()
	exportEng = export.NewEngine()

	runID = fmt.Sprintf("T%d", time.Now().UnixNano())
	os.Exit(m.Run())
}

func truncate(tables ...string) {
	_ = shared.TruncateAll(dbPool, tables...)
}

func pollUntilDone(ctx context.Context, t *testing.T, jobID int64) *progress.Progress {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		p, err := progEng.GetProgress(ctx, jobID)
		require.NoError(t, err)
		switch p.Status {
		case progress.StatusCompleted, progress.StatusFailed, progress.StatusCancelled:
			return p
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("timed out waiting for import to complete")
	return nil
}

func generateTemplate(t *testing.T, s schema.ModuleSchema) []byte {
	t.Helper()
	var buf bytes.Buffer
	err := templateEng.Generate(s, nil, &buf)
	require.NoError(t, err)
	require.Greater(t, buf.Len(), 100, "template XLSX too small")
	return buf.Bytes()
}

func templateColumns(s schema.ModuleSchema) []schema.ColumnSchema {
	var cols []schema.ColumnSchema
	for _, c := range s.Columns {
		if c.Template {
			cols = append(cols, c)
		}
	}
	return cols
}

func fillXLSX(t *testing.T, templateData []byte, s schema.ModuleSchema, rows []map[string]interface{}) []byte {
	t.Helper()
	wb, err := excelize.OpenReader(bytes.NewReader(templateData))
	require.NoError(t, err)
	defer func() { _ = wb.Close() }()

	cols := templateColumns(s)
	sheet := s.ModuleName

	for i, row := range rows {
		excelRow := i + 2
		for j, col := range cols {
			colLetter, _ := excelize.ColumnNumberToName(j + 1)
			cell := fmt.Sprintf("%s%d", colLetter, excelRow)
			if val, ok := row[col.Name]; ok {
				_ = wb.SetCellValue(sheet, cell, val)
			}
		}
	}

	var buf bytes.Buffer
	err = wb.Write(&buf)
	require.NoError(t, err)
	return buf.Bytes()
}

func doRoundtrip(t *testing.T, module string, s schema.ModuleSchema, rows []map[string]interface{}, expectedInsert, expectedErr int) {
	t.Helper()
	ctx := context.Background()

	tplData := generateTemplate(t, s)
	filled := fillXLSX(t, tplData, s, rows)

	preview, err := importEng.Preview(ctx, module, module+".xlsx", bytes.NewReader(filled))
	require.NoError(t, err)
	require.NotEmpty(t, preview.Token)
	assert.Equal(t, module, preview.Module)
	assert.Equal(t, len(rows), preview.TotalRows)
	assert.Equal(t, expectedInsert, preview.InsertCount)
	assert.Equal(t, expectedErr, preview.ErrorCount)

	jobID, err := importEng.StartImport(ctx, preview.Token, 0, 0)
	require.NoError(t, err)
	require.Greater(t, jobID, int64(0))

	p := pollUntilDone(ctx, t, jobID)
	assert.Equal(t, progress.StatusCompleted, p.Status)
}

// ============================================================================
// Brands — 100 rows (business key: Name)
// ============================================================================
func TestRoundtrip_Brands_100Rows(t *testing.T) {
	truncate("brands")
	prefix := runID + "_BR"

	rows := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		rows[i] = map[string]interface{}{
			"Name":        fmt.Sprintf("%s_Brand_%d", prefix, i+1),
			"Description": fmt.Sprintf("Roundtrip test brand %d", i+1),
			"IsActive":    "true",
		}
	}

	doRoundtrip(t, "brands", brand.Schema, rows, 100, 0)

	var count int
	err := dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM brands WHERE name LIKE $1", prefix+"%").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

// ============================================================================
// Categories — 100 rows (business key: Name)
// ============================================================================
func TestRoundtrip_Categories_100Rows(t *testing.T) {
	truncate("categories")
	prefix := runID + "_CAT"

	rows := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		rows[i] = map[string]interface{}{
			"Name":        fmt.Sprintf("%s_Category_%d", prefix, i+1),
			"Slug":        fmt.Sprintf("rt-cat-%d", i+1),
			"Description": fmt.Sprintf("Roundtrip test category %d", i+1),
			"IsActive":    "true",
		}
	}

	doRoundtrip(t, "categories", category.Schema, rows, 100, 0)

	var count int
	err := dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM categories WHERE name LIKE $1", prefix+"%").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

// ============================================================================
// UOMs — 100 rows (business key: Code)
// ============================================================================
func TestRoundtrip_UOMs_100Rows(t *testing.T) {
	truncate("units_of_measure")
	prefix := runID + "_UOM"

	rows := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		code := fmt.Sprintf("U%d", i+1)
		rows[i] = map[string]interface{}{
			"Code":        code,
			"Name":        fmt.Sprintf("%s_Name_%d", prefix, i+1),
			"Description": fmt.Sprintf("Roundtrip test UOM %d", i+1),
			"IsActive":    "true",
		}
	}

	doRoundtrip(t, "uoms", uom.Schema, rows, 100, 0)

	var count int
	err := dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM units_of_measure").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

// ============================================================================
// Customers — 100 rows (business keys: Name, Phone)
// ============================================================================
func TestRoundtrip_Customers_100Rows(t *testing.T) {
	truncate("customers")
	prefix := runID + "_CUST"

	rows := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		rows[i] = map[string]interface{}{
			"Name":     fmt.Sprintf("%s_Customer_%d", prefix, i+1),
			"Phone":    fmt.Sprintf("08%09d", i+1),
			"Email":    fmt.Sprintf("cust%d@roundtrip.test", i+1),
			"Address":  fmt.Sprintf("Test Address %d", i+1),
			"Note":     fmt.Sprintf("Roundtrip test customer %d", i+1),
			"IsActive": "true",
		}
	}

	doRoundtrip(t, "customers", customer.Schema, rows, 100, 0)

	var count int
	err := dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM customers WHERE name LIKE $1", prefix+"%").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

// ============================================================================
// Products — 100 rows (needs existing categories, brands, UOMs)
// ============================================================================
func TestRoundtrip_Products_100Rows(t *testing.T) {
	truncate("products", "categories", "brands", "units_of_measure")
	ctx := context.Background()
	refPrefix := runID + "_REF"

	brandRepo := brand.NewRepository(dbPool)
	catRepo := category.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)

	for i := 1; i <= 10; i++ {
		_ = brandRepo.Create(ctx, &brand.Brand{
			Name:        fmt.Sprintf("%s_Brand_%d", refPrefix, i),
			Description: fmt.Sprintf("Ref brand %d", i),
			IsActive:    true,
		})
		_ = catRepo.CreateCategory(ctx, &category.Category{
			Name:        fmt.Sprintf("%s_Category_%d", refPrefix, i),
			Description: fmt.Sprintf("Ref category %d", i),
			IsActive:    true,
		})
		_ = uomRepo.Create(ctx, &uom.UnitOfMeasure{
			Code:        fmt.Sprintf("RU%d", i),
			Name:        fmt.Sprintf("%s_UOM_%d", refPrefix, i),
			Description: fmt.Sprintf("Ref UOM %d", i),
			IsActive:    true,
		})
	}

	pPrefix := runID + "_PROD"
	rows := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		refIdx := (i % 10) + 1
		rows[i] = map[string]interface{}{
			"SKU":           fmt.Sprintf("%s_SKU_%d", pPrefix, i+1),
			"Name":          fmt.Sprintf("%s_Product_%d", pPrefix, i+1),
			"Barcode":       "",
			"Category":      fmt.Sprintf("%s_Category_%d", refPrefix, refIdx),
			"Brand":         fmt.Sprintf("%s_Brand_%d", refPrefix, refIdx),
			"Price":         25000,
			"Cost":          18000,
			"Status":        "active",
			"UnitOfMeasure": fmt.Sprintf("RU%d", refIdx),
		}
	}

	tplData := generateTemplate(t, product.Schema)
	filled := fillXLSX(t, tplData, product.Schema, rows)

	preview, err := importEng.Preview(ctx, "products", "products.xlsx", bytes.NewReader(filled))
	require.NoError(t, err)
	require.NotEmpty(t, preview.Token)
	assert.Equal(t, 100, preview.TotalRows)
	assert.Equal(t, 100, preview.InsertCount)
	assert.Equal(t, 0, preview.ErrorCount)

	jobID, err := importEng.StartImport(ctx, preview.Token, 0, 0)
	require.NoError(t, err)
	require.Greater(t, jobID, int64(0))

	p := pollUntilDone(ctx, t, jobID)
	assert.Equal(t, progress.StatusCompleted, p.Status)

	var count int
	err = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM products WHERE sku LIKE $1", pPrefix+"%").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

// testCategoryRefRepo wraps *category.Repository to satisfy product's
// consumer-side ref ports, mirroring the adapters in internal/wiring.
type testCategoryRefRepo struct {
	repo *category.Repository
}

func (r *testCategoryRefRepo) GetCategoryIDByName(ctx context.Context, name string) (int, error) {
	return r.repo.GetCategoryIDByName(ctx, name)
}

func (r *testCategoryRefRepo) GetCategoryIDsByNames(ctx context.Context, names []string) (map[string]int, error) {
	return r.repo.GetCategoryIDsByNames(ctx, names)
}

func (r *testCategoryRefRepo) GetAllCategoriesForExport(ctx context.Context) ([]product.CategoryRef, error) {
	categories, err := r.repo.GetAllCategoriesForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]product.CategoryRef, len(categories))
	for i, c := range categories {
		result[i] = product.CategoryRef{ID: c.ID, Name: c.Name}
	}
	return result, nil
}

type testBrandRefRepo struct {
	repo *brand.Repository
}

func (r *testBrandRefRepo) GetIDByName(ctx context.Context, name string) (int, error) {
	return r.repo.GetIDByName(ctx, name)
}

func (r *testBrandRefRepo) GetIDsByNames(ctx context.Context, names []string) (map[string]int, error) {
	return r.repo.GetIDsByNames(ctx, names)
}

func (r *testBrandRefRepo) GetAllForExport(ctx context.Context) ([]product.BrandRef, error) {
	brands, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]product.BrandRef, len(brands))
	for i, b := range brands {
		result[i] = product.BrandRef{ID: b.ID, Name: b.Name}
	}
	return result, nil
}

type testUOMRefRepo struct {
	repo *uom.Repository
}

func (r *testUOMRefRepo) GetIDByCode(ctx context.Context, code string) (int, error) {
	return r.repo.GetIDByCode(ctx, code)
}

func (r *testUOMRefRepo) GetIDsByCodes(ctx context.Context, codes []string) (map[string]int, error) {
	return r.repo.GetIDsByCodes(ctx, codes)
}

func (r *testUOMRefRepo) GetAllForExport(ctx context.Context) ([]product.UOMRef, error) {
	units, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]product.UOMRef, len(units))
	for i, u := range units {
		result[i] = product.UOMRef{ID: u.ID, Code: u.Code}
	}
	return result, nil
}
