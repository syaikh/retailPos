// Package permissions is the single source of truth for permission codes on
// the backend.
//
// Sync source: database (permissions table, 72 live codes) — see
// docs/audits/permission-matrix-final.md (approved 2026-08-04) and
// web/src/shared/constants/permissions.ts (frontend mirror).
//
// Sprint 1 rules:
//   - Do NOT add/remove/rename permission codes here without a migration.
//   - Route registration must use these typed constants — passing a raw
//     string to RequirePermission is a compile error.
//   - Changes only via backlog, not mid-sprint.
package permissions

// Code is a typed permission code. Using a distinct type (rather than a
// plain string) makes any raw string literal a compile error at call sites
// such as RequirePermission(perm(...)) and modulePerms.
type Code string

// String returns the raw permission code value.
func (c Code) String() string {
	return string(c)
}

const (
	DashboardView Code = "dashboard.view"

	UserView   Code = "user.view"
	UserCreate Code = "user.create"
	UserUpdate Code = "user.update"
	UserDelete Code = "user.delete"

	RoleView   Code = "role.view"
	RoleCreate Code = "role.create"
	RoleUpdate Code = "role.update"
	RoleDelete Code = "role.delete"

	AuditView Code = "audit.view"

	ReportView Code = "report.view"

	ProductView   Code = "product.view"
	ProductCreate Code = "product.create"
	ProductUpdate Code = "product.update"
	ProductDelete Code = "product.delete"
	ProductExport Code = "product.export"
	ProductImport Code = "product.import"

	CategoryView   Code = "category.view"
	CategoryCreate Code = "category.create"
	CategoryUpdate Code = "category.update"
	CategoryDelete Code = "category.delete"
	CategoryExport Code = "category.export"
	CategoryImport Code = "category.import"

	SaleView   Code = "sale.view"
	SaleCreate Code = "sale.create"
	SalePark   Code = "sale.park"

	ShiftView   Code = "shift.view"
	ShiftCreate Code = "shift.create"
	ShiftReview Code = "shift.review"
	ShiftAudit  Code = "shift.audit"

	CustomerView   Code = "customer.view"
	CustomerCreate Code = "customer.create"
	CustomerUpdate Code = "customer.update"
	CustomerDelete Code = "customer.delete"
	CustomerExport Code = "customer.export"
	CustomerImport Code = "customer.import"

	PricingView   Code = "pricing.view"
	PricingCreate Code = "pricing.create"
	PricingUpdate Code = "pricing.update"
	PricingDelete Code = "pricing.delete"

	InventoryAdjust Code = "inventory.adjust"

	StoreView   Code = "store.view"
	StoreCreate Code = "store.create"
	StoreUpdate Code = "store.update"
	StoreDelete Code = "store.delete"

	CustomerGroupView   Code = "customer_group.view"
	CustomerGroupCreate Code = "customer_group.create"
	CustomerGroupUpdate Code = "customer_group.update"
	CustomerGroupDelete Code = "customer_group.delete"

	PurchaseOrderView    Code = "purchase_order.view"
	PurchaseOrderCreate  Code = "purchase_order.create"
	PurchaseOrderUpdate  Code = "purchase_order.update"
	PurchaseOrderDelete  Code = "purchase_order.delete"
	PurchaseOrderConfirm Code = "purchase_order.confirm"
	PurchaseOrderReceive Code = "purchase_order.receive"
	PurchaseOrderCancel  Code = "purchase_order.cancel"

	StockOpnameView    Code = "stock_opname.view"
	StockOpnameCreate  Code = "stock_opname.create"
	StockOpnameAssign  Code = "stock_opname.assign"
	StockOpnameCount   Code = "stock_opname.count"
	StockOpnameSubmit  Code = "stock_opname.submit"
	StockOpnameRecount Code = "stock_opname.recount"
	StockOpnameCancel  Code = "stock_opname.cancel"
	StockOpnameExport  Code = "stock_opname.export"
	StockOpnameVerify  Code = "stock_opname.verify"
	StockOpnamePost    Code = "stock_opname.post"
	StockOpnameClose   Code = "stock_opname.close"
	StockOpnameReport  Code = "stock_opname.report"

	StorageLocationView   Code = "storage_location.view"
	StorageLocationCreate Code = "storage_location.create"
	StorageLocationUpdate Code = "storage_location.update"
	StorageLocationDelete Code = "storage_location.delete"
)

// All returns every registered permission code.
func All() []Code {
	return []Code{
		DashboardView,
		UserView, UserCreate, UserUpdate, UserDelete,
		RoleView, RoleCreate, RoleUpdate, RoleDelete,
		AuditView,
		ReportView,
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
}

var known = func() map[Code]struct{} {
	m := make(map[Code]struct{}, len(All()))
	for _, c := range All() {
		m[c] = struct{}{}
	}
	return m
}()

// Exists reports whether the code is a registered permission.
func Exists(c Code) bool {
	_, ok := known[c]
	return ok
}
