package sale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/secregtest"
	"retail-pos-system/internal/shared"
)

type lookupSummary struct {
	ID            int    `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	CashierID     int    `json:"cashier_id"`
	CashierName   string `json:"cashier_name"`
	TotalAmount   int    `json:"total_amount"`
	Status        string `json:"status"`
}

type lookupPage struct {
	Data  []lookupSummary `json:"data"`
	Total int             `json:"total"`
}

// TestLookup_CrossCashier_SearchableByForeignCashier asserts that sale.lookup
// returns transactions from all cashiers (not scoped to the caller), while the
// traditional sale.view endpoint remains scoped to the caller's own sales.
func TestLookup_CrossCashier_SearchableByForeignCashier(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := newTestRepo(t)
	prodID := insertRegressionProduct(ctx, t, "LOOKUP-PROD", "Lookup Product", 10000, 6000, 10)

	// saleA belongs to a foreign cashier; saleB is reassigned to the caller.
	saleA := createRegressionSale(ctx, t, repo, "INV-LOOKUP-A", prodID, 1, 10000, 6000)
	saleB := createRegressionSale(ctx, t, repo, "INV-LOOKUP-B", prodID, 2, 20000, 7000)

	lookupRouter := setupSaleRouterWithPerms(t, []string{permissions.SaleLookup.String()})
	_, err := dbPool.Exec(ctx, `UPDATE sales SET cashier_id = $1 WHERE id = $2`, int(testCashierID), saleB.ID)
	require.NoError(t, err)

	// Cross-cashier lookup surfaces BOTH foreign and own sales.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sales/lookup", nil)
	lookupRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var page lookupPage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	invoices := make(map[string]bool)
	for _, s := range page.Data {
		invoices[s.InvoiceNumber] = true
	}
	assert.True(t, invoices[saleA.InvoiceNumber], "foreign cashier sale must be visible via lookup")
	assert.True(t, invoices[saleB.InvoiceNumber], "own sale must be visible via lookup")
	assert.GreaterOrEqual(t, page.Total, 2)

	// The traditional My Transactions view stays scoped to the caller's own sales.
	historyRouter := setupSaleRouterWithPerms(t, []string{permissions.SaleView.String()})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/sales", nil)
	historyRouter.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var hist lookupPage
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &hist))
	histInvoices := make(map[string]bool)
	for _, s := range hist.Data {
		histInvoices[s.InvoiceNumber] = true
	}
	assert.True(t, histInvoices[saleB.InvoiceNumber], "own sale must be visible via history")
	assert.False(t, histInvoices[saleA.InvoiceNumber], "foreign cashier sale must NOT be visible via history-lookup")
}

// TestLookup_RedactedShape asserts the lookup endpoint returns a redacted
// summary only — no items, cost, customer PII, or payment details.
func TestLookup_RedactedShape(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := newTestRepo(t)
	prodID := insertRegressionProduct(ctx, t, "LOOKUP-REDACT-PROD", "Lookup Redact Product", 10000, 6000, 10)
	createRegressionSale(ctx, t, repo, "INV-LOOKUP-REDACT", prodID, 1, 10000, 6000)

	r := setupSaleRouterWithPerms(t, []string{permissions.SaleLookup.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sales/lookup", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	secregtest.Check(t, w.Body.Bytes(),
		secregtest.Visible("data.0.invoice_number"),
		secregtest.Visible("data.0.cashier_id"),
		secregtest.Visible("data.0.total_amount"),
		secregtest.Visible("data.0.status"),
		secregtest.Absent("data.0.items"),
		secregtest.Absent("data.0.customer_name"),
		secregtest.Absent("data.0.payments"),
		secregtest.Absent("data.0.subtotal"),
		secregtest.Absent("data.0.discount"),
		secregtest.Absent("data.0.tax"),
	)
}

// TestLookup_RequiresPermission asserts that sale.lookup is enforced at the
// route level — a caller without it gets 403, not a silent empty list.
func TestLookup_RequiresPermission(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo(t)
	ctx := context.Background()
	prodID := insertRegressionProduct(ctx, t, "LOOKUP-PERM-PROD", "Lookup Perm Product", 10000, 6000, 10)
	createRegressionSale(ctx, t, repo, "INV-LOOKUP-PERM", prodID, 1, 10000, 6000)

	r := setupSaleRouterWithPerms(t, []string{permissions.SaleView.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sales/lookup", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestLookup_SearchByInvoice exercises the search + pagination path of the
// lookup endpoint to ensure it behaves like a real query surface.
func TestLookup_SearchByInvoice(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := newTestRepo(t)
	prodID := insertRegressionProduct(ctx, t, "LOOKUP-SEARCH-PROD", "Lookup Search Product", 10000, 6000, 10)
	createRegressionSale(ctx, t, repo, "INV-LOOKUP-SEARCH", prodID, 1, 10000, 6000)

	r := setupSaleRouterWithPerms(t, []string{permissions.SaleLookup.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sales/lookup?search="+strconv.Itoa(12345), nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var empty lookupPage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &empty))
	assert.Equal(t, 0, empty.Total)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/sales/lookup?search=INV-LOOKUP-SEARCH", nil)
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var found lookupPage
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &found))
	assert.Equal(t, 1, found.Total)
	assert.Equal(t, "INV-LOOKUP-SEARCH", found.Data[0].InvoiceNumber)
}
