package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"retail-pos-system/internal/permissions"

	"github.com/jackc/pgx/v5"
)

// RequiredRoles are the roles a store must staff with at least one active
// account before the onboarding wizard considers it ready to serve. Order is
// presentation-only (manager first, finance last).
var RequiredRoles = []string{
	permissions.RoleManager,
	permissions.RoleSupervisor,
	permissions.RoleCashier,
	permissions.RoleInventoryStaff,
	permissions.RoleFinance,
}

// CatalogStats reports the health of the shared catalogue. The product list is
// global, so these numbers are identical across stores; they still gate
// readiness because a store cannot sell from an empty (or fully out-of-stock)
// catalogue.
type CatalogStats struct {
	ActiveProducts    int `json:"active_products"`
	ZeroStockProducts int `json:"zero_stock_products"`
}

// Readiness is the computed onboarding state of a store. It is never stored —
// it is derived on read (docs/design/store-onboarding-wizard.md) so it can
// never drift from the underlying facts.
type Readiness struct {
	StoreID          int            `json:"store_id"`
	Name             string         `json:"name"`
	IsActive         bool           `json:"is_active"`
	AddressSet       bool           `json:"address_set"`
	PhoneSet         bool           `json:"phone_set"`
	Staff            map[string]int `json:"staff"`
	RequiredRoles    []string       `json:"required_roles"`
	Catalog          CatalogStats   `json:"catalog"`
	StorageLocations int            `json:"storage_locations"`
	Ready            bool           `json:"ready"`
	Blockers         []string       `json:"blockers"`
}

type Repo interface {
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Store, int, error)
	GetByID(ctx context.Context, id int) (*Store, error)
	GetAllActive(ctx context.Context, storeID *int) ([]Store, error)
	GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error)
	GetAllWarehouses(ctx context.Context, storeID *int) ([]Warehouse, error)
	Create(ctx context.Context, s *Store) error
	Update(ctx context.Context, s *Store) error
	Delete(ctx context.Context, id int) error
	ReadinessDetails(ctx context.Context, storeID int) (*ReadinessDetails, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

// wrapRepoErr classifies repository failures: a missing row becomes
// ErrNotFound; anything else is wrapped in ErrInternal so handlers map
// database outages to 500 rather than misreporting them as 400/404.
func wrapRepoErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("%w: %w", ErrInternal, err)
}

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Store, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive, storeID)
}

func (s *Service) GetByID(ctx context.Context, id int) (*Store, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoErr(err)
	}
	return st, nil
}

func (s *Service) GetAllActive(ctx context.Context, storeID *int) ([]Store, error) {
	return s.repo.GetAllActive(ctx, storeID)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Store, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	st := &Store{
		Name:     name,
		Address:  strings.TrimSpace(req.Address),
		Phone:    strings.TrimSpace(req.Phone),
		IsActive: true,
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, wrapRepoErr(err)
	}
	created, err := s.repo.GetByID(ctx, st.ID)
	if err != nil {
		return nil, wrapRepoErr(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int, req UpdateRequest) (*Store, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoErr(err)
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		existing.Name = name
	}
	if req.Address != nil {
		existing.Address = strings.TrimSpace(*req.Address)
	}
	if req.Phone != nil {
		existing.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, wrapRepoErr(err)
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoErr(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return wrapRepoErr(err)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return wrapRepoErr(err)
	}
	return nil
}

func (s *Service) GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error) {
	return s.repo.GetWarehouseByID(ctx, id)
}

func (s *Service) GetAllWarehouses(ctx context.Context, storeID *int) ([]Warehouse, error) {
	return s.repo.GetAllWarehouses(ctx, storeID)
}

// Readiness computes whether a store has everything it needs to serve a
// customer. Blockers are stable machine codes the frontend maps to copy:
// "store.address", "store.phone", "store.inactive", "staff.<role>",
// "storage_location" and "catalog".
func (s *Service) Readiness(ctx context.Context, id int) (*Readiness, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoErr(err)
	}
	details, err := s.repo.ReadinessDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	staff := details.Staff
	if staff == nil {
		staff = map[string]int{}
	}
	for role, count := range staff {
		if count <= 0 {
			delete(staff, role)
		}
	}

	addressSet := strings.TrimSpace(st.Address) != ""
	phoneSet := strings.TrimSpace(st.Phone) != ""

	blockers := make([]string, 0)
	if !st.IsActive {
		blockers = append(blockers, "store.inactive")
	}
	if !addressSet {
		blockers = append(blockers, "store.address")
	}
	if !phoneSet {
		blockers = append(blockers, "store.phone")
	}
	for _, role := range RequiredRoles {
		if staff[role] == 0 {
			blockers = append(blockers, "staff."+role)
		}
	}
	if details.StorageLocations == 0 {
		blockers = append(blockers, "storage_location")
	}
	if details.ActiveProducts == 0 || details.ZeroStockProducts == details.ActiveProducts {
		blockers = append(blockers, "catalog")
	}

	return &Readiness{
		StoreID:       st.ID,
		Name:          st.Name,
		IsActive:      st.IsActive,
		AddressSet:    addressSet,
		PhoneSet:      phoneSet,
		Staff:         staff,
		RequiredRoles: RequiredRoles,
		Catalog: CatalogStats{
			ActiveProducts:    details.ActiveProducts,
			ZeroStockProducts: details.ZeroStockProducts,
		},
		StorageLocations: details.StorageLocations,
		Ready:            len(blockers) == 0,
		Blockers:         blockers,
	}, nil
}
