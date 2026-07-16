package supplier

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id int) (*Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive)
}

func (s *Service) Create(ctx context.Context, supplier *Supplier) error {
	if err := validateSupplier(supplier); err != nil {
		return err
	}
	return s.repo.Create(ctx, supplier)
}

func (s *Service) Update(ctx context.Context, supplier *Supplier) error {
	if err := validateSupplier(supplier); err != nil {
		return err
	}
	return s.repo.Update(ctx, supplier)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) LinkProduct(ctx context.Context, ps *ProductSupplier) error {
	if err := validateProductSupplier(ps); err != nil {
		return err
	}
	return s.repo.LinkProduct(ctx, ps)
}

func (s *Service) UnlinkProduct(ctx context.Context, productID, supplierID int) error {
	return s.repo.UnlinkProduct(ctx, productID, supplierID)
}

func (s *Service) GetProductSupplier(ctx context.Context, productID, supplierID int) (*ProductSupplier, error) {
	return s.repo.GetProductSupplier(ctx, productID, supplierID)
}

func (s *Service) GetPreferredSupplier(ctx context.Context, productID int) (*ProductSupplier, error) {
	return s.repo.GetPreferredSupplier(ctx, productID)
}

func (s *Service) SetPreferredSupplier(ctx context.Context, productID, supplierID int) error {
	return s.repo.SetPreferredSupplier(ctx, productID, supplierID)
}

func (s *Service) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	if err := validateProductSupplier(ps); err != nil {
		return err
	}
	return s.repo.UpdateProductSupplier(ctx, ps)
}

func (s *Service) GetSuppliersByProductID(ctx context.Context, productID int) ([]ProductSupplier, error) {
	return s.repo.GetSuppliersByProductID(ctx, productID)
}

func (s *Service) GetProductsBySupplierID(ctx context.Context, supplierID int) ([]ProductSupplier, error) {
	return s.repo.GetProductsBySupplierID(ctx, supplierID)
}

func validateSupplier(supplier *Supplier) error {
	if supplier.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidSupplier)
	}
	if supplier.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidSupplier)
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
