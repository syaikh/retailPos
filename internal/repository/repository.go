package repository

import (
	"context"
	"time"

	"retail-pos-system/internal/domain"

	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	GetByID(id int) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetAllUsers(ctx context.Context, limit, offset int, search string, sortBy string, sortDir string, roleID int, isActive *bool) ([]domain.User, int, error)
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, user *domain.User) error
	DeleteUser(ctx context.Context, id int) error
	UpdateLastLogin(ctx context.Context, userID int) error
}

type RoleRepository interface {
	GetRoleByID(ctx context.Context, id int) (*domain.Role, error)
	GetAllRoles(ctx context.Context) ([]domain.Role, error)
	GetRolePermissions(ctx context.Context, roleID int) ([]domain.Permission, error)
	GetAllPermissions(ctx context.Context) ([]domain.Permission, error)
	UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error
	CreateRole(ctx context.Context, role *domain.Role) error
	UpdateRole(ctx context.Context, role *domain.Role) error
	DeleteRole(ctx context.Context, id int) error
}

type CategoryRepository interface {
	GetAllCategories(ctx context.Context, limit, offset int, search string) ([]domain.Category, int, error)
	GetCategoryByID(ctx context.Context, id int) (*domain.Category, error)
	CreateCategory(ctx context.Context, category *domain.Category) error
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id int) error
	HasActiveProducts(ctx context.Context, categoryID int) (bool, error)
	SlugExists(ctx context.Context, slug string, excludeID int) (bool, error)
}

type ProductStockRepository interface {
	GetStock(ctx context.Context, productID int) (int, error)
	GetStockByProductID(ctx context.Context, productID int) (*domain.ProductStock, error)
	GetAllStock(ctx context.Context, limit, offset int, search string, storeID *int) ([]domain.ProductStock, int, error)
	AdjustStock(ctx context.Context, productID int, quantityChange int, userID *int, notes string) error
	UpsertStock(ctx context.Context, productID int, storeID *int, quantity int) error
}

type ProductRepository interface {
	GetProductByID(ctx context.Context, id int, storeID *int) (*domain.Product, error)
	GetProductBySKU(ctx context.Context, sku string, storeID *int) (*domain.Product, error)
	GetDeletedProductByBarcode(ctx context.Context, barcode string, storeID *int) (*domain.Product, error)
	CreateProduct(ctx context.Context, product *domain.Product) error
	UpdateProduct(ctx context.Context, product *domain.Product, storeID *int) error
	RestoreProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, id int, storeID *int) error
	GetAllProducts(ctx context.Context, limit, offset int, search string, categoryIDs []int, sortBy, sortDir string, maxStock *int, storeID *int, status string) ([]domain.Product, int, error)
	GetNextSKU(ctx context.Context) (string, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)
	GetCategoryIDByName(ctx context.Context, name string) (int, error)
	AdjustStock(ctx context.Context, productID int, quantityChange int, userID *int, notes string) error
	GetStockByProductID(ctx context.Context, productID int) (*domain.ProductStock, error)
	
	// Brand operations
	GetBrandByID(ctx context.Context, id int) (*domain.Brand, error)
	GetAllBrands(ctx context.Context) ([]domain.Brand, error)
	CreateBrand(ctx context.Context, brand *domain.Brand) error
	UpdateBrand(ctx context.Context, brand *domain.Brand) error
	DeleteBrand(ctx context.Context, id int) error
	GetBrandIDByName(ctx context.Context, name string) (int, error)
	
	// Tax class operations
	GetTaxClassByID(ctx context.Context, id int) (*domain.TaxClass, error)
	GetAllTaxClasses(ctx context.Context) ([]domain.TaxClass, error)
	GetTaxClassIDByName(ctx context.Context, name string) (int, error)
	
	// Unit of measure operations
	GetUnitOfMeasureByID(ctx context.Context, id int) (*domain.UnitOfMeasure, error)
	GetAllUnitsOfMeasure(ctx context.Context) ([]domain.UnitOfMeasure, error)
	GetUnitOfMeasureIDByCode(ctx context.Context, code string) (int, error)
	
	// Warehouse operations
	GetWarehouseByID(ctx context.Context, id int) (*domain.Warehouse, error)
	GetAllWarehouses(ctx context.Context, storeID *int) ([]domain.Warehouse, error)
}

type SaleRepository interface {
	CreateSale(ctx context.Context, tx pgx.Tx, sale *domain.Sale, items []domain.SaleItem) error
	GetSaleByID(ctx context.Context, id int) (*domain.Sale, error)
	GetAllSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate string, storeID *int) ([]domain.Sale, int, error)
	GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*domain.PeriodComparison, error)
	GetAvailableYears(ctx context.Context, storeID *int) ([]int, error)
	GetNextInvoiceNumber(ctx context.Context) (string, error)
	BeginTx(ctx context.Context) (pgx.Tx, error)
	GetLiveDashboardStats(ctx context.Context, storeID *int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error)
}

type PaymentMethodRepository interface {
	GetAllActive(ctx context.Context) ([]domain.PaymentMethod, error)
	GetPaymentMethodByCode(ctx context.Context, code string) (*domain.PaymentMethod, error)
	GetPaymentMethodByID(ctx context.Context, id int) (*domain.PaymentMethod, error)
}

type PermissionRepository interface {
	GetByRoleID(ctx context.Context, roleID int) ([]domain.Permission, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	GetAll(ctx context.Context, limit, offset int, userID *int, search string, action string, entityType string, startDate *time.Time, endDate *time.Time) ([]domain.AuditLog, int, error)
}

type InventoryRepository interface {
	GetAll(ctx context.Context, limit, offset int, search string, categoryID *int, storeID *int) ([]domain.Product, int, error)
}

type CustomerRepository interface {
	GetCustomerByID(ctx context.Context, id int) (*domain.Customer, error)
	GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool) ([]domain.Customer, int, error)
	CreateCustomer(ctx context.Context, customer *domain.Customer) error
	UpdateCustomer(ctx context.Context, customer *domain.Customer, id int) error
	DeleteCustomer(ctx context.Context, id int) error
	GetByPhone(ctx context.Context, phone string) (*domain.Customer, error)
}
