package supplier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
)

func ctxWithPermissions(perms ...string) *gin.Context {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/suppliers/1/products", nil)
	c.Set("permissions", perms)
	return c
}

// TestPresentProductSupplier_HidesUnitCost pins the separation between
// supplier.view (who a product can be bought from) and product.cost.view (what
// it costs). supplier.view alone must not reveal a purchase price — the number
// is the same figure internal/product gates behind product.cost.view.
func TestPresentProductSupplier_HidesUnitCost(t *testing.T) {
	ps := ProductSupplier{ProductID: 7, SupplierID: 3, UnitCost: 8500, LeadTimeDays: 2}

	t.Run("without product.cost.view the field is absent, not null", func(t *testing.T) {
		payload, err := json.Marshal(presentProductSupplier(ps, false))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(payload, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, present := got["unit_cost"]; present {
			t.Errorf("unit_cost leaked without product.cost.view: %s", payload)
		}
		// The rest of the link is still readable; the gate is not a blank 403.
		if got["product_id"] != float64(7) || got["supplier_id"] != float64(3) {
			t.Errorf("non-cost fields missing from %s", payload)
		}
		if got["lead_time_days"] != float64(2) {
			t.Errorf("lead_time_days missing from %s", payload)
		}
	})

	t.Run("with product.cost.view the cost is returned", func(t *testing.T) {
		payload, err := json.Marshal(presentProductSupplier(ps, true))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(payload, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got["unit_cost"] != float64(8500) {
			t.Errorf("unit_cost = %v, want 8500 (payload %s)", got["unit_cost"], payload)
		}
	})
}

func TestPresentProductSuppliers_AppliesToEveryRow(t *testing.T) {
	list := []ProductSupplier{
		{ProductID: 1, SupplierID: 1, UnitCost: 100},
		{ProductID: 2, SupplierID: 2, UnitCost: 200},
	}
	payload, err := json.Marshal(presentProductSuppliers(list, false))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := countKey(string(payload), `"unit_cost"`); got != 0 {
		t.Errorf("unit_cost appeared %d times without product.cost.view: %s", got, payload)
	}

	payload, err = json.Marshal(presentProductSuppliers(list, true))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := countKey(string(payload), `"unit_cost"`); got != len(list) {
		t.Errorf("unit_cost appeared %d times, want %d", got, len(list))
	}
}

func countKey(s, key string) int {
	n := 0
	for i := 0; i+len(key) <= len(s); i++ {
		if s[i:i+len(key)] == key {
			n++
		}
	}
	return n
}

// TestCanViewUnitCost_UsesProductCostView documents that the gate is the product
// cost permission and not any supplier code, so a future role granted
// supplier.view does not silently gain purchase prices.
func TestCanViewUnitCost_UsesProductCostView(t *testing.T) {
	if canViewUnitCost(ctxWithPermissions(string(permissions.SupplierView))) {
		t.Error("supplier.view alone must not reveal unit cost")
	}
	if canViewUnitCost(ctxWithPermissions(string(permissions.SupplierView), string(permissions.SupplierUpdate))) {
		t.Error("supplier.write permissions alone must not reveal unit cost")
	}
	if !canViewUnitCost(ctxWithPermissions(string(permissions.ProductCostView))) {
		t.Error("product.cost.view should reveal unit cost")
	}
	// No superadmin bypass: canViewUnitCost is a plain permission read, and the
	// baseline does grant superadmin product.cost.view, so no exemption is needed.
	if canViewUnitCost(ctxWithPermissions()) {
		t.Error("a caller with no cost permission must not see unit cost")
	}
}
