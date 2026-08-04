package permissions

import (
	"strings"
	"testing"
)

func TestRegistryCount(t *testing.T) {
	got := len(All())
	if got != 72 {
		t.Fatalf("registry has %d codes, want 72 (sync with database/permission-matrix-final.md)", got)
	}
}

func TestRegistryUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(All()))
	for _, c := range All() {
		s := c.String()
		if strings.TrimSpace(s) == "" {
			t.Errorf("empty permission code found")
			continue
		}
		if _, dup := seen[s]; dup {
			t.Errorf("duplicate permission code: %s", s)
		}
		seen[s] = struct{}{}
	}
}

func TestRegistryFormat(t *testing.T) {
	for _, c := range All() {
		parts := strings.Split(c.String(), ".")
		if len(parts) != 2 {
			t.Errorf("code %q is not in <module>.<action> format", c)
		}
		for _, p := range parts {
			if p == "" {
				t.Errorf("code %q contains an empty segment", c)
			}
		}
	}
}

func TestExists(t *testing.T) {
	if !Exists(ProductView) {
		t.Errorf("Exists(ProductView) = false, want true")
	}
	if !Exists(StockOpnamePost) {
		t.Errorf("Exists(StockOpnamePost) = false, want true")
	}
	if Exists(Code("product.nonexistent")) {
		t.Errorf("Exists(nonexistent) = true, want false")
	}
	if Exists(Code("")) {
		t.Errorf("Exists(empty) = true, want false")
	}
}

func TestSpotCheckCodes(t *testing.T) {
	spot := []Code{
		DashboardView, UserView, UserCreate, UserUpdate, UserDelete,
		RoleView, RoleCreate, RoleUpdate, RoleDelete,
		AuditView, ReportView,
		ProductView, ProductCreate, ProductUpdate, ProductDelete, ProductExport, ProductImport,
		CategoryView, CategoryCreate, CategoryUpdate, CategoryDelete, CategoryExport, CategoryImport,
		SaleView, SaleCreate, SalePark,
		ShiftView, ShiftCreate, ShiftReview, ShiftAudit,
		CustomerView, CustomerCreate, CustomerUpdate, CustomerDelete, CustomerExport, CustomerImport,
		PricingView, PricingCreate, PricingUpdate, PricingDelete,
		InventoryAdjust,
		StoreView, StoreCreate, StoreUpdate, StoreDelete,
		CustomerGroupView, CustomerGroupCreate, CustomerGroupUpdate, CustomerGroupDelete,
		PurchaseOrderView, PurchaseOrderCreate, PurchaseOrderUpdate, PurchaseOrderDelete, PurchaseOrderConfirm, PurchaseOrderReceive, PurchaseOrderCancel,
		StockOpnameView, StockOpnameCreate, StockOpnameAssign, StockOpnameCount, StockOpnameSubmit, StockOpnameRecount, StockOpnameCancel, StockOpnameExport, StockOpnameVerify, StockOpnamePost, StockOpnameClose, StockOpnameReport,
		StorageLocationView, StorageLocationCreate, StorageLocationUpdate, StorageLocationDelete,
	}
	if len(spot) != 72 {
		t.Fatalf("spot-check list has %d entries, want 72", len(spot))
	}
	for _, c := range spot {
		if !Exists(c) {
			t.Errorf("spot-check code %q not registered", c)
		}
	}
}

func TestAllCoversSpotCheck(t *testing.T) {
	allSet := make(map[Code]struct{}, len(All()))
	for _, c := range All() {
		allSet[c] = struct{}{}
	}
	spot := []Code{ProductImport, StockOpnameVerify, PurchaseOrderReceive, StorageLocationDelete, CustomerGroupCreate}
	for _, c := range spot {
		if _, ok := allSet[c]; !ok {
			t.Errorf("code %q missing from All()", c)
		}
	}
}
