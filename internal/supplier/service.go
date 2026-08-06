package supplier

import (
	"context"
	"fmt"
	"regexp"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
)

type service struct {
	repo *Repository
}

func NewService(repo *Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetByID(ctx context.Context, id int) (*Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive)
}

func (s *service) Create(ctx context.Context, supplier *Supplier) error {
	if err := validateSupplier(supplier); err != nil {
		return err
	}
	return s.repo.Create(ctx, supplier)
}

func (s *service) Update(ctx context.Context, supplier *Supplier) error {
	if err := validateSupplier(supplier); err != nil {
		return err
	}
	return s.repo.Update(ctx, supplier)
}

func (s *service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) LinkProduct(ctx context.Context, ps *ProductSupplier) error {
	if err := validateProductSupplier(ps); err != nil {
		return err
	}
	return s.repo.LinkProduct(ctx, ps)
}

func (s *service) UnlinkProduct(ctx context.Context, productID, supplierID int) error {
	return s.repo.UnlinkProduct(ctx, productID, supplierID)
}

func (s *service) GetProductSupplier(ctx context.Context, productID, supplierID int) (*ProductSupplier, error) {
	return s.repo.GetProductSupplier(ctx, productID, supplierID)
}

func (s *service) GetPreferredSupplier(ctx context.Context, productID int) (*ProductSupplier, error) {
	return s.repo.GetPreferredSupplier(ctx, productID)
}

func (s *service) SetPreferredSupplier(ctx context.Context, productID, supplierID int) error {
	return s.repo.SetPreferredSupplier(ctx, productID, supplierID)
}

func (s *service) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	if err := validateProductSupplier(ps); err != nil {
		return err
	}
	return s.repo.UpdateProductSupplier(ctx, ps)
}

func (s *service) GetSuppliersByProductID(ctx context.Context, productID int) ([]ProductSupplier, error) {
	return s.repo.GetSuppliersByProductID(ctx, productID)
}

func (s *service) GetProductsBySupplierID(ctx context.Context, supplierID int) ([]ProductSupplier, error) {
	return s.repo.GetProductsBySupplierID(ctx, supplierID)
}

func (s *service) BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error) {
	return s.repo.BulkUpdate(ctx, ids, isActive)
}

func (s *service) BulkDelete(ctx context.Context, ids []int) (int, error) {
	return s.repo.BulkDelete(ctx, ids)
}

func validateSupplier(supplier *Supplier) error {
	if supplier.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidSupplier)
	}
	if supplier.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidSupplier)
	}
	if supplier.Email != nil && *supplier.Email != "" {
		if !emailRegex.MatchString(*supplier.Email) {
			return fmt.Errorf("%w: invalid email format", ErrInvalidSupplier)
		}
	}
	if supplier.Phone != nil && *supplier.Phone != "" {
		phone := regexp.MustCompile(`[\s\-\(\)]`).ReplaceAllString(*supplier.Phone, "")
		if !phoneRegex.MatchString(phone) {
			return fmt.Errorf("%w: invalid phone format", ErrInvalidSupplier)
		}
	}
	return nil
}

func validateProductSupplier(ps *ProductSupplier) error {
	if ps.ProductID <= 0 {
		return fmt.Errorf("%w: product_id is required", ErrInvalidSupplier)
	}
	if ps.SupplierID <= 0 {
		return fmt.Errorf("%w: supplier_id is required", ErrInvalidSupplier)
	}
	if ps.UnitCost < 0 {
		return fmt.Errorf("%w: unit_cost must be non-negative", ErrInvalidSupplier)
	}
	return nil
}
