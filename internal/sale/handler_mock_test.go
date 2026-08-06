package sale

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"
)

func init() {
	_ = os.Setenv("JWT_SECRET", "test-secret-for-sale-mock-tests")
}

type mockService struct {
	createSaleFn               func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error
	createSaleWithParkedSaleFn func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error
	getSaleByIDFn              func(ctx context.Context, id int, storeID *int) (*Sale, error)
	listSalesFn                func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error)
	getSalesForExportFn        func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error)
	streamSalesExportCSVFn     func(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error
	getNextInvoiceNumberFn     func(ctx context.Context) (string, error)
	getAllPaymentMethodsFn     func(ctx context.Context) ([]PaymentMethod, error)
	getPaymentMethodByCodeFn   func(ctx context.Context, code string) (*PaymentMethod, error)
	parkSaleFn                 func(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int) error
	recallSaleFn               func(ctx context.Context, saleID int) (*Sale, error)
	cancelParkedSaleFn         func(ctx context.Context, saleID int) error
	listParkedSalesFn          func(ctx context.Context, cashierID int) ([]Sale, error)
	getParkedSaleByIDFn        func(ctx context.Context, saleID int, cashierID int) (*Sale, error)

	createOrGetOpenCartFn           func(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error)
	getOpenCartFn                   func(ctx context.Context, cashierID int) (*CartSession, error)
	getCartByIDFn                   func(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	listHeldCartsFn                 func(ctx context.Context, cashierID int) ([]CartSession, error)
	updateCartCustomerFn            func(ctx context.Context, cartID int, customerID *int, cashierID int) (*CartSession, error)
	addCartItemFn                   func(ctx context.Context, cartID int, productID, quantity int, customerGroupID *int, cashierID int) (*CartSession, error)
	updateCartItemQuantityFn        func(ctx context.Context, cartID, itemID, quantity int, cashierID int) (*CartSession, error)
	removeCartItemFn                func(ctx context.Context, cartID, itemID int, cashierID int) (*CartSession, error)
	holdCartFn                      func(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	resumeCartFn                    func(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	checkoutCartFn                  func(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error)
	checkoutCartWithPaymentMethodFn func(ctx context.Context, cartID int, paymentMethod string, cashierID int) (*Sale, error)

	setCartConfigFn    func(cfg CartConfig)
	setPriceStoreFn    func(ps ProductPriceGetter)
	setPriceResolverFn func(r PriceResolver)
}

func (m *mockService) CreateSale(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
	return m.createSaleFn(ctx, sale, items, payments)
}
func (m *mockService) CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
	if m.createSaleWithParkedSaleFn != nil {
		return m.createSaleWithParkedSaleFn(ctx, sale, items, parkedSaleID, payments)
	}
	return m.createSaleFn(ctx, sale, items, payments)
}
func (m *mockService) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	if m.getSaleByIDFn != nil {
		return m.getSaleByIDFn(ctx, id, storeID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
	return m.listSalesFn(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, paymentMethods, storeID, minTotal, maxTotal, cashierID)
}
func (m *mockService) GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error) {
	return m.getSalesForExportFn(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}
func (m *mockService) StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
	return m.streamSalesExportCSVFn(ctx, w, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeID)
}
func (m *mockService) GetNextInvoiceNumber(ctx context.Context) (string, error) {
	return m.getNextInvoiceNumberFn(ctx)
}
func (m *mockService) GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return m.getAllPaymentMethodsFn(ctx)
}
func (m *mockService) GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error) {
	return m.getPaymentMethodByCodeFn(ctx, code)
}
func (m *mockService) ParkSale(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int) error {
	return m.parkSaleFn(ctx, sale, items, recalledSaleID)
}
func (m *mockService) RecallSale(ctx context.Context, saleID int) (*Sale, error) {
	return m.recallSaleFn(ctx, saleID)
}
func (m *mockService) CancelParkedSale(ctx context.Context, saleID int) error {
	return m.cancelParkedSaleFn(ctx, saleID)
}
func (m *mockService) ListParkedSales(ctx context.Context, cashierID int) ([]Sale, error) {
	return m.listParkedSalesFn(ctx, cashierID)
}
func (m *mockService) GetParkedSaleByID(ctx context.Context, saleID int, cashierID int) (*Sale, error) {
	return m.getParkedSaleByIDFn(ctx, saleID, cashierID)
}

func (m *mockService) CreateOrGetOpenCart(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
	if m.createOrGetOpenCartFn != nil {
		return m.createOrGetOpenCartFn(ctx, cashierID, storeID, shiftID, customerID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) GetOpenCart(ctx context.Context, cashierID int) (*CartSession, error) {
	if m.getOpenCartFn != nil {
		return m.getOpenCartFn(ctx, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) GetCartByID(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
	if m.getCartByIDFn != nil {
		return m.getCartByIDFn(ctx, cartID, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) ListHeldCarts(ctx context.Context, cashierID int) ([]CartSession, error) {
	if m.listHeldCartsFn != nil {
		return m.listHeldCartsFn(ctx, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) UpdateCartCustomer(ctx context.Context, cartID int, customerID *int, cashierID int) (*CartSession, error) {
	if m.updateCartCustomerFn != nil {
		return m.updateCartCustomerFn(ctx, cartID, customerID, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) AddCartItem(ctx context.Context, cartID int, productID, quantity int, customerGroupID *int, cashierID int) (*CartSession, error) {
	if m.addCartItemFn != nil {
		return m.addCartItemFn(ctx, cartID, productID, quantity, customerGroupID, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) UpdateCartItemQuantity(ctx context.Context, cartID, itemID, quantity int, cashierID int) (*CartSession, error) {
	if m.updateCartItemQuantityFn != nil {
		return m.updateCartItemQuantityFn(ctx, cartID, itemID, quantity, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) RemoveCartItem(ctx context.Context, cartID, itemID int, cashierID int) (*CartSession, error) {
	if m.removeCartItemFn != nil {
		return m.removeCartItemFn(ctx, cartID, itemID, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) HoldCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
	if m.holdCartFn != nil {
		return m.holdCartFn(ctx, cartID, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) ResumeCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
	if m.resumeCartFn != nil {
		return m.resumeCartFn(ctx, cartID, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) CheckoutCart(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
	if m.checkoutCartFn != nil {
		return m.checkoutCartFn(ctx, cartID, payments, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockService) CheckoutCartWithPaymentMethod(ctx context.Context, cartID int, paymentMethod string, cashierID int) (*Sale, error) {
	if m.checkoutCartWithPaymentMethodFn != nil {
		return m.checkoutCartWithPaymentMethodFn(ctx, cartID, paymentMethod, cashierID)
	}
	return nil, fmt.Errorf("not mocked")
}

func (m *mockService) SetCartConfig(cfg CartConfig) {
	if m.setCartConfigFn != nil {
		m.setCartConfigFn(cfg)
	}
}

func (m *mockService) SetPriceStore(ps ProductPriceGetter) {
	if m.setPriceStoreFn != nil {
		m.setPriceStoreFn(ps)
	}
}

func (m *mockService) SetPriceResolver(r PriceResolver) {
	if m.setPriceResolverFn != nil {
		m.setPriceResolverFn(r)
	}
}

func (m *mockService) SetStockDeducer(sd StockDeducer) {
}

type mockAuditCreator struct {
	createAuditLogFn func(ctx context.Context, log *audit.Log) error
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.Log) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupSaleHandler(svc Service, auditSvc audit.Creator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPaymentMethodsPublicRoutes(r.Group("/public"))
	return r
}

// setupSaleHandlerUser builds a handler with a configurable userID context
// value (nil omits the key entirely) so the auth branches of the sale
// handlers can be exercised.
func setupSaleHandlerUser(svc Service, auditSvc audit.Creator, userID interface{}) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != nil {
			c.Set("userID", userID)
		}
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPaymentMethodsPublicRoutes(r.Group("/public"))
	return r
}

// TestSaleHandler_UserContextBranches covers the missing/invalid userID
// branches in CreateSale, ParkSale, ListParkedSales and GetParkedSaleByID.
func TestSaleHandler_UserContextBranches(t *testing.T) {
	svc := &mockService{}
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}]}`

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"create sale", http.MethodPost, "/sales", `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"payment_method":"cash"}`},
		{"park sale", http.MethodPost, "/sales/parked", body},
		{"list parked", http.MethodGet, "/sales/parked", ""},
		{"get parked by id", http.MethodGet, "/sales/parked/1", ""},
	}

	t.Run("missing userID", func(t *testing.T) {
		r := setupSaleHandlerUser(svc, nil, nil)
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				var req *http.Request
				if tc.body == "" {
					req = httptest.NewRequest(tc.method, tc.path, nil)
				} else {
					req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
					req.Header.Set("Content-Type", "application/json")
				}
				r.ServeHTTP(w, req)
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "user not authenticated")
			})
		}
	})

	t.Run("non-int userID", func(t *testing.T) {
		r := setupSaleHandlerUser(svc, nil, "not-an-int")
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				var req *http.Request
				if tc.body == "" {
					req = httptest.NewRequest(tc.method, tc.path, nil)
				} else {
					req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
					req.Header.Set("Content-Type", "application/json")
				}
				r.ServeHTTP(w, req)
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "invalid user ID in context")
			})
		}
	})
}

func TestSaleHandler_CreateSale_Success(t *testing.T) {
	var capturedSale *Sale
	svc := &mockService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
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
	body := `{"items":[{"product_id":1,"quantity":2,"subtotal":20000}],"payment_method":"cash","tax":1100}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, capturedSale)
	assert.Equal(t, "INV-001", capturedSale.InvoiceNumber)
	assert.Equal(t, 1, capturedSale.CashierID)
	assert.Equal(t, 1100, capturedSale.Tax, "tax should be captured from request body")
}

func TestSaleHandler_CreateSale_WithShiftID(t *testing.T) {
	var capturedSale *Sale
	svc := &mockService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
			sale.ID = 1
			capturedSale = sale
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-008", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "cash", Name: "Cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	shiftID := 42
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"shift_id":42,"payment_method":"cash"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, capturedSale)
	require.NotNil(t, capturedSale.ShiftID)
	assert.Equal(t, shiftID, *capturedSale.ShiftID)
}

func TestSaleHandler_CreateSale_WithAuditLog(t *testing.T) {
	auditCalled := false
	svc := &mockService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
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
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
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
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_CreateSale_WithCartSessionID_DelegatesToCheckout(t *testing.T) {
	checkoutCalled := false
	var capturedPayments []CreatePaymentRequest
	svc := &mockService{
		checkoutCartFn: func(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
			checkoutCalled = true
			assert.Equal(t, 5, cartID)
			assert.Equal(t, 1, cashierID)
			capturedPayments = payments
			return &Sale{ID: 9, InvoiceNumber: "INV-CART-1", TotalAmount: 10000}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"cart_session_id":5,"payments":[{"payment_method_code":"CASH","amount":10000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, checkoutCalled, "CheckoutCart should be invoked")
	require.Len(t, capturedPayments, 1)
	assert.Equal(t, "CASH", capturedPayments[0].PaymentMethodCode)
	assert.Equal(t, 10000, capturedPayments[0].Amount)
}

func TestSaleHandler_CreateSale_WithCartSessionID_ErrorPropagates(t *testing.T) {
	svc := &mockService{
		checkoutCartFn: func(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
			return nil, ErrCartNotFound
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"cart_session_id":999,"payments":[{"payment_method_code":"CASH","amount":10000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSaleHandler_CreateSale_WithCartSessionID_AuditLogged(t *testing.T) {
	auditCalled := false
	svc := &mockService{
		checkoutCartFn: func(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
			return &Sale{ID: 9, InvoiceNumber: "INV-CART-2", TotalAmount: 10000}, nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "create", log.Action)
			assert.Equal(t, "sale", log.EntityType)
			return nil
		},
	}
	r := setupSaleHandler(svc, auditSvc)
	body := `{"cart_session_id":5,"payments":[{"payment_method_code":"CASH","amount":10000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, auditCalled, "audit log should be created for cart-based sale")
}

func TestSaleHandler_CreateSale_NegativeDiscount(t *testing.T) {
	svc := &mockService{
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
	svc := &mockService{
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
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrInvalidPaymentMethod
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-005", nil
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
	svc := &mockService{
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
	svc := &mockService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
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
	svc := &mockService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
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
	svc := &mockService{
		createSaleFn: func(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error {
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
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
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
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
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
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?min_total=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "min_total must be between")
}

func TestSaleHandler_GetSalesHistory_InvalidMaxTotal(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?max_total=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "max_total must be between")
}

func TestSaleHandler_GetSalesHistory_MinExceedsMax(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?min_total=100&max_total=50", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "min_total cannot exceed max_total")
}

func TestSaleHandler_GetSalesHistory_InvalidMinTotalOutOfRange(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?min_total=999999999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_GetSalesHistory_InvalidMaxTotalOutOfRange(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?max_total=999999999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_GetSalesHistory_WithFilters(t *testing.T) {
	var capturedSearch string
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
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
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
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
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
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
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 20, capturedLimit)
	assert.Equal(t, 0, capturedOffset)
}

func TestSaleHandler_GetSalesHistory_OutOfRangeLimit(t *testing.T) {
	var capturedLimit int
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
			capturedLimit = limit
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?limit=999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 20, capturedLimit)
}

func TestSaleHandler_GetSalesHistory_NegativeOffset(t *testing.T) {
	var capturedOffset int
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
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
	svc := &mockService{
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
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid sale id")
}

func TestSaleHandler_GetSaleByID_NotFound(t *testing.T) {
	svc := &mockService{
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
	svc := &mockService{
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

func TestSaleHandler_GetSalesHistory_WithCashierID(t *testing.T) {
	var capturedCashierID *int
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
			capturedCashierID = cashierID
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?cashier_id=5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedCashierID)
	assert.Equal(t, 5, *capturedCashierID)
}

func TestSaleHandler_GetSalesHistory_InvalidCashierID(t *testing.T) {
	svc := &mockService{
		listSalesFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
			return []Sale{}, 0, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales?cashier_id=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_ExportSales_CSV(t *testing.T) {
	svc := &mockService{
		streamSalesExportCSVFn: func(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{"Invoice Number", "Date", "Customer", "Items", "Payment Method", "Total Amount"})
			_ = cw.Write([]string{"INV-001", "2024-01-01T00:00:00+07:00", "", "0", "", "10000"})
			cw.Flush()
			return nil
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
	svc := &mockService{
		getSalesForExportFn: func(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error) {
			return []ExportRow{}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export?format=xlsx", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
}

func TestSaleHandler_ExportSales_CSV_Error(t *testing.T) {
	svc := &mockService{
		streamSalesExportCSVFn: func(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
			return assert.AnError
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_ExportSales_WithFilters(t *testing.T) {
	svc := &mockService{
		streamSalesExportCSVFn: func(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
			return nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/export?search=test&payment_methods=cash&min_total=100&max_total=50000&start_date=2024-01-01&end_date=2024-12-31", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_ListPaymentMethods_Success(t *testing.T) {
	svc := &mockService{
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
	svc := &mockService{
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
	svc := &mockService{
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
	svc := &mockService{
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

func TestSaleHandler_ParkSale_Success(t *testing.T) {
	var capturedSale *Sale
	svc := &mockService{
		parkSaleFn: func(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int) error {
			sale.ID = 10
			capturedSale = sale
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-PARK-001", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "CASH", Name: "Cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":2,"subtotal":20000}],"payment_method":"CASH"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, capturedSale)
	assert.Equal(t, "parked", capturedSale.Status)
}

func TestSaleHandler_ParkSale_EmptyItems(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[],"payment_method":"CASH"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_GetParkedSales_Success(t *testing.T) {
	svc := &mockService{
		listParkedSalesFn: func(ctx context.Context, cashierID int) ([]Sale, error) {
			return []Sale{
				{ID: 1, InvoiceNumber: "INV-001", Status: "parked", TotalAmount: 20000},
				{ID: 2, InvoiceNumber: "INV-002", Status: "recalled", TotalAmount: 30000},
			}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/parked", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Sale `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
}

func TestSaleHandler_GetParkedSaleByID_Success(t *testing.T) {
	svc := &mockService{
		getParkedSaleByIDFn: func(ctx context.Context, id int, cashierID int) (*Sale, error) {
			return &Sale{ID: id, InvoiceNumber: "INV-001", Status: "parked", Items: []Item{{ProductID: 1, Quantity: 2}}}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/parked/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaleHandler_GetParkedSaleByID_NotFound(t *testing.T) {
	svc := &mockService{
		getParkedSaleByIDFn: func(ctx context.Context, id int, cashierID int) (*Sale, error) {
			return nil, ErrSaleNotFound
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/parked/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSaleHandler_GetParkedSaleByID_InvalidID(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/parked/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_RecallSale_Success(t *testing.T) {
	svc := &mockService{
		recallSaleFn: func(ctx context.Context, saleID int) (*Sale, error) {
			return &Sale{ID: saleID, InvoiceNumber: "INV-001", Status: "recalled", Items: []Item{{ProductID: 1, Quantity: 2}}}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked/1/recall", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Sale `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "recalled", resp.Data.Status)
}

func TestSaleHandler_RecallSale_AlreadyRecalled(t *testing.T) {
	svc := &mockService{
		recallSaleFn: func(ctx context.Context, saleID int) (*Sale, error) {
			return nil, ErrSaleNotFound
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked/1/recall", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSaleHandler_RecallSale_InvalidID(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked/abc/recall", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_CancelParkedSale_Success(t *testing.T) {
	svc := &mockService{
		cancelParkedSaleFn: func(ctx context.Context, saleID int) error {
			return nil
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/sales/parked/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSaleHandler_CancelParkedSale_NotFound(t *testing.T) {
	svc := &mockService{
		cancelParkedSaleFn: func(ctx context.Context, saleID int) error {
			return ErrSaleNotFound
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/sales/parked/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSaleHandler_CancelParkedSale_InvalidID(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/sales/parked/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSaleHandler_CreateSale_ParkedSaleNotRecalled covers the conflict
// response when a checkout references a parked sale that is not in the
// recalled state.
func TestSaleHandler_CreateSale_ParkedSaleNotRecalled(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrParkedSaleNotRecalled
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-NOT-RECALLED-001", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "CASH", Name: "Cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":10000}],"parked_sale_id":1,"payment_method":"CASH"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "parked sale already checked out or cancelled")
}

// TestSaleHandler_ParkSale_BindJSONError covers malformed JSON on the park
// endpoint.
func TestSaleHandler_ParkSale_BindJSONError(t *testing.T) {
	svc := &mockService{}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked", strings.NewReader(`{"items":`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaleHandler_CreateSale_WithParkedSaleID(t *testing.T) {
	var capturedParkedSaleID *int
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			capturedParkedSaleID = parkedSaleID
			sale.ID = 20
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-PARKED-CHECKOUT-001", nil
		},
		getPaymentMethodByCodeFn: func(ctx context.Context, code string) (*PaymentMethod, error) {
			return &PaymentMethod{Code: "CASH", Name: "Cash"}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":2,"subtotal":20000}],"payment_method":"CASH","parked_sale_id":5}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, capturedParkedSaleID)
	assert.Equal(t, 5, *capturedParkedSaleID)
}

func TestSaleHandler_CreateSale_SplitPayments_Success(t *testing.T) {
	var capturedPayments []CreatePaymentRequest
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			capturedPayments = payments
			sale.ID = 30
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-SPLIT-001", nil
		},
		getSaleByIDFn: func(ctx context.Context, id int, storeID *int) (*Sale, error) {
			return &Sale{ID: 30, InvoiceNumber: "INV-SPLIT-001", Payments: []Payment{
				{PaymentMethodCode: "CASH", Amount: 30000},
				{PaymentMethodCode: "QRIS", Amount: 20000},
			}}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"CASH","amount":30000},{"payment_method_code":"QRIS","amount":20000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, capturedPayments, 2)
	assert.Equal(t, "CASH", capturedPayments[0].PaymentMethodCode)
	assert.Equal(t, 30000, capturedPayments[0].Amount)
	assert.Equal(t, "QRIS", capturedPayments[1].PaymentMethodCode)
	assert.Equal(t, 20000, capturedPayments[1].Amount)
}

func TestSaleHandler_CreateSale_SplitPayments_WithReference(t *testing.T) {
	var capturedPayments []CreatePaymentRequest
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			capturedPayments = payments
			sale.ID = 31
			return nil
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-SPLIT-REF-001", nil
		},
		getSaleByIDFn: func(ctx context.Context, id int, storeID *int) (*Sale, error) {
			return &Sale{ID: 31}, nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":100000}],"payments":[{"payment_method_code":"CASH","amount":60000},{"payment_method_code":"CARD","amount":40000,"reference_number":"TXN-123"}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, capturedPayments, 2)
	assert.Equal(t, "CARD", capturedPayments[1].PaymentMethodCode)
	assert.Equal(t, 40000, capturedPayments[1].Amount)
	assert.Equal(t, "TXN-123", capturedPayments[1].ReferenceNumber)
}

func TestSaleHandler_CreateSale_MissingPayments(t *testing.T) {
	svc := &mockService{
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-NOPAY-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "payments or payment_method is required")
}

func TestSaleHandler_CreateSale_SplitPayment_TotalMismatch(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrPaymentTotalMismatch
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-MISM-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"CASH","amount":30000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "total payments do not match")
}

func TestSaleHandler_CreateSale_SplitPayment_DuplicateMethod(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrDuplicatePaymentMethod
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-DUP-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"QRIS","amount":25000},{"payment_method_code":"QRIS","amount":25000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate payment method")
}

func TestSaleHandler_CreateSale_SplitPayment_MultipleCash(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrMultipleCashPayments
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-MC-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"CASH","amount":25000},{"payment_method_code":"CASH","amount":25000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "only one cash payment")
}

func TestSaleHandler_CreateSale_SplitPayment_MaxExceeded(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrMaxPaymentsExceeded
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-MAX-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":100000}],"payments":[{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000},{"payment_method_code":"CASH","amount":10000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "maximum number")
}

func TestSaleHandler_CreateSale_SplitPayment_ReferenceRequired(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrPaymentReferenceRequired
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-REF-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"CARD","amount":50000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "reference number is required")
}

func TestSaleHandler_CreateSale_SplitPayment_InvalidMethod(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrInvalidPaymentMethod
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-INV-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"BITCOIN","amount":50000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid payment method")
}

func TestSaleHandler_CreateSale_SplitPayment_InactiveMethod(t *testing.T) {
	svc := &mockService{
		createSaleWithParkedSaleFn: func(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest) error {
			return ErrPaymentMethodInactive
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-INA-001", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":1,"subtotal":50000}],"payments":[{"payment_method_code":"INACTIVE","amount":50000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "payment method is not active")
}

func TestSaleHandler_ParkSale_ServiceError(t *testing.T) {
	svc := &mockService{
		parkSaleFn: func(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int) error {
			return fmt.Errorf("unexpected error")
		},
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "INV-PARK-ERR", nil
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":2,"subtotal":20000}],"payment_method":"CASH"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_ParkSale_AutoInvoiceError(t *testing.T) {
	svc := &mockService{
		getNextInvoiceNumberFn: func(ctx context.Context) (string, error) {
			return "", fmt.Errorf("invoice generation failed")
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{"items":[{"product_id":1,"quantity":2,"subtotal":20000}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_GetParkedSales_ServiceError(t *testing.T) {
	svc := &mockService{
		listParkedSalesFn: func(ctx context.Context, cashierID int) ([]Sale, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/parked", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_GetParkedSaleByID_ServiceError(t *testing.T) {
	svc := &mockService{
		getParkedSaleByIDFn: func(ctx context.Context, id int, cashierID int) (*Sale, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sales/parked/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_RecallSale_ServiceError(t *testing.T) {
	svc := &mockService{
		recallSaleFn: func(ctx context.Context, saleID int) (*Sale, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	r := setupSaleHandler(svc, nil)
	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sales/parked/1/recall", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSaleHandler_CancelParkedSale_ServiceError(t *testing.T) {
	svc := &mockService{
		cancelParkedSaleFn: func(ctx context.Context, saleID int) error {
			return fmt.Errorf("db error")
		},
	}
	r := setupSaleHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/sales/parked/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
