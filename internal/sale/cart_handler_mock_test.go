package sale

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"
)

// mockCartFixture builds a mock service wired for successful cart flows so
// individual tests can override the fn they want to exercise.
func mockCartFixture() *mockService {
	cart := &CartSession{
		ID:        7,
		CashierID: 1,
		Status:    "open",
		Items: []CartItem{{
			ID:            1,
			CartSessionID: 7,
			ProductID:     2,
			ProductName:   "Mock Product",
			Quantity:      2,
			UnitPrice:     10000,
			Cost:          5000,
		}},
	}
	sale := &Sale{
		ID:            99,
		InvoiceNumber: "INV-MOCK-001",
		TotalAmount:   20000,
		Items:         []Item{{ProductID: 2, Quantity: 2, UnitPrice: 10000, Cost: 5000}},
	}
	return &mockService{
		createOrGetOpenCartFn: func(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
			return cart, nil
		},
		getOpenCartFn: func(ctx context.Context, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		getCartByIDFn: func(ctx context.Context, cartID, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		listHeldCartsFn: func(ctx context.Context, cashierID int) ([]CartSession, error) {
			return []CartSession{*cart}, nil
		},
		updateCartCustomerFn: func(ctx context.Context, cartID int, customerID *int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		addCartItemFn: func(ctx context.Context, cartID, productID, quantity int, customerGroupID *int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		updateCartItemQuantityFn: func(ctx context.Context, cartID, itemID, quantity int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		removeCartItemFn: func(ctx context.Context, cartID, itemID int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		holdCartFn: func(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		resumeCartFn: func(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		cancelCartFn: func(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
			return cart, nil
		},
		checkoutCartFn: func(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
			return sale, nil
		},
	}
}

// setupSaleCartHandler builds a router with cart routes only. When withUser is
// false the userID context value is omitted so 401 paths can be exercised.
func setupSaleCartHandler(svc Service, auditSvc audit.Creator, withUser bool, perms []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if withUser {
			c.Set("userID", 1)
		}
		c.Set("storeID", nil)
		c.Set("permissions", perms)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterCartRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestSaleCartHandler_CreateOrGetOpenCart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart", strings.NewReader(`{"shift_id":5}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid json falls back to empty request", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, false, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.createOrGetOpenCartFn = func(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSaleCartHandler_GetOpenCart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("not found", func(t *testing.T) {
		svc := mockCartFixture()
		svc.getOpenCartFn = func(ctx context.Context, cashierID int) (*CartSession, error) {
			return nil, ErrCartNotFound
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"data":null`)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, false, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSaleCartHandler_ListHeldCarts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart/held", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.listHeldCartsFn = func(ctx context.Context, cashierID int) ([]CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart/held", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSaleCartHandler_GetCart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart/7", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart/abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, false, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart/7", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.getCartByIDFn = func(ctx context.Context, cartID, cashierID int) (*CartSession, error) {
			return nil, ErrCartNotOwned
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pos/cart/7", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestSaleCartHandler_AddCartItem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/items", strings.NewReader(`{"product_id":2,"quantity":2}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("success with audit log", func(t *testing.T) {
		auditCalls := 0
		auditSvc := &mockAuditCreator{createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalls++
			assert.Equal(t, "add", log.Action)
			return nil
		}}
		r := setupSaleCartHandler(mockCartFixture(), auditSvc, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/items", strings.NewReader(`{"product_id":2,"quantity":3}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, auditCalls)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/items", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing quantity fails binding", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/items", strings.NewReader(`{"product_id":2}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("open cart error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.createOrGetOpenCartFn = func(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/items", strings.NewReader(`{"product_id":2,"quantity":2}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("insufficient stock error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.addCartItemFn = func(ctx context.Context, cartID, productID, quantity int, customerGroupID *int, cashierID int) (*CartSession, error) {
			return nil, ErrInsufficientStock
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/items", strings.NewReader(`{"product_id":2,"quantity":2}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestSaleCartHandler_UpdateCartItemQuantity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/items/1", strings.NewReader(`{"quantity":5}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid item id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/items/abc", strings.NewReader(`{"quantity":5}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/items/1", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("open cart error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.getOpenCartFn = func(ctx context.Context, cashierID int) (*CartSession, error) {
			return nil, ErrCartNotFound
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/items/1", strings.NewReader(`{"quantity":5}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("item not found error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.updateCartItemQuantityFn = func(ctx context.Context, cartID, itemID, quantity int, cashierID int) (*CartSession, error) {
			return nil, ErrCartItemNotFound
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/items/99", strings.NewReader(`{"quantity":5}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestSaleCartHandler_RemoveCartItem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pos/cart/items/1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid item id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pos/cart/items/abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("open cart error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.getOpenCartFn = func(ctx context.Context, cashierID int) (*CartSession, error) {
			return nil, ErrCartExpired
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pos/cart/items/1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.removeCartItemFn = func(ctx context.Context, cartID, itemID int, cashierID int) (*CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pos/cart/items/1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSaleCartHandler_UpdateCartCustomer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/7/customer", strings.NewReader(`{"customer_id":3}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/abc/customer", strings.NewReader(`{"customer_id":3}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/7/customer", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.updateCartCustomerFn = func(ctx context.Context, cartID int, customerID *int, cashierID int) (*CartSession, error) {
			return nil, ErrCartNotOwned
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/pos/cart/7/customer", strings.NewReader(`{"customer_id":3}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestSaleCartHandler_HoldCart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/hold", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/abc/hold", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.holdCartFn = func(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/hold", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSaleCartHandler_ResumeCart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/resume", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/abc/resume", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.resumeCartFn = func(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/resume", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSaleCartHandler_CancelCart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/cancel", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"id":7`)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/abc/cancel", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.cancelCartFn = func(ctx context.Context, cartID int, cashierID int) (*CartSession, error) {
			return nil, fmt.Errorf("boom")
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/cancel", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSaleCartHandler_CheckoutCart(t *testing.T) {
	t.Run("success with detail", func(t *testing.T) {
		svc := mockCartFixture()
		svc.getSaleByIDFn = func(ctx context.Context, id int, storeID *int) (*Sale, error) {
			return &Sale{ID: 99, InvoiceNumber: "INV-MOCK-001", TotalAmount: 20000}, nil
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/checkout", strings.NewReader(`{"payments":[{"method":"CASH","amount":20000}]}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success falls back to cart sale", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/checkout", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success with audit log", func(t *testing.T) {
		auditCalls := 0
		auditSvc := &mockAuditCreator{createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalls++
			assert.Equal(t, "checkout", log.Action)
			return nil
		}}
		r := setupSaleCartHandler(mockCartFixture(), auditSvc, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/checkout", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, 1, auditCalls)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/abc/checkout", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupSaleCartHandler(mockCartFixture(), nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/checkout", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("payment mismatch error", func(t *testing.T) {
		svc := mockCartFixture()
		svc.checkoutCartFn = func(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error) {
			return nil, ErrPaymentTotalMismatch
		}
		r := setupSaleCartHandler(svc, nil, true, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pos/cart/7/checkout", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSaleCartHandler_NonIntUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "not-an-int")
		c.Set("storeID", nil)
		c.Set("permissions", []string{})
		c.Next()
	})
	h := NewHandler(mockCartFixture(), nil)
	h.RegisterCartRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pos/cart", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user ID in context")
}
