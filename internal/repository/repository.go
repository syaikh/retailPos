package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"retail-pos-system/internal/domain"
)

type UserRepository interface {
	GetByID(id int) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetAllUsers(ctx context.Context, limit, offset int, search string) ([]domain.User, int, error)
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, user *domain.User) error
	DeleteUser(ctx context.Context, id int) error
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

type ProductRepository interface {
	GetProductByID(ctx context.Context, id int, storeID *int) (*domain.Product, error)
	GetProductBySKU(ctx context.Context, sku string, storeID *int) (*domain.Product, error)
	GetAllProducts(ctx context.Context, limit, offset int, search string, categoryID *int, sortBy, sortDir string, maxStock *int, storeID *int) ([]domain.Product, int, error)
	CreateProduct(ctx context.Context, product *domain.Product) error
	UpdateProduct(ctx context.Context, product *domain.Product, storeID *int) error
	DeleteProduct(ctx context.Context, id int, storeID *int) error
}

type SaleRepository interface {
	CreateSale(ctx context.Context, tx pgx.Tx, sale *domain.Sale, items []domain.SaleItem) error
	GetSaleByID(ctx context.Context, id int) (*domain.Sale, error)
	GetAllSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate string, storeID *int) ([]domain.Sale, int, error)
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

type PermissionRepository interface {
	GetByRoleID(ctx context.Context, roleID int) ([]domain.Permission, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	GetAll(ctx context.Context, limit, offset int, userID *int) ([]domain.AuditLog, int, error)
}

type InventoryRepository interface {
	GetAll(ctx context.Context, limit, offset int, search string, categoryID *int, storeID *int) ([]domain.Product, int, error)
}
