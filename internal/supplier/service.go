package supplier

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
)

type Repo interface {
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, isConsignment *bool) ([]Supplier, int, error)
	GetByID(ctx context.Context, id int) (*Supplier, error)
	GetByCode(ctx context.Context, code string) (*Supplier, error)
	GetNextSupplierCode(ctx context.Context) (string, error)
	Create(ctx context.Context, supplier *Supplier) error
	Update(ctx context.Context, supplier *Supplier) error
	Delete(ctx context.Context, id int) error
	BulkUpdate(ctx context.Context, ids []int, isActive bool, updatedBy *int) (int, error)
	BulkDelete(ctx context.Context, ids []int) (int, error)
	CountUsage(ctx context.Context, supplierID int) (SupplierUsage, error)
	GetPreferredSupplier(ctx context.Context, productID int, storeID *int) (*ProductSupplier, error)
	GetProductsBySupplierID(ctx context.Context, supplierID int, storeID *int) ([]ProductSupplier, error)
	GetProductSupplier(ctx context.Context, productID, supplierID int, storeID *int) (*ProductSupplier, error)
	GetSuppliersByProductID(ctx context.Context, productID int, storeID *int) ([]ProductSupplier, error)
	LinkProduct(ctx context.Context, ps *ProductSupplier) error
	UnlinkProduct(ctx context.Context, productID, supplierID int, storeID *int) error
	SetPreferredSupplier(ctx context.Context, productID, supplierID int, storeID *int) error
	UpdateProductSupplier(ctx context.Context, ps *ProductSupplier, storeID *int) error
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetByID(ctx context.Context, id int) (*Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, isConsignment *bool) ([]Supplier, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive, isConsignment)
}

func (s *service) Create(ctx context.Context, supplier *Supplier) error {
	if supplier.Code == "" {
		code, err := s.repo.GetNextSupplierCode(ctx)
		if err != nil {
			return err
		}
		supplier.Code = code
	}
	if err := validateSupplier(supplier); err != nil {
		return err
	}
	return s.repo.Create(ctx, supplier)
}

func (s *service) Update(ctx context.Context, supplier *Supplier) error {
	if err := validateSupplier(supplier); err != nil {
		return err
	}
	// Deactivation is the destructive half of an update: it hides the supplier
	// from every picker while in-flight orders still name it. Only a true
	// active -> inactive transition is guarded, so re-saving an already-inactive
	// supplier is not blocked by references that predate it.
	old, err := s.repo.GetByID(ctx, supplier.ID)
	if err != nil {
		return err
	}
	if old.IsActive && !supplier.IsActive {
		usage, err := s.repo.CountUsage(ctx, supplier.ID)
		if err != nil {
			return err
		}
		if usage.InFlight() > 0 {
			return &SupplierInUseError{Usage: usage, Deactivating: true}
		}
	}
	return s.repo.Update(ctx, supplier)
}

// GetUsage returns the cross-module reference breakdown for a supplier so the
// UI can warn about the blast radius before a delete or deactivate. It is a
// read: a missing supplier is a 404, not a zero breakdown.
func (s *service) GetUsage(ctx context.Context, id int) (SupplierUsage, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return SupplierUsage{}, err
	}
	return s.repo.CountUsage(ctx, id)
}

func (s *service) Delete(ctx context.Context, id int) error {
	// Confirm the supplier exists first: CountUsage reports zeros for an unknown
	// id, which would otherwise turn a delete of a missing supplier into a 200.
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	usage, err := s.repo.CountUsage(ctx, id)
	if err != nil {
		return err
	}
	if usage.Total() > 0 {
		return &SupplierInUseError{Usage: usage}
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) LinkProduct(ctx context.Context, ps *ProductSupplier) error {
	if err := validateProductSupplier(ps); err != nil {
		return err
	}
	return s.repo.LinkProduct(ctx, ps)
}

func (s *service) UnlinkProduct(ctx context.Context, productID, supplierID int, storeID *int) error {
	return s.repo.UnlinkProduct(ctx, productID, supplierID, storeID)
}

func (s *service) GetProductSupplier(ctx context.Context, productID, supplierID int, storeID *int) (*ProductSupplier, error) {
	return s.repo.GetProductSupplier(ctx, productID, supplierID, storeID)
}

func (s *service) GetPreferredSupplier(ctx context.Context, productID int, storeID *int) (*ProductSupplier, error) {
	return s.repo.GetPreferredSupplier(ctx, productID, storeID)
}

func (s *service) SetPreferredSupplier(ctx context.Context, productID, supplierID int, storeID *int) error {
	return s.repo.SetPreferredSupplier(ctx, productID, supplierID, storeID)
}

func (s *service) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier, storeID *int) error {
	if err := validateProductSupplier(ps); err != nil {
		return err
	}
	return s.repo.UpdateProductSupplier(ctx, ps, storeID)
}

func (s *service) GetSuppliersByProductID(ctx context.Context, productID int, storeID *int) ([]ProductSupplier, error) {
	return s.repo.GetSuppliersByProductID(ctx, productID, storeID)
}

func (s *service) GetProductsBySupplierID(ctx context.Context, supplierID int, storeID *int) ([]ProductSupplier, error) {
	return s.repo.GetProductsBySupplierID(ctx, supplierID, storeID)
}

func (s *service) BulkUpdate(ctx context.Context, ids []int, isActive bool, updatedBy *int) (int, error) {
	if !isActive {
		blocked, usage, err := s.blockedForDeactivation(ctx, ids)
		if err != nil {
			return 0, err
		}
		if len(blocked) > 0 {
			return 0, &SupplierInUseError{Usage: usage, Deactivating: true, BlockedIDs: blocked}
		}
	}
	return s.repo.BulkUpdate(ctx, ids, isActive, updatedBy)
}

// blockedForDeactivation returns the ids in the batch that are currently active
// yet still have in-flight work, plus their aggregated usage. A supplier already
// inactive is skipped: deactivating it again changes nothing, so its historical
// references must not fail the batch.
func (s *service) blockedForDeactivation(ctx context.Context, ids []int) ([]int, SupplierUsage, error) {
	var blocked []int
	var agg SupplierUsage
	for _, id := range ids {
		old, err := s.repo.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrSupplierNotFound) {
				continue
			}
			return nil, SupplierUsage{}, err
		}
		if !old.IsActive {
			continue
		}
		usage, err := s.repo.CountUsage(ctx, id)
		if err != nil {
			return nil, SupplierUsage{}, err
		}
		if usage.InFlight() > 0 {
			blocked = append(blocked, id)
			agg.OpenPurchaseOrders += usage.OpenPurchaseOrders
			agg.ActiveConsignments += usage.ActiveConsignments
		}
	}
	return blocked, agg, nil
}

func (s *service) BulkDelete(ctx context.Context, ids []int) (int, error) {
	blocked := make([]int, 0, len(ids))
	var agg SupplierUsage
	for _, id := range ids {
		usage, err := s.repo.CountUsage(ctx, id)
		if err != nil {
			return 0, err
		}
		if usage.Total() > 0 {
			blocked = append(blocked, id)
			agg.ProductLinks += usage.ProductLinks
			agg.OpenPurchaseOrders += usage.OpenPurchaseOrders
			agg.ActiveConsignments += usage.ActiveConsignments
		}
	}
	if len(blocked) > 0 {
		return 0, &SupplierInUseError{Usage: agg, BlockedIDs: blocked}
	}
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
