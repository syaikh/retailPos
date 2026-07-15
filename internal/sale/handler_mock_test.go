package sale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
)

type mockSaleService struct {
	createSaleFn             func(ctx context.Context, sale *Sale, items []SaleItem) error
	getSaleByIDFn            func(ctx context.Context, id int, storeID *int) (*Sale, error)
	listSalesFn              func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error)
	getSalesForExportFn      func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error)
	getNextInvoiceNumberFn   func(ctx context.Context) (string, error)
	getAllPaymentMethodsFn   func(ctx context.Context) ([]PaymentMethod, error)
	getPaymentMethodByCodeFn func(ctx context.Context, code string) (*PaymentMethod, error)
}

func (m *mockSaleService) CreateSale(ctx context.Context, sale *Sale, items []SaleItem) error {
	return m.createSaleFn(ctx, sale, items)
}
func (m *mockSaleService) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	return m.getSaleByIDFn(ctx, id, storeID)
}
func (m *mockSaleService) ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
	return m.listSalesFn(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, paymentMethods, storeID, minTotal, maxTotal)
}
func (m *mockSaleService) GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
	return m.getSalesForExportFn(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}
func (m *mockSaleService) GetNextInvoiceNumber(ctx context.Context) (string, error) {
	return m.getNextInvoiceNumberFn(ctx)
}
func (m *mockSaleService) GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return m.getAllPaymentMethodsFn(ctx)
}
func (m *mockSaleService) GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error) {
	return m.getPaymentMethodByCodeFn(ctx, code)
}

type mockAuditCreator struct {
	createAuditLogFn func(ctx context.Context, log *audit.AuditLog) error
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.AuditLog) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupSaleHandler(svc SaleService, auditSvc audit.AuditCreator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPaymentMethodsPublicRoutes(r.Group("/public"))
	return r
}

func TestSaleHandler_CreateSale_Success(t *testing.T) {
	var capturedSale *Sale
	svc := &mockSaleService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []SaleItem) error {
			sale.ID = 1
			capturedSale = sale
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-001", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash", Name: "Cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":2,"subtotal":20000}],"payment_method":"cash"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, capturedSale)
	assert.Equal(t, "INV-001", capturedSale.InvoiceNumber)
	assert.Equal(t, 1, capturedSale.CashierID)
}

func TestSaleHandler_CreateSale_WithAuditLog(t *testing.T) {
	auditCalled := false
	svc := &mockSaleService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []SaleItem) error {
			sale.ID = 1
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-002", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash"}, nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
			auditCalled = true
			assert.Equal(t, "create", log.Action)
			assert.Equal(t, "sale", log.EntityType)
			return nil
		},
	}
	r := setupSaleHandler(svc, auditSvc)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"payment_method":"cash"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestSaleHandler_CreateSale_BindJSONError(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_CreateSale_NegativeDiscount(t *testing.T) {
	svc := &mockSaleService{
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-003", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"discount":-5}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "discount must not be negative")
}

func TestSaleHandler_CreateSale_DiscountExceedsSubtotal(t *testing.T) {
	svc := &mockSaleService{
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-004", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"discount":20000}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "discount must not exceed subtotal")
}

func TestSaleHandler_CreateSale_InvalidPaymentMethod(t *testing.T) {
	svc := &mockSaleService{
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-005", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return nil, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"payment_method":"invalid"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid payment method")
}

func TestSaleHandler_CreateSale_InvoiceNumberError(t *testing.T) {
	svc := &mockSaleService{
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "", assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to generate invoice number")
}

func TestSaleHandler_CreateSale_InsufficientStock(t *testing.T) {
	svc := &mockSaleService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []SaleItem) error {
			return ErrInsufficientStock
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-006", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"payment_method":"cash"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "insufficient stock")
}

func TestSaleHandler_CreateSale_ServiceError(t *testing.T) {
	svc := &mockSaleService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []SaleItem) error {
			return assert.AnError
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-007", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"payment_method":"cash"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_CreateSale_ProvidedInvoiceNumber(t *testing.T) {
	var capturedSale *Sale
	svc := &mockSaleService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []SaleItem) error {
			capturedSale = sale
			sale.ID = 1
			return nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"invoice_number":"CUSTOM-INV","items":[{"product_id":1,"quantity":1,"subtotal":10000}],"payment_method":"cash"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, capturedSale)
	assert.Equal(t, "CUSTOM-INV", capturedSale.InvoiceNumber)
}

func TestSaleHandler_GetSalesHistory_Success(t *testing.T) {
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			return []Sale{{ID: 1, InvoiceNumber: "INV-001"}}, 1, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])
}

func TestSaleHandler_GetSalesHistory_Error(t *testing.T) {
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			return nil, 0, assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_GetSalesHistory_InvalidMinTotal(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?min_total=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "min_total must be between")
}

func TestSaleHandler_GetSalesHistory_InvalidMaxTotal(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?max_total=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "max_total must be between")
}

func TestSaleHandler_GetSalesHistory_MinExceedsMax(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?min_total=100&max_total=50", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "min_total cannot exceed max_total")
}

func TestSaleHandler_GetSalesHistory_InvalidMinTotalOutOfRange(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?min_total=999999999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_GetSalesHistory_InvalidMaxTotalOutOfRange(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?max_total=999999999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_GetSalesHistory_WithFilters(t *testing.T) {
	var capturedSearch string
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			capturedSearch = search
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?search=test&sort_by=total_amount&sort_dir=ASC&payment_methods=cash,card&start_date=2024-01-01&end_date=2024-01-31", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test", capturedSearch)
}

func TestSaleHandler_GetSalesHistory_InvalidSortBy(t *testing.T) {
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			assert.Equal(t, "created_at", sortBy, "invalid sort_by should default to created_at")
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?sort_by=invalid_col", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_GetSalesHistory_InvalidSortDir(t *testing.T) {
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			assert.Equal(t, "DESC", sortDir, "invalid sort_dir should default to DESC")
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?sort_dir=SIDEWAYS", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_GetSalesHistory_DefaultLimitOffset(t *testing.T) {
	var capturedLimit, capturedOffset int
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 50, capturedLimit)
	assert.Equal(t, 0, capturedOffset)
}

func TestSaleHandler_GetSalesHistory_OutOfRangeLimit(t *testing.T) {
	var capturedLimit int
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			capturedLimit = limit
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?limit=999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 50, capturedLimit)
}

func TestSaleHandler_GetSalesHistory_NegativeOffset(t *testing.T) {
	var capturedOffset int
	svc := &mockSaleService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int) ([]Sale, int, error) {
			capturedOffset = offset
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?offset=-5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 0, capturedOffset)
}

func TestSaleHandler_GetSaleByID_Success(t *testing.T) {
	svc := &mockSaleService{
		getSaleByIDFn: func(ctx context.Context, id int, storeID *int) (*Sale, error) {
			return &Sale{ID: 1, InvoiceNumber: "INV-001"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_GetSaleByID_InvalidID(t *testing.T) {
	svc := &mockSaleService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid sale id")
}

func TestSaleHandler_GetSaleByID_NotFound(t *testing.T) {
	svc := &mockSaleService{
		getSaleByIDFn: func(ctx context.Context, id int, storeID *int) (*Sale, error) {
			return nil, ErrSaleNotFound
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSaleHandler_GetSaleByID_ServiceError(t *testing.T) {
	svc := &mockSaleService{
		getSaleByIDFn: func(ctx context.Context, id int, storeID *int) (*Sale, error) {
			return nil, assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_ExportSales_CSV(t *testing.T) {
	svc := &mockSaleService{
		getSalesForExportFn: func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
			return []SaleExportRow{{InvoiceNumber: "INV-001", TotalAmount: 10000}}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

func TestSaleHandler_ExportSales_XLSX(t *testing.T) {
	svc := &mockSaleService{
		getSalesForExportFn: func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
			return []SaleExportRow{}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export?format=xlsx", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
}

func TestSaleHandler_ExportSales_Error(t *testing.T) {
	svc := &mockSaleService{
		getSalesForExportFn: func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
			return nil, assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_ExportSales_WithFilters(t *testing.T) {
	svc := &mockSaleService{
		getSalesForExportFn: func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
			return []SaleExportRow{}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export?search=test&payment_methods=cash&min_total=100&max_total=50000&start_date=2024-01-01&end_date=2024-12-31", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_ListPaymentMethods_Success(t *testing.T) {
	svc := &mockSaleService{
		getAllPaymentMethodsFn: func(ctx context.Context) ([]PaymentMethod, error) {
			return []PaymentMethod{{Code: "cash", Name: "Cash"}, {Code: "card", Name: "Card"}}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/public/payment-methods", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]interface{})
	assert.Equal(t, 2, len(data))
}

func TestSaleHandler_ListPaymentMethods_Error(t *testing.T) {
	svc := &mockSaleService{
		getAllPaymentMethodsFn: func(ctx context.Context) ([]PaymentMethod, error) {
			return nil, assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/public/payment-methods", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_GetPaymentMethodByCode_Success(t *testing.T) {
	svc := &mockSaleService{
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash", Name: "Cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/payment-methods/cash", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_GetPaymentMethodByCode_NotFound(t *testing.T) {
	svc := &mockSaleService{
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return nil, assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/payment-methods/nonexistent", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "payment method not found")
}
