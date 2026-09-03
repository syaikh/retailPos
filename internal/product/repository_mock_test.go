package product

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// stubStockWriter is a no-op StockWriter used by the pgxmock tests:
// the product_stock SQL now lives in the inventory-provided writer, so the
// mock tests only exercise the product-owned products SQL and assert the rows
// the repository delegated to the port.
type stubStockWriter struct {
	calls      []shared.StockRowSet
	batchCalls [][]shared.StockRowSet
}

func (s *stubStockWriter) SetStoreStock(_ context.Context, _ pgx.Tx, item shared.StockRowSet) error {
	s.calls = append(s.calls, item)
	return nil
}

func (s *stubStockWriter) SetStoreStockBatch(_ context.Context, _ pgx.Tx, items []shared.StockRowSet) error {
	s.batchCalls = append(s.batchCalls, items)
	return nil
}

func productFullRow(id int, sku, name string, price, cost, stock int, status string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "sku", "name", "barcode", "category_id", "category_name",
		"price", "cost", "stock", "status", "store_id",
		"brand_id", "brand_name", "unit_of_measure_id", "unit_of_measure", "weight_grams", "description",
		"tax_class_id", "tax_rate",
		"supplier_id", "supplier_name",
		"created_at", "updated_at",
	}).AddRow(
		id, sku, name, nil, nil, nil,
		price, cost, stock, status, nil,
		nil, nil, nil, nil, nil, nil,
		nil, nil,
		nil, nil,
		time.Now(), time.Now(),
	)
}

func productFullRowAllFields(id int, sku, name string, price, cost, stock int, status string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "sku", "name", "barcode", "category_id", "category_name",
		"price", "cost", "stock", "status", "store_id",
		"brand_id", "brand_name", "unit_of_measure_id", "unit_of_measure", "weight_grams", "description",
		"tax_class_id", "tax_rate",
		"supplier_id", "supplier_name",
		"created_at", "updated_at",
	}).AddRow(
		id, sku, name, "123456", 10, "Electronics",
		price, cost, stock, status, 1,
		5, "BrandA", 3, "pcs", 500, "A product",
		2, 0.11,
		1, "SupplierX",
		time.Now(), time.Now(),
	)
}

func emptyProductCols() *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "sku", "name", "barcode", "category_id", "category_name",
		"price", "cost", "stock", "status", "store_id",
		"brand_id", "brand_name", "unit_of_measure_id", "unit_of_measure", "weight_grams", "description",
		"tax_class_id", "tax_rate", "supplier_id", "supplier_name", "created_at", "updated_at",
	})
}

func TestRepo_GetProductPrice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT price FROM products").WithArgs(1).WillReturnRows(
		pgxmock.NewRows([]string{"price"}).AddRow(10000))

	repo := NewRepository(mock)
	price, err := repo.GetProductPrice(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 10000, price)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductPrice_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT price FROM products").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetProductPrice(context.Background(), 999)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductPrices(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, price FROM products").WithArgs(1, 2).WillReturnRows(
		pgxmock.NewRows([]string{"id", "price"}).AddRow(1, 10000).AddRow(2, 20000))

	repo := NewRepository(mock)
	prices, err := repo.GetProductPrices(context.Background(), []int{1, 2})
	require.NoError(t, err)
	assert.Equal(t, 10000, prices[1])
	assert.Equal(t, 20000, prices[2])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductPrices_Empty(t *testing.T) {
	repo := NewRepository(nil)
	prices, err := repo.GetProductPrices(context.Background(), []int{})
	require.NoError(t, err)
	assert.Empty(t, prices)
}

func TestRepo_GetProductByID_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := productFullRowAllFields(1, "SKU-001", "Widget", 10000, 5000, 100, "active")
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	p, err := repo.GetProductByID(context.Background(), 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, p.ID)
	assert.Equal(t, "SKU-001", p.SKU)
	require.NotNil(t, p.Barcode)
	assert.Equal(t, "123456", *p.Barcode)
	require.NotNil(t, p.CategoryID)
	assert.Equal(t, 10, *p.CategoryID)
	require.NotNil(t, p.StoreID)
	assert.Equal(t, 1, *p.StoreID)
	require.NotNil(t, p.BrandID)
	assert.Equal(t, 5, *p.BrandID)
	require.NotNil(t, p.TaxRate)
	assert.Equal(t, 0.11, *p.TaxRate)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetProductByID(context.Background(), 999, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductByID_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := productFullRow(1, "SKU-001", "Widget", 10000, 5000, 100, "active")
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1, 2).WillReturnRows(rows)

	repo := NewRepository(mock)
	sid := 2
	p, err := repo.GetProductByID(context.Background(), 1, &sid)
	require.NoError(t, err)
	assert.Equal(t, 1, p.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductBySKU_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := productFullRow(1, "SKU-001", "Widget", 10000, 5000, 100, "active")
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs("SKU-001").WillReturnRows(rows)

	repo := NewRepository(mock)
	p, err := repo.GetProductBySKU(context.Background(), "SKU-001", nil)
	require.NoError(t, err)
	assert.Equal(t, "SKU-001", p.SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductBySKU_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs("NOPE").WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetProductBySKU(context.Background(), "NOPE", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_DeleteProduct(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE products SET deleted_at").WithArgs(1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	err = repo.DeleteProduct(context.Background(), 1, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_DeleteProduct_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE products SET deleted_at").WithArgs(1, 2).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	sid := 2
	err = repo.DeleteProduct(context.Background(), 1, &sid)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetNextSKU(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT nextval").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(42))

	repo := NewRepository(mock)
	sku, err := repo.GetNextSKU(context.Background())
	require.NoError(t, err)
	year := time.Now().In(shared.JakartaLocation()).Year()
	assert.Equal(t, fmt.Sprintf("SKU-%d-000042", year), sku)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetNextSKU_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT nextval").WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetNextSKU(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetTaxClassByID_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name", "rate_percent", "description", "is_active", "created_at"}).
		AddRow(1, "PPN", 11.0, "Pajak Pertambahan Nilai", true, time.Now())
	mock.ExpectQuery("SELECT.+FROM tax_classes").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	tc, err := repo.GetTaxClassByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "PPN", tc.Name)
	assert.Equal(t, 11.0, tc.RatePercent)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetTaxClassByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT.+FROM tax_classes").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetTaxClassByID(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllTaxClasses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name", "rate_percent", "description", "is_active", "created_at"}).
		AddRow(1, "PPN", 11.0, "", true, time.Now()).
		AddRow(2, "PPH", 2.0, "", true, time.Now())
	mock.ExpectQuery("SELECT.+FROM tax_classes").WillReturnRows(rows)

	repo := NewRepository(mock)
	tcs, err := repo.GetAllTaxClasses(context.Background())
	require.NoError(t, err)
	assert.Len(t, tcs, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetTaxClassIDByName_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id FROM tax_classes").WithArgs("PPN").WillReturnRows(
		pgxmock.NewRows([]string{"id"}).AddRow(1))

	repo := NewRepository(mock)
	id, err := repo.GetTaxClassIDByName(context.Background(), "PPN")
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_CreateProduct(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO products").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), time.Now()))
	mock.ExpectCommit()

	stub := &stubStockWriter{}
	repo := NewRepository(mock)
	repo.SetProductStockWriter(stub)
	p := &Product{
		SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000,
		Stock: 100, Status: "active",
	}
	err = repo.CreateProduct(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, 1, p.ID)
	assert.Equal(t, []shared.StockRowSet{{ProductID: 1, Quantity: 100}}, stub.calls)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_CreateProduct_WithOptionals(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO products").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(2, time.Now(), time.Now()))
	mock.ExpectCommit()

	bc := "123456"
	cid := 10
	sid := 1
	bid := 5
	uomid := 3
	wg := 500
	tcid := 2
	disc := 5.0
	desc := "A widget"

	stub := &stubStockWriter{}
	repo := NewRepository(mock)
	repo.SetProductStockWriter(stub)
	p := &Product{
		SKU: "SKU-002", Name: "Widget Pro", Barcode: &bc, CategoryID: &cid,
		Price: 20000, Cost: 10000, Stock: 50, Status: "active",
		StoreID: &sid, BrandID: &bid, UnitOfMeasureID: &uomid, WeightGrams: &wg,
		TaxClassID: &tcid, DefaultDiscountPct: &disc, Description: &desc,
	}
	err = repo.CreateProduct(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, 2, p.ID)
	assert.Equal(t, []shared.StockRowSet{{ProductID: 2, StoreID: &sid, Quantity: 50}}, stub.calls)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_UpdateProduct(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products SET sku").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	p := &Product{
		ID: 1, SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000,
		Stock: 100, Status: "active",
	}
	err = repo.UpdateProduct(context.Background(), p, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_UpdateProduct_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products SET sku").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	sid := 2
	stub := &stubStockWriter{}
	repo := NewRepository(mock)
	repo.SetProductStockWriter(stub)
	p := &Product{
		ID: 1, SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000, Stock: 50, Status: "active",
		StoreID: &sid,
	}
	err = repo.UpdateProduct(context.Background(), p, &sid)
	require.NoError(t, err)
	assert.Equal(t, []shared.StockRowSet{{ProductID: 1, StoreID: &sid, Quantity: 50}}, stub.calls)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_RestoreProduct(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products SET sku").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	repo := NewRepository(mock)
	p := &Product{
		ID: 1, SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000,
		Stock: 100, Status: "active",
	}
	err = repo.RestoreProduct(context.Background(), p)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_RestoreProduct_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products SET sku").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	sid := 1
	stub := &stubStockWriter{}
	repo := NewRepository(mock)
	repo.SetProductStockWriter(stub)
	p := &Product{
		ID: 1, SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000, Stock: 100, Status: "active",
		StoreID: &sid,
	}
	err = repo.RestoreProduct(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, []shared.StockRowSet{{ProductID: 1, StoreID: &sid, Quantity: 100}}, stub.calls)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_CreateProduct_UnwiredStockWriter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO products").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), time.Now()))
	mock.ExpectRollback()

	repo := NewRepository(mock)
	p := &Product{SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000, Stock: 100, Status: "active"}
	err = repo.CreateProduct(context.Background(), p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "product_stock writer not wired")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_CreateProduct_BeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	p := &Product{SKU: "SKU-001", Name: "Widget"}
	err = repo.CreateProduct(context.Background(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_UpdateProduct_BeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	p := &Product{ID: 1, SKU: "SKU-001", Name: "Widget"}
	err = repo.UpdateProduct(context.Background(), p, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_RestoreProduct_BeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	p := &Product{ID: 1, SKU: "SKU-001", Name: "Widget"}
	err = repo.RestoreProduct(context.Background(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_NoFilters(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(10, 0).WillReturnRows(
		productFullRow(1, "SKU-001", "Widget", 10000, 5000, 100, "active"))

	repo := NewRepository(mock)
	products, total, err := repo.GetAllProducts(context.Background(), 10, 0, "", nil, "", "", nil, nil, "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, products, 1)
	assert.Equal(t, "SKU-001", products[0].SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithSearch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WithArgs("widget").WillReturnRows(
		pgxmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs("widget", 10, 0).WillReturnRows(
		productFullRow(1, "SKU-001", "Widget", 10000, 5000, 100, "active"))

	repo := NewRepository(mock)
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "widget", nil, "", "", nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithCategoryIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WithArgs(1, 2).WillReturnRows(
		pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1, 2, 10, 0).WillReturnRows(emptyProductCols())

	repo := NewRepository(mock)
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "", []int{1, 2}, "", "", nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WithArgs(1).WillReturnRows(
		pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1, 10, 0).WillReturnRows(emptyProductCols())

	repo := NewRepository(mock)
	sid := 1
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "", nil, "", "", nil, &sid, "", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WithArgs("active").WillReturnRows(
		pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs("active", 10, 0).WillReturnRows(emptyProductCols())

	repo := NewRepository(mock)
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "", nil, "", "active", nil, nil, "active", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithMaxStock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	ms := 10
	mock.ExpectQuery("SELECT COUNT").WithArgs(10).WillReturnRows(
		pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(10, 10, 0).WillReturnRows(emptyProductCols())

	repo := NewRepository(mock)
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "", nil, "", "", &ms, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithSortByValid(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(10, 0).WillReturnRows(emptyProductCols())

	repo := NewRepository(mock)
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "", nil, "v.name", "ASC", nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProducts_WithSortByInvalid(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(10, 0).WillReturnRows(emptyProductCols())

	repo := NewRepository(mock)
	_, _, err = repo.GetAllProducts(context.Background(), 10, 0, "", nil, "evil_col", "ASC", nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductsByIDs_Empty(t *testing.T) {
	repo := NewRepository(nil)
	products, err := repo.GetProductsByIDs(context.Background(), []int{}, nil)
	require.NoError(t, err)
	assert.Empty(t, products)
}

func TestRepo_GetProductsByIDs_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := productFullRowAllFields(1, "SKU-001", "Widget A", 10000, 5000, 100, "active")
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1, 2).WillReturnRows(rows)

	repo := NewRepository(mock)
	products, err := repo.GetProductsByIDs(context.Background(), []int{1, 2}, nil)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "SKU-001", products[0].SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductsByIDs_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetProductsByIDs(context.Background(), []int{1}, nil)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductsByIDs_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	badRows := pgxmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1).WillReturnRows(badRows)

	repo := NewRepository(mock)
	_, err = repo.GetProductsByIDs(context.Background(), []int{1}, nil)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetProductsByIDs_MultipleRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{
		"id", "sku", "name", "barcode", "category_id", "category_name",
		"price", "cost", "stock", "status", "store_id",
		"brand_id", "brand_name", "unit_of_measure_id", "unit_of_measure", "weight_grams", "description",
		"tax_class_id", "tax_rate",
		"supplier_id", "supplier_name",
		"created_at", "updated_at",
	}).AddRow(
		1, "SKU-001", "Widget A", "123456", 10, "Electronics",
		10000, 5000, 100, "active", 1,
		5, "BrandA", 3, "pcs", 500, "A product",
		2, 0.11,
		nil, nil,
		time.Now(), time.Now(),
	).AddRow(
		2, "SKU-002", "Widget B", "789012", 10, "Electronics",
		20000, 10000, 50, "active", 1,
		5, "BrandA", 3, "pcs", 500, "B product",
		2, 0.11,
		1, "SupplierX",
		time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(1, 2).WillReturnRows(rows)

	repo := NewRepository(mock)
	products, err := repo.GetProductsByIDs(context.Background(), []int{1, 2}, nil)
	require.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, "SKU-001", products[0].SKU)
	assert.Equal(t, "SKU-002", products[1].SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProductsForExport(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := productFullRowAllFields(1, "SKU-001", "Widget", 10000, 5000, 100, "active")
	mock.ExpectQuery("SELECT.+FROM v_products_full").WillReturnRows(rows)

	repo := NewRepository(mock)
	products, err := repo.GetAllProductsForExport(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "SKU-001", products[0].SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetAllProductsForExport_StoreScoped(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storeID := 3
	rows := productFullRowAllFields(1, "SKU-001", "Widget", 10000, 5000, 100, "active")
	mock.ExpectQuery("SELECT.+FROM v_products_full").WithArgs(storeID).WillReturnRows(rows)

	repo := NewRepository(mock)
	products, err := repo.GetAllProductsForExport(context.Background(), &storeID)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "SKU-001", products[0].SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_UpdateProduct_ExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products SET sku").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	p := &Product{ID: 1, SKU: "SKU-001", Name: "Widget", Price: 10000, Cost: 5000, Stock: 50, Status: "active"}
	err = repo.UpdateProduct(context.Background(), p, nil)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_SetCache(t *testing.T) {
	repo := NewRepository(nil)
	repo.SetCache(nil)
	assert.Nil(t, repo.cache)
}
