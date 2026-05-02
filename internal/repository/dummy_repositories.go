package repository

import (
	"context"
	"retail-pos-system/internal/domain"
)

// Dummy implementations hanya agar server bisa compile & jalan tanpa DB di development.
// Nanti akan diganti dengan implementasi real (postgres_repository.go) saat ada koneksi DB.

type DummyUserRepo struct{}
func (d *DummyUserRepo) GetByID(id int) (*domain.User, error) { return &domain.User{ID: 1, Username:"admin", RoleID:1}, nil }
func (d *DummyUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) { return &domain.User{ID:1, Username:username, RoleID:1}, nil }
func (d *DummyUserRepo) GetAllUsers(ctx context.Context, limit, offset int, search string) ([]domain.User, int, error) { return []domain.User{{ID:1,Username:"admin"}}, 1, nil }
func (d *DummyUserRepo) CreateUser(ctx context.Context, user *domain.User) error { return nil }
func (d *DummyUserRepo) UpdateUser(ctx context.Context, user *domain.User) error { return nil }
func (d *DummyUserRepo) DeleteUser(ctx context.Context, id int) error { return nil }

type DummyRoleRepo struct{}
func (d *DummyRoleRepo) GetRoleByID(ctx context.Context, id int) (*domain.Role, error) { return &domain.Role{ID:1, Name:"admin"}, nil }
func (d *DummyRoleRepo) GetAllRoles(ctx context.Context) ([]domain.Role, error) { return []domain.Role{{ID:1, Name:"admin"}}, nil }
func (d *DummyRoleRepo) GetRolePermissions(ctx context.Context, roleID int) ([]domain.Permission, error) { return []domain.Permission{{ID:1, Code:"product.view"}}, nil }
func (d *DummyRoleRepo) GetAllPermissions(ctx context.Context) ([]domain.Permission, error) { return []domain.Permission{{ID:1, Code:"product.view"}}, nil }
func (d *DummyRoleRepo) UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error { return nil }
func (d *DummyRoleRepo) CreateRole(ctx context.Context, role *domain.Role) error { return nil }
func (d *DummyRoleRepo) UpdateRole(ctx context.Context, role *domain.Role) error { return nil }
func (d *DummyRoleRepo) DeleteRole(ctx context.Context, id int) error { return nil }

type DummyProductRepo struct{}
func (d *DummyProductRepo) GetProductByID(ctx context.Context, id int, storeID *int) (*domain.Product, error) { return &domain.Product{ID:id, SKU:"D001", Name:"Dummy"}, nil }
func (d *DummyProductRepo) GetProductBySKU(ctx context.Context, sku string, storeID *int) (*domain.Product, error) { return &domain.Product{SKU:sku, Name:"Dummy"}, nil }
func (d *DummyProductRepo) GetAllProducts(ctx context.Context, limit, offset int, search string, categoryID *int, sortBy, sortDir string, maxStock *int, storeID *int) ([]domain.Product, int, error) { return []domain.Product{{ID:1,SKU:"D001"}}, 1, nil }
func (d *DummyProductRepo) CreateProduct(ctx context.Context, product *domain.Product) error { return nil }
func (d *DummyProductRepo) UpdateProduct(ctx context.Context, product *domain.Product, storeID *int) error { return nil }
func (d *DummyProductRepo) DeleteProduct(ctx context.Context, id int, storeID *int) error { return nil }

type DummySaleRepo struct{}
func (d *DummySaleRepo) CreateSale(ctx context.Context, tx interface{}, sale *domain.Sale, items []domain.SaleItem) error { return nil }
func (d *DummySaleRepo) GetSaleByID(ctx context.Context, id int) (*domain.Sale, error) { return &domain.Sale{ID:id}, nil }
func (d *DummySaleRepo) GetAllSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate string, storeID *int) ([]domain.Sale, int, error) { return []domain.Sale{{ID:1}}, 1, nil }
func (d *DummySaleRepo) BeginTx(ctx context.Context) (interface{}, error) { return nil, nil } // dummy tx

type DummyPermissionRepo struct{}
func (d *DummyPermissionRepo) GetByRoleID(ctx context.Context, roleID int) ([]domain.Permission, error) { return []domain.Permission{{ID:1,Code:"product.view"}}, nil }

type DummyAuditLogRepo struct{}
func (d *DummyAuditLogRepo) Create(ctx context.Context, log *domain.AuditLog) error { return nil }
func (d *DummyAuditLogRepo) GetAll(ctx context.Context, limit, offset int, userID *int) ([]domain.AuditLog, int, error) { return []domain.AuditLog{}, 0, nil }

type DummyInventoryRepo struct{}
func (d *DummyInventoryRepo) GetAll(ctx context.Context, limit, offset int, search string, categoryID *int, storeID *int) ([]domain.Product, int, error) { return []domain.Product{}, 0, nil }
