package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Repo interface {
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Store, int, error)
	GetByID(ctx context.Context, id int) (*Store, error)
	GetAllActive(ctx context.Context, storeID *int) ([]Store, error)
	GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error)
	GetAllWarehouses(ctx context.Context, storeID *int) ([]Warehouse, error)
	Create(ctx context.Context, s *Store) error
	Update(ctx context.Context, s *Store) error
	Delete(ctx context.Context, id int) error
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
