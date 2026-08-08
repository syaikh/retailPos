package store

import (
	"context"
	"fmt"
	"strings"
)

type Repo interface {
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Store, int, error)
	GetByID(ctx context.Context, id int) (*Store, error)
	GetAllActive(ctx context.Context) ([]Store, error)
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

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Store, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive)
}

func (s *Service) GetByID(ctx context.Context, id int) (*Store, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAllActive(ctx context.Context) ([]Store, error) {
	return s.repo.GetAllActive(ctx)
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
		return nil, err
	}
	return s.repo.GetByID(ctx, st.ID)
}

func (s *Service) Update(ctx context.Context, id int, req UpdateRequest) (*Store, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("store not found")
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
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("store not found")
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error) {
	return s.repo.GetWarehouseByID(ctx, id)
}

func (s *Service) GetAllWarehouses(ctx context.Context) ([]Warehouse, error) {
	return s.repo.GetAllWarehouses(ctx, nil)
}
