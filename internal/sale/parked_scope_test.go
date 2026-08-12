package sale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
)

// newParkedScopeService wires a real repository + service (no mocks) so the
// P2-6 D4 parked-sale ownership/role rules run against PostgreSQL.
func newParkedScopeService(t *testing.T) Service {
	t.Helper()
	repo := newTestRepo(t)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)
	svc := NewService(repo, bus)
	svc.SetStockDeducer(inventory.StockDeducer{})
	svc.SetShiftTotalUpdater(shift.TotalUpdater{})
	svc.SetCartConfig(CartConfig{HoldTTLHours: 24})
	svc.SetPriceResolver(newPricingTestResolver())
	return svc
}

// parkedScopeAuth authenticates the caller with an explicit role + userID so the
// cashier/manager/elevated parked-sale scoping rules can be exercised over HTTP.
func parkedScopeAuth(role string, userID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("username", role+"_parked_scope_user")
		c.Set("roleID", 1)
		c.Set("role", role)
		c.Set("permissions", []string{})
		c.Set("storeID", nil)
		c.Next()
	}
}

// setupParkedScopeRouter builds the full sale router authenticated as the given
// role/user. The returned mockAuditCreator lets tests capture audit entries.
func setupParkedScopeRouter(t *testing.T, role string, userID int) (*gin.Engine, *mockAuditCreator) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newTestRepo(t)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)
	svc := NewService(repo, bus)
	svc.SetStockDeducer(inventory.StockDeducer{})
	svc.SetShiftTotalUpdater(shift.TotalUpdater{})
	svc.SetCartConfig(CartConfig{HoldTTLHours: 24})
	svc.SetPriceResolver(newPricingTestResolver())
	auditSvc := &mockAuditCreator{}
	h := NewHandler(svc, auditSvc)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), parkedScopeAuth(role, userID), testPermMiddleware)
	return r, auditSvc
}

// parkedCompleteSale builds a valid completion payload (server-authoritative
// pricing) for the given product, mirroring what the cashier/manager completes.
func parkedCompleteSale(invoice string, cashierID, prodID int) (*Sale, []Item, []CreatePaymentRequest) {
	return &Sale{
			InvoiceNumber: invoice,
			CashierID:     cashierID,
			Subtotal:      10000,
			TotalAmount:   10000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}, []Item{{
			ProductID: prodID,
			Quantity:  1,
			UnitPrice: 10000,
			Subtotal:  10000,
			DPPAmount: 10000,
			TaxAmount: 0,
		}}, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 10000}}
}

// TestSaleRepository_ParkedSaleOwnerScopeIDOR proves the repository enforces
// owner scoping on recall/cancel/fetch so one cashier cannot touch another
// cashier's parked sale, while managers/elevated roles (nil scope) can.
func TestSaleRepository_ParkedSaleOwnerScopeIDOR(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo(t)

	cashierA := insertTestCashierNamed(ctx, t, "parked_scope_repo_a")
	cashierB := insertTestCashierNamed(ctx, t, "parked_scope_repo_b")
	prodID := insertTestProduct(ctx, t, "SCOPE-PROD-001", "Scope Product", 10000, 50)

	parkedA := createParkedSale(ctx, t, repo, cashierA, "INV-SCOPE-A-001", "parked", prodID, 1, 10000)

	t.Run("other cashier cannot recall", func(t *testing.T) {
		_, err := repo.RecallSale(ctx, parkedA.ID, &cashierB, nil)
		require.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("other cashier cannot cancel", func(t *testing.T) {
		err := repo.CancelParkedSale(ctx, parkedA.ID, &cashierB, nil)
		require.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("other cashier cannot fetch by id", func(t *testing.T) {
		_, err := repo.GetParkedSaleByID(ctx, parkedA.ID, &cashierB, nil)
		require.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("other cashier list is scoped to own sales", func(t *testing.T) {
		sales, err := repo.GetParkedSales(ctx, &cashierB, nil)
		require.NoError(t, err)
		assert.Empty(t, sales)
	})

	t.Run("owner can recall own sale", func(t *testing.T) {
		recalled, err := repo.RecallSale(ctx, parkedA.ID, &cashierA, nil)
		require.NoError(t, err)
		assert.Equal(t, "recalled", recalled.Status)
	})

	t.Run("manager (nil scope) can recall and cancel any cashier's sale", func(t *testing.T) {
		parkedB := createParkedSale(ctx, t, repo, cashierB, "INV-SCOPE-B-001", "parked", prodID, 1, 10000)
		parkedC := createParkedSale(ctx, t, repo, cashierB, "INV-SCOPE-C-001", "parked", prodID, 1, 10000)

		_, err := repo.RecallSale(ctx, parkedB.ID, nil, nil)
		require.NoError(t, err)
		err = repo.CancelParkedSale(ctx, parkedC.ID, nil, nil)
		require.NoError(t, err)

		sales, err := repo.GetParkedSales(ctx, nil, nil)
		require.NoError(t, err)
		var invoices []string
		for _, s := range sales {
			invoices = append(invoices, s.InvoiceNumber)
		}
		assert.Contains(t, invoices, parkedA.InvoiceNumber, "recalled stays visible to manager")
		assert.Contains(t, invoices, parkedB.InvoiceNumber, "recalled stays visible to manager")
		assert.NotContains(t, invoices, parkedC.InvoiceNumber, "cancelled is excluded")
	})
}

// TestSaleRepository_ParkedSaleReturnsCustomerAndNote proves the customer and
// hold_note set at park time survive the list and by-id reads (P2-6 E1).
func TestSaleRepository_ParkedSaleReturnsCustomerAndNote(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo(t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProduct(ctx, t, "NOTE-PROD-001", "Note Product", 10000, 50)

	var customerID int
	err := dbPool.QueryRow(ctx, `INSERT INTO customers (name, phone, email) VALUES ('Parked Caller', '0812-PARKED', 'parked@test.com') RETURNING id`).Scan(&customerID)
	require.NoError(t, err)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	sale := &Sale{
		InvoiceNumber: "INV-NOTE-001",
		CashierID:     cashierID,
		Subtotal:      20000,
		TotalAmount:   20000,
		PaymentMethod: "CASH",
		Status:        "parked",
		CustomerID:    &customerID,
		HoldNote:      "hold for rina",
	}
	items := []Item{{ProductID: prodID, Quantity: 2, UnitPrice: 10000, Subtotal: 20000, DPPAmount: 20000, TaxAmount: 0}}
	require.NoError(t, repo.CreateSale(ctx, tx, sale, items))
	require.NoError(t, tx.Commit(ctx))

	t.Run("list returns customer and note", func(t *testing.T) {
		sales, err := repo.GetParkedSales(ctx, &cashierID, nil)
		require.NoError(t, err)
		require.Len(t, sales, 1)
		assert.Equal(t, "hold for rina", sales[0].HoldNote)
		require.NotNil(t, sales[0].CustomerID)
		assert.Equal(t, customerID, *sales[0].CustomerID)
	})

	t.Run("by id returns customer and note", func(t *testing.T) {
		byID, err := repo.GetParkedSaleByID(ctx, sale.ID, &cashierID, nil)
		require.NoError(t, err)
		assert.Equal(t, "hold for rina", byID.HoldNote)
		require.NotNil(t, byID.CustomerID)
		assert.Equal(t, customerID, *byID.CustomerID)
	})
}

// TestSaleService_ParkedSaleManagerRules proves the P2-6 D4 service rules:
// manager recall-only, no blanket completion, scoped consumption for cashiers,
// and elevated cancel-any.
func TestSaleService_ParkedSaleManagerRules(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc := newParkedScopeService(t)

	cashierA := insertTestCashierNamed(ctx, t, "parked_scope_svc_a")
	cashierB := insertTestCashierNamed(ctx, t, "parked_scope_svc_b")
	managerID := insertTestCashierNamed(ctx, t, "parked_scope_svc_manager")
	prodID := insertTestProduct(ctx, t, "SCOPE-SVC-PROD", "Scope Svc Product", 10000, 50)

	parkedA := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-SVC-A-001", "parked", prodID, 1, 10000)
	parkedB := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-SVC-B-001", "parked", prodID, 1, 10000)

	manager := Caller{Role: "manager", UserID: managerID}
	superadmin := Caller{Role: "superadmin"}
	admin := Caller{Role: "admin"}
	cashierBCaller := Caller{UserID: cashierB}

	t.Run("manager cannot cancel a parked sale", func(t *testing.T) {
		err := svc.CancelParkedSale(ctx, parkedA.ID, manager)
		require.ErrorIs(t, err, ErrPermissionDenied)
	})

	t.Run("manager completion without parked_sale_id is denied", func(t *testing.T) {
		sale, items, payments := parkedCompleteSale("INV-SVC-DENIED", managerID, prodID)
		err := svc.CreateSaleWithParkedSale(ctx, sale, items, nil, payments, manager)
		require.ErrorIs(t, err, ErrPermissionDenied)
	})

	t.Run("manager can recall another cashier's sale", func(t *testing.T) {
		recalled, err := svc.RecallSale(ctx, parkedB.ID, manager)
		require.NoError(t, err)
		assert.Equal(t, "recalled", recalled.Status)
	})

	t.Run("manager can complete a recalled sale", func(t *testing.T) {
		sale, items, payments := parkedCompleteSale("INV-SVC-MGR-DONE", managerID, prodID)
		err := svc.CreateSaleWithParkedSale(ctx, sale, items, &parkedB.ID, payments, manager)
		require.NoError(t, err)
	})

	t.Run("cashier cannot consume another cashier's recalled sale", func(t *testing.T) {
		parkedC := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-SVC-C-001", "parked", prodID, 1, 10000)
		recalledC, err := svc.RecallSale(ctx, parkedC.ID, manager)
		require.NoError(t, err)

		sale, items, payments := parkedCompleteSale("INV-SVC-STOLEN", cashierB, prodID)
		err = svc.CreateSaleWithParkedSale(ctx, sale, items, &recalledC.ID, payments, cashierBCaller)
		require.ErrorIs(t, err, ErrSaleNotFound, "consume is owner-scoped for cashiers")
	})

	t.Run("cashier can complete own recalled sale", func(t *testing.T) {
		parkedD := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-SVC-D-001", "parked", prodID, 1, 10000)
		recalledD, err := svc.RecallSale(ctx, parkedD.ID, Caller{UserID: cashierA})
		require.NoError(t, err)

		sale, items, payments := parkedCompleteSale("INV-SVC-OWN-DONE", cashierA, prodID)
		err = svc.CreateSaleWithParkedSale(ctx, sale, items, &recalledD.ID, payments, Caller{UserID: cashierA})
		require.NoError(t, err)
	})

	t.Run("elevated roles can cancel any parked sale", func(t *testing.T) {
		parkedE := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-SVC-E-001", "parked", prodID, 1, 10000)
		parkedF := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-SVC-F-001", "parked", prodID, 1, 10000)
		require.NoError(t, svc.CancelParkedSale(ctx, parkedE.ID, superadmin))
		require.NoError(t, svc.CancelParkedSale(ctx, parkedF.ID, admin))
	})
}

// TestParkedScope_HandlerOwnershipIDOR exercises the HTTP surface: another
// cashier must get 404 on recall/cancel/detail, manager recall works and is
// audited, and manager cancel is forbidden.
func TestParkedScope_HandlerOwnershipIDOR(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo(t)

	cashierA := insertTestCashierNamed(ctx, t, "parked_http_cashier_a")
	cashierB := insertTestCashierNamed(ctx, t, "parked_http_cashier_b")
	managerID := insertTestCashierNamed(ctx, t, "parked_http_manager")
	prodID := insertTestProduct(ctx, t, "SCOPE-HTTP-PROD", "HTTP Scope Product", 10000, 50)

	parkedA := createParkedSale(ctx, t, repo, cashierA, "INV-HTTP-A-001", "parked", prodID, 1, 10000)

	t.Run("other cashier gets 404 everywhere", func(t *testing.T) {
		r, _ := setupParkedScopeRouter(t, "cashier", cashierB)

		req := httptest.NewRequest("POST", fmt.Sprintf("/sales/parked/%d/recall", parkedA.ID), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		req = httptest.NewRequest("DELETE", fmt.Sprintf("/sales/parked/%d", parkedA.ID), nil)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		req = httptest.NewRequest("GET", fmt.Sprintf("/sales/parked/%d", parkedA.ID), nil)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		req = httptest.NewRequest("GET", "/sales/parked", nil)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var list struct {
			Data []Sale `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Empty(t, list.Data, "cashier must not see another cashier's parked sales")
	})

	t.Run("manager recall succeeds and is audited, cancel forbidden", func(t *testing.T) {
		r, auditSvc := setupParkedScopeRouter(t, "manager", managerID)
		var logs []*audit.Log
		auditSvc.createAuditLogFn = func(c context.Context, l *audit.Log) error {
			logs = append(logs, l)
			return nil
		}

		req := httptest.NewRequest("POST", fmt.Sprintf("/sales/parked/%d/recall", parkedA.ID), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp struct {
			Data Sale `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "recalled", resp.Data.Status)

		var recallAudit *audit.Log
		for _, l := range logs {
			if l.Action == "recall_sale" && l.EntityID != nil && *l.EntityID == parkedA.ID {
				recallAudit = l
			}
		}
		require.NotNil(t, recallAudit, "manager recall must write recall_sale audit log")

		req = httptest.NewRequest("DELETE", fmt.Sprintf("/sales/parked/%d", parkedA.ID), nil)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("superadmin can cancel any parked sale", func(t *testing.T) {
		r, _ := setupParkedScopeRouter(t, "superadmin", managerID)
		parkedS := createParkedSale(ctx, t, repo, cashierA, "INV-HTTP-S-001", "parked", prodID, 1, 10000)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/sales/parked/%d", parkedS.ID), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

// TestParkedScope_HandlerCompletion covers the dedicated complete route: a
// manager completing a recalled sale gets 201 + audit, another cashier gets 404.
func TestParkedScope_HandlerCompletion(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo(t)

	cashierA := insertTestCashierNamed(ctx, t, "parked_complete_cashier_a")
	cashierB := insertTestCashierNamed(ctx, t, "parked_complete_cashier_b")
	managerID := insertTestCashierNamed(ctx, t, "parked_complete_manager")
	prodID := insertTestProduct(ctx, t, "COMPLETE-PROD-001", "Complete Product", 10000, 50)

	body := fmt.Sprintf(`{"items":[{"product_id":%d,"quantity":1}],"payments":[{"payment_method_code":"CASH","amount":10000}]}`, prodID)

	t.Run("manager completion is created and audited", func(t *testing.T) {
		parked := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-HTTP-COMP-001", "parked", prodID, 1, 10000)
		parked = mustRecallParked(t, repo, parked)

		r, auditSvc := setupParkedScopeRouter(t, "manager", managerID)
		var logs []*audit.Log
		auditSvc.createAuditLogFn = func(c context.Context, l *audit.Log) error {
			logs = append(logs, l)
			return nil
		}

		req := httptest.NewRequest("POST", fmt.Sprintf("/sales/parked/%d/complete", parked.ID), strings.NewReader(body))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		var resp struct {
			Data Sale `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "completed", resp.Data.Status)
		assert.Equal(t, managerID, resp.Data.CashierID)

		found := false
		for _, l := range logs {
			if l.Action == "complete_parked_sale" && l.EntityID != nil && *l.EntityID == resp.Data.ID {
				found = true
			}
		}
		assert.True(t, found, "manager completion must write complete_parked_sale audit log")
	})

	t.Run("other cashier cannot complete a recalled sale", func(t *testing.T) {
		parked := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-HTTP-COMP-002", "parked", prodID, 1, 10000)
		parked = mustRecallParked(t, repo, parked)

		r, _ := setupParkedScopeRouter(t, "cashier", cashierB)
		req := httptest.NewRequest("POST", fmt.Sprintf("/sales/parked/%d/complete", parked.ID), strings.NewReader(body))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		// The aborted completion must not consume the parked sale or persist a
		// completed sale: the parked sale is still recalled and owner-accessible.
		byOwner, err := repo.GetParkedSaleByID(ctx, parked.ID, &cashierA, nil)
		require.NoError(t, err)
		assert.Equal(t, "recalled", byOwner.Status)
	})

	t.Run("owner cashier completion still works", func(t *testing.T) {
		parked := createParkedSale(ctx, t, newTestRepo(t), cashierA, "INV-HTTP-COMP-003", "parked", prodID, 1, 10000)
		parked = mustRecallParked(t, repo, parked)

		r, _ := setupParkedScopeRouter(t, "cashier", cashierA)
		req := httptest.NewRequest("POST", fmt.Sprintf("/sales/parked/%d/complete", parked.ID), strings.NewReader(body))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	})
}

// mustRecallParked recalls a parked sale via the owner scope, using the fresh
// repo where the parked sale was created (newTestRepo is stateless/shared pool).
func mustRecallParked(t *testing.T, repo *Repository, sale *Sale) *Sale {
	t.Helper()
	recalled, err := repo.RecallSale(context.Background(), sale.ID, nil, nil)
	require.NoError(t, err)
	return recalled
}

// createParkedSaleWithStore mirrors createParkedSale but stamps a store_id so
// the store-scoping rules (P2-6 D4) can be exercised.
func createParkedSaleWithStore(ctx context.Context, t *testing.T, repo *Repository, cashierID, storeID int, invoice, status string, prodID, qty, price int) *Sale {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	sale := &Sale{
		InvoiceNumber: invoice,
		CashierID:     cashierID,
		StoreID:       &storeID,
		Subtotal:      price * qty,
		TotalAmount:   price * qty,
		PaymentMethod: "CASH",
		Status:        status,
	}
	items := []Item{{
		ProductID: prodID,
		Quantity:  qty,
		UnitPrice: price,
		Subtotal:  price * qty,
		DPPAmount: price * qty,
		TaxAmount: 0,
	}}
	require.NoError(t, repo.CreateSale(ctx, tx, sale, items))
	require.NoError(t, tx.Commit(ctx))
	return sale
}

// TestSaleRepository_ParkedSaleStoreScope proves that manager/elevated access
// (nil owner scope) is still confined to the caller's store, so a store-scoped
// manager cannot read or mutate another store's parked sales (P2-6 D4).
func TestSaleRepository_ParkedSaleStoreScope(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo(t)

	cashierA := insertTestCashierNamed(ctx, t, "parked_store_cashier_a")
	prodID := insertTestProduct(ctx, t, "STORE-SCOPE-PROD", "Store Scope Product", 10000, 50)

	var storeA, storeB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Store A') RETURNING id`).Scan(&storeA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Store B') RETURNING id`).Scan(&storeB))

	createParkedSaleWithStore(ctx, t, repo, cashierA, storeA, "INV-STORE-A-001", "parked", prodID, 1, 10000)
	parkedB := createParkedSaleWithStore(ctx, t, repo, cashierA, storeB, "INV-STORE-B-001", "parked", prodID, 1, 10000)

	t.Run("manager list is scoped to own store", func(t *testing.T) {
		sales, err := repo.GetParkedSales(ctx, nil, &storeA)
		require.NoError(t, err)
		require.Len(t, sales, 1)
		assert.Equal(t, "INV-STORE-A-001", sales[0].InvoiceNumber)
	})

	t.Run("manager get by id is scoped to own store", func(t *testing.T) {
		_, err := repo.GetParkedSaleByID(ctx, parkedB.ID, nil, &storeA)
		require.ErrorIs(t, err, ErrSaleNotFound)
		sale, err := repo.GetParkedSaleByID(ctx, parkedB.ID, nil, &storeB)
		require.NoError(t, err)
		assert.Equal(t, parkedB.ID, sale.ID)
	})

	t.Run("manager recall is scoped to own store", func(t *testing.T) {
		_, err := repo.RecallSale(ctx, parkedB.ID, nil, &storeA)
		require.ErrorIs(t, err, ErrSaleNotFound)
		recalled, err := repo.RecallSale(ctx, parkedB.ID, nil, &storeB)
		require.NoError(t, err)
		assert.Equal(t, "recalled", recalled.Status)
	})

	t.Run("manager cancel is scoped to own store", func(t *testing.T) {
		parkedC := createParkedSaleWithStore(ctx, t, repo, cashierA, storeB, "INV-STORE-C-001", "parked", prodID, 1, 10000)
		err := repo.CancelParkedSale(ctx, parkedC.ID, nil, &storeA)
		require.ErrorIs(t, err, ErrSaleNotFound)
		err = repo.CancelParkedSale(ctx, parkedC.ID, nil, &storeB)
		require.NoError(t, err)
	})

	t.Run("cross-store completion is rejected", func(t *testing.T) {
		recalled := createParkedSaleWithStore(ctx, t, repo, cashierA, storeB, "INV-STORE-D-001", "parked", prodID, 1, 10000)
		recalled = mustRecallParked(t, repo, recalled)

		svc := newParkedScopeService(t)
		sale, items, payments := parkedCompleteSale("INV-STORE-D-DONE", cashierA, prodID)
		storeAID := storeA
		err := svc.CreateSaleWithParkedSale(ctx, sale, items, &recalled.ID, payments, Caller{Role: "manager", StoreID: &storeAID})
		require.ErrorIs(t, err, ErrParkedSaleNotRecalled, "manager must not complete another store's recalled sale")

		storeBID := storeB
		err = svc.CreateSaleWithParkedSale(ctx, sale, items, &recalled.ID, payments, Caller{Role: "manager", StoreID: &storeBID})
		require.NoError(t, err, "manager completes a recalled sale in their own store")
	})
}
