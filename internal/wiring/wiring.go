package wiring

import (
	"context"
	"time"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/customer"
	"retail-pos-system/internal/customergroup"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/export"
	ieh "retail-pos-system/internal/platform/importexport/handler"
	"retail-pos-system/internal/platform/importexport/history"
	importer "retail-pos-system/internal/platform/importexport/import"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"
	"retail-pos-system/internal/platform/importexport/validation"
	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/purchase"
	"retail-pos-system/internal/report"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
	"retail-pos-system/internal/stockopname"
	"retail-pos-system/internal/storagelocation"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/supplier"
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
	"retail-pos-system/pkg/cache"
	"retail-pos-system/pkg/websocket"
)

type authAdapter struct {
	svc *user.AuthService
}

func (a *authAdapter) ValidateToken(tokenString string) (*websocket.Claims, error) {
	claims, err := a.svc.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &websocket.Claims{
		ID:       claims.ID,
		Role:     claims.Role,
		StoreID:  claims.StoreID,
		Username: claims.Username,
	}, nil
}

type productLookupAdapter struct {
	repo *product.Repository
}

func (a *productLookupAdapter) GetProductByID(ctx context.Context, id int) (string, string, int, *int, error) {
	p, err := a.repo.GetProductByID(ctx, id, nil)
	if err != nil {
		return "", "", 0, nil, err
	}
	return p.SKU, p.Name, p.Stock, p.StoreID, nil
}

type productPriceAdapter struct {
	repo *product.Repository
}

func (a *productPriceAdapter) GetProductPrice(ctx context.Context, productID int) (int, error) {
	return a.repo.GetProductPrice(ctx, productID)
}

type productNameLookupAdapter struct {
	repo *product.Repository
}

func (a *productNameLookupAdapter) GetProductNamesByIDs(ctx context.Context, ids []int) (map[int]purchase.ProductInfo, error) {
	products, err := a.repo.GetProductsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[int]purchase.ProductInfo, len(products))
	for _, p := range products {
		result[p.ID] = purchase.ProductInfo{Name: p.Name, SKU: p.SKU}
	}
	return result, nil
}

type supplierNameLookupAdapter struct {
	repo *supplier.Repository
}

func (a *supplierNameLookupAdapter) GetSupplierNamesByIDs(ctx context.Context, ids []int) (map[int]purchase.SupplierInfo, error) {
	suppliers, err := a.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[int]purchase.SupplierInfo, len(suppliers))
	for _, s := range suppliers {
		result[s.ID] = purchase.SupplierInfo{Name: s.Name}
	}
	return result, nil
}

func (a *supplierNameLookupAdapter) GetSupplierIDsByName(ctx context.Context, name string) ([]int, error) {
	return a.repo.GetIDsByName(ctx, name)
}

type priceResolverAdapter struct {
	resolver *pricing.Resolver
}

func (a *priceResolverAdapter) ResolveSnapshotsBatch(ctx context.Context, items []sale.ResolveItem) ([]sale.PriceSnapshot, error) {
	pricingItems := make([]pricing.ResolveItem, len(items))
	for i, it := range items {
		pricingItems[i] = pricing.ResolveItem{
			ProductID:       it.ProductID,
			Quantity:        it.Quantity,
			CustomerGroupID: it.CustomerGroupID,
			StoreID:         it.StoreID,
		}
	}
	snaps, err := a.resolver.ResolveSnapshotsBatch(ctx, pricingItems)
	if err != nil {
		return nil, err
	}
	result := make([]sale.PriceSnapshot, len(snaps))
	for i, snap := range snaps {
		result[i] = sale.PriceSnapshot{
			ProductID:     snap.ProductID,
			ProductName:   snap.ProductName,
			UnitPrice:     snap.UnitPrice,
			OriginalPrice: snap.OriginalPrice,
			Discount:      snap.Discount,
			PricingType:   sale.PricingType(snap.PricingType),
			Cost:          snap.Cost,
			TaxClassID:    snap.TaxClassID,
			TaxRate:       snap.TaxRate,
			SnapshotAt:    snap.SnapshotAt,
		}
		if snap.Rule != nil {
			result[i].Rule = &sale.PricingRule{
				ID:          snap.Rule.ID,
				Name:        snap.Rule.Name,
				PricingType: sale.PricingType(snap.Rule.PricingType),
			}
		}
	}
	return result, nil
}

type Dependencies struct {
	UserRepo            *user.Repository
	ProductRepo         *product.Repository
	PurchaseRepo        *purchase.Repository
	SaleRepo            *sale.Repository
	InventoryRepo       *inventory.Repository
	CustomerRepo        *customer.Repository
	CategoryRepo        *category.Repository
	BrandRepo           *brand.Repository
	UOMRepo             *uom.Repository
	AuditRepo           *audit.Repository
	ReportRepo          *report.Repository
	PricingRepo         *pricing.Repository
	SupplierRepo        *supplier.Repository
	CustomerGroupRepo   *customergroup.Repository
	StoreRepo           *store.Repository
	ShiftRepo           *shift.Repository
	StockOpnameRepo     *stockopname.Repository
	StorageLocationRepo *storagelocation.Repository

	UserSvc            *user.Service
	AuthSvc            *user.AuthService
	ProductSvc         *product.Service
	PurchaseSvc        *purchase.Service
	SaleSvc            *sale.Service
	InventorySvc       *inventory.Service
	CustomerSvc        *customer.Service
	CategorySvc        *category.Service
	BrandSvc           *brand.Service
	UOMSvc             *uom.Service
	AuditSvc           *audit.Service
	ReportSvc          *report.Service
	PricingSvc         *pricing.Service
	SupplierSvc        *supplier.Service
	CustomerGroupSvc   *customergroup.Service
	StoreSvc           *store.Service
	ShiftSvc           *shift.Service
	StockOpnameSvc     *stockopname.Service
	StorageLocationSvc *storagelocation.Service

	UserH            *user.Handler
	AuthH            *user.AuthHandler
	ProductH         *product.Handler
	PurchaseH        *purchase.Handler
	SaleH            *sale.Handler
	InventoryH       *inventory.Handler
	CustomerH        *customer.Handler
	CategoryH        *category.Handler
	BrandH           *brand.Handler
	UOMH             *uom.Handler
	AuditH           *audit.Handler
	ReportH          *report.Handler
	PricingH         *pricing.Handler
	SupplierH        *supplier.Handler
	CustomerGroupH   *customergroup.Handler
	StoreH           *store.Handler
	ShiftH           *shift.Handler
	StockOpnameH     *stockopname.Handler
	StorageLocationH *storagelocation.Handler

	IEH *ieh.Handler

	PricingResolver *pricing.Resolver

	Bus   *eventbus.Bus
	Hub   *websocket.Hub
	Cache *cache.Cache
}

type Providers struct {
	DB     shared.DBPool
	Config *config.Config
}

func Initialize(p Providers) *Dependencies {
	d := &Dependencies{}

	d.Cache = cache.New(10*time.Minute, 30*time.Second)

	d.Bus = eventbus.New()
	d.Bus.SetDeadLetterStore(eventbus.NewPgDeadLetterStore(p.DB))

	d.UserRepo = user.NewRepository(p.DB)
	d.UserRepo.SetCache(d.Cache)
	d.ProductRepo = product.NewRepository(p.DB)
	d.ProductRepo.SetCache(d.Cache)
	d.PurchaseRepo = purchase.NewRepository(p.DB)
	d.SaleRepo = sale.NewRepository(p.DB)
	d.InventoryRepo = inventory.NewRepository(p.DB)
	d.CustomerRepo = customer.NewRepository(p.DB)
	d.CategoryRepo = category.NewRepository(p.DB)
	d.CategoryRepo.SetCache(d.Cache)
	d.BrandRepo = brand.NewRepository(p.DB)
	d.BrandRepo.SetCache(d.Cache)
	d.UOMRepo = uom.NewRepository(p.DB)
	d.UOMRepo.SetCache(d.Cache)
	d.AuditRepo = audit.NewRepository(p.DB)
	d.ReportRepo = report.NewRepository(p.DB)
	d.ReportRepo.SetCache(d.Cache)
	d.PricingRepo = pricing.NewRepository(p.DB)
	d.SupplierRepo = supplier.NewRepository(p.DB)
	d.CustomerGroupRepo = customergroup.NewRepository(p.DB)
	d.StoreRepo = store.NewRepository(p.DB)
	d.ShiftRepo = shift.NewRepository(p.DB)
	d.StockOpnameRepo = stockopname.NewRepository(p.DB)
	d.StorageLocationRepo = storagelocation.NewRepository(p.DB)

	d.AuditSvc = audit.NewService(d.AuditRepo)
	d.UserSvc = user.NewService(d.UserRepo)
	d.AuthSvc = user.NewAuthService(d.UserRepo, d.AuditSvc, p.Config)
	d.ProductSvc = product.NewService(d.ProductRepo, d.CategoryRepo, d.BrandRepo, d.UOMRepo, d.Bus)
	d.PurchaseSvc = purchase.NewService(d.PurchaseRepo, d.Bus)
	d.PurchaseSvc.SetProductLookup(&productNameLookupAdapter{repo: d.ProductRepo})
	d.PurchaseSvc.SetSupplierLookup(&supplierNameLookupAdapter{repo: d.SupplierRepo})
	d.SaleSvc = sale.NewService(d.SaleRepo, d.Bus)
	d.SaleSvc.SetCartConfig(sale.CartConfig{HoldTTLHours: p.Config.CartHoldTTLHours})

	priceStore := &productPriceAdapter{repo: d.ProductRepo}
	d.SaleSvc.SetPriceStore(priceStore)
	d.PricingResolver = pricing.NewResolver(d.PricingRepo)
	d.SaleSvc.SetPriceResolver(&priceResolverAdapter{resolver: d.PricingResolver})

	d.InventorySvc = inventory.NewService(d.InventoryRepo, d.Bus)
	d.CustomerSvc = customer.NewService(d.CustomerRepo)
	d.CategorySvc = category.NewService(d.CategoryRepo)
	d.BrandSvc = brand.NewService(d.BrandRepo)
	d.UOMSvc = uom.NewService(d.UOMRepo)
	d.ReportSvc = report.NewService(d.ReportRepo, d.Bus)
	d.PricingSvc = pricing.NewService(d.PricingRepo)
	d.SupplierSvc = supplier.NewService(d.SupplierRepo)
	d.CustomerGroupSvc = customergroup.NewService(d.CustomerGroupRepo)
	d.StoreSvc = store.NewService(d.StoreRepo)
	d.ShiftSvc = shift.NewService(d.ShiftRepo)
	d.StockOpnameSvc = stockopname.NewService(d.StockOpnameRepo, d.Bus)
	d.StorageLocationSvc = storagelocation.NewService(d.StorageLocationRepo)

	d.UserH = user.NewHandler(d.UserSvc, d.AuditSvc)
	d.AuthH = user.NewAuthHandler(d.AuthSvc, d.AuditSvc)
	d.ProductH = product.NewHandler(d.ProductSvc, d.AuditSvc)
	d.PurchaseH = purchase.NewHandler(d.PurchaseSvc, d.AuditSvc)
	d.SaleH = sale.NewHandler(d.SaleSvc, d.AuditSvc)
	d.InventoryH = inventory.NewHandler(d.InventorySvc, d.AuditSvc)
	d.CustomerH = customer.NewHandler(d.CustomerSvc, d.AuditSvc)
	d.CategoryH = category.NewHandler(d.CategorySvc, d.AuditSvc)
	d.BrandH = brand.NewHandler(d.BrandSvc, d.AuditSvc)
	d.UOMH = uom.NewHandler(d.UOMSvc, d.AuditSvc)
	d.AuditH = audit.NewHandler(d.AuditSvc)
	d.ReportH = report.NewHandler(d.ReportSvc)
	d.PricingH = pricing.NewHandler(d.PricingSvc, d.PricingResolver, d.AuditSvc)
	d.PricingH.SetProductSearcher(d.PricingRepo)
	d.SupplierH = supplier.NewHandler(d.SupplierSvc, d.AuditSvc)
	d.CustomerGroupH = customergroup.NewHandler(d.CustomerGroupSvc, d.AuditSvc)
	d.StoreH = store.NewHandler(d.StoreSvc, d.AuditSvc)
	d.ShiftH = shift.NewHandler(d.ShiftSvc, d.AuditSvc)
	d.StockOpnameH = stockopname.NewHandler(d.StockOpnameSvc, d.AuditSvc)
	d.StorageLocationH = storagelocation.NewHandler(d.StorageLocationSvc, d.AuditSvc)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(category.Schema)
	_ = schemaReg.Register(brand.Schema)
	_ = schemaReg.Register(uom.Schema)
	_ = schemaReg.Register(customer.Schema)
	_ = schemaReg.Register(product.Schema)
	_ = schemaReg.Register(store.Schema)
	_ = schemaReg.Register(customergroup.Schema)
	_ = schemaReg.Register(pricing.Schema)
	_ = schemaReg.Register(supplier.Schema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(category.NewAdapter(d.CategoryRepo))
	_ = adapterReg.Register(brand.NewAdapter(d.BrandRepo))
	_ = adapterReg.Register(uom.NewAdapter(d.UOMRepo))
	_ = adapterReg.Register(customer.NewAdapter(d.CustomerRepo))
	_ = adapterReg.Register(product.NewAdapter(d.ProductRepo, d.CategoryRepo, d.BrandRepo, d.UOMRepo))
	_ = adapterReg.Register(store.NewAdapter(d.StoreRepo))
	_ = adapterReg.Register(customergroup.NewAdapter(d.CustomerGroupRepo))
	_ = adapterReg.Register(pricing.NewAdapter(d.PricingRepo))
	_ = adapterReg.Register(supplier.NewAdapter(d.SupplierRepo))

	valPipeline := validation.NewDefaultPipeline()
	progStore := progress.NewPgRepository(p.DB)
	progEng := progress.NewEngine(progStore)
	historyStore := history.NewStore(p.DB)
	importEng := importer.NewEngine(schemaReg, valPipeline, adapterReg, progEng, historyStore)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()
	d.IEH = ieh.NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, historyStore)

	authAdpt := &authAdapter{svc: d.AuthSvc}
	d.Hub = websocket.NewHub(authAdpt)

	wsProductLookup := &productLookupAdapter{repo: d.ProductRepo}
	d.Bus.Subscribe(websocket.NewSaleCreatedListener(d.Hub))
	d.Bus.Subscribe(websocket.NewProductUpdatedListener(d.Hub))
	d.Bus.Subscribe(websocket.NewPOReceivedListener(d.Hub))
	d.Bus.Subscribe(websocket.NewPOCreatedListener(d.Hub))
	d.Bus.Subscribe(websocket.NewPOConfirmedListener(d.Hub))
	d.Bus.Subscribe(websocket.NewPOCancelledListener(d.Hub))
	d.Bus.Subscribe(websocket.NewStockOpnameStatusListener(d.Hub))
	d.Bus.Subscribe(websocket.NewStockAdjustedListener(d.Hub, wsProductLookup))
	d.Bus.Subscribe(d.ReportRepo.NewSaleCreatedListener())
	d.Bus.Subscribe(inventory.NewPurchaseReceiptListener(d.InventoryRepo, d.InventorySvc))

	return d
}
